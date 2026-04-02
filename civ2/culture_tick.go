package civ2

import (
	"log"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
)

func (m *Civ) tickCultures(nDays int) {
	// Clean up the cultures.
	m.Cultures.Clean()

	// Clear the culture locations.
	m.Cultures.ResetRegions()

	// Gather seeds (where population centers are).
	// We use the culture's origin ID as the seed point.
	seeds := make([]int, 0, len(m.Cultures.Objects))
	for _, c := range m.Cultures.Objects {
		seeds = append(seeds, c.ID)
	}

	// Expand cultures.
	m.expandCultures(seeds)

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

func (m *Civ) expandCultures(seeds []int) {
	// 1. Map culture origin ID to culture object.
	idToCulture := make(map[int]*Culture)
	for _, c := range m.Cultures.Objects {
		idToCulture[c.ID] = c
	}

	// 2. Performance weights.
	rCellType := m.GetRegCellTypes()
	elevs := m.Elevation.GetValues()
	maxElev := m.Elevation.Max

	territoryWeightFunc := m.getTerritoryWeightFunc()
	biomeWeight := m.getTerritoryBiomeWeightFunc()

	// 3. Expand territories.
	m.Cultures.Regions = m.regPlaceNTerritoriesCustom(m.Cultures.Regions, seeds, func(o, u, v int) float64 {
		c := idToCulture[o]
		if c == nil {
			return -1
		}

		// Get the cost to expand to this biome.
		gotBiome := m.GetAzgaarRegionBiome(v, elevs[v]/maxElev)
		biomePenalty := biomeWeight(o, u, v) * float64(genbiome.AzgaarBiomeMovementCost[gotBiome]) / 100

		// Check if we have a non-native biome, if so we apply an additional penalty.
		biomePenalty *= c.Type.BiomeCost(gotBiome)

		cellTypePenalty := c.Type.CellTypeCost(rCellType[v])
		return biomePenalty + cellTypePenalty*territoryWeightFunc(o, u, v)/c.Type.Expansionism()
	})
}

func (m *Civ) tickCulture(c *Culture, nDays int) {
	// Develop the culture, depending on the location.
	m.developSkills(c, m.getRegionStats(c), nDays)
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
