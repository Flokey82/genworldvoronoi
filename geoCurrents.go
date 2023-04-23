package genworldvoronoi

import (
	"log"
	"math"
	"sort"
)

// assignOceanCurrents will calculate the ocean currents for the map.
// NOTE: THIS IS NOT WORKING YET!!!!!
// For interesting approaches, see:
// https://forhinhexes.blogspot.com/search/label/Currents
// https://github.com/FreezeDriedMangos/realistic-planet-generation-and-simulation
func (m *Geo) assignOceanCurrents() {
	regCurrentVec := make([][2]float64, m.mesh.numRegions)

	// Let's calculate the ocean currents.
	// Build the region to region neighbor vectors.
	regToRegNeighborVec := m.getRegionToNeighborVec()

	// Calculate the pressure in each ocean region.
	regPressure := m.calcCurrentPressure(regCurrentVec)

	// deflectAndSplit is a function that takes a region and
	// returns a vector. It is used to calculate the ocean currents.
	// If the region's current vector is pointing towards a land region,
	// the vector is deflected towards the closes ocean region (or the region
	// with the lowest pressure).
	deflectAndSplit := func(reg int, useLowPressure bool) [2]float64 {
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
		var oceanRegions, landRegions []int
		var oceanRegionDot, landRegionDot []float64
		var oceanRegionVec, landRegionVec [][2]float64
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
				landRegionVec = append(landRegionVec, nbVec)
			} else {
				oceanRegions = append(oceanRegions, nb)
				oceanRegionDot = append(oceanRegionDot, dot)
				oceanRegionVec = append(oceanRegionVec, nbVec)
			}
		}

		// If there are no land or ocean regions, return the current vector.
		if len(landRegions) == 0 || len(oceanRegions) == 0 {
			return vec
		}

		if useLowPressure {
			// Here we deflect the current vector towards the lowest pressure
			// ocean region if we have a high pressure ocean region.
			p := regPressure[reg]
			if p > 0 {
				// Find the lowest pressure ocean region.
				minPressure := 1.0
				minPressureIdx := -1
				for i := range oceanRegions {
					if regPressure[oceanRegions[i]] < minPressure {
						minPressure = regPressure[oceanRegions[i]]
						minPressureIdx = i
					}
				}

				// Rotate the current vector towards the closest ocean region by
				// averaging the current vector and the vector to the closest ocean
				// region.
				if minPressureIdx != -1 && minPressure < p {
					oceanReg := oceanRegions[minPressureIdx]
					vec = add2(vec, scale2(regToRegNeighborVec[reg][oceanReg], p-minPressure))
				}
			} else {
				// Find the highest pressure ocean region.
				maxPressure := -1.0
				maxPressureIdx := -1
				for i := range oceanRegions {
					if regPressure[oceanRegions[i]] > maxPressure {
						maxPressure = regPressure[oceanRegions[i]]
						maxPressureIdx = i
					}
				}

				// Rotate the current vector towards the closest ocean region by
				// averaging the current vector and the vector to the closest ocean
				// region.
				if maxPressureIdx != -1 && maxPressure > p {
					oceanReg := oceanRegions[maxPressureIdx]
					vec = add2(vec, scale2(regToRegNeighborVec[reg][oceanReg], p-maxPressure))
				}
			}
			vec = normalize2(vec)
		} else {
			// Deflect the current vector towards the closest ocean region.
			maxDot1 := -1.0
			maxDot1Idx := -1
			for i := range oceanRegions {
				if oceanRegionDot[i] > maxDot1 {
					maxDot1 = oceanRegionDot[i]
					maxDot1Idx = i
				}
			}

			// NOTE: Maybe we shouldn't normalize the vectors here, but use the
			// pressure build up to better determine the strength of the currents
			// and their actual influence on each other. We can normalize the
			// vectors after each iteration.
			if maxDot1Idx != -1 {
				oceanReg := oceanRegions[maxDot1Idx]
				// Scale the new vector and normalize it.
				vecDot := oceanRegionVec[maxDot1Idx]
				if vecDot == zero2 {
					vecDot = normal2(vec)
				}
				added := scale2(regToRegNeighborVec[reg][oceanReg], 1.0)
				regCurrentVec[oceanReg] = normal2(add2(regCurrentVec[oceanReg], added))

				// Since we want the vector to snake along the coast, we need to
				// rotate the current vector towards the closest ocean region.
				// Find the highest dot product (closest to the current vector).

				// Rotate the current vector towards the closest ocean region by
				// averaging the current vector and the vector to the closest ocean
				// region.
				vec = normalize2(add2(vec, oceanRegionVec[maxDot1Idx]))
			}

			// Normalize the current vector.
			vec = normalize2(vec)
		}

		// Return the current vector.
		return vec
	}

	// propagateCurrent propagates the current vector from the given region to
	// all neighboring regions until the current vector is zero.
	var propagateCurrent func(reg int)
	propagateCurrent = func(reg int) {
		useDot := true
		if regCurrentVec[reg] == zero2 {
			return
		}
		// Propagate the current vector to all neighboring regions.
		for _, neighbor := range m.GetRegNeighbors(reg) {
			// Skip elevation above sea level and regions with a current vector
			// already set.
			if m.Elevation[neighbor] > 0 || regCurrentVec[neighbor] != zero2 {
				continue
			}
			// Set the current vector using the dot product to scale the vector
			// towards the neighboring region.
			if useDot {
				dot := dot2(regCurrentVec[reg], regToRegNeighborVec[reg][neighbor])
				regCurrentVec[neighbor] = add2(regCurrentVec[reg], scale2(regToRegNeighborVec[reg][neighbor], dot))
			} else {
				regCurrentVec[neighbor] = add2(regCurrentVec[reg], scale2(regToRegNeighborVec[reg][neighbor], 0.5))
			}
			regCurrentVec[neighbor] = normalize2(regCurrentVec[neighbor])

			// Propagate the current vector to the neighboring region.
			propagateCurrent(neighbor)
		}
	}

	// Reinforce the primary currents.
	m.seedOceanCurrents(regCurrentVec)
	for r := 0; r < m.mesh.numRegions; r++ {
		// Check if we deflected the current vector.
		regCurrentVec[r] = deflectAndSplit(r, false)
	}

	// Now interpolate all set vectors with all other set vectors.
	// This is done to make the ocean currents more realistic.
	for i := 0; i < 100; i++ {
		// Reinforce the primary currents.
		m.seedOceanCurrents(regCurrentVec)
		/*
			// Calculate the pressure in each ocean region.
			regPressure = m.calcCurrentPressure(regCurrentVec)
			// Reinforce the primary currents.
			for r := 0; r < m.mesh.numRegions; r++ {
				// Skip elevation above sea level.
				if m.Elevation[r] > 0 {
					continue
				}
				propagateCurrent(r)
			}*/

		// TODO: Loop through regions and resolve the pressure differentials.

		// Sort the regions by their "downstream" direction.
		regions := m.getCurrentSortOrder(regCurrentVec, false)
		// Calculate the pressure in each ocean region.
		regPressure = m.calcCurrentPressure(regCurrentVec)
		// Sort the regions by pressure (highest pressure first)
		sort.Slice(regions, func(i, j int) bool {
			return regPressure[regions[i]] > regPressure[regions[j]]
		})
		for _, r := range regions {
			if m.Elevation[r] > 0 {
				continue
			}
			regCurrentVec[r] = deflectAndSplit(r, true)
		}

		// Average the vectors with all set neighbor vectors.
		for _, r := range regions {
			if m.Elevation[r] > 0 {
				continue
			}

			// Average with all set neighbor vectors.
			var sumVec [2]float64
			var numVec int
			for _, nb := range m.GetRegNeighbors(r) {
				if m.Elevation[nb] > 0 || regCurrentVec[nb] == zero2 {
					continue
				}
				sumVec = add2(sumVec, regCurrentVec[nb])
				numVec++
			}
			if numVec > 0 {
				if regCurrentVec[r] != zero2 {
					sumVec = add2(sumVec, regCurrentVec[r])
					numVec++
				}
				regCurrentVec[r] = normalize2(sumVec)
			}
			if regCurrentVec[r] == zero2 {
				continue
			}
			regCurrentVec[r] = deflectAndSplit(r, true)
		}
	}

	m.RegionToOceanVec = regCurrentVec
}

