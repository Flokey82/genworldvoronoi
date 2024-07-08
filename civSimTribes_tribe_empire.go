package genworldvoronoi

import "log"

// getEmpireNeighbors returns all neighboring empires of the given empire.
func (s *simState) getEmpireNeighbors(e *Empire) []*Empire {
	var neighbors []*Empire
	for _, nbID := range s.m.getEmpireNeighbors(e) {
		if nb := s.m.GetEmpire(nbID); nb != nil {
			neighbors = append(neighbors, nb)
		} else {
			log.Printf("!!!%s has a neighboring empire with ID %d and it could not be found", e.String(), nbID)
		}
	}
	return neighbors
}

// getEmpireNeighborsCityStates returns all neighboring city states that are not part of another empire.
func (s *simState) getEmpireNeighborsCityStates(e *Empire) []*CityState {
	var neighbors []*CityState
	// Find all neighboring city states that are not part of another empire.
	for _, nbState := range s.m.getEmpireCityStateNeighbors(e) {
		if s.m.getCityStateEmpire(nbState) == -1 {
			neighbors = append(neighbors, nbState)
		} else {
			log.Printf("!!!%s has a neighboring city state: %s but it's part of another empire.", e.String(), nbState.String())
		}
	}
	return neighbors
}

func (s *simState) handleEmpire(t *Tribe) {
	log.Printf("!!!%s has become an empire.", t.String())
	// Find neighboring empires.
	for _, nbEmpire := range s.getEmpireNeighbors(t.Empire) {
		log.Printf("!!!%s has a neighboring empire: %s (score %.2f)", t.String(), nbEmpire.String(), t.Empire.compare(nbEmpire))
	}

	// Find all neighboring city states that are not part of another empire.
	for _, nbState := range s.getEmpireNeighborsCityStates(t.Empire) {
		log.Printf("!!!%s has a neighboring city state: %s.", t.String(), nbState.String())
	}
}

func (s *simState) handleTradeEmpire(t *Tribe) {
	// Get neighboring empires.
	nbEmpires := s.getEmpireNeighbors(t.Empire)
	var tradeProposals []*tradeProposal

	// TODO: Calculate available resources by looking at all member city states.
	genEmpireLocalRes := func(e *Empire) LocalResouces {
		localRes := LocalResouces{}
		for _, c := range e.Cities {
			localRes.Add(s.m.getResources(c.ID, true))
		}
		return localRes
	}

	// Loop through all the neighboring empires and propose trades.
	srcRes := genEmpireLocalRes(t.Empire)
	for _, nb := range nbEmpires {
		// Determine what resources we have and what resources we need.
		dstRes := genEmpireLocalRes(nb)
		imp := dstRes.Remove(srcRes)
		exp := srcRes.Remove(dstRes)
		tradeProposals = append(tradeProposals, &tradeProposal{
			id:   nb.ID,
			exp:  exp,
			imp:  imp,
			dist: s.m.GetDistance(t.Empire.ID, nb.ID),
		})
	}

	// Log the trade proposals.
	for _, tp := range tradeProposals {
		dstEmpire := s.m.GetEmpire(tp.id)
		log.Printf("!!!%s has proposed a trade with %s, dist %.2f", t.String(), dstEmpire.String(), tp.dist)
		if tp.exp.HasAny() {
			dstEmpire.ResourceStorage.Add(tp.exp)
			log.Printf("  Export: %s", tp.exp.String())
		}
		if tp.imp.HasAny() {
			t.Empire.ResourceStorage.Add(tp.exp)
			log.Printf("  Import: %s", tp.imp.String())
		}
	}

	// Trade with unaligned city states.
	nbCityStates := s.getEmpireNeighborsCityStates(t.Empire)
	tradeProposals = tradeProposals[:0]
	for _, nb := range nbCityStates {
		// Determine what resources we have and what resources we need.
		exp, imp := s.compareResources(t.RegionID, nb.ID)
		tradeProposals = append(tradeProposals, &tradeProposal{
			id:   nb.ID,
			exp:  exp,
			imp:  imp,
			dist: s.m.GetDistance(t.Empire.ID, nb.ID),
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
			t.Empire.ResourceStorage.Add(tp.exp)
			log.Printf("  Import: %s", tp.imp.String())
		}
	}
}
