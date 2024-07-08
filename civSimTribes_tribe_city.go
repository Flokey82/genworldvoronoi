package genworldvoronoi

import (
	"log"
)

func (s *simState) handleCity(t *Tribe) {
	log.Printf("!!!%s has settled in region %d.", t.String(), t.RegionID)

	// Do city stuff (Diplomacy, defense, etc.)

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

func (s *simState) handleTradeCity(t *Tribe) {
	// TODO: Figure out what we actually need and what we have in excess.
	const tradeRadius = 900.0 // km

	tradeCities := s.getNearbyCities(t.RegionID, tradeRadius)

	idToCity := make(map[int]*City)
	for _, c := range tradeCities {
		idToCity[c.city.ID] = c.city
	}

	var tradeProposals []*tradeProposal
	// Loop through all the trade cities and propose trades.
	// If we have resources that the other city needs, we propose a trade for export.
	// If the other city has resources that we need, we propose a trade for import.
	for _, tc := range tradeCities {
		// Determine what resources we have and what resources we need.
		exp, imp := s.compareResources(t.RegionID, tc.city.ID)
		tradeProposals = append(tradeProposals, &tradeProposal{
			id:   tc.city.ID,
			exp:  exp,
			imp:  imp,
			dist: tc.dist,
		})
	}

	// Log the trade proposals.
	for _, tp := range tradeProposals {
		dstCity := idToCity[tp.id]
		log.Printf("!!!%s has proposed a trade with %s, dist %.2f", t.String(), dstCity.String(), tp.dist)
		if tp.exp.HasAny() {
			dstCity.ResourceStorage.Add(tp.exp)
			log.Printf("  Export: %s", tp.exp.String())
		}
		if tp.imp.HasAny() {
			t.Settlement.ResourceStorage.Add(tp.exp)
			log.Printf("  Import: %s", tp.imp.String())
		}

		score := t.Settlement.compare(dstCity)
		log.Printf("  Score: %.2f", score)
	}
}
