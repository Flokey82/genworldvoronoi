package genworldvoronoi

import (
	"math"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/go_gens/gameconstants"
)

// getTempFalloffFromAltitude returns the temperature falloff at a given altitude in meters
// above sea level. (approx. 9.8 °C per 1000 m)
// NOTE: This is definitely not correct :)
// Source: https://www.quora.com/At-what-rate-does-temperature-drop-with-altitude
func getTempFalloffFromAltitude(height float64) float64 {
	if height < 0 {
		return 0.0
	}
	return gameconstants.EarthElevationTemperatureFalloff * height
}

const (
	minTemp          = genbiome.MinTemperatureC
	maxTemp          = genbiome.MaxTemperatureC
	rangeTemp        = maxTemp - minTemp
	maxPrecipitation = genbiome.MaxPrecipitationDM // 450cm
)

// getMeanAnnualTemp returns the temperature at a given latitude within the range of
// -15 °C to +30°C because that's the range in which the Whittaker biomes are defined.
// For this I assume that light hits the globe exactly from a 90° angle with respect
// to the planitary axis.
// See: https://www.scratchapixel.com/lessons/3d-basic-rendering/introduction-to-shading/shading-normals (facing ratio)
// See: http://www-das.uwyo.edu/~geerts/cwx/notes/chap16/geo_clim.html
// NOTE: -35 °C to +31 °C would be ideally the temp gradient (according to real-life data), but we don't have (yet) any biomes defined for this.
func getMeanAnnualTemp(lat float64) float64 {
	return (math.Sin(degToRad(90-math.Abs(lat))))*rangeTemp + minTemp
}

const maxAltitudeFactor = gameconstants.EarthMaxElevation // How tall is the tallest mountain with an elevation of 1.0?

// getRegTemperature returns the average yearly temperature of the given region at the surface.
func (m *Geo) getRegTemperature(r int, maxElev float64) float64 {
	// TODO: Fix maxElev caching!!!
	return getMeanAnnualTemp(m.LatLon[r][0]) - getTempFalloffFromAltitude(maxAltitudeFactor*m.Elevation[r]/maxElev)
}

// getRTemperature returns the average yearly temperature of the given triangle at the surface.
func (m *Geo) getTriTemperature(t int, maxElev float64) float64 {
	// TODO: Fix maxElev caching!!!
	return getMeanAnnualTemp(m.TriLatLon[t][0]) - getTempFalloffFromAltitude(maxAltitudeFactor*m.triElevation[t]/maxElev)
}

func (m *Geo) initRegionAirTemperature() {
	_, maxElev := minMax(m.Elevation)
	for r := 0; r < m.SphereMesh.numRegions; r++ {
		m.AirTemperature[r] = m.getRegTemperature(r, maxElev)
	}
}

func (m *Geo) assignRegionAirTemperature() {
	// TODO: Deduplicate this code with assignRegionWaterTemperature.
	newTemperature := make([]float64, m.SphereMesh.numRegions)
	baseTemperature := make([]float64, m.SphereMesh.numRegions)

	outregs := make([]int, 0, 8)
	for r := 0; r < m.SphereMesh.numRegions; r++ {
		// base
		// lat := m.LatLon[r][0]
		// absLat := math.Abs(lat)
		// lon :=  m.LatLon[r][1]
		// absLat := math.Abs(lat - m.getSunLattitude())
		startTemp := m.AirTemperature[r]
		newTemperature[r] = startTemp
		baseTemperature[r] = startTemp

		// diffusion
		neighbors := m.SphereMesh.r_circulate_r(outregs, r)
		neighborAverage := newTemperature[r]
		neighborCount := 1
		for i := 0; i < len(neighbors); i++ {
			nr := neighbors[i]
			neighborAverage += newTemperature[nr]
			neighborCount++
		}
		neighborAverage /= float64(neighborCount)

		newTemperature[r] = 0.75*newTemperature[r] + 0.25*neighborAverage

		// newTemperature[r] = clamp(0, 1, newTemperature[r] - map.r_clouds[r]/2)
	}

	const (
		transferIn  = 0.001
		transferOut = 1.0 - transferIn
		numSteps    = 5
	)

	// Initialize the movedTemp slice.
	movedTemp := make([]float64, m.SphereMesh.numRegions)
	movedCount := make([]int, m.SphereMesh.numRegions)
	for step := 0; step < numSteps; step++ {
		for r, nt := range newTemperature {
			// add in the "pulled temp"
			movedTemp[r] += nt
			movedCount[r]++

			pr := m.getPreviousNeighbor(outregs, r, m.RegionToWindVecLocal[r])
			movedTemp[r] += m.AirTemperature[pr]
			movedCount[r]++

			// add in pushed temp
			nr := m.getClosestNeighbor(outregs, r, m.RegionToWindVecLocal[r])
			if nr == r {
				continue
			}
			// const heldHeat = newTemperature[r] - baseTemperature[r]
			// const potentialHeat = newTemperature[r] - baseTemperature[nr]

			// movedTemp[nr] = movedTemp[nr]? movedTemp[nr] : 0
			// movedTemp[r] -= map.r_currents[r]*heldHeat
			// movedTemp[nr] += map.r_currents[r]*potentialHeat
			// movedTemp[nr] = movedTemp[nr]? movedTemp[nr] : []float64{}
			movedTemp[nr] += transferOut*m.AirTemperature[r] + transferIn*nt
			movedCount[nr]++
		}

		for r, mc := range movedCount {
			if mc > 0 {
				newTemperature[r] = movedTemp[r] / float64(mc)
				// Reset movedTemp after every step for the next step.
				movedTemp[r] = 0
				movedCount[r] = 0
			}
			// if (movedTemp[r] !== undefined && movedTemp[r].length > 0) newTemperature[r] = movedTemp[r].reduce((acc, temp) => acc + temp, 0) / movedTemp[r].length
		}
		m.AirTemperature = newTemperature
	}
	m.AirTemperature = newTemperature
}

