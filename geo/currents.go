package geo

import (
	"log"
	"math"
	"sort"

	"github.com/Flokey82/genworldvoronoi/various"
)

// assignOceanCurrents will calculate the ocean currents for the map.
// NOTE: THIS IS NOT WORKING YET!!!!!
// For interesting approaches, see:
// https://forhinhexes.blogspot.com/search/label/Currents
// https://github.com/FreezeDriedMangos/realistic-planet-generation-and-simulation
func (m *Geo) assignOceanCurrents() {
	regCurrentVec := make([][2]float64, m.SphereMesh.NumRegions)

	// Let's calculate the ocean currents.
	// Build the region to region neighbor vectors.
	regToRegNeighborVec := m.getRegionToNeighborVec()

	// Calculate the pressure in each ocean region.
	regPressure := m.CalcCurrentPressure(regCurrentVec)

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
			nbVec := various.CalcVecFromLatLong(m.LatLon[reg][0], m.LatLon[reg][1], m.LatLon[nb][0], m.LatLon[nb][1])
			nbVec = various.Normal2(nbVec)

			// Get the dot product of the current vector and the
			// vector to the neighbor.
			dot := various.Dot2(vec, nbVec)

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
					vec = various.Add2(vec, various.Scale2(regToRegNeighborVec[reg][oceanReg], p-minPressure))
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
					vec = various.Add2(vec, various.Scale2(regToRegNeighborVec[reg][oceanReg], p-maxPressure))
				}
			}
			vec = various.Normalize2(vec)
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
				if vecDot == various.Zero2 {
					vecDot = various.Normal2(vec)
				}
				added := various.Scale2(regToRegNeighborVec[reg][oceanReg], 1.0)
				regCurrentVec[oceanReg] = various.Normal2(various.Add2(regCurrentVec[oceanReg], added))

				// Since we want the vector to snake along the coast, we need to
				// rotate the current vector towards the closest ocean region.
				// Find the highest dot product (closest to the current vector).

				// Rotate the current vector towards the closest ocean region by
				// averaging the current vector and the vector to the closest ocean
				// region.
				vec = various.Normalize2(various.Add2(vec, oceanRegionVec[maxDot1Idx]))
			}

			// Normalize the current vector.
			vec = various.Normalize2(vec)
		}

		// Return the current vector.
		return vec
	}

	// propagateCurrent propagates the current vector from the given region to
	// all neighboring regions until the current vector is zero.
	var propagateCurrent func(reg int)
	propagateCurrent = func(reg int) {
		useDot := true
		if regCurrentVec[reg] == various.Zero2 {
			return
		}
		// Propagate the current vector to all neighboring regions.
		for _, neighbor := range m.GetRegNeighbors(reg) {
			// Skip elevation above sea level and regions with a current vector
			// already set.
			if m.Elevation[neighbor] > 0 || regCurrentVec[neighbor] != various.Zero2 {
				continue
			}
			// Set the current vector using the dot product to scale the vector
			// towards the neighboring region.
			if useDot {
				dot := various.Dot2(regCurrentVec[reg], regToRegNeighborVec[reg][neighbor])
				regCurrentVec[neighbor] = various.Add2(regCurrentVec[reg], various.Scale2(regToRegNeighborVec[reg][neighbor], dot))
			} else {
				regCurrentVec[neighbor] = various.Add2(regCurrentVec[reg], various.Scale2(regToRegNeighborVec[reg][neighbor], 0.5))
			}
			regCurrentVec[neighbor] = various.Normalize2(regCurrentVec[neighbor])

			// Propagate the current vector to the neighboring region.
			propagateCurrent(neighbor)
		}
	}

	// Reinforce the primary currents.
	m.seedOceanCurrents(regCurrentVec)
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
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
			for r := 0; r < m.mesh.NumRegions; r++ {
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
		regPressure = m.CalcCurrentPressure(regCurrentVec)
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
				if m.Elevation[nb] > 0 || regCurrentVec[nb] == various.Zero2 {
					continue
				}
				sumVec = various.Add2(sumVec, regCurrentVec[nb])
				numVec++
			}
			if numVec > 0 {
				if regCurrentVec[r] != various.Zero2 {
					sumVec = various.Add2(sumVec, regCurrentVec[r])
					numVec++
				}
				regCurrentVec[r] = various.Normalize2(sumVec)
			}
			if regCurrentVec[r] == various.Zero2 {
				continue
			}
			regCurrentVec[r] = deflectAndSplit(r, true)
		}
	}

	m.RegionToOceanVec = regCurrentVec
}

