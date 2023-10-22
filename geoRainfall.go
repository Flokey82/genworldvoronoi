package genworldvoronoi

import (
	"log"
	"math"
	"sort"

	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/go_gens/vectors"
)

type biomesParams struct {
	raininess   float64 // 0, 2
	rainShadow  float64 // 0.1, 2
	evaporation float64 // 0, 1
}

const (
	moistTransferDirect   = 0
	moistTransferIndirect = 1
	moistOrderWind        = 0
	moistOrderOther       = 1
)

// assignRainfall is an overengineered logic that is supposed to calculate the transfer
// of moisture across the globe based on global winds using distinct approaches.
// Unfortunately, this is highly bugged and not as useful as the simpler version
// 'assignRainfallBasic'.
func (m *Geo) assignRainfall(numSteps, transferMode, sortOrder int) {
	biomesParam := biomesParams{
		raininess:   0.9,
		rainShadow:  0.9,
		evaporation: 0.9,
	}

	// 1. Initialize
	// 1.1. Determine all sea regions.
	var seaRegs, landRegs []int
	isSea := make([]bool, m.SphereMesh.numRegions)
	for r := 0; r < m.SphereMesh.numRegions; r++ {
		if m.Elevation[r] < 0 {
			isSea[r] = true
			seaRegs = append(seaRegs, r)
		} else {
			landRegs = append(landRegs, r)
		}
	}

	var sortOrderRegs []int
	if sortOrder == moistOrderOther {
		// 1.2. Sort all regions by distance to ocean. Lowest to highest.
		distOrderRegs := make([]int, m.SphereMesh.numRegions)
		for r := 0; r < m.SphereMesh.numRegions; r++ {
			distOrderRegs[r] = r
		}
		regDistanceSea := m.assignDistanceField(seaRegs, make(map[int]bool))
		sort.Slice(distOrderRegs, func(a, b int) bool {
			if regDistanceSea[distOrderRegs[a]] == regDistanceSea[distOrderRegs[b]] {
				return m.Elevation[distOrderRegs[a]] < m.Elevation[distOrderRegs[b]]
			}
			return regDistanceSea[distOrderRegs[a]] < regDistanceSea[distOrderRegs[b]]
		})
		sortOrderRegs = distOrderRegs
	} else {
		// 1.2. Sort the indices in wind-order so we can ensure that we push the moisture
		// in their logical sequence across the globe.
		_, sortOrderRegs = m.getWindSortOrder() // Works reasonably well.
	}

	// 1.3. Get wind vector for every region
	regWindVec := m.RegionToWindVec
	_, maxH := minMax(m.Elevation)

	calcRainfall := func(r int, humidity float64) float64 {
		regElev := m.Elevation[r]
		if regElev < 0 {
			regElev = 0 // Set to sea-level
		}
		heightVal := 1 - (regElev / maxH)
		if humidity > heightVal {
			return biomesParam.rainShadow * (humidity - heightVal)
		}
		return 0
	}

	for step := 0; step < numSteps; step++ {
		log.Println(step)
		// Evaporation.

		// 2. Assign initial moisture of 1.0 to all regions below or at sea level or replenish
		// moisture through evaporation if our moisture is below 0.
		for _, r := range seaRegs {
			if m.Moisture[r] < 1.0 {
				m.Moisture[r] = 1.0
			}
			// m.r_rainfall[r] += biomesParam.raininess * m.r_moisture[r]
		}

		// Rivers should experience some evaporation.
		for r, fluxval := range m.Flux {
			if m.Moisture[r] < fluxval && m.Moisture[r] < 1.0 {
				m.Moisture[r] = 1.0 // TODO: Should depend on available water.
			}
		}

		// Water pools should experience some evaporation.
		for r, poolval := range m.Waterpool {
			if poolval > 0 && m.Moisture[r] < 1.0 {
				m.Moisture[r] = 1.0 // TODO: Should depend on available water.
			}
		}
		// m.interpolateRainfallMoisture(1)

		// 3. Transfer moisture based on wind vectors.
		switch transferMode {
		case moistTransferDirect:
			// 3.1.B For each region, calculate dot product of Vec r -> r_neighbor and wind vector of r.
			//       This will give us the amount of moisture we transfer to the neighbor region.
			// NOTE: This variant copies moisture from the current region to the neighbors that are in wind direction.
			outRegs := make([]int, 0, 8)
			for _, r := range sortOrderRegs {
				count := 0
				// Get XYZ Position of r.
				regXYZ := various.ConvToVec3(m.XYZ[r*3 : r*3+3])
				// Convert to polar coordinates.
				regLat := m.LatLon[r][0]
				regLon := m.LatLon[r][1]

				// Add wind vector to neighbor lat/lon to get the "wind vector lat long" or something like that..
				regToWindVec3 := various.ConvToVec3(various.LatLonToCartesian(regLat+regWindVec[r][1], regLon+regWindVec[r][0])).Normalize()
				for _, nbReg := range m.SphereMesh.r_circulate_r(outRegs, r) {
					if isSea[nbReg] {
						continue
					}
					// Calculate dot product of wind vector to vector r -> neighbor_r.
					// Get XYZ Position of r_neighbor.
					regToNbVec3 := various.ConvToVec3(m.XYZ[nbReg*3 : nbReg*3+3])

					// Calculate Vector between r and neighbor_r.
					va := vectors.Sub3(regToNbVec3, regXYZ).Normalize()

					// Calculate Vector between r and wind_r.
					vb := vectors.Sub3(regToWindVec3, regXYZ).Normalize()

					// Calculate dot product between va and vb.
					// This will give us how much the current region lies within the wind direction of the
					// current neighbor.
					// See: https://www.scratchapixel.com/lessons/3d-basic-rendering/introduction-to-shading/shading-normals
					dotV := vectors.Dot3(va, vb)
					if dotV > 0 {
						// Only positive dot products mean that we lie within 90°, so 'in wind direction'.
						count++
						humidity := m.Moisture[nbReg] + m.Moisture[r]*dotV
						rainfall := m.Rainfall[nbReg] // + biomesParam.raininess*m.r_moisture[r]*dotV
						orographicRainfall := calcRainfall(nbReg, humidity)
						if orographicRainfall > 0.0 {
							rainfall += biomesParam.raininess * orographicRainfall
							humidity -= orographicRainfall
						}
						// TODO: Calculate max humidity at current altitude, temperature, rain off the rest.
						// WARNING: The humidity calculation is off.
						// humidity = math.Min(humidity, 1.0)
						// rainfall = math.Min(rainfall, 1.0)
						m.Rainfall[nbReg] = rainfall
						m.Moisture[nbReg] = humidity
					}
				}
			}
		case moistTransferIndirect:
			// 3.2. For each region, calculate dot product of Vec r -> r_neighbor and wind vector of r_neighbor.
			//    This will give us the amount of moisture we transfer from the neighbor region.
			// NOTE: This variant copies moisture to the current region from the neighbors depending on their wind direction.
			outRegs := make([]int, 0, 8)
			for _, r := range sortOrderRegs {
				count := 0
				sum := 0.0
				// Get XYZ Position of r as vector3
				regVec3 := various.ConvToVec3(m.XYZ[r*3 : r*3+3])
				for _, nbReg := range m.SphereMesh.r_circulate_r(outRegs, r) {
					// Calculate dot product of wind vector to vector r -> neighbor_r.
					// Get XYZ Position of r_neighbor.
					regToNbVec3 := various.ConvToVec3(m.XYZ[nbReg*3 : nbReg*3+3])

					// Convert to polar coordinates.
					rLat := m.LatLon[nbReg][0]
					rLon := m.LatLon[nbReg][1]

					// Add wind vector to neighbor lat/lon to get the "wind vector lat long" or something like that..
					nbToWindVec3 := various.ConvToVec3(various.LatLonToCartesian(rLat+regWindVec[nbReg][1], rLon+regWindVec[nbReg][0])).Normalize()

					// Calculate Vector between r and neighbor_r.
					va := vectors.Sub3(regVec3, regToNbVec3).Normalize()

					// Calculate Vector between neightbor_r and wind_neighbor_r.
					vb := vectors.Sub3(nbToWindVec3, regToNbVec3).Normalize()

					// Calculate dot product between va and vb.
					// This will give us how much the current region lies within the wind direction of the
					// current neighbor.
					// See: https://www.scratchapixel.com/lessons/3d-basic-rendering/introduction-to-shading/shading-normals
					dotV := vectors.Dot3(va, vb)
					if dotV > 0 {
						// Only positive dot products mean that we lie within 90°, so 'in wind direction'.
						count++
						sum += m.Moisture[nbReg] * dotV
					}
				}

				var humidity, rainfall float64
				humidity = m.Moisture[r]
				if count > 0 {
					// TODO: Calculate max humidity at current altitude, temperature, rain off the rest.
					// WARNING: The humidity calculation is off.
					humidity = math.Min(humidity+sum, 1.0) // / float64(count)
					rainfall = math.Min(rainfall+biomesParam.raininess*sum, 1.0)
				}
				if m.Elevation[r] <= 0.0 {
					// evaporation := biomesParam.evaporation * (-m.r_elevation[r])
					// humidity = evaporation
					humidity = m.Moisture[r]
				}
				orographicRainfall := calcRainfall(r, humidity)
				if orographicRainfall > 0.0 {
					rainfall += biomesParam.raininess * orographicRainfall
					humidity -= orographicRainfall
				}
				m.Rainfall[r] = rainfall
				m.Moisture[r] = humidity
			}
		}

		// 4. Average moisture and rainfall.
		// m.interpolateRainfallMoisture(1)
	}
}

