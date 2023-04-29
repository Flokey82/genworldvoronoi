package genworldvoronoi

import (
	"math"

	"github.com/Flokey82/go_gens/vectors"
)

// getGlobalWindVector returns a vector for the global wind at the given latitude.
// NOTE: This is based on the trade winds on... well, earth.
// See: https://en.wikipedia.org/wiki/Trade_winds
func getGlobalWindVector(lat float64) [2]float64 {
	// Based on latitude, we calculate the wind vector angle.
	var degree float64
	if latAbs := math.Abs(lat); latAbs >= 0 && latAbs <= 30 {
		// +30° ... 0°, 0° ... -30° -> Primitive Hadley Cell.
		// In a Hadley cell, we turn the wind vector until we are exactly parallel with the equator once we reach 0° Lat.
		// TODO: This is probably not perfectly parallel at the equator.
		change := 90 * latAbs / 30
		if lat > 0 {
			degree = 180 + change // Northern hemisphere.
		} else {
			degree = 180 - change // Southern hemisphere.
		}
	} else if latAbs > 30 && latAbs <= 60 {
		// +60° ... +30°, -30° ... -60° -> Primitive Mid Latitude Cell.
		// In a mid latitude cell, we turn the wind vector until we are exactly parallel with the 60° Lat.
		// TODO: This is probably not a full 90° turn. Fix this
		change := 90 * (latAbs - 30) / 30
		if lat > 0 {
			degree = 90 - change // Northern hemisphere.
		} else {
			degree = 270 + change // Southern hemisphere.
		}
	} else {
		// NOTE: This is buggy or at least "not nice".
		// +90° ... +60°, -60° ... -90° -> Primitive Hadley Cell.
		// In a polar cell, we turn the wind vector until we are exactly parallel with the equator once we reach 60° Lat.
		change := 90 * (latAbs - 60) / 30
		if lat > 0 {
			degree = 180 + change // Northern hemisphere.
		} else {
			degree = 180 - change // Southern hemisphere.
		}
	}
	rad := degToRad(degree)
	return [2]float64{math.Cos(rad), math.Sin(rad)}
}

const (
	localWindModeTemperature = iota
	localWindModeAltitude
	localWindModeMixed
)

