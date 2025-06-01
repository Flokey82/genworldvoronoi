package geo

import (
	"log"
	"math"
	"sort"

	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/go_gens/utils"
)

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

/*
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
*/

// FindBestNeighbor finds the best neighbor for a region based on the given
// fitness function.
func (m *Geo) FindBestNeighbor(r int, fitFunc func(int) float64) (int, float64) {
	nbs := m.R_circulate_r(nil, r)
	bestScore := 0.0
	bestRegion := -1
	for _, nb := range nbs {
		score := fitFunc(nb)
		log.Printf("Region %d has score %f", nb, score)
		if score > bestScore {
			bestScore = score
			bestRegion = nb
		}
	}
	return bestRegion, bestScore
}

// RegionScore is a struct that holds a region and its score.
type RegionScore struct {
	Region int
	Score  float64
}

// FindBestNeighbors returns a list of the best neighbors for a region based on
// the given fitness function sorted by score.
func (m *Geo) FindBestNeighbors(r int, fitFunc func(int) float64) []RegionScore {
	nbs := m.R_circulate_r(nil, r)
	var scores []RegionScore
	for _, nb := range nbs {
		score := fitFunc(nb)
		if score < 0 {
			continue
		}
		scores = append(scores, RegionScore{Region: nb, Score: score})
	}
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})
	return scores
}

// getFitnessProximityToWater returns a fitness function with high scores for
// terrain close to water.
func (m *Geo) GetFitnessProximityToWater() func(int) float64 {
	elevs := m.Elevation.GetValues()

	var seedWater []int
	for r := range elevs {
		if m.IsRegLakeOrWaterBody(r) || m.IsRegBigRiver(r) {
			seedWater = append(seedWater, r)
		}
	}

	// Make sure we normalize the distance field so that the highest value is 1.
	distWater := m.DistMountainToWater.GetValues()
	maxDist := m.DistMountainToWater.Max
	return func(r int) float64 {
		if m.IsRegLakeOrWaterBody(r) || distWater[r] < 0 {
			return -1.0
		}
		if math.IsInf(distWater[r], 0) {
			return 0
		}
		return 1 - distWater[r]/maxDist
	}
}

// GetFitnessSteepMountains returns a fitness function with high scores for
// steep terrain close to mountains.
func (m *Geo) GetFitnessSteepMountains() func(int) float64 {
	elevs := m.Elevation.GetValues()
	steepness := m.GetSteepness()
	seedMountains := m.Mountain_r
	distMountains := m.AssignDistanceField(seedMountains, make(map[int]bool))
	return func(r int) float64 {
		if elevs[r] <= 0 {
			return -1.0
		}
		chance := steepness[r] * math.Sqrt(elevs[r])
		chance /= (distMountains[r] + 1) / 2
		return chance
	}
}

// GetFitnessInlandValleys returns a fitness function with high scores for
// terrain that is not steep and far away from coastlines, mountains, and
// oceans.
func (m *Geo) GetFitnessInlandValleys() func(int) float64 {
	steepness := m.GetSteepness()
	seedMountains := m.Mountain_r
	seedCoastlines := m.Coastline_r
	seedOceans := m.Ocean_r
	elev := m.Elevation.GetValues()

	// Combine all seed points so we can find the spots furthest away from them.
	var seedAll []int
	seedAll = append(seedAll, seedMountains...)
	seedAll = append(seedAll, seedCoastlines...)
	seedAll = append(seedAll, seedOceans...)
	distAll := m.AssignDistanceField(seedAll, make(map[int]bool))
	return func(r int) float64 {
		if elev[r] <= 0 {
			return -1.0
		}
		chance := 1 - steepness[r]
		chance *= distAll[r]
		return chance
	}
}

func (m *Geo) GetFitnessArableLand() func(int) float64 {
	// Prefer flat terrain with reasonable precipitation and at
	// lower altitudes.
	steepness := m.GetSteepness()
	elev := m.Elevation.GetValues()
	maxElev := m.Elevation.Max
	rains := m.Rainfall.GetValues()
	maxRain := m.Rainfall.Max
	flux := m.Flux.GetValues()
	maxFlux := m.Flux.Max
	return func(r int) float64 {
		temp := m.GetRegTemperature(r)
		if elev[r] <= 0 {
			return -1.0
		}
		irrigation := math.Max(rains[r]/maxRain, flux[r]/maxFlux)
		if irrigation <= 0.01 || temp <= 0 {
			return 0
		}
		chance := 1 - steepness[r]
		chance *= irrigation
		chance *= 1 - (elev[r]/maxElev)*(elev[r]/maxElev)

		return chance
	}
}

