package genworldvoronoi

import "log"

type tradeProposal struct {
	id   int           // id of the trading partner
	dist float64       // distance to the trading partner
	exp  LocalResouces // resources that we can export
	imp  LocalResouces // resources that we can import
}

func (m *Civ) handleTradeCity(c *City) {
	// TODO: Figure out what we actually need and what we have in excess.
	const tradeRadius = 900.0 // km

	tradeCities := m.getNearbyCities(c.ID, tradeRadius)

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
		exp, imp := m.compareResources(c.ID, tc.city.ID)
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
		log.Printf("!!!%s has proposed a trade with %s, dist %.2f", c.String(), dstCity.String(), tp.dist)
		if tp.exp.HasAny() {
			dstCity.ResourceStorage.Add(tp.exp)
			log.Printf("  Export: %s", tp.exp.String())
		}
		if tp.imp.HasAny() {
			c.ResourceStorage.Add(tp.exp)
			log.Printf("  Import: %s", tp.imp.String())
		}

		score := c.compare(dstCity)
		log.Printf("  Score: %.2f", score)
	}
}

func (m *Civ) handleTradeCityState(cs *CityState) {
	// Get neighboring city states.
	nbCityStates := m.getCityStateNeighbors(cs)
	var tradeProposals []*tradeProposal

	// TODO: Calculate available resources by looking at all member cities.
	genCityStateLocalRes := func(cs *CityState) LocalResouces {
		localRes := LocalResouces{}
		for _, c := range cs.Cities {
			localRes.Add(m.getResources(c.ID, true))
		}
		return localRes
	}

	// Loop through all the neighboring city states and propose trades.
	srcRes := genCityStateLocalRes(cs)
	for _, nb := range nbCityStates {
		// Determine what resources we have and what resources we need.
		dstRes := genCityStateLocalRes(nb)
		imp := dstRes.Remove(srcRes)
		exp := srcRes.Remove(dstRes)
		tradeProposals = append(tradeProposals, &tradeProposal{
			id:   nb.ID,
			exp:  exp,
			imp:  imp,
			dist: m.GetDistance(cs.ID, nb.ID),
		})
	}

	// Log the trade proposals.
	for _, tp := range tradeProposals {
		dstCityState := m.GetCityState(tp.id)
		log.Printf("!!!%s has proposed a trade with %s, dist %.2f", cs.String(), dstCityState.String(), tp.dist)
		if tp.exp.HasAny() {
			dstCityState.ResourceStorage.Add(tp.exp)
			log.Printf("  Export: %s", tp.exp.String())
		}
		if tp.imp.HasAny() {
			cs.ResourceStorage.Add(tp.exp)
			log.Printf("  Import: %s", tp.imp.String())
		}

		score := cs.compare(dstCityState)
		log.Printf("  Score: %.2f", score)
	}
}

func (m *Civ) handleTradeEmpire(emp *Empire) {
	// Get neighboring empires.
	nbEmpires := m.getEmpireNeighbors(emp)
	var tradeProposals []*tradeProposal

	// TODO: Calculate available resources by looking at all member city states.
	genEmpireLocalRes := func(e *Empire) LocalResouces {
		localRes := LocalResouces{}
		for _, c := range e.Cities {
			localRes.Add(m.getResources(c.ID, true))
		}
		return localRes
	}

	// Loop through all the neighboring empires and propose trades.
	srcRes := genEmpireLocalRes(emp)
	for _, nb := range nbEmpires {
		// Determine what resources we have and what resources we need.
		dstRes := genEmpireLocalRes(nb)
		imp := dstRes.Remove(srcRes)
		exp := srcRes.Remove(dstRes)
		tradeProposals = append(tradeProposals, &tradeProposal{
			id:   nb.ID,
			exp:  exp,
			imp:  imp,
			dist: m.GetDistance(emp.ID, nb.ID),
		})
	}

	// Log the trade proposals.
	for _, tp := range tradeProposals {
		dstEmpire := m.GetEmpire(tp.id)
		log.Printf("!!!%s has proposed a trade with %s, dist %.2f", emp.String(), dstEmpire.String(), tp.dist)
		if tp.exp.HasAny() {
			dstEmpire.ResourceStorage.Add(tp.exp)
			log.Printf("  Export: %s", tp.exp.String())
		}
		if tp.imp.HasAny() {
			emp.ResourceStorage.Add(tp.exp)
			log.Printf("  Import: %s", tp.imp.String())
		}
	}

	// Trade with unaligned city states.
	nbCityStates := m.getEmpireCityStateNeighbors(emp, true)
	tradeProposals = tradeProposals[:0]
	for _, nb := range nbCityStates {
		// Determine what resources we have and what resources we need.
		exp, imp := m.compareResources(emp.ID, nb.ID)
		tradeProposals = append(tradeProposals, &tradeProposal{
			id:   nb.ID,
			exp:  exp,
			imp:  imp,
			dist: m.GetDistance(emp.ID, nb.ID),
		})
	}

	// Log the trade proposals.
	for _, tp := range tradeProposals {
		dstCityState := m.GetCityState(tp.id)
		log.Printf("!!!%s has proposed a trade with %s, dist %.2f", emp.String(), dstCityState.String(), tp.dist)
		if tp.exp.HasAny() {
			dstCityState.ResourceStorage.Add(tp.exp)
			log.Printf("  Export: %s", tp.exp.String())
		}
		if tp.imp.HasAny() {
			emp.ResourceStorage.Add(tp.exp)
			log.Printf("  Import: %s", tp.imp.String())
		}
	}
}
