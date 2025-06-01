package civ

import (
	"math/rand"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/genworldvoronoi/various"
)

type CultureType int

// Culture types.
const (
	CultureTypeWildland CultureType = iota
	CultureTypeGeneric
	CultureTypeRiver
	CultureTypeLake
	CultureTypeNaval
	CultureTypeNomadic
	CultureTypeHunting
	CultureTypeHighland
	CultureTypeCount // Number of culture types.
)

// String returns the string representation of a given culture type.
func (c CultureType) String() string {
	switch c {
	case CultureTypeWildland:
		return "Wildland"
	case CultureTypeGeneric:
		return "Generic"
	case CultureTypeRiver:
		return "River"
	case CultureTypeLake:
		return "Lake"
	case CultureTypeNaval:
		return "Naval"
	case CultureTypeNomadic:
		return "Nomadic"
	case CultureTypeHunting:
		return "Hunting"
	case CultureTypeHighland:
		return "Highland"
	default:
		return "Unknown"
	}
}

const maxExpansionism = 1.5 * 1.5

// Expansionism returns the expansionism of a given culture type.
func (t CultureType) Expansionism() float64 {
	// TODO: This is a random attractiveness value of the capital.
	// https://azgaar.wordpress.com/2017/11/21/settlements/
	// I introduced two custom parameters — disbalance and power.
	// Each capital has unique attractiveness power, which is randomly
	// assigned to it based on a disbalance value. Disbalance is the same
	// for all capitals, it only controls the randomness of power
	// definition. Calculating a distance to the closest capital we
	// multiply this value by capital’s power. If capital located not on
	// the same island, we double the distance as it should not be easy
	// for city to get an overseas possessions. As all capitals have
	// different “powers”, the regions vary in area. For some reasons
	// user may want regions having almost the same area, so the disbalance
	// value could be changed.
	powerInputValue := 1.0
	base := 1.0 // Generic
	switch t {
	case CultureTypeLake:
		base = 0.8
	case CultureTypeNaval:
		base = 1.5
	case CultureTypeRiver:
		base = 0.9
	case CultureTypeNomadic:
		base = 1.5
	case CultureTypeHunting:
		base = 0.7
	case CultureTypeHighland:
		base = 1.2
	}
	return various.RoundToDecimals(((rand.Float64()*powerInputValue)/2+1)*base, 1)
}

const maxMartialism = 1.5 * 1.5

// Martialism returns the martialism of a given culture type.
func (t CultureType) Martialism() float64 {
	powerInputValue := 1.0
	base := 1.0 // Generic
	switch t {
	case CultureTypeLake:
		base = 0.8
	case CultureTypeNaval:
		base = 1.5
	case CultureTypeRiver:
		base = 0.9
	case CultureTypeNomadic:
		base = 1.4
	case CultureTypeHunting:
		base = 1.4
	case CultureTypeHighland:
		base = 1.1
	}
	return various.RoundToDecimals(((rand.Float64()*powerInputValue)/2+1)*base, 1)
}

const maxSpirituality = 1.2 * 1.5

// Spirituality returns the spirituality of a given culture type.
// TODO: Replace this with a more meaningful value.
func (t CultureType) Spirituality() float64 {
	powerInputValue := 1.0
	base := 1.0 // Generic
	switch t {
	case CultureTypeLake:
		base = 1.2
	case CultureTypeNaval:
		base = 1.2
	case CultureTypeRiver:
		base = 1.2
	case CultureTypeNomadic:
		base = 1.2
	case CultureTypeHunting:
		base = 1.2
	case CultureTypeHighland:
		base = 1.2
	}
	return various.RoundToDecimals(((rand.Float64()*powerInputValue)/2+1)*base, 1)
}

const maxOpenness = 1.5 * 1.5