// getCurrentSortOrder returns the regions sorted by their position and downstream direction.
func (m *Geo) getCurrentSortOrder(revVecs [][2]float64, reverse bool) []int {
	_, orderedRegs := m.getVectorSortOrder(revVecs, reverse)
	return orderedRegs
}

func (m *Geo) assignOceanCurrentsInflowOutflow() {
	// Calculate the inflow and outflow of each ocean region, which can be used
	// to calculate the ocean currents.

	// We start off by setting the primary ocean current vectors and then iterate
	// over the regions to calculate the inflow and outflow vectors, depending
	// on the pressure in the region.
	regCurrentVec := make([][2]float64, m.mesh.numRegions)

	// Build the region to region neighbor vectors.
	regToRegNeighborVec := m.getRegionToNeighborVec()

	// Loop a few times to establis an equilibrium.
	for i := 0; i < 100; i++ {
		// Set the primary ocean current vectors.
		m.seedOceanCurrents(regCurrentVec)

		// Calculate the pressure in each ocean region.
		regPressure := m.calcCurrentPressure(regCurrentVec)

		// Calculate the inflow and outflow vectors based on the pressure difference.
		for reg := 0; reg < m.mesh.numRegions; reg++ {
			// If the region is not an ocean region, skip it.
			if m.Elevation[reg] > 0 {
				continue
			}
			// Calculate the inflow and outflow vectors.
			// The inflow vector is the sum of the vectors of the neighbors
			// that have a positive pressure.
			// The outflow vector is the current vector.
			inflowVec := [2]float64{0, 0}
			outflowVec := [2]float64{0, 0}
			for _, neighbor := range m.GetRegNeighbors(reg) {
				// Skip neighbors that are not ocean regions.
				if m.Elevation[neighbor] > 0 {
					continue
				}
				// If the neghbor has no current, we can skip it.
				if regCurrentVec[neighbor] == [2]float64{0, 0} {
					continue
				}
				// Calculate the dot product of the vector from the neighbor to the
				// current region and the current vector of the neighbor.
				// This will tell us how much of the current vector of the neighbor
				// is flowing into the current region.
				dot := dot2(normalize2(regToRegNeighborVec[neighbor][reg]), normalize2(regCurrentVec[neighbor]))

				// If the dot product is positive, the neighbor is flowing into the
				// current region, so it is part of the inflow vector.
				if dot > 0 {
					inflowVec = add2(inflowVec, scale2(regCurrentVec[neighbor], dot))
				} else if dot < 0 {
					// If the dot product is negative, the neighbor is flowing out of
					// the current region, so it is part of the outflow vector.
					outflowVec = add2(outflowVec, scale2(regCurrentVec[neighbor], -dot))
				}
			}

			// The difference in magnitude between the inflow and outflow vectors
			// indicates the pressure difference.
			lenIn := len2(inflowVec)
			lenOut := len2(outflowVec)
			diff := lenIn - lenOut

			log.Println("reg", reg, "pressure", regPressure[reg], "inflow", lenIn, "outflow", lenOut, "diff", diff)

			// If we have a pressure difference, we need to adjust the current vector.
			if regPressure[reg] != 0 {
				// Loop through all neighbors and adjust the current vector.
				for _, neighbor := range m.GetRegNeighbors(reg) {
					if m.Elevation[neighbor] > 0 {
						continue
					}
					// Skip higher pressure regions.
					if regPressure[neighbor] > regPressure[reg] {
						continue
					}
					// Calculate the dot product of the vector from the neighbor to the
					// current region and the current vector of the neighbor.
					// This will tell us how much of the current vector of the neighbor
					// is flowing into the current region.
					dot := dot2(normalize2(regToRegNeighborVec[neighbor][reg]), normalize2(regCurrentVec[neighbor]))
					// If the dot product is positive, the neighbor is flowing into the
					// current region, so it is part of the inflow vector.
					if dot > 0 {
						// Calculate the amount of the inflow vector that will be
						// transferred to the neighbor.
						transfer := dot * regPressure[reg]
						// Add the transfer to the neighbor's current vector.
						regCurrentVec[neighbor] = add2(regCurrentVec[neighbor], scale2(inflowVec, transfer))
						// Subtract the transfer from the current region's current vector.
						regCurrentVec[reg] = add2(regCurrentVec[reg], scale2(inflowVec, -transfer))
					}
				}
			}
		}

		// Normalize the current vectors.
		for reg := 0; reg < m.mesh.numRegions; reg++ {
			// If the region is not an ocean region, skip it.
			if m.Elevation[reg] > 0 {
				continue
			}
			// Normalize the current vector.
			regCurrentVec[reg] = normal2(regCurrentVec[reg])
		}
	}
	m.RegionToOceanVec = regCurrentVec
}

