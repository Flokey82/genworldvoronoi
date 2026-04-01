package civ2

import (
	"log"
	"sort"

	"github.com/Flokey82/genworldvoronoi/civ"
)

func (m *Civ) tickCityStates(nDays int) {
	rNbs := make([]int, 0, 10)
	for _, cs := range m.CityStates.Objects {
		// City states might collapse if they lose on influence.
		if cs.Capital == nil || cs.Capital.Population == 0 {
			continue
		}

		log.Printf("City state %d", cs.ID)
		
		// Centralized economic tick.
		m.tickEconomyBase(cs, nDays)

		// Tick diplomatic relations.
		m.tickDiplomacyBase(cs, nDays)

		// Tick leadership.
		m.tickLeadership(cs, nDays)

		// We might expand or contract our influence.
		m.expandCityState(cs, rNbs)

		// We might found an empire.
		if m.Empires.GetAt(cs.ID) == nil && cs.Capital.Population > 10000 {
			m.foundEmpire(cs)
		}
	}
}

func (m *Civ) expandCityState(cs *CityState, rNbs []int) {
	// Check the neighbours of the controlled regions.
	var numNewRegions int
	var limitExpansionToSettled bool
	var possibleDestinations []int
	seenRegions := make(map[int]bool)
	for _, r := range cs.Regions {
		seenRegions[r] = true
	}

	for _, r := range cs.Regions {
		for _, nb := range m.R_circulate_r(rNbs, r) {
			if m.Elevation.Values[nb] < 0 {
				// We cannot expand to water.
				continue
			}
			if _, ok := seenRegions[nb]; ok || m.CityStates.GetAt(nb) != nil {
				continue
			}
			seenRegions[nb] = true
			if limitExpansionToSettled && m.Settlements.GetIDAt(nb) == -1 && m.Cities.GetIDAt(nb) == -1 {
				continue
			}
			if c := m.Cities.Get(nb); c != nil && c.Population >= cs.Capital.Population {
				continue
			}
			// We found a settlement or a city, so we can expand.
			// Check if the settlement is willing to join us.
			possibleDestinations = append(possibleDestinations, nb)
			numNewRegions++
		}
	}
	if numNewRegions > 0 {
		log.Printf("City state %d has expanded to %d new regions", cs.ID, numNewRegions)
	}
	const limitExpansion = 5
	// Sort candidates by distance.
	distances := make(map[int]float64)
	for _, r := range possibleDestinations {
		distances[r] = m.GetDistance(cs.Capital.ID, r)
	}
	// Sort the possible destinations by distance.
	sort.Slice(possibleDestinations, func(i, j int) bool {
		return distances[possibleDestinations[i]] < distances[possibleDestinations[j]]
	})
	for i, r := range possibleDestinations {
		if i >= limitExpansion {
			break
		}
		m.CityStates.PlaceObjectAt(cs, r)
		cs.AddRegion(r)
	}
}

func (m *Civ) foundEmpire(cs *CityState) {
	e := &Empire{
		BaseEntity: BaseEntity{
			ID:                cs.ID,
			Name:              "Empire of " + cs.Capital.Name,
			Culture:           cs.Culture,
			Type:              civ.ObjectTypeEmpire,
			Storage:           NewStorage(),
			GoverningPeople:   newGoverningPeople(),
			Infrastructure:    NewInfrastructure(),
			ConstructionQueue: NewConstructionQueue(),
			Military:          cs.Military,
		},
		Capital: cs.Capital,
		Founded: m.Geo.Calendar.GetYear(),
	}

	// Place the empire in the world and sync regions.
	m.Empires.PlaceObjectAt(e, e.ID)
	for _, r := range cs.Regions {
		m.Empires.PlaceObjectAt(e, r)
		e.AddRegion(r)
	}
}
