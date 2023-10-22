package genworldvoronoi

import (
	"math"
	"math/rand"
	"sort"
)

// disaster represents a disaster that can occur in a region.
type disaster struct {
	Name           string  // Name of the disaster
	Probability    float64 // Probability of the disaster occurring
	PopulationLoss float64 // Percentage of the population that will be lost
}

// adjustProbability adjusts the probability of a disaster based on the
// probability of the region.
func (d disaster) adjustProbability(p float64) disaster {
	return disaster{
		Name:           d.Name,
		Probability:    d.Probability * p,
		PopulationLoss: d.PopulationLoss,
	}
}

// TODO: Move non-geographical disasters to a separate file or move this
// code to a more genericly named file.
var (
	disNone       = disaster{"None", 0, 0}
	disStorm      = disaster{"Storm", 0.99, 0.01}
	disFire       = disaster{"Fire", 0.98, 0.02}
	disRockslide  = disaster{"Rockslide", 0.97, 0.03}
	disCaveIn     = disaster{"Cave In", 0.95, 0.05}
	disWildfire   = disaster{"Wildfire", 0.93, 0.07}
	disDrought    = disaster{"Drought", 0.9, 0.1}
	disFamine     = disaster{"Famine", 0.85, 0.15}
	disDisease    = disaster{"Disease", 0.75, 0.25}
	disEarthquake = disaster{"Earthquake", 0.7, 0.3}
	disFlood      = disaster{"Flood", 0.65, 0.35}
	disVolcano    = disaster{"Volcanic Eruption", 0.4, 0.6}
	disPlague     = disaster{"Plague", 0.2, 0.8}
	disSandstorm  = disaster{"Sandstorm", 0.9, 0.1}
)

func randDisaster(dis []disaster) disaster {
	// Pick a random disaster given their respective probabilities.
	// TODO: Replace this with region specific disasters and disasters
	// that are likely based on local industry, population density, etc.

	// Sort the disasters by their ascending probability.
	sort.Slice(dis, func(i, j int) bool {
		return dis[i].Probability < dis[j].Probability
	})
	r := rand.Float64() * sumDisasterProbabilities(dis)
	for _, d := range dis {
		r -= d.Probability
		if r <= 0 {
			return d
		}
	}
	return disNone
}

func sumDisasterProbabilities(dis []disaster) float64 {
	// Get the sum of the probabilities of the disasters.
	var sum float64
	for _, d := range dis {
		sum += d.Probability
	}
	return sum
}

// GeoDisasterChance is the chance of a disaster in a region based on the
// geographical properties of the region.
type GeoDisasterChance struct {
	Earthquake float64 // 0.0-1.0
	Flood      float64 // 0.0-1.0
	Volcano    float64 // 0.0-1.0
	RockSlide  float64 // 0.0-1.0
}

func (c GeoDisasterChance) getDisasters() []disaster {
	// Get the disasters that occur in this region.
	var ds []disaster
	if c.Earthquake > 0.0001 {
		ds = append(ds, disEarthquake.adjustProbability(c.Earthquake))
	}
	if c.Flood > 0.0001 {
		ds = append(ds, disFlood.adjustProbability(c.Flood))
	}
	if c.Volcano > 0.0001 {
		ds = append(ds, disVolcano.adjustProbability(c.Volcano))
	}
	if c.RockSlide > 0.0001 {
		ds = append(ds, disRockslide.adjustProbability(c.RockSlide))
	}
	return ds
}

func (m *Geo) getGeoDisasterFunc() func(int) GeoDisasterChance {
	// Get the chance function for each disaster.
	earthquakeChance := m.getEarthquakeChance()
	floodChance := m.getFloodChance()
	volcanoEruptionChance := m.getVolcanoEruptionChance()
	rockSlideAvalancheChance := m.getRockSlideAvalancheChance()
	return func(reg int) GeoDisasterChance {
		// Get the chance of a disaster in this region.
		// NOTE: This is a very simple way of combining the chances.
		// TODO: Add chance of tropical storms, wildfires, etc.
		return GeoDisasterChance{
			Earthquake: earthquakeChance[reg],
			Flood:      floodChance[reg],
			Volcano:    volcanoEruptionChance[reg],
			RockSlide:  rockSlideAvalancheChance[reg],
		}
	}
}