func (m *Geo) calcCurrentPressure(currentVecs [][2]float64) []float64 {
	// Calculate the pressure in each region based on inflow and outflow.
	// The pressure is the sum of the inflow and outflow vectors.
	// A non-zero pressure means that the current is not balanced.
	// The pressure is used to calculate the ocean currents.

	// Build the region to region neighbor vectors.

	// TODO: Use the function (but this leads to odd results?????)
	regToRegNeighborVec := make([]map[int][2]float64, m.mesh.numRegions)
	for reg := 0; reg < m.mesh.numRegions; reg++ {
		regToRegNeighborVec[reg] = make(map[int][2]float64)
		for _, neighbor := range m.GetRegNeighbors(reg) {
			// TODO: This will cause artifacts around +/- 180 degrees.
			regToRegNeighborVec[reg][neighbor] = normalize2(calcVecFromLatLong(m.LatLon[reg][0], m.LatLon[reg][1], m.LatLon[neighbor][0], m.LatLon[neighbor][1]))
		}
	}

	// Calculate the pressure in each region.
	pressure := make([]float64, m.mesh.numRegions)
	for reg := 0; reg < m.mesh.numRegions; reg++ {
		// If the region is not an ocean region, skip it.
		if m.Elevation[reg] > 0 {
			continue
		}
		// We need to iterate over the neighbors and see if a current vector
		// is set. If so, we need to add the dot product times the magnitude
		// of the neighbor vector to the pressure.
		for _, neighbor := range m.GetRegNeighbors(reg) {
			// Skip the neighbor if it is not an ocean region.
			if m.Elevation[neighbor] > 0 {
				continue
			}

			// Incoming current.
			if currentVecs[neighbor] != [2]float64{0, 0} {
				// Calculate the dot product of the vector from the neighbor to the
				// current region and the current vector of the neighbor.
				// This will tell us how much of the current vector of the neighbor
				// is flowing into the current region.
				// We multiply the dot product with the magnitude of the neighbor
				// vector to get the pressure.
				// Only add the pressure if the dot product is positive.
				dot := dot2(regToRegNeighborVec[neighbor][reg], normalize2(currentVecs[neighbor]))
				if dot > 0 {
					pressure[reg] += dot * len2(currentVecs[neighbor])
				}
			}

			// Outgoing current.
			if currentVecs[reg] != [2]float64{0, 0} {
				// Now do the opposite. If our current streams into the neighbor,
				// we need to subtract the pressure from the current region.
				dot := dot2(regToRegNeighborVec[reg][neighbor], normalize2(currentVecs[reg]))
				if dot > 0 {
					pressure[reg] -= dot * len2(currentVecs[reg])
				}
			}
		}
		// The remaining pressure indicates unbalanced currents.
	}
	return pressure
}

