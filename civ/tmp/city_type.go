package civ

import (
	"log"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
)

// getRegCityType returns the optimal type of city for a given region.
func (m *Civ) getRegCityType(r int) CityType {
	// TODO: Cache this somehow.
	if m.GetFitnessArableLand()(r) > 0.5 {
		return CityTypeFarming
	}

	// If we have a lot of metals, gems, etc. we have a mining town.
	if m.HasType(r, geo.ResourceTypeMetal) || m.HasType(r, geo.ResourceTypeGem) {
		return CityTypeMining
	}

	// If we have stone, we have a quarry.
	if m.HasType(r, geo.ResourceTypeStone) {
		return CityTypeQuarry
	}

	// TODO: Add more types of cities.
	return CityTypeDefault
}

// CityType represents the type of a city.
type CityType string

// The different types of cities.
const (
	CityTypeDefault     CityType = "town"
	CityTypeTrading     CityType = "trading"
	CityTypeMining      CityType = "mining"
	CityTypeMiningGems  CityType = "mining (gems)"
	CityTypeQuarry      CityType = "quarry"
	CityTypeFarming     CityType = "agricultural"
	CityTypeDesertOasis CityType = "desert oasis"
)

// FoundingPopulation returns the starting population of a city type.
func (t CityType) FoundingPopulation() int {
	switch t {
	case CityTypeDefault:
		return 100
	case CityTypeTrading:
		return 80
	case CityTypeQuarry, CityTypeMining, CityTypeMiningGems:
		return 20
	case CityTypeFarming:
		return 20
	case CityTypeDesertOasis:
		return 20
	default:
		log.Fatalf("unknown city type: %s", t)
	}
	return 0
}

// GetDistanceSeedFunc returns the distance seed function for a city type.
func (t CityType) GetDistanceSeedFunc(m *Civ) func() []int {
	// For now we just maximize the distance to cities of the same type.
	return func() []int {
		var cities []int
		for _, c := range m.Cities.Objects {
			if c.Type == t {
				cities = append(cities, c.ID)
			}
		}
		return cities
	}
}

// GetFitnessFunction returns the fitness function for a city type.
func (t CityType) GetFitnessFunction(m *Civ) func(int) float64 {
	// TODO: Create different fitness functions for different types of settlement.
	//   - Capital
	//   - Cities / Settlements
	//     ) Proximity to capital!
	//   - Agricultural
	//   - Mining
	//   - ...
	switch t {
	case CityTypeDefault:
		fa := m.GetFitnessClimate()
		fb := m.GetFitnessCityDefault()
		return func(r int) float64 {
			return fa(r) * fb(r)
		}
	case CityTypeTrading:
		return m.GetFitnessTradingTowns()
	case CityTypeQuarry:
		fa := m.GetFitnessSteepMountains()
		fb := m.GetFitnessClimate()
		fc := m.GetFitnessProximityToWater()
		fd := m.getFitnessProximityToCities(CityTypeMining, CityTypeMiningGems, CityTypeQuarry)
		return func(r int) float64 {
			if !m.HasType(r, geo.ResourceTypeStone) {
				return -1.0
			}
			return fd(r) * (fa(r)*fb(r) + fc(r)) / 2
		}
	case CityTypeMining:
		fa := m.GetFitnessSteepMountains()
		fb := m.GetFitnessClimate()
		fc := m.GetFitnessProximityToWater()
		fd := m.getFitnessProximityToCities(CityTypeMining, CityTypeMiningGems, CityTypeQuarry)
		return func(r int) float64 {
			if !m.HasType(r, geo.ResourceTypeMetal) {
				return -1.0
			}
			return fd(r) * (fa(r)*fb(r) + fc(r)) / 2
		}
	case CityTypeMiningGems:
		fa := m.GetFitnessSteepMountains()
		fb := m.GetFitnessClimate()
		fc := m.GetFitnessProximityToWater()
		fd := m.getFitnessProximityToCities(CityTypeMining, CityTypeMiningGems, CityTypeQuarry)
		return func(r int) float64 {
			if !m.HasType(r, geo.ResourceTypeGem) {
				return -1.0
			}
			return fd(r) * (fa(r)*fb(r) + fc(r)) / 2
		}
	case CityTypeFarming:
		return m.GetFitnessArableLand()
	case CityTypeDesertOasis:
		// TODO: Improve this fitness function.
		// Right now the oasis are placed at the very edges of
		// deserts, as there is the "best" climate.
		// However, we want them to be trade hubs for desert
		// crossings... so we'll need to place them in the middle
		// of deserts instead.
		fa := m.GetFitnessClimate()
		bf := m.GetRegWhittakerModBiomeFunc()
		return func(r int) float64 {
			biome := bf(r)
			if biome == genbiome.WhittakerModBiomeColdDesert ||
				biome == genbiome.WhittakerModBiomeSubtropicalDesert {
				return fa(r)
			}
			return 0
		}
	default:
		log.Fatalf("unknown city type: %s", t)
	}
	return nil
}
