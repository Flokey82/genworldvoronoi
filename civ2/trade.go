package civ2

import (
	"fmt"
)

const (
	tradeRadius      = 900.0 // km
	surplusThreshold = 100   // Amount above which we consider a resource for export
	needThreshold    = 10    // Amount below which we consider a resource for import
)

// AnalyzeNeeds returns a list of resources the entity is low on.
func (m *Civ) AnalyzeNeeds(e peopleThing) []ResAmount {
	s := e.GetStorage()
	if s == nil {
		return nil
	}

	var needs []ResAmount
	// For now, let's just focus on Food and Wood as primary needs if they are low.
	if s.Resources[ResFood] < needThreshold {
		needs = append(needs, ResAmount{Res: ResFood, Amount: needThreshold - s.Resources[ResFood]})
	}

	// Check for any resource type we might be missing entirely but know about.
	// This is a simplified version.
	return needs
}

// AnalyzeSurplus returns a list of resources the entity has in excess.
func (m *Civ) AnalyzeSurplus(e peopleThing) []ResAmount {
	s := e.GetStorage()
	if s == nil {
		return nil
	}

	var surplus []ResAmount
	for res, amount := range s.Resources {
		if amount > surplusThreshold {
			surplus = append(surplus, ResAmount{Res: res, Amount: amount - surplusThreshold})
		}
	}
	return surplus
}

// EstablishTradeRoute attempts to create a trade connection between two entities.
func (m *Civ) EstablishTradeRoute(src, dst peopleThing) bool {
	// Simple distance check first.
	dist := m.Geo.GetDistance(src.GetID(), dst.GetID())
	if dist > tradeRadius {
		return false
	}

	// Pathfinding and other logic would go here.
	// For now, we'll create a direct "virtual" path if they are within range.
	route := &TradeRoute{
		Source:      src,
		Destination: dst,
		Active:      true,
	}

	// Determine what to trade.
	srcSurplus := m.AnalyzeSurplus(src)
	dstNeeds := m.AnalyzeNeeds(dst)

	for _, s := range srcSurplus {
		for _, n := range dstNeeds {
			if s.Res == n.Res {
				amount := min3(s.Amount, n.Amount, 10) // Trade in small batches
				route.Exp = append(route.Exp, ResAmount{Res: s.Res, Amount: amount})
			}
		}
	}

	if len(route.Exp) > 0 || len(route.Imp) > 0 {
		m.TradeRoutes = append(m.TradeRoutes, route)
		m.History.AddEvent("trade", fmt.Sprintf("Established trade route between %d and %d", src.GetID(), dst.GetID()), src.Ref())
		return true
	}

	return false
}

// tickTrade processes all active trade routes.
func (m *Civ) tickTrade(nDays int) {
	for i := 0; i < len(m.TradeRoutes); i++ {
		r := m.TradeRoutes[i]
		if !r.Active {
			continue
		}

		// Perform the actual transfer.
		srcStorage := r.Source.GetStorage()
		dstStorage := r.Destination.GetStorage()

		for _, item := range r.Exp {
			if srcStorage.RemoveResource(item.Res, item.Amount) {
				dstStorage.AddResource(item.Res, item.Amount)
				
				// Generate wealth/gold for both parties.
				r.Source.GetGoverningPeople().Gold += item.Amount
				r.Destination.GetGoverningPeople().Gold += item.Amount / 2
			}
		}

		// TODO: Validate if the route should stay active.
	}
}

func min3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}