// getCurrentSortOrder returns the regions sorted by their position and downstream direction.
func (m *Geo) getCurrentSortOrder(revVecs [][2]float64, reverse bool) []int {
	_, orderedRegs := m.GetVectorSortOrder(revVecs, reverse)
	return orderedRegs
}

func (m *Geo) assignOceanCurrentsInflowOutflow() {
	// Calculate the inflow and outflow of each ocean region, which can be used
	// to calculate the ocean currents.

	// We start off by setting the primary ocean current vectors and then iterate
	// over the regions to calculate the inflow and outflow vectors, depending
	// on the pressure in the region.
	regCurrentVec := make([][2]float64, m.SphereMesh.NumRegions)

	// Build the region to region neighbor vectors.
	regToRegNeighborVec := m.getRegionToNeighborVec()

	// Loop a few times to establis an equilibrium.
	for i := 0; i < 100; i++ {
		// Set the primary ocean current vectors.
		m.seedOceanCurrents(regCurrentVec)

		// Calculate the pressure in each ocean region.
		regPressure := m.CalcCurrentPressure(regCurrentVec)

		// Calculate the inflow and outflow vectors based on the pressure difference.
		for reg := 0; reg < m.SphereMesh.NumRegions; reg++ {
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
				dot := various.Dot2(various.Normalize2(regToRegNeighborVec[neighbor][reg]), various.Normalize2(regCurrentVec[neighbor]))

				// If the dot product is positive, the neighbor is flowing into the
				// current region, so it is part of the inflow vector.
				if dot > 0 {
					inflowVec = various.Add2(inflowVec, various.Scale2(regCurrentVec[neighbor], dot))
				} else if dot < 0 {
					// If the dot product is negative, the neighbor is flowing out of
					// the current region, so it is part of the outflow vector.
					outflowVec = various.Add2(outflowVec, various.Scale2(regCurrentVec[neighbor], -dot))
				}
			}

			// The difference in magnitude between the inflow and outflow vectors
			// indicates the pressure difference.
			lenIn := various.Len2(inflowVec)
			lenOut := various.Len2(outflowVec)
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
					dot := various.Dot2(various.Normalize2(regToRegNeighborVec[neighbor][reg]), various.Normalize2(regCurrentVec[neighbor]))
					// If the dot product is positive, the neighbor is flowing into the
					// current region, so it is part of the inflow vector.
					if dot > 0 {
						// Calculate the amount of the inflow vector that will be
						// transferred to the neighbor.
						transfer := dot * regPressure[reg]
						// Add the transfer to the neighbor's current vector.
						regCurrentVec[neighbor] = various.Add2(regCurrentVec[neighbor], various.Scale2(inflowVec, transfer))
						// Subtract the transfer from the current region's current vector.
						regCurrentVec[reg] = various.Add2(regCurrentVec[reg], various.Scale2(inflowVec, -transfer))
					}
				}
			}
		}

		// Normalize the current vectors.
		for reg := 0; reg < m.SphereMesh.NumRegions; reg++ {
			// If the region is not an ocean region, skip it.
			if m.Elevation[reg] > 0 {
				continue
			}
			// Normalize the current vector.
			regCurrentVec[reg] = various.Normal2(regCurrentVec[reg])
		}
	}
	m.RegionToOceanVec = regCurrentVec
}