func (m *Geo) getEarthquakeChance() []float64 {
	// Get distance field from fault lines using the plate compression.
	compression := m.propagateCompression(m.RegionCompression)

	// Now get the chance of earthquake for each region.
	earthquakeChance := make([]float64, m.SphereMesh.NumRegions)
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		earthquakeChance[r] = math.Abs(compression[r])
	}
	return earthquakeChance
}

func (m *Geo) getFloodChance() []float64 {
	// Now get the chance of flood for each region.
	floodChance := make([]float64, m.SphereMesh.NumRegions)
	_, maxFlux := minMax(m.Flux)
	steepness := m.GetSteepness()
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		// We use the flux of water and the steepness in the region
		// to determine the chance of a flood.
		// NOTE: This should also apply to lakes.
		floodChance[r] = (1 - steepness[r]) * m.Flux[r] / maxFlux
	}

	// Normalize the flood chance.
	_, maxFloodChance := minMax(floodChance)
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		floodChance[r] /= maxFloodChance
	}
	return floodChance
}

func (m *Geo) getVolcanoEruptionChance() []float64 {
	return m.getDownhillDisaster(m.RegionIsVolcano, 0.05)
}

func (m *Geo) getRockSlideAvalancheChance() []float64 {
	return m.getDownhillDisaster(m.RegionIsMountain, 0.1)
}

func (m *Geo) getDownhillDisaster(origins map[int]bool, steepnessLimit float64) []float64 {
	steepness := m.GetSteepness()
	downhill := m.GetDownhill(true)

	// Start at the origin regions and go downhill until the terrain is too
	// flat or we reach the ocean.
	chance := make([]float64, m.SphereMesh.NumRegions)
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		if !origins[r] {
			continue
		}

		// Go downhill until the steepness is too low or we reach the ocean.
		rdh := r
		danger := 1.0
		for rdh != -1 && steepness[rdh] > steepnessLimit && m.Elevation[rdh] > 0 {
			// Add the danger of the region to the chance of being affected by a
			// downhill disaster.
			chance[rdh] += danger
			danger *= steepness[rdh]
			rdh = downhill[rdh]
		}
	}
	return chance
}

func (m *Civ) getDisasterFunc() func(r int) []disaster {
	// TODO: Use the proper fitness functions above to determine the chance
	// of a disaster?

	// distRegion := math.Sqrt(4 * math.Pi / float64(m.mesh.NumRegions))
	// biomeFunc := m.getRegWhittakerModBiomeFunc()
	_, maxElev := minMax(m.Elevation)
	var volcanoes, mountains, faultlines []int
	isBigRiver := make(map[int]bool)
	isFireDanger := make(map[int]bool)
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		if m.RegionIsMountain[r] {
			mountains = append(mountains, r)
		}
		if m.RegionIsVolcano[r] {
			volcanoes = append(volcanoes, r)
		}
		if m.RegionCompression[r] > 0 {
			faultlines = append(faultlines, r)
		}
		if m.isRegBigRiver(r) {
			isBigRiver[r] = true
		}
		// Determine if there is danger of fire by checking if the region is
		// hot and relatively dry while still having vegetation.
		temp := m.getRegTemperature(r, maxElev)
		if temp > 25 && m.Moisture[r] < 0.2 {
			isFireDanger[r] = true
		}
	}

	// Get distance field from volanoes.
	distVolcanoes := m.assignDistanceField(mountains, make(map[int]bool))
	// Get distance field from fault lines.
	distMountains := m.assignDistanceField(mountains, make(map[int]bool))
	// Get distance field from mountains.
	distFaultlines := m.assignDistanceField(faultlines, make(map[int]bool))

	// TODO: Instead, introduce a new property of disasters that determines
	// how likely they are to occur. Then, we can take in account how far
	// away the disaster is from the region.
	return func(regionID int) []disaster {
		// Now get the disasters that might affect the region.
		var ds []disaster
		// Check if the region is close to a volcano.
		if distVolcanoes[regionID] < 3 {
			ds = append(ds, disVolcano)
		}
		// Check if the region is close to a mountain.
		if distMountains[regionID] < 3 {
			ds = append(ds, disRockslide)
		}
		// Check if the region is close to a fault line.
		if distFaultlines[regionID] < 3 {
			ds = append(ds, disEarthquake)
		}
		// Check if the region is at a big river.
		if isBigRiver[regionID] {
			ds = append(ds, disFlood)
		}
		// Check if we have a fire danger.
		if isFireDanger[regionID] {
			ds = append(ds, disWildfire)
		}
		return ds
	}
}
