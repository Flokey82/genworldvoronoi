package genworldvoronoi

import (
	"log"
	"sort"
	"time"
)

const (
	FluxVolVariantBasic           = 0
	FluxVolVariantBasicWithDrains = 1
	FluxVolVariantWalk1           = 2
	FluxVolVariantWalk2           = 3
)

// assignFlux will populate r_flux by summing up the rainfall for each region from highest to
// lowest using r_downhill to reconstruct the downhill path that water would follow.
// NOTE: This is based on mewo2's terrain generation code
// See: https://github.com/mewo2/terrain
func (m *Geo) assignFlux(skipBelowSea bool) {
	m.Flux = m.getFlux(skipBelowSea)
}

// getFlux calculates and returns the water flux values for each region.
func (m *Geo) getFlux(skipBelowSea bool) []float64 {
	// Determines which flux calculation algorithm we use.
	variant := FluxVolVariantBasic

	// Initialize flux values with r_rainfall.
	regFlux := make([]float64, m.SphereMesh.numRegions)
	for i := 0; i < m.SphereMesh.numRegions; i++ {
		if m.Elevation[i] >= 0 || !skipBelowSea {
			regFlux[i] = m.Rainfall[i]
		}
	}

	switch variant {
	case FluxVolVariantBasic:
		// This is most basic flux calculation.
		// Sort regions by elevation in descending order.
		idxs := make([]int, len(regFlux))
		for i := range regFlux {
			idxs[i] = i
		}
		sort.Slice(idxs, func(a, b int) bool {
			return m.Elevation[idxs[a]] > m.Elevation[idxs[b]]
		})

		// Highest elevation first.
		for _, r := range idxs {
			// Skip calculation if we are below sea level or there is no downhill
			// neighbor where the water could flow to.
			// NOTE: In this case we allow water to flow to sea level.
			if (m.Elevation[r] < 0 && skipBelowSea) || m.Downhill[r] < 0 {
				continue
			}

			// Add the flux of the region to the downhill neighbor.
			regFlux[m.Downhill[r]] += regFlux[r]
		}
	case FluxVolVariantBasicWithDrains:
		// Basic variant copying the flux to the downhill neighbor or the drainage.
		// Initialize map for identifying drains and populate initial state of sorted index.
		drains := make(map[int]bool)
		idxs := make([]int, m.SphereMesh.numRegions)
		for i := range idxs {
			if m.Drainage[i] >= 0 {
				drains[m.Drainage[i]] = true
				// r_flux[m.r_drainage[i]] += m.r_rainfall[i]
			}
			idxs[i] = i
		}

		// Sort index array.
		sort.Slice(idxs, func(a, b int) bool {
			return (m.Elevation[idxs[b]] + m.Waterpool[idxs[b]]) < (m.Elevation[idxs[a]] + m.Waterpool[idxs[a]])
		})

		// Copy flux to known drainage point or next lowest neighbor.
		for _, j := range idxs {
			// Do not copy flux if we are below sea level.
			// NOTE: In this case we allow water to flow to sea level.
			if m.Elevation[j] < 0 && skipBelowSea {
				continue
			}

			// Check if we are entering a pool that drains somewhere else.
			if m.Drainage[j] >= 0 {
				// If there is a drainage point set for the current region,
				// which indicates that this region is part of a lake.
				// In this case we copy the flux directly to the region where
				// this region drains into.
				regFlux[m.Drainage[j]] += regFlux[j]
			} else if m.Downhill[j] >= 0 {
				// Add the flux of the region to the downhill neighbor.
				regFlux[m.Downhill[j]] += regFlux[j]
			}
		}
	case FluxVolVariantWalk1:
		// This seems incomplete as it will only calculate the flux
		// if a drainage point is set.
		// I put in a quick fix as I type this, but I didn't test the
		// result, so no guarantees.
		regFluxTmp := make([]float64, m.SphereMesh.numRegions)
		for j, fl := range regFlux {
			seen := make(map[int]bool)
			drain := m.Drainage[j]
			if drain == -1 {
				drain = m.Downhill[j]
			}
			for drain != -1 {
				// NOTE: In this case we allow water to flow to sea level.
				if m.Elevation[drain] < 0 && skipBelowSea {
					break
				}
				regFluxTmp[drain] += fl
				if m.Drainage[drain] >= 0 && !seen[drain] {
					drain = m.Drainage[drain]
				} else if m.Downhill[drain] >= 0 {
					drain = m.Downhill[drain]
				} else {
					drain = -1
				}
				seen[drain] = true
			}
		}

		// Copy the flux to the resulting flux map.
		for r, fl := range regFluxTmp {
			regFlux[r] += fl
		}
	case FluxVolVariantWalk2:
		// This variant will walk downhill for each region until we
		// can't find neither a downhill neighbor nor a drainage point.
		regFluxTmp := make([]float64, m.SphereMesh.numRegions)
		for j, fl := range regFlux {
			// Seen will keep track of the regions that we have
			// already visited for this region. This will prevent
			// any infinite recursions that might be caused by
			// drains that loop back to themselves.
			//
			// Not ideal, but it is what it is.
			seen := make(map[int]bool)

			// Start at region r.
			r := j

			// Not sure why I kept the chain of visited regions but I
			// assume it is useful for debugging.
			// var chain []int
			for {
				// Note that we've visited the region.
				seen[r] = true

				// Check if we have a drainage point.
				if m.Drainage[r] >= 0 {
					r = m.Drainage[r] // continue with drainage point
				} else {
					r = m.Downhill[r] // use downhill neighbor
				}

				// If we couldn't find a region to drain into, or if
				// we are below sea level, stop here.
				// NOTE: In this case we allow water to flow to sea level.
				if r < 0 || m.Elevation[r] < 0 && skipBelowSea {
					break
				}
				// Abort if we have already visited r to avoid circular
				// references.
				if seen[r] {
					break
				}
				// chain = append(chain, r)
				regFluxTmp[r] += fl
			}
			// Not sure why this was here.
			// r_flux[m.r_drainage[j]] += r_flux[j]
		}

		// Copy the flux to the resulting flux map.
		for r, fl := range regFluxTmp {
			regFlux[r] += fl
		}
	}
	return regFlux
}

