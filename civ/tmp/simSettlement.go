package civ

import (
	"container/heap"
	"math"
	"math/rand"
	"sort"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
)

func (m *Civ) InitSimSettling() {
	// Set up the suitability of the regions for population growth.
	m.calculateSuitability()
	// Initialize exhaustion of resources.
	// This will keep track of the exhaustion of resources for each region.
	m.SoilExhaustion = make([]float64, m.NumRegions)

	// Find the best place for the cradle of civilization.
	// Since we only have one species for now (humans), we will just start
	// with a 'steppe' region, and then expand from there incrementally.
	// Now we pick a suitable region to start with (steppe/grassland).
	bestRegion := m.pickCradleOfCivilization(genbiome.WhittakerModBiomeTemperateGrassland)
	if bestRegion == -1 {
		panic("no suitable region found")
	}

	// Initial population.
	const initialPopulation = 100
	m.Population = initRegionSlice(m.NumRegions)
	m.Population[bestRegion] = initialPopulation
}

func (m *Civ) tickSimSettlement(rNbs []int) {
	// Get elevation values.
	elevs := m.Elevation.GetValues()

	// Regenerate the land.
	// TODO: Deduplicate.
	for i, ex := range m.SoilExhaustion {
		if ex < 0.9 {
			m.SoilExhaustion[i] = 0
		} else if ex != 0 {
			m.SoilExhaustion[i] *= 0.9
		}
	}

	// Use this as a temporary slice to store the new populations.
	newPops := make([]int, m.NumRegions)
	copy(newPops, m.Population)

	// Calculate the population growth rate for each region.
	for r, pop := range m.Population {
		if pop <= 0 {
			continue
		}
		// Calculate the population growth rate for the tribe.
		// Use the exponential growth model.
		newPop := float64(pop) * math.Pow(math.E, growthRateSettled)
		if diff := newPop - float64(pop); diff < 1 {
			// Use rand to potentially grow the population by one.
			if rand.Float64() < diff {
				newPop++
			}
		}
		newPops[r] = int(newPop)

		// If the population is too high, we might need to move some people to other regions.

		// Exhaust the resources of the region.
		maxRegionPop := m.maxPopReg(r)
		if maxRegionPop <= 0 {
			maxRegionPop = 1
		}

		// Calculate the required resources for the tribe.
		resExhaustion := float64(m.Population[r]) / float64(maxRegionPop)

		// Exhaust the resources of the region.
		m.SoilExhaustion[r] += resExhaustion

		// Reduce the max sustainable population of the region.
		maxRegionPop -= int(resExhaustion)

		if newPops[r] > maxRegionPop {
			// log.Printf("Region %d has overpopulation. Population: %d, Max population: %d", i, newPops[i], maxRegionPop)
			// We need to move some people to other regions.
			diff := newPops[r] - maxRegionPop
			diff = newPops[r] / 2

			// Calculate the max population for each neighboring region.
			maxPopPerRegion := make([]int, 0, 6)
			neigbors := m.R_circulate_r(rNbs, r)
			neighborIndices := make([]int, 0, len(neigbors))

			for idx, nb := range neigbors {
				neighborIndices = append(neighborIndices, idx)
				if elevs[nb] <= 0 {
					// Ocean regions are not suitable for population growth.
					maxPopPerRegion = append(maxPopPerRegion, 0)
					continue
				}
				capacity := m.maxPopReg(nb) - int(m.SoilExhaustion[nb]) - m.Population[nb] // NOTE: We use the original population here.
				maxPopPerRegion = append(maxPopPerRegion, capacity)
			}

			// Sort the neighbor indices by max population.
			sort.Slice(neighborIndices, func(i, j int) bool {
				return maxPopPerRegion[neighborIndices[i]] < maxPopPerRegion[neighborIndices[j]]
			})

			// Move the excess population to the best suitable region.
			// If the new region can sustain the population, we move the population there.
			if nb0 := neigbors[neighborIndices[0]]; maxPopPerRegion[nb0] >= diff {
				// Move the population to the best suitable region.
				newPops[r] -= diff
				newPops[nb0] += diff

				// If the region has not been settled yet, we need to update the time of settlement.
				if m.Settled[nb0] == 0 {
					m.Settled[nb0] = m.Geo.Calendar.GetYear()
				}
			} else {
				// TODO: Find a better way to distribute the population.
				// Split the excess population into one or more parts with a minimum population of 10.
				remainingPopulation := diff
				for _, idx := range neighborIndices {
					nbr := neigbors[idx]
					movePop := min(remainingPopulation, maxPopPerRegion[nbr])
					remainingPopulation -= movePop
					newPops[r] -= movePop
					newPops[nbr] += movePop

					// If the region has not been settled yet, we need to update the time of settlement.
					if m.Settled[nbr] == 0 {
						m.Settled[nbr] = m.Geo.Calendar.GetYear()
					}

					// If there is no more population to move, we can break.
					if remainingPopulation <= 0 {
						break
					}
				}

				// If there is still remaining population, we need to kill some of the population.
				if remainingPopulation > 0 {
					newPops[r] -= remainingPopulation
				}
			}
		}
		m.Population = newPops
	}
}