func (m *Geo) CalcCurrentPressure(currentVecs [][2]float64) []float64 {
	// Calculate the pressure in each region based on inflow and outflow.
	// The pressure is the sum of the inflow and outflow vectors.
	// A non-zero pressure means that the current is not balanced.
	// The pressure is used to calculate the ocean currents.

	// Build the region to region neighbor vectors.

	// TODO: Use the function (but this leads to odd results?????)
	regToRegNeighborVec := make([]map[int][2]float64, m.SphereMesh.NumRegions)
	for reg := 0; reg < m.SphereMesh.NumRegions; reg++ {
		regToRegNeighborVec[reg] = make(map[int][2]float64)
		for _, neighbor := range m.GetRegNeighbors(reg) {
			// TODO: This will cause artifacts around +/- 180 degrees.
			regToRegNeighborVec[reg][neighbor] = various.Normalize2(various.CalcVecFromLatLong(m.LatLon[reg][0], m.LatLon[reg][1], m.LatLon[neighbor][0], m.LatLon[neighbor][1]))
		}
	}

	// Calculate the pressure in each region.
	pressure := make([]float64, m.SphereMesh.NumRegions)
	for reg := 0; reg < m.SphereMesh.NumRegions; reg++ {
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
				dot := various.Dot2(regToRegNeighborVec[neighbor][reg], various.Normalize2(currentVecs[neighbor]))
				if dot > 0 {
					pressure[reg] += dot * various.Len2(currentVecs[neighbor])
				}
			}

			// Outgoing current.
			if currentVecs[reg] != [2]float64{0, 0} {
				// Now do the opposite. If our current streams into the neighbor,
				// we need to subtract the pressure from the current region.
				dot := various.Dot2(regToRegNeighborVec[reg][neighbor], various.Normalize2(currentVecs[reg]))
				if dot > 0 {
					pressure[reg] -= dot * various.Len2(currentVecs[reg])
				}
			}
		}
		// The remaining pressure indicates unbalanced currents.
	}
	return pressure
}

