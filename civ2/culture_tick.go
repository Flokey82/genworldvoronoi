package civ2

import (
	"log"

	"github.com/Flokey82/genworldvoronoi/geo"
)

func (m *Civ) tickCultures(nDays int) {
	// Clean up the cultures.
	m.Cultures.Clean()

	// TODO: Clear the culture locations and reassign them based on settlement and city locations.
	// This way cultures withdraw when regions / cities / settlements are abandoned.
	m.Cultures.ResetRegions()

	// Now loop over all tribes, settlements, cities, anything with an assigned culture
	// and assign the culture to the region.
	for _, t := range m.Tribes.Objects {
		if t.Population == 0 {
			continue
		}
		m.Cultures.PlaceObjectAt(t.Culture, t.RegionID)
	}
	for _, s := range m.Settlements.Objects {
		if s.Population == 0 {
			continue
		}
		m.Cultures.PlaceObjectAt(s.Culture, s.ID)
	}
	for _, c := range m.Cities.Objects {
		if c.Population == 0 {
			continue
		}
		m.Cultures.PlaceObjectAt(c.Culture, c.ID)
	}

	for _, c := range m.Cultures.Objects {
		m.tickCulture(c, nDays)

		// Log the culture and its skills.
		if len(c.Skills) == 0 {
			log.Printf("Culture %s has no skills.", c.Name)
			continue
		}
		log.Printf("Culture %s has the following skills:", c.Name)
		for _, s := range c.Skills {
			log.Printf("  %s", s.Name)
		}
	}
}

func (m *Civ) tickCulture(c *Culture, nDays int) {
	// Develop the culture, depending on the location.
	// TODO: Instead of expecting one specific region, the culture should be able to develop in multiple regions.
	// For this we'd check all regions where the culture is present and calculate
	// the average properties of the regions.
	m.developSkills(c, m.getRegionStats(c), nDays)

	// TODO: Expand the culture from population centers.
}

type regionStats struct {
	numRegions      int                      // total number of regions
	numRiver        int                      // number of regions with rivers
	numLake         int                      // number of regions with lakes
	numCoastal      int                      // number of regions with coast
	numResourceType map[geo.ResourceType]int // number of regions with specific resource types
}

func (m *Civ) getRegionStats(c *Culture) *regionStats {
	stats := &regionStats{
		numResourceType: make(map[geo.ResourceType]int),
	}
	cID := c.GetID()
	rNbs := make([]int, 0, 6)
	for r, rcID := range m.Cultures.Regions {
		if rcID != cID {
			continue
		}
		stats.numRegions++

		// Check if the region has a river.
		if m.IsRegRiver(r) {
			stats.numRiver++
		}

		// Check if the region has a neighboring lake.
		for _, nb := range m.R_circulate_r(rNbs, r) {
			if m.IsRegLake(nb) {
				stats.numLake++
				break
			}
		}

		// Check if the region is coastal.
		if m.RegCellTypes.Values[r] == geo.CellTypeCoastalLand {
			stats.numCoastal++
		}

		// Check for each resource type.
		for rt := geo.ResourceType(0); rt < geo.ResourceTypeMax; rt++ {
			if m.ResourceLocations.HasType(r, rt) {
				stats.numResourceType[rt]++
			}
		}
	}
	return stats
}