func (m *Civ) GenerateTimeOfSettlement() {
	// First we pick a "suitable" region where the cradle of civilization
	// will be located.
	// Since we only have one species for now (humans), we will just start
	// with a 'steppe' region, and then expand from there incrementally.
	// Now we pick a suitable region to start with (steppe/grassland).
	bestRegion := m.pickCradleOfCivilization(genbiome.WhittakerModBiomeTemperateGrassland)
	if bestRegion == -1 {
		panic("no suitable region found")
	}

	// How long it takes for the civilization to expand to a region is
	// determined by the characteristics of the region and if there are
	// more suitable regions nearby. So we will use a priority queue
	// to determine the next region to expand to.
	var queue geo.AscPriorityQueue
	heap.Init(&queue)

	// 'settleTime' is the time when a region was settled.
	settleTime := initTimeSlice(m.SphereMesh.NumRegions)

	// We will start with a settlement time of 0.
	settleTime[bestRegion] = 0

	// terrainDifficulty returns high scores for difficult terrain.
	terrainDifficulty := m.getTerritoryWeightFunc()

	// Get elevation values.
	elevs := m.Elevation.GetValues()

	// terrainArable returns high scores if the terrain is arable.
	//terrainArable := m.getFitnessArableLand()

	weight := func(o, u, v int) float64 {
		// Terrain weight.
		// TODO: We should use a slightly different weight function
		// that doesn't treat up- and downhill differently.
		// Also, the penalty should be way higher for "impassable"
		// terrain.
		terrDifficulty := terrainDifficulty(bestRegion, u, v)

		// TODO: The duration that it takes to settle a region should
		// depend on how many regions there are in total (the size of
		// the regions).
		const (
			baseDifficultyTime = 2000 // Time to settle land per difficulty.

			// Time to cross land and sea.
			timeToCrossToLand = 1   // Crossing from sea to land.
			timeToCrossSea    = 20  // Crossing from sea to sea.
			timeToCrossToSea  = 200 // Crossing from land to sea (time to build a boat).
		)

		// If the terrain weight is positive (or zero), the destination region is land.
		if terrDifficulty >= 0 {
			// Settlement on land takes a fraction of 2000 years per (unit) region.
			// 'terrWeight' already takes the actual distance between the regions
			// into account.
			return float64(settleTime[u]) + baseDifficultyTime*terrDifficulty // * (1-terrainArable(v))
		}

		// If the terrain weight is negative, the source- and/or destination region is ocean.
		// This means, we need boats to get there, which will require more time.
		// TODO: For crossing the ocean, we need to wait for boats to be invented?
		var timeReqired float64
		if elevs[v] > 0 {
			// If we were at sea and arrive at land, we only need a year to disembark.
			timeReqired = timeToCrossToLand
		} else if elevs[v] <= 0 && elevs[u] <= 0 {
			// Once we are traveling at sea, we travel at a speed of 20 years
			// per (unit) region.
			timeReqired = timeToCrossSea
		} else {
			// We were on land, but the destination is at sea,
			// it takes us 200 years to build a boat.
			timeReqired = timeToCrossToSea
		}

		// Calculate the actual distance between the two regions,
		// so we are independent of the mesh resolution.
		actualDist := m.GetDistance(u, v)
		return float64(settleTime[u]) + timeReqired*actualDist
	}

	// Now add the region neighbors to the queue.
	out_r := make([]int, 0, 8)
	for _, n := range m.R_circulate_r(out_r, bestRegion) {
		heap.Push(&queue, &geo.QueueEntry{
			Origin:      bestRegion,
			Score:       weight(bestRegion, bestRegion, n),
			Destination: n,
		})
	}

	// Expand settlements until we have settled all regions.
	for queue.Len() > 0 {
		u := heap.Pop(&queue).(*geo.QueueEntry)

		// Check if the region has already been settled.
		if settleTime[u.Destination] >= 0 {
			continue
		}

		// The higher the score, the more difficult it is to settle there,
		// and the longer it took to settle there.
		settleTime[u.Destination] = int64(u.Score)
		for _, v := range m.SphereMesh.R_circulate_r(out_r, u.Destination) {
			// Check if the region has already been settled.
			if settleTime[v] >= 0 {
				continue
			}
			newdist := weight(u.Origin, u.Destination, v)
			if newdist < 0 {
				continue
			}
			heap.Push(&queue, &geo.QueueEntry{
				Score:       newdist,
				Origin:      u.Destination,
				Destination: v,
			})
		}
	}

	m.Settled = settleTime
}
