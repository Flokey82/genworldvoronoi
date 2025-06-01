package geo

import (
	"math"

	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/go_gens/utils"
)

type biomesParams struct {
	raininess   float64 // 0, 2
	rainShadow  float64 // 0.1, 2
	evaporation float64 // 0, 1
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

	maxPool := utils.MaxArray(m.Waterpool)

	// Get the elevation values.
	elevs := m.Elevation.GetValues()

	minElev, maxElev := m.Elevation.Min, m.Elevation.Max
	if minElev == 0 {
		minElev = 1
	}

	// Sort the indices in wind-order so we can ensure that we push the moisture
	// in their logical sequence across the globe.
	_, windOrderRegs := m.GetWindSortOrder()
	regWindVec := m.RegionToWindVecLocal

	// calcRainfall returns the amount of rain shed given the region and humidity.
	calcRainfall := func(r int, humidity float64) float64 {
		elev := elevs[r]
		if elev < 0 {
			elev = 0 // Set to sea-level
		}
		heightVal := 1 - (elev / maxElev)
		if humidity > heightVal {
			return biomesParam.rainShadow * (humidity - heightVal)
		}
		return 0
	}

	// Create new slices for the rainfall and moisture values.
	rainfall := make([]float64, m.SphereMesh.NumRegions)
	moisture := make([]float64, m.SphereMesh.NumRegions)
	copy(rainfall, m.Rainfall.Values)
	copy(moisture, m.Moisture.Values)

	// Evaporation.
	// 1. Assign initial moisture of 1.0 to all regions below or at sea level or replenish
	// moisture through evaporation if our moisture is below 0.
	for r, h := range elevs {
		if h <= 0 {
			moisture[r] = max(moisture[r], humidityFromSea)
		}
	}

	// Rivers should experience some evaporation.
	maxFlux := m.Flux.Max
	flux := m.getFluxNoCache(true)
	if evaporateRivers {
		for r, fluxval := range flux {
			if m.IsRegBigRiver(r) {
				evaporation := humidityFromRiver * fluxval / maxFlux
				moisture[r] = max(moisture[r], evaporation)
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
				moisture[r] = max(moisture[r], evaporation)
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
			for _, nbReg := range m.SphereMesh.R_circulate_r(outRegs, r) {
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
		various.KickOffChunkWorkers(len(dotToNeighbors), dotChunkProcessor)
	} else {
		dotChunkProcessor(0, len(dotToNeighbors))
	}

	// Calculate the humidity for each region and transport it to the up-wind regions.
	for i := 0; i < stepsTransport; i++ {
		for _, r := range windOrderRegs {
			// Calculate humidityVal.
			var humidityVal float64

			// Use the cached dot product of the wind vector and the vector from the region to its neighbors
			// to determine how much moisture to transport from the neighbor regions.
			for i, nbReg := range m.SphereMesh.R_circulate_r(outRegs, r) {
				dotV := dotToNeighbors[r][i]

				// Check if the neighbor region is up-wind (that the wind blows from neighbor_r to r) / dotV is positive.
				if dotV > 0.0 {
					humidityVal += moisture[nbReg] * math.Sqrt(dotV)
				}
			}

			// Evaporation.
			if elevs[r] <= 0 {
				evaporation := biomesParam.evaporation * humidityFromSea * elevs[r] / minElev
				humidityVal = math.Max(humidityVal, evaporation)
			} else if evaporateRivers && m.IsRegRiver(r) {
				evaporation := biomesParam.evaporation * humidityFromRiver * flux[r] / maxFlux
				humidityVal = math.Max(humidityVal, evaporation)
			} else if evaporatePools && m.Waterpool[r] > 0 {
				evaporation := biomesParam.evaporation * humidityFromPool * m.Waterpool[r] / maxPool
				humidityVal = math.Max(humidityVal, evaporation)
			}

			// Calculate orographic rainfallVal caused by elevation changes.
			rainfallVal := biomesParam.raininess * calcRainfall(r, humidityVal)
			rainfall[r] = rainfallVal
			moisture[r] = humidityVal - rainfallVal
		}
	}

	// Assign the new values.
	m.Rainfall.SetValues(rainfall)
	m.Moisture.SetValues(moisture)

	// Interpolate the rainfall and moisture values.
	m.interpolateRainfallMoisture(stepsInterpolation)
}

func (m *Geo) interpolateRainfallMoisture(interpolationSteps int) {
	// Copy the moisture and rainfall values.
	moisture := make([]float64, m.SphereMesh.NumRegions)
	rainfall := make([]float64, m.SphereMesh.NumRegions)
	copy(moisture, m.Moisture.Values)
	copy(rainfall, m.Rainfall.Values)

	// Get the elevation values.
	elevs := m.Elevation.GetValues()
	outRegs := make([]int, 0, 8)
	for i := 0; i < interpolationSteps; i++ {
		regMoistureInterpol := make([]float64, m.SphereMesh.NumRegions)
		regRainfallInterpol := make([]float64, m.SphereMesh.NumRegions)
		for r := range regMoistureInterpol {
			rMoist := moisture[r]
			rRain := rainfall[r]
			var count int
			for _, nbReg := range m.SphereMesh.R_circulate_r(outRegs, r) {
				// Gravity! Water moves downwards.
				// This is not super-accurate since you'd have to take
				// in account how steep the slope is etc.
				if elevs[r] >= elevs[nbReg] {
					continue
				}
				rMoist += moisture[nbReg]
				rRain += rainfall[nbReg]
				count++
			}
			regMoistureInterpol[r] = rMoist / float64(count+1)
			regRainfallInterpol[r] = rRain / float64(count+1)
		}
		moisture = regMoistureInterpol
		rainfall = regRainfallInterpol
	}

	// Assign the new values.
	m.Moisture.SetValues(moisture)
	m.Rainfall.SetValues(rainfall)
}
