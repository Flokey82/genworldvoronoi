package genworldvoronoi

import (
	"container/list"
	"image/color"

	"github.com/Flokey82/genbiome"
)

// getAzgaarRegionBiome returns the biome for a given region as per Azgaar's map generator.
func (m *Geo) getAzgaarRegionBiome(r int, elev, maxElev float64) int {
	return genbiome.GetAzgaarBiome(int(20.0*m.Moisture[r]), int(m.getRegTemperature(r, maxElev)), int(elev*100))
}

// getRegWhittakerModBiomeFunc returns a function that returns the Whittaker biome
// for a given region.
func (m *Geo) getRegWhittakerModBiomeFunc() func(r int) int {
	_, maxElev := minMax(m.Elevation)
	_, maxMois := minMax(m.Moisture)
	return func(r int) int {
		valElev := m.Elevation[r] / maxElev
		valMois := m.Moisture[r] / maxMois
		regLat := m.LatLon[r][0]
		return getWhittakerModBiome(regLat, valElev, valMois)
	}
}

func getWhittakerModBiome(latitude, elevation, moisture float64) int {
	return genbiome.GetWhittakerModBiome(int(getMeanAnnualTemp(latitude)-getTempFalloffFromAltitude(maxAltitudeFactor*elevation)), int(moisture*maxPrecipitation))
}

func getWhittakerModBiomeColor(latitude, elevation, moisture, intensity float64) color.NRGBA {
	return genbiome.GetWhittakerModBiomeColor(int(getMeanAnnualTemp(latitude)-getTempFalloffFromAltitude(maxAltitudeFactor*elevation)), int(moisture*maxPrecipitation), intensity)
}

func (m *Geo) assignBiomeRegions() {
	// Identify connected regions with the same biome.
	m.BiomeRegions = m.identifyBiomeRegions()

	regSize := make(map[int]int)
	for _, lm := range m.BiomeRegions {
		if lm >= 0 {
			regSize[lm]++ // Only count regions that are set to a valid ID.
		}
	}
	m.BiomeRegionSize = regSize
}

// identifyBiomeRegions identifies connected regions with the same biome.
func (m *Geo) identifyBiomeRegions() []int {
	// We use a flood fill algorithm to identify regions with the same biome
	biomeToRegs := initRegionSlice(m.mesh.numRegions)
	// Set all ocean regions to -2.
	for r := range biomeToRegs {
		if m.Elevation[r] <= 0.0 {
			biomeToRegs[r] = -2
		}
	}
	biomeFunc := m.getRegWhittakerModBiomeFunc()

	// Use a queue to implement the flood fill algorithm.
	outRegs := make([]int, 0, 6)
	floodFill := func(id int) {
		queue := list.New()
		// Get the biome that is represented by the region ID.
		biome := biomeFunc(id)
		queue.PushBack(id)

		// The region ID will serve as a representative of the biome.
		biomeToRegs[id] = id

		// Now flood fill all regions with the same biome.
		for queue.Len() > 0 {
			e := queue.Front()
			if e == nil {
				break
			}
			queue.Remove(e)
			nbID := e.Value.(int)

			for _, n := range m.mesh.r_circulate_r(outRegs, nbID) {
				if biomeToRegs[n] == -1 && biomeFunc(n) == biome {
					queue.PushBack(n)

					// The region ID will serve as a representative of the biome.
					biomeToRegs[n] = id
				}
			}
		}
	}

	// Loop through all regions and pick the first region that has not been
	// assigned a biome yet. Then flood fill all regions with the same biome.
	for id := 0; id < m.mesh.numRegions; id++ {
		if biomeToRegs[id] == -1 {
			floodFill(id)
		}
	}
	return biomeToRegs
}
