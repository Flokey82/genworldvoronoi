package genworldvoronoi

import (
	"container/list"
	"log"
	"math"
	"sort"
)

const (
	FloodVariant1 = 0
	FloodVariant2 = 1
)

// assignHydrology will calculate river systems and fill sinks instead of trying to generate
// water pools.
func (m *Geo) assignHydrology() {
	maxAttempts := 3
	erosionAmount := 0.01 // Erode 1% of delta-h per pass.

	// HACK: Fill all sinks that are below sea level and a single region
	// below sea level.
Loop:
	for _, r := range m.GetSinks(false, false) {
		// Check if all neighbors are above sea level.
		lowest := math.Inf(0)
		for _, nb := range m.GetRegNeighbors(r) {
			if !m.isRegBelowOrAtSeaLevelOrPool(r) {
				continue Loop
			}
			if m.Elevation[nb] < lowest {
				lowest = m.Elevation[nb]
			}
		}
		m.Elevation[r] = lowest
	}

	// Start off by filling sinks.
	m.Elevation = m.FillSinks(true)

	// Try to flood all sinks.
	var attempts int
	m.BaseObject.assignDownhill(true)
	m.assignFlux(false)
	for {
		// Abort if we have no more sinks or ran out of attempts.
		if attempts > maxAttempts {
			m.Elevation = m.FillSinks(true)

			// Regenerate downhill.
			m.BaseObject.assignDownhill(true)

			// Regenerate flux.
			m.assignFlux(true)

			// TODO: Diffuse flux and pool.
			m.assignRainfallBasic()
			// TODO: Fill remaining sinks and re-generate downhill and flux.
			break
		}
		attempts++
		// Reset drains.
		for i := range m.Drainage {
			m.Drainage[i] = -1
		}

		// Reset pools.
		for i := range m.Waterpool {
			m.Waterpool[i] = 0
		}
		m.Elevation = m.FillSinks(true)

		// TODO: Diffuse flux and pool.
		m.assignRainfallBasic()

		// Regenerate downhill.
		m.BaseObject.assignDownhill(true)

		// Regenerate flux.
		m.assignFlux(false)

		// Erode a little.
		m.Elevation = m.Erode(erosionAmount) // NOTE: Flux would change as downhill values would change.
	}

	// TODO: Move this somewhere else.
	m.LakeSize = m.getLakeSizes()
	// TODO: Make note of oceans.
	//   - Note ocean sizes (and small waterbodies below sea level)
	m.assignWaterbodies()
}

// assignHydrologyWithFlooding will calculate river systems and water pools.
func (m *Geo) assignHydrologyWithFlooding() {
	maxAttempts := 20
	floodVariant := FloodVariant2
	skipSinksBelowSea := true

	// Reset drains.
	for i := range m.Drainage {
		m.Drainage[i] = -1
	}

	// Try to flood all sinks.
	var attempts int
	for {
		// Identify sinks above sea level.
		r_sinks := m.BaseObject.GetSinks(skipSinksBelowSea, true)

		// Abort if we have no more sinks or ran out of attempts.
		if len(r_sinks) == 0 || attempts > maxAttempts {
			m.Elevation = m.FillSinks(true)
			// Regenerate downhill.
			m.BaseObject.assignDownhill(true)

			// Regenerate flux.
			m.assignFlux(true)

			// TODO: Diffuse flux and pool.
			m.assignRainfall(4, moistTransferDirect, moistOrderWind)

			log.Println("ran out of attempts", len(r_sinks))
			// TODO: Fill remaining sinks and re-generate downhill and flux.
			break
		}
		attempts++

		// Now that we want to calculate the max. flux that accumulates in sinks,
		// we will have to disregard any drainage and waterpools.

		// Reset drains.
		for i := range m.Drainage {
			m.Drainage[i] = -1
		}

		// Reset pools.
		for i := range m.Waterpool {
			m.Waterpool[i] = 0
		}

		// Erode a little.
		// m.r_elevation = m.rErode(0.01) // NOTE: Flux would change as downhill values would change.

		// Regenerate downhill and do not skip below sea level.
		m.BaseObject.assignDownhill(false)

		// Regenerate flux.
		m.assignFlux(true)

		// Identify sinks above sea level.
		r_sinks = m.BaseObject.GetSinks(false, false)

		// Start from lowest sink.
		sort.Slice(r_sinks, func(i, j int) bool {
			return m.Elevation[r_sinks[i]] < m.Elevation[r_sinks[j]]
		})

		// Flood sink up to lowest neighbor + epsilon.
		for _, r := range r_sinks {
			//if m.r_flux[r] < m.r_rainfall[r] {
			//	continue
			//}
			switch floodVariant {
			case FloodVariant1:
				m.floodV1(r, m.Flux[r])
			case FloodVariant2:
				m.floodV2(r, m.Flux[r])
			}
		}

		// TODO: Diffuse flux and pool.
		m.assignRainfall(1, moistTransferDirect, moistOrderWind)
	}

	// TODO: Triangle downhill.
	// TODO: Make note of lakes.
	//   - Sum up regions r_pool[r] > 0
	//   - Note lake sizes (for city placement)
	m.LakeSize = m.getLakeSizes()
	// TODO: Make note of rivers.
	m.assignWaterbodies()
}

