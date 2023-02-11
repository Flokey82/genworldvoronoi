package genworldvoronoi

import (
	"math/rand"

	"github.com/Flokey82/genbiome"
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
	return roundToDecimals(((rand.Float64()*powerInputValue)/2+1)*base, 1)
}

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
	return roundToDecimals(((rand.Float64()*powerInputValue)/2+1)*base, 1)
}

// CellTypeCost returns the cost of crossing / navigating a given cell type for a given culture.
func (t CultureType) CellTypeCost(cellType int) float64 {
	// Land near coast / coastline / coastal land strip / "beach"?.
	if cellType == CellTypeCoastalLand {
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
	if cellType == CellTypeInland {
		if t == CultureTypeNaval || t == CultureTypeNomadic {
			// Small penalty for land with distance 2 to ocean for navals and nomads.
			return 1.3
		}
		// All other cultures do not have appreciable penalty.
		return 1.0
	}

	// Not water near coast (deep ocean/coastal land).
	if cellType != CellTypeCoastalWater {
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
