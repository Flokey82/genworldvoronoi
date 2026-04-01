package civ2

import (
	"container/heap"
	"math"

	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/genworldvoronoi/various"
)

// getTerritoryCultureWeightFunc returns a weight function which returns a penalty
// for expanding from a region into a region with a different culture.
func (m *Civ) getTerritoryCultureWeightFunc() func(o, u, v int) float64 {
	return func(o, u, v int) float64 {
		var penalty float64
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
		if biomeFunc(u) != biomeFunc(v) {
			penalty += 0.1 + 0.9*(1-(climatFunc(u)+climatFunc(v))/2)
		}
		return penalty
	}
}

// getTerritoryWeightFunc returns a weight function which returns a penalty
// depending on the slope of the terrain, the distance, and changes in
// flux (river crossings).
func (m *Civ) getTerritoryWeightFunc() func(o, u, v int) float64 {
	maxFlux := m.Flux.Max
	flux := m.Flux.GetValues()
	maxElev := m.Elevation.Max
	elevs := m.Elevation.GetValues()

	return func(o, u, v int) float64 {
		if (elevs[u] > 0) != (elevs[v] > 0) || elevs[v] <= 0 {
			return -1
		}

		ulat := m.LatLon[u][0]
		ulon := m.LatLon[u][1]
		vlat := m.LatLon[v][0]
		vlon := m.LatLon[v][1]
		horiz := various.Haversine(ulat, ulon, vlat, vlon) / (2 * math.Pi)

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

// regPlaceNTerritoriesCustom expands territories based on seed points and weights.
func (m *Civ) regPlaceNTerritoriesCustom(terr, seedPoints []int, weight func(o, u, v int) float64) []int {
	var queue geo.AscPriorityQueue
	heap.Init(&queue)
	outReg := make([]int, 0, 8)

	for i := 0; i < len(seedPoints); i++ {
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
