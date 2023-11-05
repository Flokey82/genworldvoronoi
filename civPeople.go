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

// placePopulationAt places a population of n randomly generated people at the given region.
// The people are assigned a culture based on the given culture function.
//
// NOTE:
// - 'cf' is a function that returns a culture for a given location.
// We use this function to determine the culture of a newborn based on the region
// it is born in.
// - This function does not assign the people to any city that might be at the region.
//
// TODO: Assign people to cities!
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

// killNPeople kills n people from the given list of people and returns the list of
// people that are still alive.
//
// NOTE: This function filters out dead people and does not kill people that are already dead.
func (m *Civ) killNPeople(people []*Person, n int, reason string) []*Person {
	var killed int
	alive := make([]*Person, 0, len(people))
	for _, i := range rand.Perm(len(people)) {
		p := people[i]
		if !p.isDead() {
			if killed >= n {
				alive = append(alive, p)
			} else {
				m.killPerson(p, reason)
				killed++
			}
		}
	}
	return alive
}

// migrateNPeopleFromTo migrates n people from the given list of people from / to the given region.
// It returns the list of people that are still alive in the original region and the list of people
// that were migrated to the new region.
//
// NOTE:
// - This function filters out dead people.
// - This function does not assign the people to any city that might be at the region.
//
// TODO:
// - Assign people to / remove people from cities!
func (m *Civ) migrateNPeopleFromTo(pFrom []*Person, rTo, n int) (pFromAfter, pToMigrate []*Person) {
	pFromAfter = make([]*Person, 0, len(pFrom))
	pToMigrate = make([]*Person, 0, n)
	var migrated int
	for _, i := range rand.Perm(len(m.People)) {
		p := m.People[i]
		if !p.isDead() {
			if migrated >= n {
				pFromAfter = append(pFromAfter, p)
			} else {
				// TODO: Also migrate spouses and children.
				migrated++
				pToMigrate = append(pToMigrate, p)
				p.Region = rTo
			}
		}
	}
	return
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

// matchMaker matches up single people based on their age and region.
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