func (m *Geo) initRegionWaterTemperature() {
	_, maxElev := minMax(m.Elevation)
	for r := 0; r < m.SphereMesh.numRegions; r++ {
		if m.Elevation[r] <= 0 {
			m.OceanTemperature[r] = m.getRegTemperature(r, maxElev)
		}
	}
}

func (m *Geo) transportRegionWaterTemperature() {
	// TODO: Deduplicate this code with assignRegionAirTemperature.
	newTemperature := make([]float64, m.SphereMesh.numRegions)
	baseTemperature := make([]float64, m.SphereMesh.numRegions)
	for r := 0; r < m.SphereMesh.numRegions; r++ {
		if m.Elevation[r] > 0 {
			newTemperature[r] = 0.5
		}
	}

	outregs := make([]int, 0, 8)
	for r := 0; r < m.SphereMesh.numRegions; r++ {
		if m.Elevation[r] > 0 {
			continue
		}

		// base
		// lat := m.LatLon[r][0]
		// absLat := math.Abs(lat)
		// lon :=  m.LatLon[r][1]
		// absLat := math.Abs(lat - m.getSunLattitude())
		startTemp := m.OceanTemperature[r]
		newTemperature[r] = startTemp
		baseTemperature[r] = startTemp

		// diffusion
		neighbors := m.SphereMesh.r_circulate_r(outregs, r)
		neighborAverage := newTemperature[r]
		neighborCount := 1
		for i := 0; i < len(neighbors); i++ {
			nr := neighbors[i]
			if m.Elevation[nr] <= 0 {
				neighborAverage += newTemperature[nr]
				neighborCount++
			}
		}
		neighborAverage /= float64(neighborCount)

		newTemperature[r] = 0.75*newTemperature[r] + 0.25*neighborAverage

		// newTemperature[r] = clamp(0, 1, newTemperature[r] - map.r_clouds[r]/2)
	}

	const (
		transferIn  = 0.001
		transferOut = 1.0 - transferIn
		numSteps    = 5
	)

	// Initialize the movedTemp slice.
	movedTemp := make([]float64, m.SphereMesh.numRegions)
	movedCount := make([]int, m.SphereMesh.numRegions)
	for step := 0; step < numSteps; step++ {
		for r, nt := range newTemperature {
			// add in the "pulled temp"
			movedTemp[r] += nt
			movedCount[r]++

			pr := m.getPreviousNeighbor(outregs, r, m.RegionToOceanVec[r])
			if m.Elevation[pr] <= 0 {
				movedTemp[r] += m.OceanTemperature[pr]
				movedCount[r]++
			}

			// add in pushed temp
			nr := m.getClosestNeighbor(outregs, r, m.RegionToOceanVec[r])
			if nr == r || m.Elevation[nr] > 0 {
				continue
			}
			// const heldHeat = newTemperature[r] - baseTemperature[r]
			// const potentialHeat = newTemperature[r] - baseTemperature[nr]

			// movedTemp[nr] = movedTemp[nr]? movedTemp[nr] : 0
			// movedTemp[r] -= map.r_currents[r]*heldHeat
			// movedTemp[nr] += map.r_currents[r]*potentialHeat
			// movedTemp[nr] = movedTemp[nr]? movedTemp[nr] : []float64{}
			movedTemp[nr] += transferOut*m.OceanTemperature[r] + transferIn*nt
			movedCount[nr]++
		}

		for r, mc := range movedCount {
			if mc > 0 {
				newTemperature[r] = movedTemp[r] / float64(mc)
				// Reset movedTemp after every step for the next step.
				movedTemp[r] = 0
				movedCount[r] = 0
			}
			// if (movedTemp[r] !== undefined && movedTemp[r].length > 0) newTemperature[r] = movedTemp[r].reduce((acc, temp) => acc + temp, 0) / movedTemp[r].length
		}
		m.OceanTemperature = newTemperature
	}
	m.OceanTemperature = newTemperature
}