func (m *Geo) seedOceanCurrents(currents [][2]float64) {
	// Seed the ocean currents with the given vectors.
	for reg := 0; reg < m.mesh.numRegions; reg++ {
		// If the region is not an ocean region, set the vector to zero.
		if m.Elevation[reg] > 0 {
			currents[reg] = [2]float64{0, 0}
			continue
		}
		lat := math.Abs(m.LatLon[reg][0])
		if lat <= 2.2 && lat >= 0.5 {
			// Initialize the currents at the equator to flow from west to east.
			currents[reg] = [2]float64{-1.0, 0.0}
		} else if lat >= 59.2 && lat <= 62.2 {
			// Initialize the currents at the arctic / antarctic circles to flow
			// from east to west.
			currents[reg] = [2]float64{1.0, 0.0}
		}
	}
}

func (m *Geo) getRegionToNeighborVec() []map[int][2]float64 {
	useFancyFunc := false
	regToNeighborVec := make([]map[int][2]float64, m.mesh.numRegions)
	for reg := 0; reg < m.mesh.numRegions; reg++ {
		regToNeighborVec[reg] = make(map[int][2]float64)
		rLat, rLon := m.LatLon[reg][0], m.LatLon[reg][1]
		for _, neighbor := range m.GetRegNeighbors(reg) {
			if useFancyFunc {
				regToNeighborVec[reg][neighbor] = normalize2(calcVecFromLatLong(rLat, rLon, m.LatLon[neighbor][0], m.LatLon[neighbor][1]))
				continue
			}
			// Calculate the vector between the current region and the neighbor from lat/long.
			nbLat, nbLon := m.LatLon[neighbor][0], m.LatLon[neighbor][1]

			vec := [2]float64{nbLon - rLon, nbLat - rLat}
			regToNeighborVec[reg][neighbor] = normal2(vec)

		}
	}
	return regToNeighborVec
}