// floodSinks fills the sinks in the map either by using the water pool
// or by using the fill sinks algorithm.
// NOTE: Use flux with FluxVolVariantBasicWithDrains for this.
// THIS IS STILL A WORK IN PROGRESS!
func (m *BaseObject) floodSinks() []float64 {
	// Get the filled sinks and assign the difference to the elevation
	// to the water pool.

	// - Get the elevation from the fill sinks algorithm.
	filledSinks := m.FillSinks(false)

	newHeight := make([]float64, len(m.Elevation))
	copy(newHeight, filledSinks)

	// Compare the unaltered elevation of the lake regions
	// with the elevation of the fill sinks algorithm to get
	// the difference, which would be the water level.
	//
	// NOTE: The surface of the lakes would not level if used
	// unaltered due to the way the fill sinks algorithm works.
	pool := make([]float64, len(m.Elevation))
	for i, v := range filledSinks {
		pool[i] = v - m.Elevation[i]
	}

	// Sort the regions by their filled elevation in ascending order.
	// This way we avoid picking a high region as the seed region.
	// The seed region is the region that is used to represent the lake.
	sortedRegs := make([]int, len(m.Elevation))
	for i := range sortedRegs {
		sortedRegs[i] = i
	}
	sort.Slice(sortedRegs, func(i, j int) bool {
		return filledSinks[sortedRegs[i]] < filledSinks[sortedRegs[j]]
	})

	outRegs := make([]int, 0, 8)

	// drainage holds mapping of region to drainage region.
	drainage := initRegionSlice(len(m.Elevation))

	// poolIDs olds mapping of region to pool ID.
	poolIDs := initRegionSlice(len(m.Elevation))

	// poolIDToLowestReg holds mapping of pool ID to lowest region
	// (for normalizing/leveling the water surface).
	poolIDToLowestReg := make(map[int]int)

	// poolIDToDrainage holds mapping of pool ID to drainage region.
	poolIDToDrainage := make(map[int]int)

	// poolIDToSize holds mapping of pool ID to number of regions
	// that are part of the same lake.
	poolIDToSize := make(map[int]int)

	// poolSeeds holds the regions that were picked to represent
	// lakes / connected regions.
	var poolSeeds []int

	// poolParty holds the regions that are part of the same pool.
	poolParty := make([]int, 0, 100)

	// Identify connected, potential lake regions.
	for _, i := range sortedRegs {
		v := pool[i]
		if v == 0 || poolIDs[i] != -1 {
			continue
		}

		// We have a water pool and it is not part of a lake yet.
		lowestReg := i
		poolIDs[i] = i

		list := list.New()
		list.PushBack(i)
		for list.Len() > 0 {
			e := list.Front()
			list.Remove(e)
			reg := e.Value.(int)
			poolParty = append(poolParty, reg)
			for _, n := range m.SphereMesh.r_circulate_r(outRegs, reg) {
				if poolIDs[n] == -1 && pool[n] > 0 {
					poolIDs[n] = i
					list.PushBack(n)
					if filledSinks[n] < filledSinks[lowestReg] {
						lowestReg = n
					}
				}
			}
		}

		// Store the lowest region of the lake, which will be used
		// to level the water surface of the lake.
		poolIDToLowestReg[i] = lowestReg

		// Find the drainage point of the lake regions, which is the
		// lowest region that is not part of the lake, bordering the lake.
		if len(poolParty) > 0 {
			lowestRegDrainage := -1
			for _, reg := range poolParty {
				for _, n := range m.SphereMesh.r_circulate_r(outRegs, reg) {
					if (lowestRegDrainage == -1 || filledSinks[n] < filledSinks[lowestRegDrainage]) && poolIDs[n] == -1 {
						lowestRegDrainage = n
					}
				}
			}
			poolIDToDrainage[i] = lowestRegDrainage
		}
		poolSeeds = append(poolSeeds, i)
		poolIDToSize[i] = len(poolParty)
		poolParty = poolParty[:0]
	}

	// Loop over the lake seeds and sum up the precipitation and flux
	// of the lake regions. If the sum is zero, we fill up the elevation
	// of the lake regions instead of the water pool.
	for _, seed := range poolSeeds {
		lowestReg := poolIDToLowestReg[seed]
		var sumPrecip, sumFlux float64
		for reg, rID := range poolIDs {
			if rID != seed {
				continue
			}
			sumPrecip += m.Rainfall[reg]
			sumFlux += m.Flux[reg]
		}

		// Fill up the elevation of the lake regions instead of the water pool if:
		// - the sum of precipitation and flux is zero
		// - the lake region does not have a drainage region
		//
		// TODO: Maybe use a threshold instead of zero?
		if sumPrecip == 0 && sumFlux == 0 || poolIDToDrainage[seed] == -1 {
			// If lake regions do not have any precipitation, we
			// fill up the elevation of the lake regions instead
			// of the water pool.
			for reg, regID := range poolIDs {
				if regID == seed {
					pool[reg] = 0
					poolIDs[reg] = -1
					drainage[reg] = -1
					newHeight[reg] = filledSinks[reg]
				}
			}
		} else {
			// Fill up the water pool to the lowest lake region.
			// Set the pool depth to the lowest lake region according to the
			// elevation returned by the fill sinks algorithm.
			for reg, poolID := range poolIDs {
				if poolID != seed {
					continue
				}

				if filledSinks[lowestReg] < m.Elevation[reg] {
					// Use the actual elevation of the region if it is higher
					// than the lowest lake region.
					pool[reg] = 0
					poolIDs[reg] = -1
					drainage[reg] = -1
					newHeight[reg] = filledSinks[reg]
					// TODO: Check if this is also the seed region.
					// If so, we need to update the seed region.
					// if reg == seed {
					//   panic("seed is filled region")
					// }
				} else {
					// If the the actual elevation of the region is higher than the
					// lowest lake region, we fill up the water pool to the actual
					pool[reg] = (m.Elevation[lowestReg] + pool[lowestReg]) - m.Elevation[reg]
					newHeight[reg] = m.Elevation[reg]

					// Set the drainage of the lake regions to the drainage region
					// which is the lowest region of the lake neighbors.
					// TODO: Check of this region's downhill chain leads to the ocean.
					drainage[reg] = poolIDToDrainage[seed]
				}
			}
		}
	}
	m.Elevation = newHeight
	m.Drainage = drainage
	m.Waterpool = pool
	return pool
}

