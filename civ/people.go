package civ

import (
	"log"
	"math"
	"math/rand"
	"sort"

	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/go_gens/utils"
)

// tickPeople ticks the given people for the given number of days and returns
// the list of alive people (including newborns).
//
// NOTE: 'cf' is a function that returns a culture for a given location.
// We use this function to determine the culture of a newborn based on the region
// it is born in.
func (m *Civ) tickPeople(people []*Person, nDays int, cf func(int) *Culture, limitPop, reg int) []*Person {
	logStats := false

	// Separate the people into alive and dead.
	alive := make([]*Person, 0, len(people))
	dead := make([]*Person, 0, len(people))
	var peopleCount int
	for _, p := range people {
		// Tick, and check if we had a child.
		if child := m.tickPerson(p, nDays, cf); child != nil {
			alive = append(alive, child)
		}

		// Check if we are still alive.
		if !p.Death.IsSet() {
			alive = append(alive, p)
			peopleCount++
			if p.Prengancy != nil {
				peopleCount++
			}
		} else {
			dead = append(dead, p)
		}
		// TODO: Increase mortality and fertility rates based on prosperity, culture, etc.
	}

	if peopleCount >= limitPop {
		// Kill the oldest people first.
		sort.Slice(alive, func(a, b int) bool {
			return alive[a].Age < alive[b].Age
		})
		toKill := alive[limitPop:]
		alive = alive[:limitPop]
		for _, p := range toKill {
			m.killPerson(p, "overpopulation")
		}
		dead = append(dead, toKill...)
	} else {
		// TODO: Initiate pregnancies if we are below the population limit.
		// If we are above the population limit, we should kill some people.
		for _, pIdx := range rand.Perm(len(alive)) {
			p := alive[pIdx]
			if peopleCount >= limitPop {
				break
			}
			// Check if the person can get pregnant.
			if p.isOfChildbearingAge() && p.canBePregnant() {
				// Approximately once every year if no children.
				// TODO: Figure out proper chance of birth.
				chance := 365
				if p.Age > 40 {
					// Over 40, it becomes more and more unlikely.
					// TODO: Genetic variance?
					chance *= (p.Age - 40)
				}

				// The more children, the less likely it becomes
				// that more children are on the way.
				//
				// NOTE: Not because of biological reasons, but
				// who wants more children after having some.
				chance *= len(p.Children) + 1
				if rand.Intn(chance) < nDays {
					p.newPersonPregnancy(m.getNextPersonID(), p.Spouse)
					peopleCount++
				}
			}
		}

		// Add some random people to the population.
		missing := limitPop - peopleCount
		if missing > 0 {
			// Add some random people to the population.
			alive = append(alive, m.placePopulationAt(reg, missing, cf)...)
		}
	}

	// Pair up single people.
	m.matchMaker(alive)

	// Log some stats.
	if logStats {
		m.LogPopulationStats(alive)
		log.Println("dead people")
		m.LogPopulationStats(dead)
		log.Println("dropped people", len(dead), "alive", len(alive), "added", len(alive)-len(people))
	}
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
		localPop = append(localPop, m.newRandomPersonAt(r, culture, randGender(), nil))
	}

	// Match up people.
	m.matchMaker(localPop)

	// Add people to world.
	m.People = append(m.People, localPop...)
	return localPop
}

// movePopulationTo moves the given list of people to the given region.
func (m *Civ) movePopulationTo(p []*Person, r int) {
	for _, p := range p {
		m.updatePersonLocation(p, r)
	}
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
	pToMigrate = m.PickNPeopleToMigrate(pFrom, nil, nil, n)
	seenMigrate := make(map[*Person]bool)
	for _, p := range pToMigrate {
		seenMigrate[p] = true
		p.Region = rTo
	}
	for _, p := range pFrom {
		if !seenMigrate[p] {
			pFromAfter = append(pFromAfter, p)
		}
	}
	return
}

