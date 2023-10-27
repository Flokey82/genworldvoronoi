package genworldvoronoi

import (
	"fmt"
	"math"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
)

// SpeciesTolerances defines the environmental tolerances of a species.
type SpeciesTolerances struct {
	Ecosphere       EcosphereType // Ocean, River, Land, Lake
	TempRange       [2]float64    // Min, Max temperature
	HumRange        [2]float64    // Min, Max humidity
	RainRange       [2]float64    // Min, Max rain
	ElevRange       [2]float64    // Min, Max elevation (this should be maybe in meters?)
	SteepRange      [2]float64    // Min, Max steepness
	PreferredBiomes []int         // Only applies to non-marine species.
}

func (s *SpeciesTolerances) String() string {
	var str string
	str += fmt.Sprintf("TEMP: %.2f°C - %.2f°C,\n", s.TempRange[0], s.TempRange[1])
	str += fmt.Sprintf("HUMD: %.2fdm - %.2fdm,\n", s.HumRange[0]*geo.MaxPrecipitation, s.HumRange[1]*geo.MaxPrecipitation)
	str += fmt.Sprintf("RAIN: %.2fdm - %.2fdm,\n", s.RainRange[0]*geo.MaxPrecipitation, s.RainRange[1]*geo.MaxPrecipitation)
	str += fmt.Sprintf("ELEV: %.2f-%.2f\n", s.ElevRange[0]*geo.MaxAltitudeFactor, s.ElevRange[1]*geo.MaxAltitudeFactor)
	str += fmt.Sprintf("STEE: %.2f-%.2f\n", s.SteepRange[0], s.SteepRange[1])
	if len(s.PreferredBiomes) > 0 {
		str += "biomes:\n"
		for _, b := range s.PreferredBiomes {
			str += fmt.Sprintf(" %s\n", genbiome.WhittakerModBiomeToString(b))
		}
	}
	return str
}

func (b *Bio) getTolerancesForRegionFunc() func(int) SpeciesTolerances {
	minElev, maxElev := minMax(b.Elevation)
	_, maxHum := minMax(b.Moisture)
	_, maxRain := minMax(b.Rainfall)
	steep := b.GetSteepness()
	return func(r int) SpeciesTolerances {
		s := SpeciesTolerances{
			Ecosphere: b.getEcosphere(r),
		}

		// minMaxRange returns a min and max range for the given value with the given variance.
		minMaxRange := func(val, minVal, maxVal, variation float64) [2]float64 {
			nVar := b.rand.Float64() * variation
			varMin := nVar * b.rand.Float64()
			newMin := math.Max(minVal, val-varMin)
			newMax := math.Min(maxVal, val+(nVar-varMin))
			return [2]float64{newMin, newMax}
		}

		// The possible variation of the ranges.
		variation := 0.2

		// Prefered elevation range.
		if s.Ecosphere == EcosphereTypeOcean {
			s.ElevRange = minMaxRange(b.Elevation[r], minElev, 0, variation)
		} else {
			s.ElevRange = minMaxRange(b.Elevation[r], 0, maxElev, variation)
		}

		// Preferred temperature range.
		s.TempRange = minMaxRange(b.GetRegTemperature(r, maxElev), float64(geo.MinTemp), float64(geo.MaxTemp), variation)

		// Preferred humidity range.
		s.HumRange = minMaxRange(b.Moisture[r]/maxHum, 0, 1, variation)

		// Preferred rain range.
		s.RainRange = minMaxRange(geo.MaxPrecipitation*b.Rainfall[r]/maxRain, 0, geo.MaxPrecipitation, variation)

		// Preferred steepness range.
		s.SteepRange = minMaxRange(steep[r], 0, 1, variation)

		// If we are not in the ocean, we probably have a preferred biome.
		if s.Ecosphere != EcosphereTypeOcean && b.rand.Float64() < 0.7 {
			s.PreferredBiomes = []int{b.GetRegWhittakerModBiomeFunc()(r)}
		}
		return s
	}
}