// floodV1 is the first variant of the flood fill algorithm, which finds
// drainage points for sinks and generates lakes.
// Don't ask me how this works in detail as I do not know.
//
// Only thing I know is that it is based on Nick McDonald's old flood fill
// he used in simple_hydrology.
//
// TODO: Return remaining volume
func (m *Geo) floodV1(r int, dVol float64) {
	const (
		volumeFactor = 100.0 // "Water Deposition Rate"
		epsilon      = 1e-3
		minVol       = 0.01
		drainage     = 0.01
	)

	plane := m.Elevation[r] + m.Waterpool[r]
	initialplane := plane

	// Floodset contains all regions that are part of a floodplain.
	set := make([]int, 0, 1024)

	// Abort after 200 attempts.
	fail := 200

	// Keep track of the regions we have visitad during a flood fill attempt.
	tried := make([]bool, m.SphereMesh.numRegions)
	var drain int
	var drainfound bool
	var fill func(i int)
	fill = func(i int) {
		// Out of bounds, or region has been visited ("tried") previously.
		if i < 0 || tried[i] {
			return
		}
		tried[i] = true

		// Wall / Boundary
		currHeight := m.Elevation[i] + m.Waterpool[i]
		if plane < currHeight {
			return
		}

		// Drainage Point
		if initialplane > currHeight {
			if !drainfound || currHeight < m.Waterpool[drain]+m.Elevation[drain] {
				// No Drain yet or lower drain.
				drain = i
			}

			drainfound = true
			return
		}

		// Part of the Pool
		set = append(set, i)
		nbs := m.GetRegNeighbors(i)

		// Pre-sort neighbors by height (elevation + water pool).
		//
		// NOTE: The regions are sorted in ascending order, so the first
		// region in the list will be the lowest one.
		sort.Slice(nbs, func(si, sj int) bool {
			return m.Elevation[nbs[si]]+m.Waterpool[nbs[si]] < m.Elevation[nbs[sj]]+m.Waterpool[nbs[sj]]
		})

		// Expand floodset by attempting to fill all neighbors.
		for _, nbReg := range nbs {
			fill(nbReg)
		}
	}

	// Iterate
	for dVol > minVol && fail != 0 {
		set = set[:0]

		// Reset the visited regions.
		for i := range tried {
			tried[i] = false
		}

		// Reset the drain and drainfound flag.
		drain = 0
		drainfound = false

		// Perform flooding of initial region.
		fill(r)

		// Drainage Point
		if drainfound {
			// Set the New Waterlevel (Slowly)
			plane = (1.0-drainage)*initialplane + drainage*(m.Elevation[drain]+m.Waterpool[drain])

			// Compute the New Height
			for _, s := range set {
				if plane > m.Elevation[s] {
					m.Waterpool[s] = plane - m.Elevation[s]
					m.Drainage[s] = drain
				} else {
					m.Waterpool[s] = 0.0
					m.Drainage[s] = -1
				}
			}
			// Remove Sediment
			// d.sediment *= 0.1
			log.Println(r, "found drain!")
			break
		}

		// Get Volume under Plane
		// So we sum up the difference between plane and (height[s]+pool[s]) which
		// gives up the total missing volume required for a full flood.
		var totalVol float64
		for _, s := range set {
			totalVol += volumeFactor * (plane - (m.Elevation[s] + m.Waterpool[s]))
		}
		// log.Println("totalVol", totalVol, "dVol", dVol, "setLen", len(set))
		// We can fill the volume of the sink.
		if totalVol <= dVol && initialplane < plane {
			// Raise water level to plane height.
			for _, s := range set {
				m.Waterpool[s] = plane - m.Elevation[s]
			}

			// Adjust flux Volume
			dVol -= totalVol
			totalVol = 0.0
		} else {
			fail-- // Plane was too high.
		}

		// Adjust planes.
		if plane > initialplane {
			initialplane = plane
		}
		// log.Println("plane before", plane)
		plane += 0.5 * (dVol - totalVol) / float64(len(set)) / volumeFactor
		log.Println(r, "plane after", plane)
	}
}