// PickNPeopleToMigrate selects n people from the given list of people, making sure that spouses
// and children are also migrated. There is a chance, depending on the person's traits, that they
// will not migrate or that they will intentionally migrate without their spouse or children.
func (m *Civ) PickNPeopleToMigrate(people, required, disallowed []*Person, n int) []*Person {
	if len(required) > n {
		panic("required people can't be more than n")
	}

	var toMigrate []*Person

	selectedPerson := make(map[*Person]bool)

	getDependents := func(p *Person) []*Person {
		log.Printf("Getting dependents for %s (%d)", p.Name(), p.Age)
		// If the person is cruel, they might want to migrate without their spouse and children.
		if p.Traits.HasTrait(geneticshuman.TraitCruel) && rand.Intn(100) < 50 {
			// TODO: Move spouse to ex-spouse if they end up migrating without them?
			// If so, change the opinion of the spouse, the children, and the relatives.
			log.Printf("Person %s (%d) is cruel, so they won't migrate with their spouse and children", p.Name(), p.Age)
			return nil
		}

		// Get the dependents willing to migrate.
		var dependents []*Person
		if spouse := p.Spouse; spouse != nil && spouse.Region == p.Region && !spouse.isDead() {
			dependents = append(dependents, spouse)
		}

		for _, c := range p.Children {
			if c.Region == p.Region && !c.isDead() && c.Age < 16 {
				dependents = append(dependents, c)
			}
		}

		var filtered []*Person
		for _, dep := range dependents {
			if selectedPerson[dep] {
				// We already selected this dependent.
				log.Printf("Person %s (%d) is already selected", dep.Name(), dep.Age)
				continue
			}

			// Check if the dependent is willing to migrate.
			// If not, we don't migrate them.
			opinionPOfDep := p.Opinions.GetOpinion(dep)
			opinionDepOfP := dep.Opinions.GetOpinion(p)

			totalOp := opinionPOfDep + opinionDepOfP
			if totalOp < 0 {
				// They don't like each other.
				// TODO: Make age a factor in this.
				log.Printf("Person %s (%d) doesn't like %s (%d) (%.2f), so they won't migrate together", p.Name(), p.Age, dep.Name(), dep.Age, totalOp)
				continue
			}

			filtered = append(filtered, dep)
		}

		return filtered
	}

	// First mark all disallowed people as selected.
	// TODO: Dependents?
	var disallowedCount int
	for _, p := range disallowed {
		selectedPerson[p] = true
		disallowedCount++
	}

	// If we have can still remove some people, lets remove dependents of disallowed people.
	maxDisallowed := len(people) - n
	if disallowedCount < maxDisallowed {
		for _, p := range disallowed {
			for _, d := range getDependents(p) {
				selectedPerson[d] = true
				disallowedCount++
				if disallowedCount >= maxDisallowed {
					break
				}
			}
		}
	}

	// Add all required people to the list of people to migrate.
	for _, p := range required {
		toMigrate = append(toMigrate, p)
		selectedPerson[p] = true

		// Add dependents to the list of people to migrate.
		dependents := getDependents(p)
		for _, d := range dependents {
			toMigrate = append(toMigrate, d)
			selectedPerson[d] = true
		}
	}

	// If we have already exceeded the number of people we need to migrate, we need to trim the list.
	if len(toMigrate) > n {
		// The required people HAVE to migrate, so we just leave some of the dependents behind
		// that are old enough to stay behind.
		isRequired := make(map[*Person]bool)
		for _, p := range required {
			isRequired[p] = true
		}

		var newToMigrate []*Person
		for _, p := range toMigrate {
			if !isRequired[p] && p.Age > 16 {
				continue
			}
			newToMigrate = append(newToMigrate, p)
			if len(newToMigrate) >= n {
				break
			}
		}
		return newToMigrate
	}

	// First sort the list of people by their likelihood to migrate.
	// This will be an aggregate score based on:
	// - traits
	// - age
	// - family situation
	// - opinion of the current region
	// - opinion of the destination region

	minScore := math.Inf(-1)

	calcMigrationScore := func(p *Person) float64 {
		// If the person is too young to migrate on their own, we return a low score.
		if p.Age < 12 {
			return minScore
		}

		var score float64

		// NOTE: Some of the traits effects would depend on the situation.
		// If the people are fleeing a war, a brave person might be more likely
		// to stay and fight, while a cowardly person might be more likely to flee.
		//
		// If the people are following a vision or a prophecy, a trusting person
		// or a brave person might be more likely to follow the vision, while a
		// paranoid person might be more likely to stay behind.
		//
		// If the person is cowardly or paranoid, they might not want to migrate.
		// Right now we assume that we flee a natural disaster, a famine, etc.
		if p.Traits.HasTrait(geneticshuman.TraitCowardly) ||
			p.Traits.HasTrait(geneticshuman.TraitParanoid) {
			score -= 1.0
		}
		if p.Traits.HasTrait(geneticshuman.TraitBrave) ||
			p.Traits.HasTrait(geneticshuman.TraitAmbitious) {
			score += 1.0
		}

		// If the person is content, they might not want to migrate.
		if p.Traits.HasTrait(geneticshuman.TraitContent) {
			score -= 0.5
		}

		// If the person is trusting, they might be more likely to migrate with
		// others.
		if p.Traits.HasTrait(geneticshuman.TraitTrusting) {
			score += 0.5
		}

		return score
	}

	migrationScore := make(map[*Person]float64)

	for _, p := range people {
		migrationScore[p] = calcMigrationScore(p)
	}

	// Sort the people by their migration score.
	sort.Slice(people, func(a, b int) bool {
		return migrationScore[people[a]] > migrationScore[people[b]]
	})

	// Iterate over the people and pick people and their dependents to migrate.
	for _, p := range people {
		if selectedPerson[p] {
			// We already selected this person.
			continue
		}

		// Get the number of dependents that are willing to migrate.
		dependents := getDependents(p)
		if len(dependents)+1 > n {
			// We can't migrate this person and their dependents.
			continue
		}
		toMigrate = append(toMigrate, p)
		selectedPerson[p] = true

		for _, d := range dependents {
			toMigrate = append(toMigrate, d)
			selectedPerson[d] = true
		}

		log.Printf("Selected %s (%d) to migrate with %d dependents", p.Name(), p.Age, len(dependents))
		if len(toMigrate) >= n {
			break
		}
	}

	// Now if we have less than n people, we can add some more people.
	if len(toMigrate) < n {
		for _, p := range people {
			if selectedPerson[p] {
				// We already selected this person.
				continue
			}
			log.Println("Selected", p.Name(), "to migrate without dependents")
			toMigrate = append(toMigrate, p)
			selectedPerson[p] = true
			if len(toMigrate) >= n {
				break
			}
		}
	}

	return toMigrate
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
		if single[a].Age == single[b].Age {
			// Higher popularity first.
			return single[a].Popularity < single[b].Popularity
		}
		return single[a].Age > single[b].Age
	})

	// Pair up singles.
	for i, p := range single {
		if !p.isEligibleSingle() {
			continue // Not single anymore.
		}

		popularityFactor := 1.0
		// If we are ambitious, we care more about the popularity of the other person.
		if p.Traits.HasTrait(geneticshuman.TraitAmbitious) {
			popularityFactor = 2.0
		} else if p.Traits.HasTrait(geneticshuman.TraitContent) ||
			p.Traits.HasTrait(geneticshuman.TraitTrusting) ||
			p.Traits.HasTrait(geneticshuman.TraitKind) ||
			p.Traits.HasTrait(geneticshuman.TraitCareless) {
			popularityFactor = 0.5
		}

		opinionFactor := 1.0
		// If we are cowardly or deceptive, we care less about our own opinion.
		if p.Traits.HasTrait(geneticshuman.TraitCowardly) ||
			p.Traits.HasTrait(geneticshuman.TraitDeceptive) {
			opinionFactor = 0.5
		} else if p.Traits.HasTrait(geneticshuman.TraitArrogant) ||
			p.Traits.HasTrait(geneticshuman.TraitCareful) ||
			p.Traits.HasTrait(geneticshuman.TraitParanoid) {
			opinionFactor = 2.0
		}

		compatibilityFactor := 1.0
		// If we are trusting, we are shy or humble we care more about compatibility.
		if p.Traits.HasTrait(geneticshuman.TraitTrusting) ||
			p.Traits.HasTrait(geneticshuman.TraitShy) ||
			p.Traits.HasTrait(geneticshuman.TraitHumble) {
			compatibilityFactor = 2.0
		} else if p.Traits.HasTrait(geneticshuman.TraitArrogant) ||
			p.Traits.HasTrait(geneticshuman.TraitCruel) ||
			p.Traits.HasTrait(geneticshuman.TraitDeceptive) {
			compatibilityFactor = 0.5
		}

		// TODO: Should we also calculate this the other way around and
		// use either the average or the sum of the two?
		// It is kinda unfair that some people get first pick.

		// Sort the rest of the people by compatibility.
		sort.Slice(single[i+1:], func(a, b int) bool {
			// TODO: We should weight the indvidual components of the comparison based
			// on how important they are to the person 'p'.
			// This is based on the personality of the person.
			// If we have an opinion of the person, rank good opinions higher.
			// If we hav no opinion, use the popularity.
			// If we have a bad opinion, rank them lower.
			opOfA := p.Opinions.GetOpinion(single[a])*opinionFactor +
				float64(single[a].Popularity)*popularityFactor +
				p.compare(single[a])*compatibilityFactor
			opOfB := p.Opinions.GetOpinion(single[b])*opinionFactor +
				float64(single[b].Popularity)*popularityFactor +
				p.compare(single[b])*compatibilityFactor
			return opOfA > opOfB
		})

		// We can skip all people up to the current person, since they have already been matched.
		for j := i + 1; j < len(single); j++ {
			pc := single[j]
			if p.Region != pc.Region || !pc.isEligibleSingle() || // Not single anymore or not in same region.
				p.Gender() == pc.Gender() || isRelated(p, pc) || // TODO: Allow same sex couples (which can adopt children/orphans).
				utils.Abs(p.Age-pc.Age) > min(p.Age, pc.Age)/3 { // At most 33% age difference.
				continue
			}

			// TODO: Sort by compatibility?
			// Right now we just check if they don't hate each other.
			opAOfB := p.Opinions.GetOpinion(pc)
			if opAOfB < 0 {
				log.Printf("Person %s (%d) doesn't like %s (%d) (%.2f)", p.Name(), p.Age, pc.Name(), pc.Age, opAOfB)
				continue // They don't like each other.
			}

			opBOfA := pc.Opinions.GetOpinion(p)
			if opBOfA < 0 {
				log.Printf("Person %s (%d) doesn't like %s (%d) (%.2f)", pc.Name(), pc.Age, p.Name(), p.Age, opBOfA)
				continue // They don't like each other.
			}

			// Get popularity of the people.
			popA := float64(p.Popularity)
			popB := float64(pc.Popularity)

			// Get Compatibility of the people.
			compA := p.compare(pc)
			compB := pc.compare(p)

			p.Spouse = pc
			pc.Spouse = p

			log.Printf("Matched %s (%d O:%.1f, P:%.1f, C:%.1f) with %s (%d O:%.1f, P:%.1f, C:%.1f)", p.Name(), p.Age, opBOfA, popA, compA, pc.Name(), pc.Age, opAOfB, popB, compB)

			// Update family name.
			//
			// TODO: This is not optimal... There should be a better way to do this.
			// The culture should determine any changes to the name.
			if p.Gender() == GenderFemale {
				p.LastName = pc.LastName
			} else {
				pc.LastName = p.LastName
			}

			// We found a match, so we can break out of the inner loop.
			break
		}
	}
}