func (m *Geo) assignRainfallBasic() {
	// NOTE: This still has issues with the wrap around at +/- 180° long
	biomesParam := biomesParams{
		raininess:   0.9,
		rainShadow:  0.9,
		evaporation: 0.9,
	}

	// Factor for evaporation from rivers, sea and pools.
	humidityFromRiver := 1.0
	humidityFromSea := 1.0
	humidityFromPool := 1.0

	// Sources of moisture.
	evaporateRivers := true // Evaporate moisture from rivers.
	evaporatePools := false // Evaporate moisture from water pools.

	// Number of steps to perform.
	stepsTransport := 2     // Number of moisture transport steps to perform.
	stepsInterpolation := 2 // Number of interpolation steps to perform on the moisture and rainfall.

	_, maxFlux := minMax(m.Flux)
	_, maxPool := minMax(m.Waterpool)
	minElev, maxElev := minMax(m.Elevation)
	if minElev == 0 {
		minElev = 1
	}

	// Sort the indices in wind-order so we can ensure that we push the moisture
	// in their logical sequence across the globe.
	_, windOrderRegs := m.getWindSortOrder()
	regWindVec := m.RegionToWindVecLocal

	// calcRainfall returns the amount of rain shed given the region and humidity.
	calcRainfall := func(r int, humidity float64) float64 {
		elev := m.Elevation[r]
		if elev < 0 {
			elev = 0 // Set to sea-level
		}
		heightVal := 1 - (elev / maxElev)
		if humidity > heightVal {
			return biomesParam.rainShadow * (humidity - heightVal)
		}
		return 0
	}

	// Evaporation.
	// 1. Assign initial moisture of 1.0 to all regions below or at sea level or replenish
	// moisture through evaporation if our moisture is below 0.
	for r, h := range m.Elevation {
		if h <= 0 {
			m.Moisture[r] = math.Max(m.Moisture[r], humidityFromSea)
		}
	}

	// Rivers should experience some evaporation.
	if evaporateRivers {
		for r, fluxval := range m.Flux {
			if m.isRegBigRiver(r) {
				evaporation := humidityFromRiver * fluxval / maxFlux
				m.Moisture[r] = math.Max(m.Moisture[r], evaporation)
			}
		}
	}

	// Water pools should experience some evaporation.
	//
	// NOTE: Currently this is not used since flood algorithms are deactivated so
	// the value for water pools is always 0.
	if evaporatePools {
		for r, poolval := range m.Waterpool {
			if poolval > 0 {
				evaporation := humidityFromPool * poolval / maxPool
				m.Moisture[r] = math.Max(m.Moisture[r], evaporation)
			}
		}
	}

	// Visit regions in wind order and copy the moisture from the neighbor regious that are
	// up-wind.
	//
	// NOTE: Since we start and stop at +- 180° long, we need to run the code several times
	// to ensure that moisture is pushed across the longitude wrap-around.
	outRegs := make([]int, 0, 8)

	// Cache the wind vectors and the dot product of the wind vector and the vector from the
	// region to its neighbors. These values do not change here, so we can safely cache them.
	//
	// This will allow us to transport moisture from the regions up-wind to the regions
	// quickly since we do not recalculate the wind vectors and the dot product for each
	// region on each iteration.

	// Calculate the wind vectors in lat/lon coordinates.
	normalizedWindVecs := make([][2]float64, len(regWindVec))
	// localWindVecWithLatLon := make([][2]float64, len(regWindVec))
	for r := range normalizedWindVecs {
		// rL := m.LatLon[r]
		//
		// Calculate the "end" of the wind vector in lat/lon coordinates.
		// v2Lat, v2Lon := addVecToLatLong(rL[0], rL[1], regWindVec[r])
		// Calculate the cartesian vector from the region to the end of the wind vector.
		// localWindVecWithLatLon[r] = normal2(calcVecFromLatLong(rL[0], rL[1], v2Lat, v2Lon))
		//
		// Old version:
		// localWindVecWithLatLon[nbReg] = normal2(calcVecFromLatLong(nL[0], nL[1], nL[0]+regWindVec[nbReg][1], nL[1]+regWindVec[nbReg][0]))

		// NOTE: The above vector is almost identical to the wind vector (dot product > 0.999)
		// so we can just use the normalized wind vector instead.
		normalizedWindVecs[r] = various.Normal2(regWindVec[r])
	}

	// Calculate the dot product of the wind vector and the vector from the region to its
	// neighbors for each region.
	dotToNeighbors := make([][]float64, len(regWindVec))

	dotChunkProcessor := func(start, end int) {
		outRegs := make([]int, 0, 8)
		for r := start; r < end; r++ {
			rL := m.LatLon[r]
			for _, nbReg := range m.SphereMesh.r_circulate_r(outRegs, r) {
				nL := m.LatLon[nbReg]
				// TODO: Check dot product of wind vector (r) and neighbour->r.
				vVec := normalizedWindVecs[nbReg]
				nVec := various.Normal2(various.CalcVecFromLatLong(nL[0], nL[1], rL[0], rL[1]))
				dotToNeighbors[r] = append(dotToNeighbors[r], various.Dot2(vVec, nVec))
			}
		}
	}

	useGoRoutines := true
	if useGoRoutines {
		kickOffChunkWorkers(len(dotToNeighbors), dotChunkProcessor)
	} else {
		dotChunkProcessor(0, len(dotToNeighbors))
	}

	// Calculate the humidity for each region and transport it to the up-wind regions.
	for i := 0; i < stepsTransport; i++ {
		for _, r := range windOrderRegs {
			// Calculate humidity.
			var humidity float64

			// Use the cached dot product of the wind vector and the vector from the region to its neighbors
			// to determine how much moisture to transport from the neighbor regions.
			for i, nbReg := range m.SphereMesh.r_circulate_r(outRegs, r) {
				dotV := dotToNeighbors[r][i]

				// Check if the neighbor region is up-wind (that the wind blows from neighbor_r to r) / dotV is positive.
				if dotV > 0.0 {
					humidity += m.Moisture[nbReg] * dotV
				}
			}

			// Evaporation.
			if m.Elevation[r] <= 0 {
				evaporation := biomesParam.evaporation * humidityFromSea * m.Elevation[r] / minElev
				humidity = math.Max(humidity, evaporation)
			} else if evaporateRivers && m.isRegBigRiver(r) {
				evaporation := biomesParam.evaporation * humidityFromRiver * m.Flux[r] / maxFlux
				humidity = math.Max(humidity, evaporation)
			} else if evaporatePools && m.Waterpool[r] > 0 {
				evaporation := biomesParam.evaporation * humidityFromPool * m.Waterpool[r] / maxPool
				humidity = math.Max(humidity, evaporation)
			}

			// Calculate orographic rainfall caused by elevation changes.
			rainfall := biomesParam.raininess * calcRainfall(r, humidity)
			m.Rainfall[r] = rainfall
			m.Moisture[r] = humidity - rainfall
		}
	}

	// Interpolate the rainfall and moisture values.
	m.interpolateRainfallMoisture(stepsInterpolation)
}

func (m *Geo) interpolateRainfallMoisture(interpolationSteps int) {
	outRegs := make([]int, 0, 8)
	for i := 0; i < interpolationSteps; i++ {
		regMoistureInterpol := make([]float64, m.SphereMesh.numRegions)
		regRainfallInterpol := make([]float64, m.SphereMesh.numRegions)
		for r := range regMoistureInterpol {
			rMoist := m.Moisture[r]
			rRain := m.Rainfall[r]
			var count int
			for _, nbReg := range m.SphereMesh.r_circulate_r(outRegs, r) {
				// Gravity! Water moves downwards.
				// This is not super-accurate since you'd have to take
				// in account how steep the slope is etc.
				if m.Elevation[r] >= m.Elevation[nbReg] {
					continue
				}
				rMoist += m.Moisture[nbReg]
				rRain += m.Rainfall[nbReg]
				count++
			}
			regMoistureInterpol[r] = rMoist / float64(count+1)
			regRainfallInterpol[r] = rRain / float64(count+1)
		}
		m.Moisture = regMoistureInterpol
		m.Rainfall = regRainfallInterpol
	}
}