// Openness returns the openness of a given culture type.
// This value reflects how open a culture is to new ideas, technologies, and
// other cultures.
func (t CultureType) Openness() float64 {
	powerInputValue := 1.0
	base := 1.0 // Generic
	switch t {
	case CultureTypeLake:
		base = 1.0
	case CultureTypeNaval:
		base = 1.5
	case CultureTypeRiver:
		base = 1.2
	case CultureTypeNomadic:
		base = 1.3
	case CultureTypeHunting:
		base = 0.9
	case CultureTypeHighland:
		base = 0.6
	}
	return various.RoundToDecimals(((rand.Float64()*powerInputValue)/2+1)*base, 1)
}

// CellTypeCost returns the cost of crossing / navigating a given cell type for a given culture.
func (t CultureType) CellTypeCost(cellType int) float64 {
	// Land near coast / coastline / coastal land strip / "beach"?.
	if cellType == geo.CellTypeCoastalLand {
		if t == CultureTypeNaval || t == CultureTypeLake {
			// Naval cultures or lake cultures have an easier time navigating
			// coastal areas or shores of lakes.
			return 1.0
		}
		if t == CultureTypeNomadic {
			// Nomadic cultures have a harder time navigating coastal areas or
			// shores of lakes.
			return 1.6
		}
		// All other cultures have a small penalty for coastal areas.
		return 1.2
	}

	// Land slightly further inland.
	if cellType == geo.CellTypeInland {
		if t == CultureTypeNaval || t == CultureTypeNomadic {
			// Small penalty for land with distance 2 to ocean for navals and nomads.
			return 1.3
		}
		// All other cultures do not have appreciable penalty.
		return 1.0
	}

	// Not water near coast (deep ocean/coastal land).
	if cellType != geo.CellTypeCoastalWater {
		if t == CultureTypeNaval || t == CultureTypeLake {
			// Penalty for mainland for naval and lake cultures
			return 2.0
		}
	}
	return 1.0
}

// BiomeCost returns the cost for traversion / expanding into a given biome.
func (t CultureType) BiomeCost(biome int) float64 {
	if t == CultureTypeHunting {
		// Non-native biome penalty for hunters.
		return 5.0
	}
	if t == CultureTypeNomadic && (biome == genbiome.AzgaarBiomeTropicalSeasonalForest ||
		biome == genbiome.AzgaarBiomeTemperateDeciduousForest ||
		biome == genbiome.AzgaarBiomeTropicalRainforest ||
		biome == genbiome.AzgaarBiomeTemperateRainforest ||
		biome == genbiome.AzgaarBiomeTaiga) {
		// Forest biome penalty for nomads.
		return 10.0
	}
	// General non-native biome penalty.
	return 2.0
}

