package civ

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

	logInfo := false

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
		if logInfo {
			log.Printf("!!!%s has proposed a trade with %s, dist %.2f", c.String(), dstCity.String(), tp.dist)
			if tp.exp.HasAny() {
				log.Printf("  Export: %s", tp.exp.String())
			}
			if tp.imp.HasAny() {
				log.Printf("  Import: %s", tp.imp.String())
			}
			score := c.compare(dstCity)
			log.Printf("  Score: %.2f", score)
		}

		if tp.exp.HasAny() {
			dstCity.ResourceStorage.Add(tp.exp)
		}
		if tp.imp.HasAny() {
			c.ResourceStorage.Add(tp.exp)
		}
	}

	// HACK HACK HACK
	// Now we generate gold for the city.
	c.GoverningPeople.Gold += c.Population

	// TODO: Expenses for the city.
}

func (m *Civ) handleTradeCityState(cs *CityState) {
	logInfo := false

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
		if logInfo {
			log.Printf("!!!%s has proposed a trade with %s, dist %.2f", cs.String(), dstCityState.String(), tp.dist)
			if tp.exp.HasAny() {
				log.Printf("  Export: %s", tp.exp.String())
			}
			if tp.imp.HasAny() {
				log.Printf("  Import: %s", tp.imp.String())
			}
			score := cs.compare(dstCityState)
			log.Printf("  Score: %.2f", score)
		}

		if tp.exp.HasAny() {
			dstCityState.ResourceStorage.Add(tp.exp)
		}
		if tp.imp.HasAny() {
			cs.ResourceStorage.Add(tp.exp)
		}
	}

	// HACK HACK HACK
	// Now we generate gold for the city state.
	// We do this by taxing the cities.
	for _, c := range cs.Cities {
		wantGold := c.Population * 20 / 100 // 20% tax
		if c.GoverningPeople.Gold < wantGold {
			log.Printf("!!!%s is not generating enough gold, %d < %d", c.String(), c.GoverningPeople.Gold, wantGold)
			wantGold = c.GoverningPeople.Gold
		}
		c.GoverningPeople.Gold -= wantGold
		cs.GoverningPeople.Gold += wantGold
	}

	// TODO: Expenses for the city state.
}

func (m *Civ) handleTradeEmpire(emp *Empire) {
	logInfo := false

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
		if logInfo {
			log.Printf("!!!%s has proposed a trade with %s, dist %.2f", emp.String(), dstEmpire.String(), tp.dist)
			if tp.exp.HasAny() {
				log.Printf("  Export: %s", tp.exp.String())
			}
			if tp.imp.HasAny() {
				log.Printf("  Import: %s", tp.imp.String())
			}
		}

		if tp.exp.HasAny() {
			dstEmpire.ResourceStorage.Add(tp.exp)
		}
		if tp.imp.HasAny() {
			emp.ResourceStorage.Add(tp.exp)
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
		if logInfo {
			log.Printf("!!!%s has proposed a trade with %s, dist %.2f", emp.String(), dstCityState.String(), tp.dist)
			if tp.exp.HasAny() {
				log.Printf("  Export: %s", tp.exp.String())
			}
			if tp.imp.HasAny() {
				log.Printf("  Import: %s", tp.imp.String())
			}
		}

		if tp.exp.HasAny() {
			dstCityState.ResourceStorage.Add(tp.exp)
		}
		if tp.imp.HasAny() {
			emp.ResourceStorage.Add(tp.exp)
		}
	}

	// HACK HACK HACK
	// Now we generate gold for the empire.
	// We do this by taxing the city states and independent cities in our territory.
	for _, c := range emp.Cities {
		// Check if they are part of a city state.
		if m.CityStates.Regions[c.ID] != -1 {
			continue
		}
		wantGold := c.Population * 20 / 100 // 20% tax
		if c.GoverningPeople.Gold < wantGold {
			log.Printf("!!!%s is not generating enough gold, %d < %d", c.String(), c.GoverningPeople.Gold, wantGold)
			wantGold = c.GoverningPeople.Gold
		}
		c.GoverningPeople.Gold -= wantGold
		emp.GoverningPeople.Gold += wantGold
	}

	for _, cs := range m.getEmpireCityStates(emp) {
		wantGold := cs.GetPopulation() * 20 / 100 // 20% tax
		if cs.GoverningPeople.Gold < wantGold {
			log.Printf("!!!%s is not generating enough gold, %d < %d", cs.String(), cs.GoverningPeople.Gold, wantGold)
			wantGold = cs.GoverningPeople.Gold
		}
		cs.GoverningPeople.Gold -= wantGold
		emp.GoverningPeople.Gold += wantGold
	}

	// TODO: Expenses for the empire.
}
