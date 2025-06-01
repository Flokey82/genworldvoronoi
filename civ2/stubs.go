package civ2

import (
	"math"

	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/genworldvoronoi/geo"
)

type Religion = civ.Religion

func (m *Civ) GetReligion(id int) *Religion {
	for _, c := range m.Religions.Objects {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func (m *Civ) GetTradeRoutes() ([][]int, [][]int) {
	return nil, nil
}

func (m *Civ) GetTradeRoutesInLatLonBB(minLat, minLon, maxLat, maxLon float64) [][]int {
	return nil
}

func (m *Civ) GenerateCityFlavorText(c *City, p geo.RegProperty) string {
	return ""
}

// CalcCityScore calculates the fitness value for settlements for all regions.
//
// 'sf': Fitness function for scoring a region.
// 'distSeedFunc': Returns a number of regions from which we maximize the distance.
func (m *Civ) CalcCityScore(sf func(int) float64, distSeedFunc func() []int) []float64 {
	elevs := m.Elevation.GetValues()
	sfCity := func(r int) float64 {
		if elevs[r] <= 0 || m.Waterpool[r] > 0 {
			return -1.0 // Skip regions below sea level or in water.
		}
		return sf(r)
	}
	return m.CalcFitnessScore(sfCity, distSeedFunc)
}

func (m *Civ) GetFitnessCityDefault() func(int) float64 {
	flux := m.Flux.GetValues()
	maxFlux := m.Flux.Max
	steepness := m.GetSteepness()
	elevs := m.Elevation.GetValues()

	// WARNING: Using this will prevent us from using the fitness function concurrently.
	out_r := make([]int, 0, 8)

	return func(r int) float64 {
		// If we are below (or at) sea level, or we are in a pool of water,
		// assign lowest score and continue.
		if elevs[r] <= 0 || m.Waterpool[r] > 0 {
			return -1.0
		}

		// Visit all neighbors and modify the score based on their properties.
		var hasWaterBodyBonus bool
		nbs := m.SphereMesh.R_circulate_r(out_r, r)

		// Initialize fitness score with the normalized flux value.
		// This will favor placing cities along (and at the end of)
		// large rivers.
		score := math.Sqrt(flux[r] / maxFlux)
		for _, nb := range nbs {
			// Add bonus if near ocean or lake.
			if m.IsRegBelowOrAtSeaLevelOrPool(nb) {
				// If a neighbor is below (or at) sea level, or a lake,
				// we increase the fitness value and reduce it by a fraction,
				// depending on the size of the lake or ocean it is part of.
				//
				// We only apply this bonus once.
				if hasWaterBodyBonus {
					continue
				}

				// If nb is part of a waterbody (ocean) or lake, we reduce the score by a constant factor.
				// The larger the waterbody/lake, the smaller the penalty, which will favor larger waterbodies.
				if wbSize := m.GetRegLakeOrWaterBodySize(nb); wbSize > 0 {
					hasWaterBodyBonus = true
					score += 0.55 * (1 - 1/(float64(wbSize)+1e-9))
				}
			} else {
				// If the sourrounding terrain is flat, we get a bonus.
				stp := steepness[nb]
				score += 0.5 * (1.0 - stp*stp) / float64(len(nbs))
			}

			// TODO:
			// - Consider biome
			// - Consider sediment/fertility of land.
			// - Add bonus for mountain proximity (mines, resources)
			// - Add bonus for proximity to other cities / trade routes.
		}

		// The steeper the terrain, the less likely it is to be settled.
		stp := steepness[r]
		score *= 1.0 - (stp * stp)
		return score
	}
}

func (m *Civ) CalcCityScoreWithDistanceField(sf func(int) float64, regDistanceC []float64) []float64 {
	elevs := m.Elevation.GetValues()
	sfCity := func(r int) float64 {
		// If we are below (or at) sea level, or we are in a pool of water,
		// assign lowest score and continue.
		if elevs[r] <= 0 || m.Waterpool[r] > 0 {
			return -1.0
		}
		return sf(r)
	}
	return m.CalcFitnessScoreWithDistanceField(sfCity, regDistanceC)
}
