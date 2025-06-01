package civ

import (
	"container/heap"
	"log"
	"math"

	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/genworldvoronoi/various"
)

// getTerritoryCultureWeightFunc returns a weight function which returns a penalty
// for expanding from a region into a region with a different culture.
// An additional penalty is applied if the destination region has a different
// culture than the origin region.
func (m *Civ) getTerritoryCultureWeightFunc() func(o, u, v int) float64 {
	return func(o, u, v int) float64 {
		var penalty float64
		// TODO: Compare culture expansionism?
		// If the destination has a higher culture expansionism than the
		// origin culture, then it's less likely to expand into that territory.
		if m.Cultures.Regions[o] != m.Cultures.Regions[v] {
			penalty += 0.25
		}
		if m.Cultures.Regions[u] != m.Cultures.Regions[v] {
			penalty += 0.75
		}
		return penalty
	}
}

// getTerritoryBiomeWeightFunc returns a weight function which returns a penalty
// for expanding from a region into a region with a different biome.
func (m *Civ) getTerritoryBiomeWeightFunc() func(o, u, v int) float64 {
	biomeFunc := m.GetRegWhittakerModBiomeFunc()
	climatFunc := m.GetFitnessClimate()
	return func(o, u, v int) float64 {
		var penalty float64

		// Try to stick with original biome?
		// if biomeFunc(o) != biomeFunc(v) {
		//	// Penalty is higher for inhospitable climates.
		//	penalty += 0.05 * (1 - (climatFunc(o)+climatFunc(v))/2)
		// }

		// Changes in biomes are also considered natural boundaries.
		if biomeFunc(u) != biomeFunc(v) {
			// Penalty is higher for inhospitable climates.
			penalty += 0.1 + 0.9*(1-(climatFunc(u)+climatFunc(v))/2)
		}
		return penalty
	}
}

// getTerritoryWeightFunc returns a weight function which returns a penalty
// depending on the slope of the terrain, the distance, and changes in
// flux (river crossings).
func (m *Civ) getTerritoryWeightFunc() func(o, u, v int) float64 {
	// Get maxFlux and maxElev for normalizing.
	maxFlux := m.Flux.Max
	flux := m.Flux.GetValues()
	maxElev := m.Elevation.Max
	elevs := m.Elevation.GetValues()

	return func(o, u, v int) float64 {
		// Don't cross from water to land and vice versa,
		// don't do anything below or at sea level.
		if (elevs[u] > 0) != (elevs[v] > 0) || elevs[v] <= 0 {
			return -1
		}

		// Calculate horizontal distance.
		ulat := m.LatLon[u][0]
		ulon := m.LatLon[u][1]
		vlat := m.LatLon[v][0]
		vlon := m.LatLon[v][1]
		horiz := various.Haversine(ulat, ulon, vlat, vlon) / (2 * math.Pi)

		// TODO: Maybe add a small penalty based on distance from the capital?
		// oLat := m.r_latLon[o][0]
		// oLon := m.r_latLon[o][1]
		// originDist := haversine(vlat, vlon, oLat, oLon) / (2 * math.Pi)

		// Calculate vertical distance.
		vert := (elevs[v] - elevs[u]) / maxElev
		if vert > 0 {
			vert /= 10
		}
		diff := 1 + 0.25*math.Pow(vert/horiz, 2)
		diff += 100 * math.Sqrt(flux[u]/maxFlux)
		if elevs[u] <= 0 {
			diff = 100
		}
		return horiz * diff
	}
}

// NOTE: The weight function takes three parameters:
// o: The origin/seed region
// u: The region we expand from
// v: The region we expand to
func (m *Civ) regPlaceNTerritoriesCustom(terr, seedPoints []int, weight func(o, u, v int) float64) []int {
	var queue geo.AscPriorityQueue
	heap.Init(&queue)
	outReg := make([]int, 0, 8)

	// If force rebuild is enabled, we will rebuild all territories from scratch.
	forceRebuild := false

	// Place the seed points and add the neighbors that we try to expand into to the queue.
	for i := 0; i < len(seedPoints); i++ {
		if !forceRebuild && terr[seedPoints[i]] >= 0 {
			continue
		}
		terr[seedPoints[i]] = seedPoints[i]
		for _, v := range m.SphereMesh.R_circulate_r(outReg, seedPoints[i]) {
			newdist := weight(seedPoints[i], seedPoints[i], v)
			if newdist < 0 {
				continue
			}
			heap.Push(&queue, &geo.QueueEntry{
				Score:       newdist,
				Origin:      seedPoints[i],
				Destination: v,
			})
		}
	}

	// Extend territories until the queue is empty.
	for queue.Len() > 0 {
		u := heap.Pop(&queue).(*geo.QueueEntry)
		if terr[u.Destination] >= 0 {
			continue
		}
		terr[u.Destination] = u.Origin
		for _, v := range m.SphereMesh.R_circulate_r(outReg, u.Destination) {
			if terr[v] >= 0 {
				continue
			}
			newdist := weight(u.Origin, u.Destination, v)
			if newdist < 0 {
				continue
			}
			heap.Push(&queue, &geo.QueueEntry{
				Score:       u.Score + newdist,
				Origin:      u.Origin,
				Destination: v,
			})
		}
	}
	return terr
}

