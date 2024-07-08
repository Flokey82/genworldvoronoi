package genworldvoronoi

import (
	"log"
)

// getCityStateNeighbors returns all neighboring city states of the given city state.
func (s *simState) getCityStateNeighbors(c *CityState) []*CityState {
	var neighbors []*CityState
	for _, nbID := range s.m.getCityStateNeighbors(c) {
		if nb := s.m.GetCityState(nbID); nb != nil {
			neighbors = append(neighbors, nb)
		} else {
			log.Printf("!!!%s has a neighboring city state with ID %d and it could not be found", c.String(), nbID)
		}
	}
	return neighbors
}

func (s *simState) handleCityState(t *Tribe) {
	log.Printf("!!!%s is a city state in region %d.", t.String(), t.RegionID)
	// TODO:
	// Check if we can add some cities to our city state.
	// If we are big and prosperous, try to negotiate protection, trade, etc. with other city states.
	// We can either try to join an empire that already exists, or try to form a new empire by
	// negotiating other city states to join us.

	// TODO: Do trade, diplomacy, etc.
	// - We create proposals and send them to other city states.
	// - Proposals are resolved in the next iteration.
	// - Handle proposals from the previous iteration.
	for _, c := range t.CityState.Cities {
		if c != t.CityState.Capital {
			log.Printf("!!!%s has a city: %s.", t.String(), c.String())
		}
	}

	// Get neighboring city states.
	// - We can propose alliances, trade, etc.
	// - Potentially attack other city states.
	for _, nbState := range s.getCityStateNeighbors(t.CityState) {
		log.Printf("!!!%s has a neighboring city state: %s.", t.String(), nbState.String())
	}

	// Now, if we are big and prosperous enough, we can establish an empire.
	// Right now, we just check the population, but we should instead check resources,
	// influence, prosperity, etc.
	if t.Type == TribeTypeCityState && t.Population > 2000 {
		// Promote to an empire.
		t.Type = TribeTypeEmpire
		log.Println("!!!Tribe", t.ID, "has become an empire.")
		t.Empire = s.m.placeEmpireAt(t.RegionID, t.Settlement)
		// Evolve the government.
		t.Leadership.ChangeType(FactionTypeCivil, t.findNaturalProgression(), s.m.History)
	}
}

func (s *simState) handleTradeCityState(t *Tribe) {
	// Get neighboring city states.
	nbCityStates := s.getCityStateNeighbors(t.CityState)
	var tradeProposals []*tradeProposal

	// TODO: Calculate available resources by looking at all member cities.
	genCityStateLocalRes := func(cs *CityState) LocalResouces {
		localRes := LocalResouces{}
		for _, c := range cs.Cities {
			localRes.Add(s.m.getResources(c.ID, true))
		}
		return localRes
	}

	// Loop through all the neighboring city states and propose trades.
	srcRes := genCityStateLocalRes(t.CityState)
	for _, nb := range nbCityStates {
		// Determine what resources we have and what resources we need.
		dstRes := genCityStateLocalRes(nb)
		imp := dstRes.Remove(srcRes)
		exp := srcRes.Remove(dstRes)
		tradeProposals = append(tradeProposals, &tradeProposal{
			id:   nb.ID,
			exp:  exp,
			imp:  imp,
			dist: s.m.GetDistance(t.CityState.ID, nb.ID),
		})
	}

	// Log the trade proposals.
	for _, tp := range tradeProposals {
		dstCityState := s.m.GetCityState(tp.id)
		log.Printf("!!!%s has proposed a trade with %s, dist %.2f", t.String(), dstCityState.String(), tp.dist)
		if tp.exp.HasAny() {
			dstCityState.ResourceStorage.Add(tp.exp)
			log.Printf("  Export: %s", tp.exp.String())
		}
		if tp.imp.HasAny() {
			t.CityState.ResourceStorage.Add(tp.exp)
			log.Printf("  Import: %s", tp.imp.String())
		}

		score := t.CityState.compare(dstCityState)
		log.Printf("  Score: %.2f", score)
	}
}