// getRegionCutureTypeFunc returns a function that returns the culture type suitable for a given region.
func GetRegionCultureTypeFunc(m *geo.Geo) func(int) CultureType {
	cellType := m.GetRegCellTypes()
	getType := m.GetRegionFeatureTypeFunc()
	//biomeFunc := m.GetRegWhittakerModBiomeFunc()

	// Get the elevation values.
	elevs := m.Elevation.GetValues()
	maxElev := m.Elevation.Max

	//log.Println("TODO: Map whittaker to azgaar biomes")

	// Return culture type based on culture center region.
	return func(r int) CultureType {
		eleVal := elevs[r] / maxElev
		gotBiome := m.GetAzgaarRegionBiome(r, eleVal)

		// Desert and grassland means a nomadic culture.
		// BUT: Grassland is extremely well suited for farming... Which is not nomadic.
		if eleVal < 0.7 && (gotBiome == genbiome.AzgaarBiomeHotDesert ||
			gotBiome == genbiome.AzgaarBiomeColdDesert ||
			gotBiome == genbiome.AzgaarBiomeGrassland) {
			return CultureTypeNomadic // high penalty in forest biomes and near coastline
		}

		// Montane cultures in high elevations and hills
		// that aren't deserts or grassland.
		if eleVal > 0.3 {
			return CultureTypeHighland // no penalty for hills and moutains, high for other elevations
		}

		// Get the region (if any) that represents the haven for this region.
		// A haven is the closest neighbor that is a water body.
		// NOTE: harborSize indicates the number of neighbors that are water.
		rHaven, harborSize := m.GetRegHaven(r)
		havenType := getType(rHaven) // Get the haven type of the region.
		regionType := getType(r)     // Get the region type of the region.

		// Ensure only larger lakes will result in the 'lake' culture type.
		if havenType == geo.FeatureTypeLake && m.WaterbodySize[rHaven] > 5 {
			return CultureTypeLake // low water cross penalty and high for growth not along coastline
		}

		// If we have a harbor (more than 1 water neighbor), or are on an island,
		// we are potentially a naval culture.
		if (harborSize > 0 && P(0.1) && havenType != geo.FeatureTypeLake) ||
			(harborSize == 1 && P(0.6)) ||
			(regionType == geo.FeatureTypeIsle && P(0.4)) {
			return CultureTypeNaval // low water cross penalty and high for non-along-coastline growth
		}

		// If we are on a big river (flux > 2*rainfall), we are a river culture.
		if m.IsRegBigRiver(r) {
			return CultureTypeRiver // no River cross penalty, penalty for non-River growth
		}

		// If we are inland (cellType > 2) and in one of the listed biomes,
		// we are a hunting culture.
		if cellType[r] > 2 && (gotBiome == genbiome.AzgaarBiomeSavanna ||
			gotBiome == genbiome.AzgaarBiomeTropicalRainforest ||
			gotBiome == genbiome.AzgaarBiomeTemperateRainforest ||
			gotBiome == genbiome.AzgaarBiomeWetland ||
			gotBiome == genbiome.AzgaarBiomeTaiga ||
			gotBiome == genbiome.AzgaarBiomeTundra || // Tundra is also nomadic?
			gotBiome == genbiome.AzgaarBiomeGlacier) {
			return CultureTypeHunting // high penalty in non-native biomes
		}

		// TODO:
		// - Wildlands?
		// - What culture would have originated in seasonal forests?
		return CultureTypeGeneric
	}
}

/*

func (m *Map) getBiomeCost(cultureCenter int, biome int, cType CultureType) int {
	_, maxElev := minMax(m.r_elevation)
	eleVal := m.r_elevation[cultureCenter] / maxElev
	gotBiome := m.getRBiomeTEMP(cultureCenter, eleVal, maxElev)
	if gotBiome == biome {
		return 10 // tiny penalty for native biome
	}
	if cType == CultureTypeHunting {
		return genbiome.AzgaarBiomeMovementCost[biome] * 5 // non-native biome penalty for hunters
	}
	if cType == CultureTypeNomadic && biome > 4 && biome < 10 {
		return genbiome.AzgaarBiomeMovementCost[biome] * 10 // forest biome penalty for nomads
	}
	return genbiome.AzgaarBiomeMovementCost[biome] * 2 // general non-native biome penalty
}

func (m *Map) getHeightCost(i int, h float64, cType CultureType) int {
	f = pack.features[cells.f[i]]
	a = cells.area[i]
	if cType == CultureTypeLake && f.Type == "lake" {
		return 10 // no lake crossing penalty for Lake cultures
	}
	if cType == CultureTypeNaval && h < 20 {
		return a * 2 // low sea/lake crossing penalty for Naval cultures
	}
	if cType == CultureTypeNomadic && h < 20 {
		return a * 50 // giant sea/lake crossing penalty for Nomads
	}
	if h < 20 {
		return a * 6 // general sea/lake crossing penalty
	}
	if cType == CultureTypeHighland && h < 44 {
		return 3000 // giant penalty for highlanders on lowlands
	}
	if cType == CultureTypeHighland && h < 62 {
		return 200 // giant penalty for highlanders on lowhills
	}
	if cType == CultureTypeHighland {
		return 0 // no penalty for highlanders on highlands
	}
	if h >= 67 {
		return 200 // general mountains crossing penalty
	}
	if h >= 44 {
		return 30 // general hills crossing penalty
	}
	return 0
}

	if r_waterbodies[i] >= 0 {
		if r_waterbody_size[r_waterbodies[i]] > m.mesh.numRegions/25 {
			return "ocean"
		}
		if r_waterbody_size[r_waterbodies[i]] > m.mesh.numRegions/100 {
			return "sea"
		}
		return "gulf"
	}
	  function defineOceanGroup(number) {
	    if (number > grid.cells.i.length / 25) return "ocean";
	    if (number > grid.cells.i.length / 100) return "sea";
	    return "gulf";
	  }

	  function defineIslandGroup(cell, number) {
	    if (cell && features[cells.f[cell - 1]].type === "lake") return "lake_island";
	    if (number > grid.cells.i.length / 10) return "continent";
	    if (number > grid.cells.i.length / 1000) return "island";
	    return "isle";
	  }*/