func (m *Civ) expandTerritoriesAggressive(terr, seedPoints []int, weight func(o, u, v int) float64) []int {
	logInfo := false

	var queue geo.AscPriorityQueue
	heap.Init(&queue)
	outReg := make([]int, 0, 8)
	// TODO: Introduce a token system that limits the number of expansions to a certain number.
	// This number should depend on the prosperity of the territories and the culture expansionism.
	seedTokens := make(map[int]int)
	for _, s := range seedPoints {
		seedTokens[s] = 1
	}

	// Place the seed points and add the neighbors that we try to expand into to the queue.
	for i := 0; i < len(seedPoints); i++ {
		terr[seedPoints[i]] = seedPoints[i]
		for _, v := range m.SphereMesh.R_circulate_r(outReg, seedPoints[i]) {
			// Add the destination region to the queue if the weight is positive.
			if newdist := weight(seedPoints[i], seedPoints[i], v); newdist >= 0 {
				heap.Push(&queue, &geo.QueueEntry{
					Score:       newdist,
					Origin:      seedPoints[i],
					Destination: v,
				})
			}
		}
	}

	seenRegionPerSeed := make(map[int]map[int]bool)
	for _, s := range seedPoints {
		seenRegionPerSeed[s] = make(map[int]bool)
	}

	// Extend territories until the queue is empty.
	for queue.Len() > 0 {
		u := heap.Pop(&queue).(*geo.QueueEntry)
		seed := terr[u.Origin]
		if seedTokens[seed] <= 0 || seenRegionPerSeed[seed][u.Destination] {
			continue
		}
		seenRegionPerSeed[seed][u.Destination] = true

		// Check if we already own the destination territory or if the destination
		// owns itself (capital city, cultural center, etc.), and if we have any tokens left.
		if terr[u.Destination] != seed {
			if terr[u.Destination] == u.Destination || seedTokens[seed] <= 0 {
				continue
			}

			// If the destination is occupied, we compare the scores of the opposing costs.
			// ws: Cost of the destination territory to expand into the origin territory.
			// So if us expanding into the region is cheaper than the region expanding into us,
			// then we can expand into the region.
			if terr[u.Destination] >= 0 {
				// Check if the defending region would have an easier time pushing back into the expanding region
				// than the attacker expanding into the destination region.
				ws := weight(terr[u.Destination], u.Destination, u.Origin)
				// TODO: This doesn't take into account the full chain of expansion costs.
				// We should either find a way to sum this up, or find a different way to figure out
				// if we can expand into the region.
				if ws < u.Score {
					// log.Printf("A Could not expand; Territory %d (%f) is stronger than %d (%f)", terr[u.Destination], ws, u.Origin, u.Score)
					continue
				}
				if logInfo {
					log.Printf("Expanding territory %d (%f) to %d (%f)", terr[u.Destination], ws, u.Origin, u.Score)
				}
			} else if logInfo {
				log.Printf("Expanding territory %d to %d", u.Origin, u.Destination)
			}

			// We can successfully expand into the destination territory, so we spend a token.
			seedTokens[seed]--
			terr[u.Destination] = seed
		}

		// If we don't own this territory, we can't expand into its neighbors.
		if terr[u.Destination] != seed {
			continue
		}

		// Add the destination region to the queue.
		// TODO: We have to avoid checking the same regions over and over again.
		// So maybe keep a record of all regions where we have visited neighbors for
		// each seed.
		for _, v := range m.SphereMesh.R_circulate_r(outReg, u.Destination) {
			newdist := weight(u.Origin, u.Destination, v)
			if newdist < 0 {
				continue
			}
			heap.Push(&queue, &geo.QueueEntry{
				Score:       u.Score + newdist,
				Origin:      u.Origin,
				Destination: v,
			})
		}
	}
	return terr
}

