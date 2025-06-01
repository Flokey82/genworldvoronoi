package civ2

import (
	"log"
	"sort"
)

func (m *Civ) tickCityStates(nDays int) {
	rNbs := make([]int, 0, 10)
	for _, cs := range m.CityStates.Objects {
		// City states might collapse if they lose on influence.
		if cs.Capital == nil || cs.Capital.Population == 0 {
			continue
		}

		log.Printf("City state %d", cs.ID)

		// We might expand or contract our influence.
		m.expandCityState(cs, rNbs)

		// We might found an empire.
		if m.Empires.GetAt(cs.ID) == nil && cs.Capital.Population > 10000 {
			m.foundEmpire(cs)
		}
	}
}

func (m *Civ) expandCityState(cs *CityState, rNbs []int) {
	// We might expand or contract our influence.
	// Look at surrounding settlements and cities that would be willing to join us
	// Check what regions we control.
	var controlledRegions []int
	seenRegions := make(map[int]bool)
	for r, id := range m.CityStates.Regions {
		if id == cs.ID {
			// We control this region, so we check the neighbours if we can expand.
			controlledRegions = append(controlledRegions, r)
			seenRegions[r] = true
		}
	}

	// Check the neighbours of the controlled regions.
	var numNewRegions int
	var limitExpansionToSettled bool
	var possibleDestinations []int
	for _, r := range controlledRegions {
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
			seenRegions[nb] = true
			possibleDestinations = append(possibleDestinations, nb)
			numNewRegions++
		}
	}
	if numNewRegions > 0 {
		log.Printf("City state %d has expanded to %d new regions", cs.ID, numNewRegions)
	}
	const limitExpansion = 5
	// Sort candidates by distance.
	distances := make([]float64, m.NumRegions)
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
	}
}

func (m *Civ) foundEmpire(cs *CityState) {
	e := &Empire{
		ID:      cs.ID,
		Capital: cs.Capital,
		Culture: cs.Culture,
		Founded: m.Geo.Calendar.GetYear(),
	}

	// Place the empire in the world.
	m.Empires.PlaceObjectAt(e, e.ID)

	// The empire will control the regions of the city state.
	for r, id := range m.CityStates.Regions {
		if id == cs.ID {
			m.Empires.PlaceObjectAt(e, r)
		}
	}
}