// Rivers - from mapgen4

// assignFlow calculates the water flux by traversing the graph generated with
// assignDownflow in reverse order (so, downhill?) and summing up the moisture.
//
// NOTE: This is the original code that Amit uses in his procedural planets project.
// He uses triangle centroids for his river generation, where I prefer to use the regions
// directly.
func (m *BaseObject) assignFlow() {
	sideFlow := m.sideFlow

	// Clear all existing water flux values.
	for i := range sideFlow {
		sideFlow[i] = 0
	}

	triFlow := m.triFlow
	triElevation := m.triElevation
	triMoisture := m.triMoisture

	// Set the flux value for each triangle above sealevel to
	// half of its moisture squared as its initial state.
	numTriangles := m.SphereMesh.numTriangles
	for t := 0; t < numTriangles; t++ {
		if triElevation[t] >= 0.0 {
			triFlow[t] = 0.5 * triMoisture[t] * triMoisture[t]
		} else {
			triFlow[t] = 0
		}
	}

	// Now traverse the flux graph in reverse order and sum up
	// the moisture of all tributaries while descending.
	orderTris := m.orderTri
	triDownflowSide := m.triDownflowSide
	halfedges := m.SphereMesh.Halfedges
	for i := len(orderTris) - 1; i >= 0; i-- {
		// TODO: Describe what's going on here.
		tributaryTri := orderTris[i]
		flowSide := triDownflowSide[tributaryTri]
		if flowSide >= 0 {
			trunkTri := (halfedges[flowSide] / 3)
			triFlow[trunkTri] += triFlow[tributaryTri]
			sideFlow[flowSide] += triFlow[tributaryTri] // TODO: isn't s_flow[flow_s] === t_flow[?]
			if triElevation[trunkTri] > triElevation[tributaryTri] {
				triElevation[trunkTri] = triElevation[tributaryTri]
			}
		}
	}
	m.triFlow = triFlow
	m.sideFlow = sideFlow
	m.triElevation = triElevation
}