/*


  // expand cultures across the map (Dijkstra-like algorithm)
  const expand = function () {
    TIME && console.time("expandCultures");
    cells = pack.cells;

    const queue = new PriorityQueue({comparator: (a, b) => a.p - b.p});
    pack.cultures.forEach(function (c) {
      if (!c.i || c.removed) return;
      queue.queue({e: c.center, p: 0, c: c.i});
    });

    const neutral = (cells.i.length / 5000) * 3000 * neutralInput.value; // limit cost for culture growth
    const cost = [];
    while (queue.length) {
      const next = queue.dequeue(),
        n = next.e,
        p = next.p,
        c = next.c;
      const type = pack.cultures[c].type;
      cells.c[n].forEach(function (e) {
        const biome = cells.biome[e];
        const biomeCost = getBiomeCost(c, biome, type);
        const biomeChangeCost = biome === cells.biome[n] ? 0 : 20; // penalty on biome change
        const heightCost = getHeightCost(e, cells.h[e], type);
        const riverCost = getRiverCost(cells.r[e], e, type);
        const typeCost = getTypeCost(cells.t[e], type);
        const totalCost =
          p + (biomeCost + biomeChangeCost + heightCost + riverCost + typeCost) / pack.cultures[c].expansionism;

        if (totalCost > neutral) return;

        if (!cost[e] || totalCost < cost[e]) {
          if (cells.s[e] > 0) cells.culture[e] = c; // assign culture to populated cell
          cost[e] = totalCost;
          queue.queue({e, p: totalCost, c});
        }
      });
    }

    TIME && console.timeEnd("expandCultures");
  };



  function getHeightCost(i, h, type) {
    const f = pack.features[cells.f[i]],
      a = cells.area[i];
    if (type === "Lake" && f.type === "lake") return 10; // no lake crossing penalty for Lake cultures
    if (type === "Naval" && h < 20) return a * 2; // low sea/lake crossing penalty for Naval cultures
    if (type === "Nomadic" && h < 20) return a * 50; // giant sea/lake crossing penalty for Nomads
    if (h < 20) return a * 6; // general sea/lake crossing penalty
    if (type === "Highland" && h < 44) return 3000; // giant penalty for highlanders on lowlands
    if (type === "Highland" && h < 62) return 200; // giant penalty for highlanders on lowhills
    if (type === "Highland") return 0; // no penalty for highlanders on highlands
    if (h >= 67) return 200; // general mountains crossing penalty
    if (h >= 44) return 30; // general hills crossing penalty
    return 0;
  }

  function getRiverCost(r, i, type) {
    if (type === "River") return r ? 0 : 100; // penalty for river cultures
    if (!r) return 0; // no penalty for others if there is no river
    return minmax(cells.fl[i] / 10, 20, 100); // river penalty from 20 to 100 based on flux
  }

  function getTypeCost(t, type) {
    if (t === 1) return type === "Naval" || type === "Lake" ? 0 : type === "Nomadic" ? 60 : 20; // penalty for coastline
    if (t === 2) return type === "Naval" || type === "Nomadic" ? 30 : 0; // low penalty for land level 2 for Navals and nomads
    if (t !== -1) return type === "Naval" || type === "Lake" ? 100 : 0; // penalty for mainland for navals
    return 0;
  }*/
