package genworldvoronoi

import "log"

func (s *simState) handleCityState(t *Tribe) {
	log.Printf("!!!%s is a city state in region %d.", t.String(), t.RegionID)
	// TODO:
	// If we are big and prosperous, try to negotiate protection, trade, etc. with other city states.
	// We can either try to join an empire that already exists, or try to form a new empire by
	// negotiating other city states to join us.
	s.handleTrade(t)

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
	for _, nbStateID := range s.m.getCityStateNeighbors(t.CityState) {
		if nbState := s.m.GetCityState(nbStateID); nbState != nil {
			log.Printf("!!!%s has a neighboring city state: %s (score %.2f)", t.String(), nbState.String(), t.CityState.compare(nbState))
		} else {
			log.Printf("!!!%s has a neighboring city state with ID %d and it could not be found", t.String(), nbStateID)
		}
	}

	if t.Population > 2000 {
		// Promote to an empire.
		t.Type = TribeTypeEmpire
		log.Println("!!!Tribe", t.ID, "has become an empire.")
		t.Empire = s.m.placeEmpireAt(t.RegionID, t.Settlement)
	}
}
