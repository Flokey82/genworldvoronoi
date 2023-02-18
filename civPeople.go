package genworldvoronoi

import (
	"log"
	"math/rand"
	"sort"

	"github.com/Flokey82/go_gens/utils"
)

// tickPeople ticks the given people for the given number of days and returns
// the list of alive people (including newborns).
//
// NOTE: 'cf' is a function that returns a culture for a given location.
// We use this function to determine the culture of a newborn based on the region
// it is born in.
func (m *Civ) tickPeople(people []*Person, nDays int, cf func(int) *Culture) []*Person {
	alive := make([]*Person, 0, len(people))
	dead := make([]*Person, 0, len(people))
	for _, p := range people {
		// Tick, and check if we had a child.
		if child := m.tickPerson(p, nDays, cf); child != nil {
			alive = append(alive, child)
		}

		// Check if we are still alive.
		if !p.Death.IsSet() {
			alive = append(alive, p)
		} else {
			dead = append(dead, p)
		}
	}

	// Pair up single people.
	m.matchMaker(alive)

	// Log some stats.
	m.LogPopulationStats(alive)
	log.Println("dead people")
	m.LogPopulationStats(dead)
	log.Println("dropped people", len(dead), "alive", len(alive), "added", len(alive)-len(people))
	return alive
}

func (m *Civ) placePopulationAt(r, n int, cf func(int) *Culture) []*Person {
	// Get the culture at the region.
	culture := cf(r)

	// Generate a number of people and match them up with each other.
	localPop := make([]*Person, 0, n)
	for i := 0; i < n; i++ {
		localPop = append(localPop, m.newRandomPersonAt(r, culture))
	}

	// Match up people.
	m.matchMaker(localPop)

	// Add people to world.
	m.People = append(m.People, localPop...)
	return localPop
}

func (m *Civ) killNPeopleAt(r, n int) {
	peopleAt := make([]*Person, 0, n)
	for _, p := range m.People {
		if p.Region == r {
			peopleAt = append(peopleAt, p)
		}
	}
	m.killNPeople(peopleAt, n)
}

func (m *Civ) killNPeople(people []*Person, n int) {
	var killed int
	var p *Person
	for _, i := range rand.Perm(len(people)) {
		if killed >= n {
			break
		}
		p = people[i]
		if !p.isDead() {
			m.killPerson(p)
			killed++
		}
	}
}

func (m *Civ) killNPeople2(people []*Person, n int) []*Person {
	var killed int
	alive := make([]*Person, 0, len(people))
	for _, i := range rand.Perm(len(people)) {
		p := people[i]
		if killed >= n {
			alive = append(alive, p)
			continue
		}
		if !p.isDead() {
			m.killPerson(p)
			killed++
		}
	}
	return alive
}

func (m *Civ) migrateNPeopleFromTo(rFrom, rTo, n int) {
	var migrated int
	var p *Person
	for _, i := range rand.Perm(len(m.People)) {
		if migrated >= n {
			break
		}
		p = m.People[i]
		// TODO: Also migrate spouses and children.
		if p.Region == rFrom && !p.isDead() {
			p.Region = rTo
			migrated++
		}
	}
}

func (m *Civ) LogPopulationStats(people []*Person) {
	// Gather statistics.
	// We gather the age of all people in the village in buckets.
	// The buckets are 10 years wide.
	var ageBuckets [40]int
	for _, p := range people {
		ageBuckets[p.Age/10]++
	}

	// Print the statistics.
	log.Println("Population stats:")
	for i, n := range ageBuckets {
		if n != 0 {
			log.Println("Age", i*10, "-", i*10+9, ":", n)
		}
	}
}

func (m *Civ) matchMaker(people []*Person) {
	// Get eligible singles (not dead, not married, and already alive).
	single := make([]*Person, 0, len(people))
	for _, p := range people {
		if m.doesPersonExist(p) && p.isEligibleSingle() {
			single = append(single, p)
		}
	}

	// Sort by age, so similar age people are more likely to be paired up quicker.
	sort.Slice(single, func(a, b int) bool {
		return single[a].Age > single[b].Age
	})

	// Pair up singles.
	for i, p := range single {
		if !p.isEligibleSingle() {
			continue // Not single anymore.
		}

		// We can skip all people up to the current person, since they have already been matched.
		for j := i + 1; j < len(single); j++ {
			pc := single[j]
			if p.Region != pc.Region || !pc.isEligibleSingle() || // Not single anymore or not in same region.
				p.Gender() == pc.Gender() || isRelated(p, pc) || // TODO: Allow same sex couples (which can adopt children/orphans).
				utils.Abs(p.Age-pc.Age) > utils.Min(p.Age, pc.Age)/3 { // At most 33% age difference.
				continue
			}
			p.Spouse = pc
			pc.Spouse = p

			// Update family name.
			//
			// TODO: This is not optimal... There should be a better way to do this.
			// The culture should determine any changes to the name.
			if p.Gender() == GenderFemale {
				p.LastName = pc.LastName
			} else {
				pc.LastName = p.LastName
			}
			// log.Println(p.Name(), "and", pc.Name(), "are in love")

			// We found a match, so we can break out of the inner loop.
			break
		}
	}
}
