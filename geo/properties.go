package geo

import (
	"math"

	"github.com/Flokey82/genworldvoronoi/various"
)

type RegProperty struct {
	ID                  int
	Elevation           float64 // 0.0-1.0
	Steepness           float64 // 0.0-1.0
	Biome               int     // biome of the region
	DistanceToCoast     float64 // graph distance to the nearest coast
	DistanceToMountain  float64 // graph distance to the nearest mountain
	DistanceToRiver     float64 // graph distance to the nearest river
	DistanceToVolcano   float64 // graph distance to the nearest volcano
	DistanceToFaultline float64 // graph distance to the nearest faultline
	Temperature         float64 // in °C
	Rainfall            float64 // in dm
	Danger              GeoDisasterChance
	HasWaterfall        bool // true if the region has a waterfall
	IsValley            bool // true if the region is a valley
	OnIsland            bool // true if the region is on an island
}

// GetRegPropertyFunc returns a function that returns the properties of a region.
// NOTE: This is probably a very greedy function.
func (m *Geo) GetRegPropertyFunc() func(int) RegProperty {
	// TODO: Add chance of tropical storms, wildfires, etc.
	disasterFunc := m.GetGeoDisasterFunc()
	steepness := m.GetSteepness()
	inlandValleyFunc := m.GetFitnessInlandValleys()
	biomeFunc := m.GetRegWhittakerModBiomeFunc()
	_, maxElev := minMax(m.Elevation)
	var oceanRegs, volcanoRegs, riverRegs, faultlineRegs []int
	stopOcean := make(map[int]bool)
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		if m.Elevation[r] <= 0 {
			oceanRegs = append(oceanRegs, r)
			stopOcean[r] = true
		}
		if m.RegionIsVolcano[r] {
			volcanoRegs = append(volcanoRegs, r)
		}
		if m.IsRegBigRiver(r) {
			riverRegs = append(riverRegs, r)
		}
		if m.RegionCompression[r] != 0 {
			faultlineRegs = append(faultlineRegs, r)
		}
	}
	distOcean := m.AssignDistanceField(oceanRegs, m.RegionIsMountain)
	distMountain := m.AssignDistanceField(m.Mountain_r, stopOcean)
	distVolcano := m.AssignDistanceField(volcanoRegs, stopOcean)
	distRiver := m.AssignDistanceField(riverRegs, stopOcean)
	distFaultline := m.AssignDistanceField(faultlineRegs, stopOcean)
	return func(id int) RegProperty {
		// Make sure that we do not have more than 2 neighbours that has a lower elevation.
		// ... because a valley should be surrounded by mountains.
		isValley := inlandValleyFunc(id) > 0.8
		if isValley {
			var count int
			for _, n := range m.GetRegNeighbors(id) {
				if m.Elevation[n] > m.Elevation[id] {
					continue
				}
				count++
				if count > 2 {
					isValley = false
					break
				}
			}
		}

		return RegProperty{
			ID:                  id,
			Elevation:           m.Elevation[id],
			Steepness:           steepness[id],
			Biome:               biomeFunc(id),
			DistanceToCoast:     distOcean[id],
			DistanceToMountain:  distMountain[id],
			DistanceToRiver:     distRiver[id],
			DistanceToVolcano:   distVolcano[id],
			DistanceToFaultline: distFaultline[id],
			Temperature:         m.GetRegTemperature(id, maxElev),
			Rainfall:            m.Rainfall[id],
			Danger:              disasterFunc(id),
			HasWaterfall:        m.RegionIsWaterfall[id],
			IsValley:            isValley,
			OnIsland:            m.LandmassSize[m.Landmasses[id]] < 15, // TODO: This should use actual geographical area.
		}
	}
}

// Landmark feature types.
const (
	FeatureTypeOcean     = "ocean"
	FeatureTypeSea       = "sea"
	FeatureTypeLake      = "lake"
	FeatureTypeGulf      = "gulf"
	FeatureTypeIsle      = "isle"
	FeatureTypeContinent = "continent"
)

// GetRegionFeatureTypeFunc returns a function that returns the feature type of
// a given region.
func (m *Geo) GetRegionFeatureTypeFunc() func(int) string {
	return func(i int) string {
		if i < 0 {
			return ""
		}
		if waterbodyID := m.Waterbodies[i]; waterbodyID >= 0 {
			switch wbSize := m.WaterbodySize[waterbodyID]; {
			case wbSize > m.SphereMesh.NumRegions/25:
				return FeatureTypeOcean
			case wbSize > m.SphereMesh.NumRegions/100:
				return FeatureTypeSea
			case wbSize > m.SphereMesh.NumRegions/500:
				return FeatureTypeGulf
			default:
				return FeatureTypeLake
			}
		}
		if landmassID := m.Landmasses[i]; landmassID >= 0 {
			if m.LandmassSize[landmassID] < m.SphereMesh.NumRegions/100 {
				return FeatureTypeIsle
			}
			return FeatureTypeContinent
		}
		return ""
	}
}