func (m *Geo) genOceanCurrents2() {
	regCurrentVec := make([][2]float64, m.mesh.numRegions)

	// Seed the ocean currents.
	m.seedOceanCurrents(regCurrentVec)

	// Build the region to region neighbor vectors.
	regToRegNeighborVec := m.getRegionToNeighborVec()

	deflectCurrent := func(r int) [2]float64 {
		if m.Elevation[r] > 0 || regCurrentVec[r] == zero2 {
			return [2]float64{0, 0}
		}

		// Check if the current vector is pointing into the land.
		// If so, we need to deflect it.
		currentVec := regCurrentVec[r]
		maxDotLand := math.Inf(-1)
		maxDotOcean := math.Inf(-1)
		maxRegOcean := -1
		for _, neighbor := range m.GetRegNeighbors(r) {
			dot := dot2(normalize2(regToRegNeighborVec[r][neighbor]), normalize2(currentVec))
			if m.Elevation[neighbor] > 0 {
				if dot > maxDotLand {
					maxDotLand = dot
				}
			} else {
				if dot > maxDotOcean || maxRegOcean < 0 {
					maxDotOcean = dot
					maxRegOcean = neighbor
				}
			}
		}

		// If the max dot product for land is greater than the max dot product
		// for ocean, we need to deflect the current vector.
		if maxDotLand >= 0 && maxRegOcean >= 0 {
			return normalize2(regToRegNeighborVec[r][maxRegOcean])
		}
		return currentVec
	}
	newVec := make([][2]float64, m.mesh.numRegions)

	// Loop once so we can see what it looks like.
	for i := 0; i < 1; i++ {
		// Deflect the current vectors.
		for r := range regCurrentVec {
			newVec[r] = deflectCurrent(r)
		}
		regCurrentVec, newVec = newVec, regCurrentVec
	}
	m.RegionToOceanVec = regCurrentVec
}

