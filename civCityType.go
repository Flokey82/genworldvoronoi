package genworldvoronoi

import (
	"log"

	"github.com/Flokey82/genbiome"
)

// getRegCityType returns the optimal type of city for a given region.
func (m *Civ) getRegCityType(r int) TownType {
	// If we have a lot of metals, gems, etc. we have a mining town.
	if m.Metals[r] > 0 || m.Gems[r] > 0 {
		return TownTypeMining
	}

	// If we have stone, we have a quarry.
	if m.Stones[r] > 0 {
		return TownTypeQuarry
	}

	// TODO: Cache this somehow.
	if m.getFitnessArableLand()(r) > 0.5 {
		return TownTypeFarming
	}
	// TODO: Add more types of cities.
	return TownTypeDefault
}

// TownType represents the type of a city.
type TownType string

// The different types of cities.
const (
	TownTypeDefault     TownType = "town"
	TownTypeTrading     TownType = "trading"
	TownTypeMining      TownType = "mining"
	TownTypeMiningGems  TownType = "mining (gems)"
	TownTypeQuarry      TownType = "quarry"
	TownTypeFarming     TownType = "agricultural"
	TownTypeDesertOasis TownType = "desert oasis"
)

// FoundingPopulation returns the starting population of a city type.
func (t TownType) FoundingPopulation() int {
	switch t {
	case TownTypeDefault:
		return 100
	case TownTypeTrading:
		return 80
	case TownTypeQuarry, TownTypeMining, TownTypeMiningGems:
		return 20
	case TownTypeFarming:
		return 20
	case TownTypeDesertOasis:
		return 20
	default:
		log.Fatalf("unknown city type: %s", t)
	}
	return 0
}

// GetDistanceSeedFunc returns the distance seed function for a city type.
func (t TownType) GetDistanceSeedFunc(m *Civ) func() []int {
	// For now we just maximize the distance to cities of the same type.
	return func() []int {
		var cities []int
		for _, c := range m.Cities {
			if c.Type == t {
				cities = append(cities, c.ID)
			}
		}
		return cities
	}
}

type fitCache struct {
	f       func(int) float64
	fGetter func() func(int) float64
}

func newFitCache(fGetter func() func(int) float64) *fitCache {
	return &fitCache{
		fGetter: fGetter,
	}
}

func (fc *fitCache) getFunc() func(int) float64 {
	if fc.f == nil {
		fc.f = fc.fGetter()
	}
	return fc.f
}

type fitCaches struct {
	climate *fitCache
	city    *fitCache
	trading *fitCache
	steep   *fitCache
	water   *fitCache
	arable  *fitCache
}

func (m *Civ) getFitCaches() *fitCaches {
	return &fitCaches{
		climate: newFitCache(m.getFitnessClimate),
		city:    newFitCache(m.getFitnessCityDefault),
		trading: newFitCache(m.getFitnessTradingTowns),
		steep:   newFitCache(m.getFitnessSteepMountains),
		water:   newFitCache(m.getFitnessProximityToWater),
		arable:  newFitCache(m.getFitnessArableLand),
	}
}

// GetFitnessFunction returns the fitness function for a city type.
func (t TownType) GetFitnessFunction(m *Civ, fitc *fitCaches) func(int) float64 {
	// TODO: Create different fitness functions for different types of settlement.
	//   - Capital
	//   - Cities / Settlements
	//     ) Proximity to capital!
	//   - Agricultural
	//   - Mining
	//   - ...
	switch t {
	case TownTypeDefault:
		fa := fitc.climate.getFunc()
		fb := fitc.city.getFunc()
		return func(r int) float64 {
			return fa(r) * fb(r)
		}
	case TownTypeTrading:
		return fitc.trading.getFunc()
	case TownTypeQuarry:
		fa := fitc.steep.getFunc()
		fb := fitc.climate.getFunc()
		fc := fitc.water.getFunc()
		fd := m.getFitnessProximityToCities(TownTypeMining, TownTypeMiningGems, TownTypeQuarry)
		return func(r int) float64 {
			if m.Stones[r] == 0 {
				return -1.0
			}
			return fd(r) * (fa(r)*fb(r) + fc(r)) / 2
		}
	case TownTypeMining:
		fa := fitc.steep.getFunc()
		fb := fitc.climate.getFunc()
		fc := fitc.water.getFunc()
		fd := m.getFitnessProximityToCities(TownTypeMining, TownTypeMiningGems, TownTypeQuarry)
		return func(r int) float64 {
			if m.Metals[r] == 0 {
				return -1.0
			}
			return fd(r) * (fa(r)*fb(r) + fc(r)) / 2
		}
	case TownTypeMiningGems:
		fa := fitc.steep.getFunc()
		fb := fitc.climate.getFunc()
		fc := fitc.water.getFunc()
		fd := m.getFitnessProximityToCities(TownTypeMining, TownTypeMiningGems, TownTypeQuarry)
		return func(r int) float64 {
			if m.Gems[r] == 0 {
				return -1.0
			}
			return fd(r) * (fa(r)*fb(r) + fc(r)) / 2
		}
	case TownTypeFarming:
		return fitc.arable.getFunc()
	case TownTypeDesertOasis:
		// TODO: Improve this fitness function.
		// Right now the oasis are placed at the very edges of
		// deserts, as there is the "best" climate.
		// However, we want them to be trade hubs for desert
		// crossings... so we'll need to place them in the middle
		// of deserts instead.
		fa := fitc.climate.getFunc()
		bf := m.getRegWhittakerModBiomeFunc()
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