// assignWindVectors constructs faux global wind cells reminiscent of a simplified earth model.
// NOTE: This function includes an experimental part that calculates local winds that are influenced
// by the topography / elevation changes. Please note that the code for local winds is incomplete.
func (m *Geo) assignWindVectors() {
	// Based on latitude of each region, we calculate the wind vector.
	regWindVec := make([][2]float64, m.numRegions)
	chunkProcessorWind := func(start, end int) {
		for r := start; r < end; r++ {
			regWindVec[r] = getGlobalWindVector(m.LatLon[r][0])
		}
	}

	useGoRoutines := true
	if useGoRoutines {
		// Use goroutines to calculate wind vectors.
		kickOffChunkWorkers(m.numRegions, chunkProcessorWind)
	} else {
		// Use single threaded calculation of wind vectors.
		chunkProcessorWind(0, m.numRegions)
	}

	// Select the mode for calculating local wind vectors.
	calcMode := localWindModeMixed

	// Get the chunk processing function for the local wind vectors.
	var chunkProcessor func(start, end int)

	// Local wind vectors.
	regWindVecLocal := make([][2]float64, m.numRegions)

	// NOTE: This is currently overridden by the altitude changes below.
	_, maxElev := minMax(m.Elevation)
	switch calcMode {
	case localWindModeTemperature:
		// Add local wind vectors based on local temperature gradients.
		//
		// NOTE: You won't be happy about the results of the temp gradient anyway
		// since everything becomes quite "patchy".
		//
		// In plain English: This is garbage code :(
		// I suspect that the wind is deflected too much by minimal temperature changes
		// and I am too lazy to really look into it.

		// Determine all sea regions.
		var seaRegs []int
		for r := 0; r < m.SphereMesh.numRegions; r++ {
			if m.Elevation[r] <= 0 {
				seaRegs = append(seaRegs, r)
			}
		}

		chunkProcessor = func(start, end int) {
			outRegs := make([]int, 0, 8)
			regDistanceSea := m.assignDistanceField(seaRegs, make(map[int]bool))
			for r := start; r < end; r++ {
				regVec := regWindVec[r]
				lat := m.LatLon[r][0]
				lon := m.LatLon[r][1]
				tempReg := getMeanAnnualTemp(lat) - getTempFalloffFromAltitude(maxAltitudeFactor*m.Elevation[r]/maxElev)
				if m.Elevation[r] < 0 {
					// TODO: Use actual distance from ocean to calculate temperature falloff.
					tempReg -= 1 / (regDistanceSea[r] + 1)
				}
				// Get temperature for r.
				v := vectors.Normalize(vectors.Vec2{
					X: regVec[0],
					Y: regVec[1],
				})
				for _, nb := range m.SphereMesh.r_circulate_r(outRegs, r) {
					nbLat := m.LatLon[nb][0]
					nbLon := m.LatLon[nb][1]
					tempNb := getMeanAnnualTemp(nbLat) - getTempFalloffFromAltitude(maxAltitudeFactor*m.Elevation[nb]/maxElev)
					if m.Elevation[nb] < 0 {
						// TODO: Use actual distance from ocean to calculate temperature falloff.
						tempNb -= 1 / (regDistanceSea[nb] + 1)
					}
					ve := calcVecFromLatLong(lat, lon, nbLat, nbLon)
					v = v.Add(vectors.Normalize(vectors.NewVec2(ve[0], ve[1])).Mul(tempNb - tempReg))
				}
				v = vectors.Normalize(v)
				regWindVecLocal[r] = [2]float64{v.X, v.Y}
			}
		}
	case localWindModeAltitude:
		// Add wind deflection based on altitude changes.
		chunkProcessor = func(start, end int) {
			outRegs := make([]int, 0, 8)
			for r := start; r < end; r++ {
				// Get the wind vector for r.
				regVec := regWindVec[r]
				// Get XYZ Position of r.
				regXYZ := convToVec3(m.XYZ[r*3 : r*3+3])
				// Get polar coordinates.
				regLat := m.LatLon[r][0]
				regLon := m.LatLon[r][1]
				h := m.Elevation[r]
				if h < 0 {
					h = 0
				}
				// Add wind vector to neighbor lat/lon to get the "wind vector lat long" or something like that..
				// rLatWind := regLat + regWindVec[r][1]
				// rLonWind := regLon + regWindVec[r][0]
				// Not sure if this is correct... Adding a 2d vector to a lat/lon breaks my brain.
				// TODO: Fix this once and for all.
				rLatWind, rLonWind := addVecToLatLong(regLat, regLon, regWindVec[r])
				rwXYZ := convToVec3(latLonToCartesian(rLatWind, rLonWind)).Normalize()
				v := vectors.Normalize(vectors.Vec2{
					X: regVec[0],
					Y: regVec[1],
				}) // v.Mul(h / maxElev)
				vw := calcVecFromLatLong(regLat, regLon, rLatWind, rLonWind)
				v0 := vectors.Normalize(vectors.Vec2{
					X: vw[0],
					Y: vw[1],
				})

				// Calculate Vector between r and wind_r.
				vb := vectors.Sub3(rwXYZ, regXYZ).Normalize()

				for _, nbReg := range m.SphereMesh.r_circulate_r(outRegs, r) {
					// if is_sea[neighbor_r] {
					//	continue
					// }
					// Calculate dot product of wind vector to vector r -> neighbor_r.
					// Get XYZ Position of r_neighbor.
					rnXYZ := convToVec3(m.XYZ[nbReg*3 : nbReg*3+3])

					// Calculate Vector between r and neighbor_r.
					va := vectors.Sub3(rnXYZ, regXYZ).Normalize()

					// Calculate dot product between va and vb.
					// This will give us how much the current region lies within the wind direction of the
					// current neighbor.
					// See: https://www.scratchapixel.com/lessons/3d-basic-rendering/introduction-to-shading/shading-normals
					dotV := vectors.Dot3(va, vb)
					hnb := m.Elevation[nbReg]
					if hnb < 0 {
						hnb = 0
					}
					if dotV > 0 {
						nbLat := m.LatLon[nbReg][0]
						nbLon := m.LatLon[nbReg][1]
						ve := calcVecFromLatLong(regLat, regLon, nbLat, nbLon)
						vx := vectors.Normalize(v0.Sub(vectors.Normalize(vectors.Vec2{
							X: ve[0],
							Y: ve[1],
						})))
						// The higher the dot product (the more direct the neighbor is in wind direction), the higher
						// the influence of an elevation change. So a steep mountain ahead will slow the wind down.
						// If a steep mountain is to the left, the wind vector will be pushed to the right.
						deltaElev := hnb - h // Positive if neighbor is higher.
						v = v.Add(vx.Mul(dotV * (deltaElev) / maxElev))
					}
				}
				v = vectors.Normalize(v)
				regWindVecLocal[r] = [2]float64{v.X, v.Y}
			}
		}
	case localWindModeMixed:
		// Adapted from:
		// https://github.com/FreezeDriedMangos/realistic-planet-generation-and-simulation/blob/main/src/Generate_Weather.js
		WATER_LEVEL := 0.0
		TEMPERATURE_INFLUENCE_FACTOR := 0.5
		ELEVATION_CHANGE_FACTOR := 1.0
		isInit := false

		chunkProcessor = func(start, end int) {
			outRegs := make([]int, 0, 8)
			for r := start; r < end; r++ {
				windDir := regWindVec[r]
				// slowdown/speedup according to elevation change
				blowsPastReg := m.getClosestNeighbor(outRegs, r, windDir)

				// Elevation change is negative if the current region is higher than the region the wind blows past.
				// This will result in wind slowing down if it blows towards a mountain and to speed up if it blows
				// towards a valley.
				elevationChange := math.Max(m.Elevation[blowsPastReg], WATER_LEVEL) - math.Max(m.Elevation[r], WATER_LEVEL)
				windSpeed := (1 - (2*elevationChange)*ELEVATION_CHANGE_FACTOR)
				windSpeed = math.Max(0.1, windSpeed)
				// map.r_wind[r] = 5*(1-(terrain.depthMap[i][j]-terrain.depthMap[k][l])/1000);
				// my windspeed: [0, 3]
				if isInit {
					regWindVecLocal[r] = setMagnitude2(windDir, windSpeed)
					continue
				}

				var acc [2]float64
				for _, nr := range m.SphereMesh.r_circulate_r(outRegs, r) {
					// Magnitude will be positive if the neighbor is warmer than the current region, which will
					// result in a wind vector pointing towards the neighbor.
					magnitude := (m.getRegTemperature(nr, maxElev) - m.getRegTemperature(r, maxElev))
					vec := setMagnitude2(m.dirVecFromToRegs(r, nr), magnitude)
					acc = add2(vec, acc)
				}
				// Add the temperature vector to the wind vector.
				windDir = add2(windDir, setMagnitude2(acc, TEMPERATURE_INFLUENCE_FACTOR))

				// Scale the wind vector to the wind speed.
				regWindVecLocal[r] = setMagnitude2(windDir, windSpeed)
			}
		}
	}

	if useGoRoutines {
		// Split the work into chunks and process them in parallel.
		kickOffChunkWorkers(m.SphereMesh.numRegions, chunkProcessor)
	} else {
		chunkProcessor(0, m.SphereMesh.numRegions)
	}

	// Average wind vectors using neighbor vectors.
	interpolationSteps := 0
	m.RegionToWindVec = m.interpolateWindVecs(regWindVec, interpolationSteps)

	interpolationStepsLocal := 4
	m.RegionToWindVecLocal = m.interpolateWindVecs(regWindVecLocal, interpolationStepsLocal)
}

// interpolateWindVecs interpolates the given wind vectors at their respective regions by
// mixing them with the wind vectors of their neighbor regions.
func (m *Geo) interpolateWindVecs(in [][2]float64, steps int) [][2]float64 {
	// Average wind vectors using neighbor vectors.
	outRegs := make([]int, 0, 8)
	for i := 0; i < steps; i++ {
		regWindVecInterpolated := make([][2]float64, m.SphereMesh.numRegions)
		for r := range regWindVecInterpolated {
			// Copy the original wind vector.
			resVec := in[r]

			// Add the wind vectors of the neighbor regions.
			var count int
			for _, nbReg := range m.SphereMesh.r_circulate_r(outRegs, r) {
				resVec[0] += in[nbReg][0]
				resVec[1] += in[nbReg][1]
				count++
			}
			resVec[0] /= float64(count + 1)
			resVec[1] /= float64(count + 1)
			regWindVecInterpolated[r] = resVec
		}
		in = regWindVecInterpolated
	}
	return in
}

func (m *Geo) getWindSortOrder() ([]float64, []int) {
	// TODO: Add bool parameter to switch between local winds and global winds.
	return m.getVectorSortOrder(m.RegionToWindVecLocal, false)
}
