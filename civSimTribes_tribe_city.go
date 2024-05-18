package genworldvoronoi

import "log"

func (s *simState) handleCity(t *Tribe) {
	log.Printf("!!!%s has settled in region %d.", t.String(), t.RegionID)

	// Do city stuff.
	// Diplomacy, trade, defense, etc.
	// - We create proposals and send them to other cities, city states, etc.
	//
	// Types of proposals:
	// - Trade proposals (resources, etc.)
	//   - Determine what resources we have and what resources we need.
	// - Diplomatic proposals (alliances, etc.)
	//   - We can propose alliances, non-aggression pacts, etc.
	//   - We can ask the nest higher level of government to protect us.
	//   - We can ask other cities for alliances, etc. (How do we decide who to ask?)
	// - Military proposals (defense, etc.)
	//   - We can propose to other cities to join us in a war.
	//   - We can propose to other cities to defend us.
	s.handleTrade(t)

	// Handle nearby cities.
	nearbyCities := s.getNearbyCities(t.RegionID, 300.0)
	for _, c := range nearbyCities {
		// TODO: Trade, diplomacy, etc will be handled different if the city is not in the same city state.
		if s.m.RegionToCityState[c.city.ID] != s.m.RegionToCityState[t.RegionID] {
			log.Printf("!!!%s has a nearby city: %s %.2fkm but it's not in the same city state.", t.String(), c.city.String(), c.dist)
		} else {
			log.Printf("!!!%s has a nearby city: %s %.2fkm.", t.String(), c.city.String(), c.dist)
		}
	}

	// Now, if we are big and prosperous enough, we can establish a city state.
	if t.Population > 1000 {
		// Promote to a city state.
		t.Type = TribeTypeCityState
		log.Println("!!!Tribe", t.ID, "has become a city state.")
		// Set up a city state.
		t.CityState = s.m.PlaceCityStateAt(t.RegionID, t.Settlement)
	}
}