func (m *Geo) seedOceanCurrents(currents [][2]float64) {
	// Seed the ocean currents with the given vectors.
	for reg := 0; reg < m.SphereMesh.NumRegions; reg++ {
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
	regToNeighborVec := make([]map[int][2]float64, m.SphereMesh.NumRegions)
	for reg := 0; reg < m.SphereMesh.NumRegions; reg++ {
		regToNeighborVec[reg] = make(map[int][2]float64)
		rLat, rLon := m.LatLon[reg][0], m.LatLon[reg][1]
		for _, neighbor := range m.GetRegNeighbors(reg) {
			if useFancyFunc {
				regToNeighborVec[reg][neighbor] = various.Normalize2(various.CalcVecFromLatLong(rLat, rLon, m.LatLon[neighbor][0], m.LatLon[neighbor][1]))
				continue
			}
			// Calculate the vector between the current region and the neighbor from lat/long.
			nbLat, nbLon := m.LatLon[neighbor][0], m.LatLon[neighbor][1]

			vec := [2]float64{nbLon - rLon, nbLat - rLat}
			regToNeighborVec[reg][neighbor] = various.Normal2(vec)

		}
	}
	return regToNeighborVec
}

func (m *Geo) genOceanCurrents2() {
	regCurrentVec := make([][2]float64, m.SphereMesh.NumRegions)

	// Seed the ocean currents.
	m.seedOceanCurrents(regCurrentVec)

	// Build the region to region neighbor vectors.
	regToRegNeighborVec := m.getRegionToNeighborVec()

	deflectCurrent := func(r int) [2]float64 {
		if m.Elevation[r] > 0 || regCurrentVec[r] == various.Zero2 {
			return [2]float64{0, 0}
		}

		// Check if the current vector is pointing into the land.
		// If so, we need to deflect it.
		currentVec := regCurrentVec[r]
		maxDotLand := math.Inf(-1)
		maxDotOcean := math.Inf(-1)
		maxRegOcean := -1
		for _, neighbor := range m.GetRegNeighbors(r) {
			dot := various.Dot2(various.Normalize2(regToRegNeighborVec[r][neighbor]), various.Normalize2(currentVec))
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
			return various.Normalize2(regToRegNeighborVec[r][maxRegOcean])
		}
		return currentVec
	}
	newVec := make([][2]float64, m.SphereMesh.NumRegions)

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
	useGoRoutines := true

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
	r_currents := make([][2]float64, m.SphereMesh.NumRegions)

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
		for r := 0; r < m.SphereMesh.NumRegions; r++ {
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
	for r, rgr := range groups {
		if rgr == -1 || m.Elevation[r] > 0 {
			continue
		}
		for _, sr := range m.SphereMesh.R_circulate_r(outRegs, r) {
			sgr := groups[sr]
			if sgr == -1 || rgr == sgr || seedSupergroup[rgr] != seedSupergroup[sgr] {
				continue // if r and sr are in different "bands", aka supergroups, do not merge them
			}
			// assign all regions belonging to the same group as sr, to the same group as r
			chunkProcessor := func(start, end int) {
				for i := start; i < end; i++ {
					if groups[i] == sgr {
						groups[i] = rgr
					}
				}
			}
			if useGoRoutines {
				// use goroutines
				various.KickOffChunkWorkers(len(groups), chunkProcessor)
			} else {
				// do not use goroutines
				chunkProcessor(0, len(groups))
			}
		}
	}

	// determine how close each region is to the edge of its group
	distFromEdge := initRegionSlice(m.SphereMesh.NumRegions)

	frontier := make([]int, 0, m.SphereMesh.NumRegions)
	groupmates := make([][]int, m.SphereMesh.NumRegions)
	for r, g := range groups {
		if g != -1 {
			groupmates[g] = append(groupmates[g], r)
		}
	}

	for _, seed := range seeds {
		frontier = frontier[:0]
		groupmatesForSeed := groupmates[seed]
		for _, r := range groupmatesForSeed {
			for _, nr := range m.SphereMesh.R_circulate_r(outRegs, r) {
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
			for _, nr := range m.SphereMesh.R_circulate_r(outRegs, curr) {
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

		chunkProcessor := func(start, end int) {
			outRegs := make([]int, 0, 8)
			for idx := start; idx < end; idx++ {
				r := groupmatesForSeed[idx]
				var inwardDirRaw [2]float64
				for _, nr := range m.SphereMesh.R_circulate_r(outRegs, r) {
					// if this neighbor has a smaller distance to edge, or belongs to a different gyre, the inward dir points away from it (so we add dirFromTo(nr, r), aka the dir away from nr)
					if groups[nr] != groups[r] {
						inwardDirRaw = various.Add2(inwardDirRaw, m.DirVecFromToRegs(nr, r))
					} else if distFromEdge[nr] < distFromEdge[r] {
						inwardDirRaw = various.Add2(inwardDirRaw, m.DirVecFromToRegs(nr, r))
					} else if distFromEdge[nr] == distFromEdge[r] {
						continue
					} else {
						// if the neighbor has a larger dist to the edge, the inward dir points towards it
						inwardDirRaw = various.Add2(inwardDirRaw, m.DirVecFromToRegs(r, nr))
					}
				}
				// normalize inward dir
				inwardDir := various.Normal2(inwardDirRaw)

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
		if useGoRoutines {
			various.KickOffChunkWorkers(len(groupmatesForSeed), chunkProcessor)
		} else {
			chunkProcessor(0, len(groupmatesForSeed))
		}
	}

	// TODO: Create a proper solution for ocean current vectors that
	// doesn't affect vectors on land.
	for i := 0; i < 4; i++ {
		m.RegionToOceanVec = m.interpolateWindVecs(r_currents, 1)
		// Reset all vectors that are not in the ocean
		for r := 0; r < m.SphereMesh.NumRegions; r++ {
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
	m.ResetRand()
	rGroup := initRegionSlice(m.SphereMesh.NumRegions)
	isSeed := make([]bool, m.SphereMesh.NumRegions)
	for _, seed := range seeds {
		isSeed[seed] = true
	}

	mesh := m.SphereMesh
	numRegions := mesh.NumRegions

	// Initialize the queue for the breadth first search with
	// the seed regions.
	queue := make([]int, len(seeds), numRegions)
	for i, r := range seeds {
		queue[i] = r
		rGroup[r] = r
	}

	// Allocate a slice for the output of mesh.R_circulate_r.
	outRegs := make([]int, 0, 6)

	// Random search adapted from breadth first search.
	// TODO: Improve the queue. Currently this is growing unchecked.
	for queueOut := 0; queueOut < len(queue); queueOut++ {
		pos := queueOut + m.Rand.Intn(len(queue)-queueOut)
		currentReg := queue[pos]
		queue[pos] = queue[queueOut]
		for _, nbReg := range mesh.R_circulate_r(outRegs, currentReg) {
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
	return m.GetClosestNeighbor(outregs, r, [2]float64{-vec[0], -vec[1]})
}