func (b *Bio) getToleranceScoreFunc(s SpeciesTolerances) func(int) float64 {
	_, maxElev := minMax(b.Elevation)
	_, maxHum := minMax(b.Moisture)
	_, maxRain := minMax(b.Rainfall)
	bf := b.GetRegWhittakerModBiomeFunc()
	steepness := b.GetSteepness()
	return func(r int) float64 { // Check what ecosphere we are in and if it matches the species.
		if !s.Ecosphere.isSet(b.getEcosphere(r)) {
			return -1.0
		}

		// Check if we require a specific biome.
		if len(s.PreferredBiomes) > 0 && !isInIntList(s.PreferredBiomes, bf(r)) {
			return -1
		}

		var tempScore, humScore, elevScore, steepScore, rainScore float64

		// Check how much we diverge from the preferred temperature range.
		if isRangeSet(s.TempRange) {
			tempScore = getRangeFit(b.GetRegTemperature(r, maxElev), s.TempRange)
			if tempScore == -1 {
				return -1
			}
		} else {
			tempScore = 0.5
		}

		// Check how much we diverge from the preferred humidity range.
		if isRangeSet(s.HumRange) {
			humScore = getRangeFit(b.Moisture[r]/maxHum, s.HumRange)
			if humScore == -1 {
				return -1
			}
		} else {
			humScore = 0.5
		}

		// Check how much we diverge from the preferred rain range.
		if isRangeSet(s.RainRange) {
			rainScore = getRangeFit(geo.MaxPrecipitation*b.Rainfall[r]/maxRain, s.RainRange)
			if rainScore == -1 {
				return -1
			}
		} else {
			rainScore = 0.5
		}

		// Check how much we diverge from the preferred elevation range.
		if isRangeSet(s.ElevRange) {
			elevScore = getRangeFit(b.Elevation[r], s.ElevRange)
			if elevScore == -1 {
				return -1
			}
		} else {
			elevScore = 0.5
		}

		// Check how much we diverge from the preferred steepness range.
		if isRangeSet(s.SteepRange) {
			steepScore = getRangeFit(steepness[r], s.SteepRange)
			if steepScore == -1 {
				return -1
			}
		} else {
			steepScore = 0.5
		}
		total := (tempScore + humScore + elevScore + steepScore + rainScore) / 5
		if total < 0.3 {
			return -1
		}
		return total
	}
}

type EcosphereType byte

const (
	EcosphereTypeOcean EcosphereType = 1 << iota
	EcosphereTypeRiver
	EcosphereTypeLake
	EcosphereTypeLand
)

func (e EcosphereType) IsWater() bool {
	return e == EcosphereTypeOcean || e == EcosphereTypeRiver || e == EcosphereTypeLake
}

func (e EcosphereType) isSet(t EcosphereType) bool {
	return e&t == t
}

// getEcosphere returns the ecosphere of the given region.
func (b *Bio) getEcosphere(r int) EcosphereType {
	// Get the ecosphere we are in.
	if b.Elevation[r] <= 0.0 {
		return EcosphereTypeOcean
	}
	if b.IsRegRiver(r) {
		return EcosphereTypeRiver
	}
	if b.IsRegLake(r) {
		return EcosphereTypeLake
	}
	return EcosphereTypeLand
}

func (e EcosphereType) String() string {
	switch e {
	case EcosphereTypeOcean:
		return "ocean"
	case EcosphereTypeRiver:
		return "river"
	case EcosphereTypeLake:
		return "lake"
	case EcosphereTypeLand:
		return "land"
	}
	return "unknown"
}

func isRangeSet(r [2]float64) bool {
	return r[0] != 0 || r[1] != 0
}

func easeInOutCubic(x float64) float64 {
	if x < 0.5 {
		return 4 * x * x * x
	}
	return 1 - math.Pow(-2*x+2, 3)/2
}