// GetRegHaven returns the closest neighbor region that is a water cell, which
// can be used as a haven, and returns the number of water neighbors, indicating
// the harbor size.
//
// If no haven is found, -1 is returned.
func (m *Geo) GetRegHaven(reg int) (int, int) {
	// get all neighbors that are below or at sea level.
	water := make([]int, 0, 8)
	for _, nb := range m.GetRegNeighbors(reg) {
		if m.Elevation[nb] <= 0.0 {
			water = append(water, nb)
		}
	}

	// No water neighbors, return -1.
	if len(water) == 0 {
		return -1, 0
	}

	// Get distances of i to each water neighbor.
	// get the closest water neighbor.
	rLatLon := m.LatLon[reg]
	closest := -1
	minDist := math.Inf(1)
	for _, nb := range water {
		nbLatLon := m.LatLon[nb]
		dist := various.Haversine(rLatLon[0], rLatLon[1], nbLatLon[0], nbLatLon[1])
		if dist < minDist {
			minDist = dist
			closest = nb
		}
	}
	// store the closest water neighbor as the haven.
	// store the number of water neighbors as the harbor.
	return closest, len(water)
}

// CellType is the type of a cell indicating the distance to the shore.
const (
	CellTypeDeepWaters   = -2
	CellTypeCoastalWater = -1
	CellTypeCoastalLand  = 1
	CellTypeInland       = 2
)

// GetRegCellTypes maps the region to its cell type.
//
// NOTE: Currently this depends on the region graph, which will break
// things once we increas or decrease the number of regions on the map as
// the distance between regions will change with the region density.
//
// Value meanings:
//
// -2: deep ocean or large lake
// -1: region is a water cell next to a land cell (lake shore/coastal water)
// +1: region is a land cell next to a water cell (lake shore/coastal land)
// +2: region is a land cell next to a coastal land cell
// >2: region is inland
func (m *Geo) GetRegCellTypes() []int {
	var oceanRegs, landRegs []int
	stop_land := make(map[int]bool)
	stop_ocean := make(map[int]bool)
	for r, elev := range m.Elevation {
		if elev <= 0.0 {
			oceanRegs = append(oceanRegs, r)
			stop_ocean[r] = true
		} else {
			landRegs = append(landRegs, r)
			stop_land[r] = true
		}
	}
	regDistanceOcean := m.AssignDistanceField(oceanRegs, stop_land)
	regDistanceLand := m.AssignDistanceField(landRegs, stop_ocean)

	cellType := make([]int, m.SphereMesh.NumRegions)
	for i := range cellType {
		// Is it water?
		if m.Elevation[i] <= 0.0 {
			// Figure out if it has a land neighbor.
			// If so, it is -1 (water near coast)
			if regDistanceLand[i] <= 1 {
				cellType[i] = CellTypeCoastalWater
			} else {
				// If not, it is -2 (water far from coast)
				cellType[i] = CellTypeDeepWaters
			}
		} else if regDistanceOcean[i] <= 1 { // Figure out if it has a water neighbor.
			// If so, it is 1 (land near coast)
			cellType[i] = CellTypeCoastalLand
		} else {
			// If not, it is >=2 (land far from coast)
			cellType[i] = int(regDistanceOcean[i])
		}
	}
	return cellType
}

func (m *BaseObject) IsRegBelowOrAtSeaLevelOrPool(r int) bool {
	return m.Elevation[r] <= 0 || m.Waterpool[r] > 0
}

func (m *BaseObject) IsRegLakeOrWaterBody(r int) bool {
	return m.IsRegWaterBody(r) || m.IsRegLake(r)
}

func (m *BaseObject) IsRegWaterBody(r int) bool {
	return m.Waterbodies[r] >= 0
}

func (m *BaseObject) IsRegLake(r int) bool {
	return m.Drainage[r] >= 0 || m.Waterpool[r] > 0
}

func (m *BaseObject) IsRegRiver(r int) bool {
	return m.Flux[r] > m.Rainfall[r]
}

func (m *BaseObject) IsRegBigRiver(r int) bool {
	return m.Flux[r] > m.Rainfall[r]*2
}
