package civ

// findNewPath will find a new path to the destination region and assign it to the tribe.
func (s *simState) findNewPath(t *Tribe, destination int) bool {
	newPath, found := PlanPath(s.navCache.GetTile(t.RegionID), s.navCache.GetTile(destination))
	if found {
		if len(newPath.Steps) == 1 {
			panic("path length is 1")
		}
		t.SetPath(newPath)
	}
	return found
}

// tribeMigrationCost returns the modified cost of migrating from one region to another and if the migration is possible.
func (s *simState) tribeMigrationCustomCost(from, to *NavTile, cost float64) (float64, bool) {
	if s.tribeAtRegion[to.ID] != nil {
		return 0, false
	}

	// Penalty if the neighbor is a city.
	if s.m.Cities.GetIDAt(to.ID) != -1 {
		cost *= 4.0
	}

	// Penalty for crossing into a new territory
	if s.m.Empires.Regions[from.ID] != s.m.Empires.Regions[to.ID] {
		cost *= 2.0
	}

	// Penalty for crossing into a new culture.
	if s.m.Cultures.Regions[from.ID] != s.m.Cultures.Regions[to.ID] {
		cost *= 2.0
	}
	return cost, true
}
