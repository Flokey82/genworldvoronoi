package genworldvoronoi

import (
	"math"
	"math/rand"
	"sort"

	"github.com/Flokey82/genbiome"
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
	for i, pop := range m.Population {
		if pop <= 0 {
			continue
		}
		// Calculate the population growth rate for the tribe.
		// Use the exponential growth model.
		newPop := float64(pop) * math.Pow(math.E, growthRate)
		if diff := newPop - float64(pop); diff < 1 {
			// Use rand to potentially grow the population by one.
			if rand.Float64() < diff {
				newPop++
			}
		}
		newPops[i] = int(newPop)

		// If the population is too high, we might need to move some people to other regions.

		// Exhaust the resources of the region.
		maxRegionPop := m.maxPopReg(i)
		if maxRegionPop <= 0 {
			maxRegionPop = 1
		}

		// Calculate the required resources for the tribe.
		resExhaustion := float64(m.Population[i]) / float64(maxRegionPop)

		// Exhaust the resources of the region.
		m.SoilExhaustion[i] += resExhaustion

		// Reduce the max sustainable population of the region.
		maxRegionPop -= int(resExhaustion)

		if newPops[i] > maxRegionPop {
			// log.Printf("Region %d has overpopulation. Population: %d, Max population: %d", i, newPops[i], maxRegionPop)
			// We need to move some people to other regions.
			diff := newPops[i] - maxRegionPop
			diff = newPops[i] / 2

			// Calculate the max population for each neighboring region.
			maxPopPerRegion := make([]int, 0, 6)
			neigbors := m.R_circulate_r(rNbs, i)
			neighborIndices := make([]int, 0, len(neigbors))

			for idx, nb := range neigbors {
				neighborIndices = append(neighborIndices, idx)
				if m.Elevation[nb] <= 0 {
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
			if maxPopPerRegion[neighborIndices[0]] >= diff {
				// Move the population to the best suitable region.
				newPops[i] -= diff
				newPops[neigbors[neighborIndices[0]]] += diff
			} else {
				// TODO: Find a better way to distribute the population.
				// Split the excess population into one or more parts with a minimum population of 10.
				remainingPopulation := diff
				for _, idx := range neighborIndices {
					movePop := min(remainingPopulation, maxPopPerRegion[idx])
					remainingPopulation -= movePop
					newPops[i] -= movePop
					newPops[neigbors[idx]] += movePop
					if remainingPopulation <= 0 {
						break
					}
				}

				// If there is still remaining population, we need to kill some of the population.
				if remainingPopulation > 0 {
					newPops[i] -= remainingPopulation
				}
			}
		}
		m.Population = newPops
	}
}
