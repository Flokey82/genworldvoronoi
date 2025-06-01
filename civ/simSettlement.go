package civ

import (
	"container/heap"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
)

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