// Flooding Algorithm Overhaul:
// Currently, I can only flood at my position as long as we are rising.
// Then I return and let the particle descend. This should only happen if I can't find a closed set to fill.
// So: Rise and fill, removing the volume as we go along.
// Then: If we find a lower point, try to rise and fill from there.
//
// See: https://github.com/weigert/SimpleHydrology/blob/master/source/water.h
func (m *Geo) floodV2(r int, dVol float64) bool {
	minVol := 0.001
	if dVol < minVol {
		return false
	}
	volumeFactor := 0.5

	// Either try to find a closed set under this plane, which has a certain volume,
	// or raise the plane till we find the correct closed set height.
	// And only if it can't be found, re-emit the particle.
	tried := make([]bool, m.SphereMesh.numRegions)
	boundary := make(map[int]float64)
	var floodset []int
	var drainfound bool
	var drain, drainedFrom int
	drainedFrom = -1

	useDrain := true // Use drainage point instead of region draining into drainage point.

	// Returns whether the set is closed at given height
	var findset func(i int, plane float64) bool
	findset = func(i int, plane float64) bool {
		// Out of Bounds or position has been tried.
		if i < 0 || tried[i] {
			return true
		}
		tried[i] = true

		// Wall / Boundary
		currHeight := m.Elevation[i] + m.Waterpool[i]
		if plane < currHeight {
			boundary[i] = currHeight
			return true
		}

		// Drainage Point
		if currHeight < plane {
			// No Drain yet
			if !drainfound || currHeight < m.Waterpool[drain]+m.Elevation[drain] {
				drain = i
			}
			drainfound = true
			return false
		}

		// Part of the Pool
		floodset = append(floodset, i)
		nbs := m.GetRegNeighbors(i)
		sort.Slice(nbs, func(si, sj int) bool {
			return m.Elevation[nbs[si]]+m.Waterpool[nbs[si]] < m.Elevation[nbs[sj]]+m.Waterpool[nbs[sj]]
		})
		for _, nbReg := range nbs {
			if !findset(nbReg, plane) {
				if drainfound { // && drainedFrom == -1
					newDrain := -1
					if useDrain {
						newDrain = drain
					} else {
						newDrain = i
					}
					if drainedFrom == -1 || m.Elevation[newDrain]+m.Waterpool[newDrain] < m.Elevation[drainedFrom]+m.Waterpool[drainedFrom] {
						drainedFrom = newDrain
					}
				}
				return false
			}
		}
		return true
	}

	plane := m.Waterpool[r] + m.Elevation[r]
	minboundFirst := r
	minboundSecond := plane
	for dVol > minVol && findset(r, plane) {
		// Find the Lowest Element on the Boundary
		minboundFirst = -1
		for bfirst, bsecond := range boundary {
			if bsecond < minboundSecond || minboundFirst == -1 {
				minboundFirst = bfirst
				minboundSecond = bsecond
			}
		}
		// Compute the Height of our Volume over the Set
		vheight := dVol * volumeFactor / float64(len(floodset))

		// Not High Enough: Fill 'er up
		if plane+vheight < minboundSecond {
			plane += vheight
		} else {
			dVol -= (minboundSecond - plane) / volumeFactor * float64(len(floodset))
			plane = minboundSecond
		}

		for _, s := range floodset {
			m.Waterpool[s] = plane - m.Elevation[s]
			if s != drainedFrom {
				m.Drainage[s] = drainedFrom // WROOOOONG?????
			}
		}
		delete(boundary, minboundFirst)
		tried[minboundFirst] = false
		r = minboundFirst
	}

	if drainfound {
		if true {
			// Search for Exposed Neighbor with Non-Zero Waterlevel
			var lowbound func(i int)
			lowbound = func(i int) {
				// Out-Of-Bounds
				if i < 0 || m.Waterpool[i] == 0 {
					return
				}
				// Below Drain Height
				if m.Elevation[i]+m.Waterpool[i] < m.Elevation[drain]+m.Waterpool[drain] {
					return
				}
				// Higher than Plane (we want lower)
				if m.Elevation[i]+m.Waterpool[i] >= plane {
					return
				}
				plane = m.Elevation[i] + m.Waterpool[i]
			}

			nbs := m.GetRegNeighbors(drain)
			sort.Slice(nbs, func(si, sj int) bool {
				return m.Elevation[nbs[si]]+m.Waterpool[nbs[si]] < m.Elevation[nbs[sj]]+m.Waterpool[nbs[sj]]
			})

			// Fill Neighbors
			for _, nbReg := range nbs {
				lowbound(nbReg)
				// Fill neighbors of neighbors
				// for _, nbs2 := range m.rNeighbors(nbs) { // ??????
				//	lowbound(nbs2)
				// }
			}
		}

		// Water-Level to Plane-Height
		for _, s := range floodset {
			// volume += ((plane > h[ind])?(h[ind] + p[ind] - plane):p[ind])/volumeFactor;
			if plane > m.Elevation[s] {
				m.Waterpool[s] = plane - m.Elevation[s]
				if s != drainedFrom {
					m.Drainage[s] = drainedFrom
				}
			} else {
				m.Waterpool[s] = 0.0
				m.Drainage[s] = -1
			}
		}

		for bfirst := range boundary {
			// volume += ((plane > h[ind])?(h[ind] + p[ind] - plane):p[ind])/volumeFactor;
			if plane > m.Elevation[bfirst] {
				m.Waterpool[bfirst] = plane - m.Elevation[bfirst]
				if bfirst != drainedFrom {
					m.Drainage[bfirst] = drainedFrom
				}
			} else {
				m.Waterpool[bfirst] = 0.0
				m.Drainage[bfirst] = -1
			}
		}
		// sediment *= oldvolume/volume;
		// sediment /= float64(len(floodset)) //Distribute Sediment in Pool
		r = drain
		return true
	}
	return false
}