func (m *Geo) assignOceanCurrents3() {
	// Adapted from:
	// https://github.com/FreezeDriedMangos/realistic-planet-generation-and-simulation/blob/main/src/Generate_Weather.js

	// Assign the ocean currents to the mesh.
	const latLeeway = 2
	seedSupergroup := make(map[int]int)

	//const numCurrentBandsPerHemisphere = 2
	//const step = 90/numCurrentBandsPerHemisphere
	//const seedLatsBands = []
	//for(let l = step / 2; l < 90; l += step) seedLatsBands.push(l)

	// initialize output variable
	r_currents := make([][2]float64, m.mesh.numRegions)

	// select seeds. all regions that are "close enough" to the center of a current band becomes a seed
	var seeds []int
	bands := [][2]float64{
		{75.0, 60.0},
		// {37.5, 22.5},
		// {15.0, 0.01},
	}
	for _, band := range bands {
		highLatBand := band[0]
		lowLatBand := band[1]
		//const highLatBand = 75.0 // 67.5
		//const lowLatBand = 60.0  // 22.5
		for r := 0; r < m.mesh.numRegions; r++ {
			if m.Elevation[r] > 0 {
				continue
			}

			lat := m.LatLon[r][0]
			//lon := m.LatLon[r][1]

			// let band = seedLatBands.filter(bandLat => Math.abs(lat-bandLat) < latLeeway)[0]
			// if (band === undefined) continue
			// seeds.push(r)

			if math.Abs(math.Abs(lat)-lowLatBand) <= latLeeway {
				seeds = append(seeds, r)
				if lat > 0 {
					seedSupergroup[r] = 0
				} else {
					seedSupergroup[r] = 1
				}
			} else if math.Abs(math.Abs(lat)-highLatBand) <= latLeeway {
				seeds = append(seeds, r)
				if lat > 0 {
					seedSupergroup[r] = 2
				} else {
					seedSupergroup[r] = 3
				}
			}
		}
	}

	// assign every region to the closest seed, accounting for obstacles (where land is an obstacle)
	// a bunch of regions assigned to the same seed are called a group
	groups := m.bfsMetaVoronoi(seeds, func(r int) bool { return m.Elevation[r] <= 0 }, true)
	outRegs := make([]int, 0, 8)

	// merge groups that are touching
	for r := 0; r < m.mesh.numRegions; r++ {
		if m.Elevation[r] > 0 {
			continue
		}
		for _, sr := range m.mesh.r_circulate_r(outRegs, r) {
			if groups[r] == groups[sr] {
				continue
			}
			if groups[r] == -1 || groups[sr] == -1 {
				continue
			}
			if seedSupergroup[groups[r]] != seedSupergroup[groups[sr]] {
				continue // if r and sr are in different "bands", aka supergroups, do not merge them
			}
			// assign all regions belonging to the same group as sr, to the same group as r
			for i := range groups {
				if groups[i] == groups[sr] {
					groups[i] = groups[r]
				}
			}
		}
	}

	// determine how close each region is to the edge of its group
	distFromEdge := initRegionSlice(m.mesh.numRegions)

	frontier := make([]int, 0, m.mesh.numRegions)
	groupmates := make([][]int, m.mesh.numRegions)
	for r, g := range groups {
		if g != -1 {
			groupmates[g] = append(groupmates[g], r)
		}
	}

	for _, seed := range seeds {
		frontier = frontier[:0]
		groupmatesForSeed := groupmates[seed]
		for _, r := range groupmatesForSeed {
			for _, nr := range m.mesh.r_circulate_r(outRegs, r) {
				if groups[nr] != groups[r] {
					frontier = append(frontier, r)
					distFromEdge[r] = 0
					break
				}
			}
		}

		for fidx := 0; fidx < len(frontier); fidx++ {
			curr := frontier[fidx]
			// frontier = frontier[1:]
			for _, nr := range m.mesh.r_circulate_r(outRegs, curr) {
				if distFromEdge[nr] < 0 {
					distFromEdge[nr] = 9999
				}
				if distFromEdge[nr] <= distFromEdge[curr]+1 {
					continue
				}
				distFromEdge[nr] = distFromEdge[curr] + 1
				frontier = append(frontier, nr)
			}
		}

		// assign current vectors
		var maxDist int
		for _, r := range groupmatesForSeed {
			if distFromEdge[r] > maxDist {
				maxDist = distFromEdge[r]
			}
		}
		for _, r := range groupmatesForSeed {
			var inwardDirRaw [2]float64
			for _, nr := range m.mesh.r_circulate_r(outRegs, r) {
				// if this neighbor has a smaller distance to edge, or belongs to a different gyre, the inward dir points away from it (so we add dirFromTo(nr, r), aka the dir away from nr)
				if groups[nr] != groups[r] {
					inwardDirRaw = add2(inwardDirRaw, m.dirVecFromToRegs(nr, r))
				} else if distFromEdge[nr] < distFromEdge[r] {
					inwardDirRaw = add2(inwardDirRaw, m.dirVecFromToRegs(nr, r))
				} else if distFromEdge[nr] == distFromEdge[r] {
					continue
				} else {
					// if the neighbor has a larger dist to the edge, the inward dir points towards it
					inwardDirRaw = add2(inwardDirRaw, m.dirVecFromToRegs(r, nr))
				}
			}
			// normalize inward dir
			inwardDir := normal2(inwardDirRaw)

			clockwise := seedSupergroup[seed] == 1 || seedSupergroup[seed] == 2
			// since currents at gyre edges are a mess, we'll decrease their magnitude
			// map.r_currents[r] = setMagnitude(perpendicular, 2*(1-distFromEdge[r]/maxDist))
			if clockwise {
				r_currents[r][0] = -inwardDir[1]
				r_currents[r][1] = inwardDir[0]
			} else {
				r_currents[r][0] = inwardDir[1]
				r_currents[r][1] = -inwardDir[0]
			}
			// if(distFromEdge[r] === 0) map.r_currents[r] = setMagnitude(perpendicular, 0.4)
		}
	}

	// TODO: Create a proper solution for ocean current vectors that
	// doesn't affect vectors on land.
	for i := 0; i < 4; i++ {
		m.RegionToOceanVec = m.interpolateWindVecs(r_currents, 1)
		// Reset all vectors that are not in the ocean
		for r := 0; r < m.mesh.numRegions; r++ {
			if m.Elevation[r] >= 0 {
				m.RegionToOceanVec[r][0] = 0.0
				m.RegionToOceanVec[r][1] = 0.0
			}
		}
		r_currents = m.RegionToOceanVec
	}
}