// assignWaterfalls finds regions that carry a river and are steep enough to be a waterfall.
func (m *BaseObject) assignWaterfalls() {
	steepness := m.GetSteepness()
	wfRegs := make(map[int]bool)
	for i, s := range steepness {
		if m.Elevation[i] <= 0.0 {
			continue
		}
		// 1.0 is the maximum steepness (90 degrees), so
		// everything above 0.9 (81 degrees) is a waterfall.
		if s > 0.9 && m.isRegBigRiver(i) {
			wfRegs[i] = true
		}
		// TODO:
		// - Also note that the downhill region is a waterfall?
		// - We should differentiate somehow between the top and the bottom of the waterfall.
		// - What if a waterfall is more than one region high?
		// - Should we assign an ID to each waterfall?
	}
	m.RegionIsWaterfall = wfRegs
}

// getRivers returns the merged river segments whose flux exceeds the provided limit.
// Each river is represented as a sequence of region indices.
func (m *BaseObject) getRivers(limit float64) [][]int {
	// Get segments that are valid river segments.
	links := m.getRiverSegments(limit)

	// Merge the segments that are connected to each other into logical region sequences.
	log.Println("start merge")
	start := time.Now()
	defer func() {
		log.Println("Done river segments in ", time.Since(start).String())
	}()
	return mergeIndexSegments(links)
}

func (m *BaseObject) getRiversInLatLonBB(limit float64, minLat, minLon, maxLat, maxLon float64) [][]int {
	// Get segments that are valid river segments.
	links := m.getRiverSegments(limit)

	// Merge the segments that are connected to each other into logical region sequences.
	log.Println("start merge")
	start := time.Now()
	defer func() {
		log.Println("Done river segments in ", time.Since(start).String())
	}()
	// Filter out all segments that are not in the bounding box.
	var filtered [][2]int
	for _, link := range links {
		lat1, lon1 := m.LatLon[link[0]][0], m.LatLon[link[0]][1]
		lat2, lon2 := m.LatLon[link[1]][0], m.LatLon[link[1]][1]

		// If both points are outside the bounding box, skip the segment.
		if (lat1 < minLat || lat1 > maxLat || lon1 < minLon || lon1 > maxLon) &&
			(lat2 < minLat || lat2 > maxLat || lon2 < minLon || lon2 > maxLon) {
			continue
		}
		filtered = append(filtered, link)
	}
	return mergeIndexSegments(filtered)
}

// getRiverSegments returns all region / downhill neighbor pairs whose flux values
// exceed the provided limit / threshold.
func (m *BaseObject) getRiverSegments(limit float64) [][2]int {
	// NOTE: Should we re-generate downhill and flux, just in case erosion
	// or other factors might have changed this?

	// Get (cached) downhill neighbors.
	dh := m.Downhill

	// Get (cached) flux values.
	flux := m.Flux

	// Adjust the limit to be a fraction of the max flux.
	// This will save us a lot of cycles when comparing
	// flux values to the limit.
	_, maxFlux := minMax(flux)
	limit *= maxFlux

	// Find all link segments that have a high enough flux value.
	var links [][2]int
	for r := 0; r < m.SphereMesh.numRegions; r++ {
		// Skip all regions that are sinks / have no downhill neighbor or
		// regions below sea level.
		if dh[r] < 0 || m.Elevation[r] < 0 {
			continue
		}

		// Skip all regions with flux values that are equal to the rainfall in the region,
		// which is the minimum flux value / the default state for regions without
		// water influx.
		// NOTE: Rivers need at least one contributor region and would therefore have a flux
		// value that is higher than the rainfall in the region.
		if flux[r] <= m.Rainfall[r] || flux[dh[r]] <= m.Rainfall[dh[r]] {
			continue
		}

		// NOTE: Right now we skip segments if both flux values are
		// below the limit.
		if flux[r] >= limit && flux[dh[r]] >= limit {
			// NOTE: The river segment always flows from seg[0] to seg[1].
			links = append(links, [2]int{r, dh[r]})
		}
	}
	return links
}

// getRiverIndices returns a mapping from regions to river ID.
func (m *Geo) getRiverIndices(limit float64) []int {
	// Set up defaults.
	rivers := make([]int, m.SphereMesh.numRegions)
	for i := range rivers {
		rivers[i] = -1 // -1 means no river
	}
	for i, riv := range m.getRivers(limit) {
		for _, idx := range riv {
			rivers[idx] = i
		}
	}
	return rivers
}
