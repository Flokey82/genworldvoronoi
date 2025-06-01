package civ

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
	// Do city stuff (Diplomacy, defense, etc.)
	s.m.handleTradeCity(t.Settlement)
	s.m.tickDiplomacyCity(t.Settlement)
	s.m.handleLeadershipCity(t.Settlement)

	// Handle nearby cities.
	nearbyCities := s.m.getNearbyCities(t.RegionID, 300.0)
	for _, c := range nearbyCities {
		// TODO: Trade, diplomacy, etc will be handled different if the city is not in the same city state.
		if s.m.CityStates.Regions[c.city.ID] != s.m.CityStates.Regions[t.RegionID] {
			log.Printf("!!!%s has a nearby city: %s %.2fkm but it's not in the same city state.", t.String(), c.city.String(), c.dist)
		} else {
			log.Printf("!!!%s has a nearby city: %s %.2fkm.", t.String(), c.city.String(), c.dist)
		}
	}

	// Now, if we are big and prosperous enough, we can establish a city state.
	// Right now, we just check the population, but we should instead check resources,
	// influence, prosperity, etc.
	if t.Type == TribeTypeCity && t.Population > 1000 {
		log.Println("!!!Tribe", t.ID, "has become a city state.")

		// Promote to / set up a city state and evolve the government.
		t.Type = TribeTypeCityState
		if t.Leadership == nil {
			log.Println("!!!City has no leadership before becoming a city state.")
		}
		t.CityState = s.m.PlaceCityStateAt(t.RegionID, t.Settlement, t.Leadership)
		if t.CityState.Leadership == nil {
			log.Println("!!!City state has no leadership.1")
		}
		t.Leadership.ChangeType(FactionTypeCivil, t.findNaturalProgression(), s.m.History)
		if t.CityState.Leadership == nil {
			log.Println("!!!City state has no leadership.2")
		}
	}
}

func (s *simState) handleCityState(t *Tribe) {
	// TODO: Do trade, diplomacy, etc.
	// - We create proposals and send them to other city states.
	// - Proposals are resolved in the next iteration.
	// - Handle proposals from the previous iteration.
	//
	// Check if we can add some cities to our city state.
	// If we are big and prosperous, try to negotiate protection, trade, etc. with other city states.
	// We can either try to join an empire that already exists, or try to form a new empire by
	// negotiating other city states to join us.
	s.m.handleTradeCityState(t.CityState)
	s.m.tickDiplomacyCityState(t.CityState)
	s.m.handleLeadershipCityState(t.CityState)
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
		log.Println("!!!Tribe", t.ID, "has become an empire.")

		// Promote to / set up an empire and evolve the government.
		t.Type = TribeTypeEmpire
		t.Empire = s.m.placeEmpireAt(t.RegionID, t.Settlement, t.Leadership)
		t.Leadership.ChangeType(FactionTypeCivil, t.findNaturalProgression(), s.m.History)
	}
}

func (s *simState) handleEmpire(t *Tribe) {
	s.m.handleTradeEmpire(t.Empire)
	s.m.tickDiplomacyEmpire(t.Empire)
	s.m.handleLeadershipEmpire(t.Empire)

	// Find neighboring empires.
	for _, nbEmpire := range s.m.getEmpireNeighbors(t.Empire) {
		log.Printf("!!!%s has a neighboring empire: %s (score %.2f)", t.String(), nbEmpire.String(), t.Empire.compare(nbEmpire))
	}

	// Find all neighboring city states that are not part of another empire.
	for _, nbState := range s.m.getEmpireCityStateNeighbors(t.Empire, true) {
		log.Printf("!!!%s has a neighboring city state: %s.", t.String(), nbState.String())
	}
}