// bfsMetaVoronoi is similar to distanceField, but instead of returning the distance to the closest seed, it returns the seed that is closest to the region.
// TODO: Optimize, fix, maybe replace.
func (m *BaseObject) bfsMetaVoronoi(seeds []int, includeCondition func(int) bool, forceIncludeIsolatedRegions bool) []int {
	// Reset the random number generator.
	m.resetRand()
	rGroup := initRegionSlice(m.mesh.numRegions)
	isSeed := make([]bool, m.mesh.numRegions)
	for _, seed := range seeds {
		isSeed[seed] = true
	}

	mesh := m.mesh
	numRegions := mesh.numRegions

	// Initialize the queue for the breadth first search with
	// the seed regions.
	queue := make([]int, len(seeds), numRegions)
	for i, r := range seeds {
		queue[i] = r
		rGroup[r] = r
	}

	// Allocate a slice for the output of mesh.r_circulate_r.
	outRegs := make([]int, 0, 6)

	// Random search adapted from breadth first search.
	// TODO: Improve the queue. Currently this is growing unchecked.
	for queueOut := 0; queueOut < len(queue); queueOut++ {
		pos := queueOut + m.rand.Intn(len(queue)-queueOut)
		currentReg := queue[pos]
		queue[pos] = queue[queueOut]
		for _, nbReg := range mesh.r_circulate_r(outRegs, currentReg) {
			if rGroup[nbReg] >= 0 || !includeCondition(nbReg) {
				continue
			}

			// Grow the assigned region to the current region.
			rGroup[nbReg] = rGroup[currentReg]
			queue = append(queue, nbReg)
		}

		// If we have consumed over 1000000 elements in the queue,
		// we reset the queue to the remaining elements.
		if queueOut > 10000 {
			n := copy(queue, queue[queueOut:])
			queue = queue[:n]
			queueOut = 0
		}
	}
	return rGroup
}

func (m *Geo) getPreviousNeighbor(outregs []int, r int, vec [2]float64) int {
	return m.getClosestNeighbor(outregs, r, [2]float64{-vec[0], -vec[1]})
}
