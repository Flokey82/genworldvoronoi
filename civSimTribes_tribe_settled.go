package genworldvoronoi

import (
	"log"
)

func (s *simState) handleSettledTribe(t *Tribe) {
	if t.Type >= TribeTypeCity {
		s.handleCity(t)
	}
	if t.Type >= TribeTypeCityState {
		s.handleCityState(t)
	}
	if t.Type >= TribeTypeEmpire {
		s.handleEmpire(t)
	}
}

func (s *simState) handleCity(t *Tribe) {
	log.Printf("!!!%s has settled in region %d.", t.String(), t.RegionID)

	// Do city stuff (Diplomacy, defense, etc.)
	s.m.handleTradeCity(t.Settlement)

	// Handle nearby cities.
	nearbyCities := s.m.getNearbyCities(t.RegionID, 300.0)
	for _, c := range nearbyCities {
		// TODO: Trade, diplomacy, etc will be handled different if the city is not in the same city state.
		if s.m.RegionToCityState[c.city.ID] != s.m.RegionToCityState[t.RegionID] {
			log.Printf("!!!%s has a nearby city: %s %.2fkm but it's not in the same city state.", t.String(), c.city.String(), c.dist)
		} else {
			log.Printf("!!!%s has a nearby city: %s %.2fkm.", t.String(), c.city.String(), c.dist)
		}
	}

	// Now, if we are big and prosperous enough, we can establish a city state.
	// Right now, we just check the population, but we should instead check resources,
	// influence, prosperity, etc.
	if t.Type == TribeTypeCity && t.Population > 1000 {
		// Promote to a city state.
		t.Type = TribeTypeCityState
		log.Println("!!!Tribe", t.ID, "has become a city state.")
		// Set up a city state.
		t.CityState = s.m.PlaceCityStateAt(t.RegionID, t.Settlement)
		// Evolve the government.
		t.Leadership.ChangeType(FactionTypeCivil, t.findNaturalProgression(), s.m.History)
	}
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
	s.m.handleTradeCityState(t.CityState)
	for _, c := range t.CityState.Cities {
		if c != t.CityState.Capital {
			log.Printf("!!!%s has a city: %s.", t.String(), c.String())
		}
	}

	// Get neighboring city states.
	// - We can propose alliances, trade, etc.
	// - Potentially attack other city states.
	for _, nbState := range s.m.getCityStateNeighbors(t.CityState) {
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

func (s *simState) handleEmpire(t *Tribe) {
	log.Printf("!!!%s has become an empire.", t.String())
	s.m.handleTradeEmpire(t.Empire)

	// Find neighboring empires.
	for _, nbEmpire := range s.m.getEmpireNeighbors(t.Empire) {
		log.Printf("!!!%s has a neighboring empire: %s (score %.2f)", t.String(), nbEmpire.String(), t.Empire.compare(nbEmpire))
	}

	// Find all neighboring city states that are not part of another empire.
	for _, nbState := range s.m.getEmpireCityStateNeighbors(t.Empire, true) {
		log.Printf("!!!%s has a neighboring city state: %s.", t.String(), nbState.String())
	}
}