/*
func (m *Civ) expandTerritoriesAggressive(terr, seedPoints []int, weight func(o, u, v int) float64) []int {
	outReg := make([]int, 0, 8)

	// TODO: Introduce a token system that limits the number of expansions to a certain number.
	// This number should depend on the prosperity of the territories and the culture expansionism.
	seedTokens := make(map[int]int)
	for _, s := range seedPoints {
		seedTokens[s] = 1
	}

	// Let's find all regions of the given territory that are border regions.
	// We return them as pairs of [origin, destination] where origin is the
	// region ID of the territory we own and destination is the region ID of
	// the neighboring territory.
	findBorderRegions := func(terrID int) [][2]int {
		var res [][2]int
		for r, va := range terr {
			if va != terrID {
				continue
			}
			for _, v := range m.SphereMesh.R_circulate_r(outReg, r) {
				if terr[v] == terrID {
					continue
				}
				res = append(res, [2]int{r, v})
			}
		}
		return res
	}

	// TODO: Extract some of this to a separate function, so that I can
	// call it for a specific culture or region on a tick.
	queue := make([]*geo.QueueEntry, 0, m.SphereMesh.NumRegions)

	// Each seed point represents the capital city of a territory.
	for _, seed := range seedPoints {
		log.Printf("Expanding territory of %d", seed)
		terr[seed] = seed

		// Find all border regions and add them to the queue.
		borderRegs := findBorderRegions(seed)
		log.Printf("Found %d border regions", len(borderRegs))
		for _, v := range borderRegs {
			newdist := weight(seed, v[0], v[1])
			if newdist < 0 {
				continue // Region is not suited for us.
			}
			// NOTE: The score usually takes into account all scores summed up
			// from the origin to the destination!!! We don't do that yet here,
			// leading to a suboptimal expansion (stringy territories, etc.)
			// We need to somehow compensate for that or just adopt the original
			// algorithm.
			queue = append(queue, &geo.QueueEntry{
				Score:       newdist,
				Origin:      v[0],
				Destination: v[1],
			})
		}
	}

	// Sort the queue by ascending score.
	// The score represents a "cost" of expanding from the origin to the destination,
	// so we want to expand into the lowest cost (score) territories first.
	sort.Slice(queue, func(i, j int) bool {
		return queue[i].Score > queue[j].Score
	})

	// Iterate over the queue and expand the territories.
	for _, u := range queue {
		seed := terr[u.Origin]
		// Check if we already own the destination territory or if the destination
		// owns itself (capital city, cultural center, etc.), and if we have any tokens left.
		if terr[u.Destination] == seed || terr[u.Destination] == u.Destination || seedTokens[seed] <= 0 {
			continue
		}

		// If the destination is occupied, we compare the scores of the opposing costs.
		// ws: Cost of the destination territory to expand into the origin territory.
		// So if us expanding into the region is cheaper than the region expanding into us,
		// then we can expand into the region.
		if terr[u.Destination] >= 0 {
			// Check if the defending region would have an easier time pushing back into the expanding region
			// than the attacker expanding into the destination region.
			ws := weight(terr[u.Destination], u.Destination, u.Origin)
			if ws < u.Score {
				log.Printf("A Could not expand; Territory %d (%f) is stronger than %d (%f)", terr[u.Destination], ws, u.Origin, u.Score)
				continue
			}
			log.Printf("Expanding territory %d (%f) to %d (%f)", terr[u.Destination], ws, u.Origin, u.Score)
		} else {
			log.Printf("Expanding territory %d to %d", u.Origin, u.Destination)
		}

		// We can successfully expand into the destination territory, so we spend a token.
		seedTokens[seed]--
		terr[u.Destination] = seed
	}
	return terr
}*/

func (m *Civ) rRelaxTerritories(terr []int, n int) {
	outReg := make([]int, 0, 8)
	for i := 0; i < n; i++ {
		// TODO: Make sure that we can put some type of constraints on
		// how much a territory can move.
		for r, t := range terr {
			if t < 0 {
				continue
			}
			var nbCountOtherTerr, nbCountSameTerr int
			otherTerr := -1
			for _, v := range m.SphereMesh.R_circulate_r(outReg, r) {
				if terr[v] != t {
					nbCountOtherTerr++
					otherTerr = terr[v]
				} else {
					nbCountSameTerr++
				}
			}
			if nbCountOtherTerr > nbCountSameTerr && otherTerr >= 0 {
				terr[r] = otherTerr
			}
		}
	}
}

// getTerritoryNeighbors returns a list of territories neighboring the
// territory with the ID 'r' based on the provided slice of len NumRegions
// which maps the index (region id) to their respective territory ID.
func (m *Civ) getTerritoryNeighbors(r int, r_terr []int) []int {
	var res []int
	seenTerritories := make(map[int]bool)
	outReg := make([]int, 0, 8)
	for i, rg := range r_terr {
		if rg != r {
			continue
		}
		for _, nb := range m.SphereMesh.R_circulate_r(outReg, i) {
			// Determine territory ID.
			terrID := r_terr[nb]
			if terrID < 0 || terrID == r || seenTerritories[terrID] {
				continue
			}
			seenTerritories[terrID] = true
			res = append(res, terrID)
		}
	}
	return res
}
