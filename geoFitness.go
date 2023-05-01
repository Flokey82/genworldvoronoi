package genworldvoronoi

import "math"

type fitCache struct {
	f       func(int) float64
	fGetter func() func(int) float64
}

func newFitCache(fGetter func() func(int) float64) *fitCache {
	return &fitCache{
		fGetter: fGetter,
	}
}

func (fc *fitCache) getFunc() func(int) float64 {
	if fc.f == nil {
		fc.f = fc.fGetter()
	}
	return fc.f
}

type fitCaches struct {
	climate *fitCache
	city    *fitCache
	trading *fitCache
	steep   *fitCache
	water   *fitCache
	arable  *fitCache
}

func (m *Civ) getFitCaches() *fitCaches {
	return &fitCaches{
		climate: newFitCache(m.getFitnessClimate),
		city:    newFitCache(m.getFitnessCityDefault),
		trading: newFitCache(m.getFitnessTradingTowns),
		steep:   newFitCache(m.getFitnessSteepMountains),
		water:   newFitCache(m.getFitnessProximityToWater),
		arable:  newFitCache(m.getFitnessArableLand),
	}
}

// getFitnessProximityToWater returns a fitness function with high scores for
// terrain close to water.
func (m *Geo) getFitnessProximityToWater() func(int) float64 {
	var seedWater []int
	for r := range m.Elevation {
		if m.isRegLakeOrWaterBody(r) || m.isRegBigRiver(r) {
			seedWater = append(seedWater, r)
		}
	}

	// Make sure we normalize the distance field so that the highest value is 1.
	distWater := m.assignDistanceField(seedWater, m.RegionIsMountain)
	_, maxDist := minMax(distWater)
	return func(r int) float64 {
		if m.isRegLakeOrWaterBody(r) || distWater[r] < 0 {
			return -1.0
		}
		if math.IsInf(distWater[r], 0) {
			return 0
		}
		return 1 - distWater[r]/maxDist
	}
}

// getFitnessSteepMountains returns a fitness function with high scores for
// steep terrain close to mountains.
func (m *Geo) getFitnessSteepMountains() func(int) float64 {
	steepness := m.GetSteepness()
	seedMountains := m.mountain_r
	distMountains := m.assignDistanceField(seedMountains, make(map[int]bool))
	return func(r int) float64 {
		if m.Elevation[r] <= 0 {
			return -1.0
		}
		chance := steepness[r] * math.Sqrt(m.Elevation[r])
		chance /= (distMountains[r] + 1) / 2
		return chance
	}
}

// getFitnessInlandValleys returns a fitness function with high scores for
// terrain that is not steep and far away from coastlines, mountains, and
// oceans.
func (m *Geo) getFitnessInlandValleys() func(int) float64 {
	steepness := m.GetSteepness()
	seedMountains := m.mountain_r
	seedCoastlines := m.coastline_r
	seedOceans := m.ocean_r

	// Combine all seed points so we can find the spots furthest away from them.
	var seedAll []int
	seedAll = append(seedAll, seedMountains...)
	seedAll = append(seedAll, seedCoastlines...)
	seedAll = append(seedAll, seedOceans...)
	distAll := m.assignDistanceField(seedAll, make(map[int]bool))
	return func(r int) float64 {
		if m.Elevation[r] <= 0 {
			return -1.0
		}
		chance := 1 - steepness[r]
		chance *= distAll[r]
		return chance
	}
}

func (m *Geo) getFitnessArableLand() func(int) float64 {
	// Prefer flat terrain with reasonable precipitation and at
	// lower altitudes.
	steepness := m.GetSteepness()
	_, maxElev := minMax(m.Elevation)
	_, maxRain := minMax(m.Rainfall)
	_, maxFlux := minMax(m.Flux)
	return func(r int) float64 {
		temp := m.getRegTemperature(r, maxElev)
		if m.Elevation[r] <= 0 {
			return -1.0
		}
		irrigation := math.Max(m.Rainfall[r]/maxRain, m.Flux[r]/maxFlux)
		if irrigation < 0.1 || temp <= 0 {
			return 0
		}
		chance := 1 - steepness[r]
		chance *= irrigation
		chance *= 1 - (m.Elevation[r]/maxElev)*(m.Elevation[r]/maxElev)
		return chance
	}
}

// getFitnessClimate returns a fitness function that returns high
// scores for regions with high rainfall high temperatures, and alternatively high flux.
func (m *Geo) getFitnessClimate() func(int) float64 {
	_, maxRain := minMax(m.Rainfall)
	_, maxElev := minMax(m.Elevation)
	_, maxFlux := minMax(m.Flux)

	return func(r int) float64 {
		temp := m.getRegTemperature(r, maxElev)
		if temp < 0 {
			return 0.1
		}
		scoreTemp := math.Sqrt(temp / maxTemp)
		scoreRain := m.Rainfall[r] / maxRain
		scoreFlux := math.Sqrt(m.Flux[r] / maxFlux)
		return 0.1 + 0.9*(scoreTemp*(scoreFlux+scoreRain)/2)
	}
}

// CalcFitnessScore calculates the fitness value for all regions based on the
// given fitness function.
//
// - 'sf' is the fitness function for scoring a region.
// - 'distSeedFunc' returns a number of regions from which we maximize the distance when
// calculating the fitness score.
func (m *Geo) CalcFitnessScore(sf func(int) float64, distSeedFunc func() []int) []float64 {
	// Get distance to other seed regions returned by the distSeedFunc.
	return m.CalcFitnessScoreWithDistanceField(sf, m.assignDistanceField(distSeedFunc(), make(map[int]bool)))
}

func (m *Geo) CalcFitnessScoreWithDistanceField(sf func(int) float64, regDistanceC []float64) []float64 {
	score := make([]float64, m.SphereMesh.numRegions)

	// Get the max distance for normalizing the distance.
	_, maxDistC := minMax(regDistanceC)

	chunkProcessor := func(start, end int) {
		// Calculate the fitness score for each region
		for r := start; r < end; r++ {
			score[r] = sf(r)

			// Check if we have a valid score.
			if score[r] == -1.0 {
				continue
			}

			// Penalty for proximity / bonus for higher distance to other seed regions.
			//
			// We multiply the score by the distance to other seed regions, amplifying
			// positive scores.
			//
			// NOTE: Originally this was done with some constant values, which might be better
			// since we are here dependent on the current score we have assigned and cannot
			// recover an initially bad score caused by a low water flux.
			if math.IsInf(regDistanceC[r], 0) {
				continue
			}
			dist := (regDistanceC[r] / maxDistC)
			score[r] *= dist // originally
		}
	}

	useGoRoutines := true
	if useGoRoutines {
		kickOffChunkWorkers(m.SphereMesh.numRegions, chunkProcessor)
	} else {
		chunkProcessor(0, m.SphereMesh.numRegions)
	}
	return score
}