// GetFitnessClimate returns a fitness function that returns high
// scores for regions with high rainfall high temperatures, and alternatively high flux.
func (m *Geo) GetFitnessClimate() func(int) float64 {
	rains := m.Rainfall.GetValues()
	maxRain := m.Rainfall.Max
	flux := m.Flux.GetValues()
	maxFlux := m.Flux.Max
	return func(r int) float64 {
		temp := m.GetRegTemperature(r)
		if temp < 0 {
			return 0.1
		}
		scoreTemp := math.Sqrt(temp / MaxTemp)
		scoreRain := rains[r] / maxRain
		scoreFlux := math.Sqrt(flux[r] / maxFlux)
		return 0.1 + 0.9*(scoreTemp*(scoreFlux+scoreRain)/2)
	}
}

// GetFitnessSurviability returns a fitness function that returns high
// scores for regions with high rainfall high temperatures, and alternatively high flux
// or proximity to oceans.
func (m *Geo) GetFitnessSurviability() func(int) float64 {
	rains := m.Rainfall.GetValues()
	maxRain := m.Rainfall.Max
	flux := m.Flux.GetValues()
	maxFlux := m.Flux.Max

	// Survivability is increased by a neighboring ocean,
	// but since it is salt water, it will only guarantee a minimum
	// survivability.
	const minOceanFlux = 0.5

	// MinSurvivability is the minimum survivability for regions.
	const minSurv = 0.1
	const remSurv = 1 - minSurv

	elevs := m.Elevation.GetValues()
	return func(r int) float64 {
		temp := m.GetRegTemperature(r)
		if temp < 0 {
			return 0.1
		}
		scoreTemp := math.Sqrt(temp / MaxTemp)
		scoreRain := rains[r] / maxRain
		scoreFlux := math.Sqrt(flux[r] / maxFlux)
		scoreWater := max(scoreFlux, scoreRain)
		if scoreWater < minOceanFlux {
			// Check if any neighboring region is an ocean.
			var scoreOcean float64
			nbs := m.SphereMesh.R_circulate_r(nil, r)
			nbFraction := 1.0 / float64(len(nbs))
			for _, n := range nbs {
				if elevs[n] <= 0 {
					scoreOcean += nbFraction
					break
				}
			}
			scoreWater = max(scoreWater, scoreOcean)
		}
		return minSurv + remSurv*scoreTemp*scoreWater
	}
}

// GetFitnessOceanProximity returns a fitness function that returns high
// scores for regions close to the ocean.
func (m *Geo) GetFitnessOceanProximity() func(int) float64 {
	// Get elevation values.
	elevs := m.Elevation.GetValues()

	var seedOceans []int
	for r, e := range elevs {
		if e <= 0 {
			seedOceans = append(seedOceans, r)
		}
	}

	distOceans := m.AssignDistanceField(seedOceans, make(map[int]bool))
	maxDist := utils.MaxArray(distOceans)
	return func(r int) float64 {
		if elevs[r] <= 0 {
			return 0.0
		}
		v := 1 - distOceans[r]/maxDist
		return v * v * v * v
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
	return m.CalcFitnessScoreWithDistanceField(sf, m.AssignDistanceField(distSeedFunc(), make(map[int]bool)))
}

func (m *Geo) CalcFitnessScoreWithDistanceField(sf func(int) float64, regDistanceC []float64) []float64 {
	score := make([]float64, m.SphereMesh.NumRegions)

	// Get the max distance for normalizing the distance.
	maxDistC := utils.MaxArray(regDistanceC)

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
		various.KickOffChunkWorkers(m.SphereMesh.NumRegions, chunkProcessor)
	} else {
		chunkProcessor(0, m.SphereMesh.NumRegions)
	}
	return score
}
