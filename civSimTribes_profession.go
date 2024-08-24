package genworldvoronoi

import (
	"log"
	"math/rand"
	"sort"
)

func (t *Tribe) AssignProfessions(m *Civ) {
	// Determine the professions needed.
	needed := t.determineProfessions()
	if len(needed) == 0 {
		return
	}

	// Assign professions to people that don't have one yet.
	for _, i := range rand.Perm(len(t.People)) {
		p := t.People[i]
		// Check if the person already has a profession, or is too young or too old.
		if p.Career != nil || p.Age < 14 || p.Age > 65 {
			continue
		}

		// Assign a required profession to the person.
		// TODO: Take into account preferences, parents, skills, etc.
		for i := range needed {
			if n := &needed[i]; n.Missing > 0 {
				// TODO: If a close relative has the same job, transfer some of the skill
				// to the new person (if we aren't careless or careful).
				p.Career = n.Profession.Career(int(m.History.GetYear())) // TODO: Add current year.
				n.Missing--
			}
		}
	}

	// Log the professions.
	log.Print("Professions:")
	for _, need := range needed {
		log.Printf("Profession %s: Needed %d, Got %d, Missing %d\n", need.Profession.Name, need.Needed, need.Got, need.Missing)
	}
}

type profNeeded struct {
	Profession *Profession
	Needed     int
	Got        int
	Missing    int
}

// Depending on the type of the tribe, and the developed skills, as well as the settlement status
// of the tribe, and its population, the professions of the tribe members will vary.
func (t *Tribe) determineProfessions() []profNeeded {
	// TODO: This should also depend on the available resources, and the settlement status of the tribe.
	var availableProfessions []*Profession
	for _, s := range Skills {
		if !t.Skills[s] || s.Profession == nil {
			continue
		}
		availableProfessions = append(availableProfessions, s.Profession)
	}

	// If we have a settlement, we gain access to some more professions.
	// TODO: These professions should be part of the settlement.
	if t.Settlement != nil {
		availableProfessions = append(availableProfessions, ProfessionMerchant)
		availableProfessions = append(availableProfessions, ProfessionHealer)
	}

	// If we have a city state, we gain access to some more professions.
	// TODO: These professions should be part of the city state.
	if t.CityState != nil {
		availableProfessions = append(availableProfessions, ProfessionAdministrator)
		availableProfessions = append(availableProfessions, ProfessionAmbassador)
		availableProfessions = append(availableProfessions, ProfessionScholar)
		availableProfessions = append(availableProfessions, ProfessionArchitect)
	}

	// If we have an empire, we gain access to some more professions.
	// TODO: These professions should be part of the empire.
	if t.Empire != nil {
		availableProfessions = append(availableProfessions, ProfessionSpy)
	}

	// If we have a religion, we gain access to some more professions.
	// TODO: These professions should be part of the religion.
	if t.Religion != nil {
		availableProfessions = append(availableProfessions, ProfessionCleric)
	}

	// TODO: Also add cultural professions.

	// Determine the number of people needed for each profession.
	neededProfessions := make(map[*Profession]int)
	gotProfessions := make(map[*Profession]int)
	var popAssigned int
	for _, p := range availableProfessions {
		neededPop := p.PerPopulation
		if neededPop == 0 {
			continue
		}
		neededProfessions[p] = t.Population / neededPop // Ceil?
		popAssigned += neededProfessions[p]
	}

	var workingAge int
	for _, p := range t.People {
		if p.Age >= 14 && p.Age <= 65 {
			workingAge++
		}
		if p.Career != nil {
			gotProfessions[p.Career.Profession]++
		}
	}

	var needed []profNeeded
	for p, n := range neededProfessions {
		needed = append(needed, profNeeded{
			Profession: p,
			Needed:     n,
			Got:        gotProfessions[p],
			Missing:    n - gotProfessions[p],
		})
	}

	// Sort the professions by how many people are needed from most to least.
	sort.Slice(needed, func(i, j int) bool {
		return needed[i].Missing > needed[j].Missing
	})

	for _, n := range needed {
		log.Printf("Profession %s: Needed %d, Got %d, Missing %d\n", n.Profession.Name, n.Needed, n.Got, n.Missing)
	}

	return needed
}
