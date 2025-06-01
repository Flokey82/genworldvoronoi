package civ

func (s *simState) handleResources(t *Tribe) {
	var storage *ComboStorage
	var incNeighbors bool
	if t.Type <= TribeTypeSettling {
		incNeighbors = false
		storage = t.ComboStorage
	} else {
		incNeighbors = true
		storage = t.Settlement.ComboStorage
	}

	// Generate resources based on the local resources.
	s.m.handleResources(t.RegionID, storage, incNeighbors, t.Population, t)

	// Spend resources.
	s.m.spendResources(t.RegionID, storage, t.Population, t)

	// If we lack food or firewood, we need to find a strategy to get more.
	// - We can trade for it.
	// - We can produce more (more efficiently, or just more of it)

	// Further we should check here how much housing we need and how much we have.

	// Build stuff.
	s.buildThings(t)

	// TODO: Maintain stuff.
	// s.maintainThings(t)

	// TODO: We have a resource budget.
	// - We produce resources
	// - We consume resources
	// - We need resources to build stuff as one-time costs.

	// So first we need to establish our current resource situation.
	// ... then we figure out what we need to build and maintain.
}
