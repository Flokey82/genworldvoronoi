package genworldvoronoi

import (
	"math"
)

// assignOceanCurrents will calculate the ocean currents for the map.
// NOTE: THIS IS NOT WORKING YET!!!!!
// For interesting approaches, see:
// https://forhinhexes.blogspot.com/search/label/Currents
// https://github.com/FreezeDriedMangos/realistic-planet-generation-and-simulation
func (m *Geo) assignOceanCurrents() {
	regCurrentVec := make([][2]float64, m.mesh.numRegions)

	// Let's calculate the ocean currents.

	// deflectAndSplit is a function that takes a region and
	// returns a vector. It is used to calculate the ocean currents.
	// If the region's current vector is pointing towards a land region,
	// the vector is deflected and split into two vectors, impacting the
	// the vectors of ocean regions in the vicinity.
	deflectAndSplit := func(reg int) [2]float64 {
		// If the region is not an ocean region, return the zero vector.
		if m.Elevation[reg] > 0 {
			return [2]float64{0.0, 0.0}
		}

		// If the region is an ocean region, calculate the vector to
		// each of its neighbors. If the neighbor in the direction of
		// the ocean current is a land region, deflect the current
		// vector and split it into two vectors to the neighboring
		// ocean regions.
		vec := regCurrentVec[reg]
		var oceanRegions []int
		var oceanRegionDot []float64
		var landRegions []int
		var landRegionDot []float64
		// Calculate the dot product of the current vector and the
		// vector to each neighbor.
		for _, nb := range m.GetRegNeighbors(reg) {
			nbVec := calcVecFromLatLong(m.LatLon[reg][0], m.LatLon[reg][1], m.LatLon[nb][0], m.LatLon[nb][1])
			nbVec = normal2(nbVec)

			// Get the dot product of the current vector and the
			// vector to the neighbor.
			dot := dot2(vec, nbVec)

			if m.Elevation[nb] > 0 {
				landRegions = append(landRegions, nb)
				landRegionDot = append(landRegionDot, dot)
			} else {
				oceanRegions = append(oceanRegions, nb)
				oceanRegionDot = append(oceanRegionDot, dot)
			}
		}

		// If there are no land regions, return the current vector.
		if len(landRegions) == 0 {
			return vec
		}

		// If there are no ocean regions, return the current vector.
		if len(oceanRegions) == 0 {
			return vec
		}

		// If there are land regions, deflect the current vector and
		// split it into two vectors to the neighboring ocean regions.
		var newVec [2]float64
		for i, nb := range oceanRegions {
			// If the dot product of the current vector and the vector
			// to the neighbor is negative, the neighbor is in the
			// opposite direction of the current vector. In this case,
			// the current vector is split into two vectors, one
			// pointing to the neighbor and one pointing away from the
			// neighbor.
			if oceanRegionDot[i] < 0 {
				newVec = add2(newVec, scale2(vec, 0.5))
				regCurrentVec[nb] = add2(regCurrentVec[nb], scale2(vec, 0.5))
			} else {
				// If the dot product of the current vector and the
				// vector to the neighbor is positive, the neighbor is
				// in the same direction as the current vector. In this
				// case, the current vector is split into two vectors,
				// both pointing to the neighbor.
				newVec = add2(newVec, scale2(vec, 0.25))
				regCurrentVec[nb] = add2(regCurrentVec[nb], scale2(vec, 0.25))
			}
		}

		// Return the new current vector.
		return newVec
	}

	for r := 0; r < m.mesh.numRegions; r++ {
		// Skip elevation above sea level.
		if m.Elevation[r] > 0 {
			continue
		}
		lat := math.Abs(m.LatLon[r][0])
		if lat <= 0.9 && lat >= 0.5 {
			// Initialize the currents at the equator to flow from west to east.
			regCurrentVec[r] = [2]float64{-1.0, 0.0}
		} else if lat >= 59.2 && lat <= 60.2 {
			// Initialize the currents at the arctic / antarctic circles to flow
			// from east to west.
			regCurrentVec[r] = [2]float64{1.0, 0.0}
		}

		// Check if we deflected the current vector.
		regCurrentVec[r] = deflectAndSplit(r)
	}

	// Now interpolate all set vectors with all other set vectors.
	// This is done to make the ocean currents more realistic.
	for i := 0; i < 100; i++ {
		for r := 0; r < m.mesh.numRegions; r++ {
			if m.Elevation[r] > 0 {
				continue
			}
			// Average with all set neighbor vectors.
			var sumVec [2]float64
			var numVec int
			for _, nb := range m.GetRegNeighbors(r) {
				if m.Elevation[nb] > 0 || regCurrentVec[nb] == [2]float64{0.0, 0.0} {
					continue
				}
				sumVec = add2(sumVec, regCurrentVec[nb])
				numVec++
			}
			if numVec > 0 {
				regCurrentVec[r] = normalize2(sumVec)
			}
			regCurrentVec[r] = deflectAndSplit(r)
		}
	}

	m.RegionToOceanVec = regCurrentVec
}
