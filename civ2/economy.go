package civ2

import (
	"fmt"
	"math"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
)

func (m *Civ) handleProduction(r int, s *Storage, pop int, origin fmt.Stringer) {
	biome := m.Geo.GetRegWhittakerModBiome(r)
	precipitation := m.Geo.Rainfall.GetValues()[r] * geo.MaxPrecipitation * 100
	steepness := m.Geo.Steepness.GetValues()[r]
	temp := m.GetRegTemperature(r)

	// Suitability for gathering.
	gatheringScore := 0.2 + 0.8*(1.0-steepness)*(math.Min(precipitation, 1000)/1000)*math.Max(0, math.Min(temp/30, 1.0))
	if biome == genbiome.WhittakerModBiomeSnow || biome == genbiome.WhittakerModBiomeSubtropicalDesert {
		gatheringScore *= 0.5
	}

	// Suitability for hunting.
	huntingScore := 0.4 + 0.6*(1.0-steepness)*(math.Min(precipitation, 1000)/1000)*math.Max(0, math.Min(temp/30, 1.0))
	if biome == genbiome.WhittakerModBiomeTemperateRainforest || biome == genbiome.WhittakerModBiomeTropicalRainforest {
		huntingScore *= 0.5
	}

	// Production rates.
	foodHunting := float64(pop) * 0.5 * huntingScore * 5.0
	foodGathering := float64(pop) * 0.9 * gatheringScore * 3.0

	s.AddResource(ResFood, int(foodHunting+foodGathering))
	s.AddResource(ResLeather, int(foodHunting*0.2))

	// Raw resources from the region.
	res := m.getResources(r, true)
	s.AddResources(res)
}

func (m *Civ) handleConsumption(s *Storage, pop int) {
	// Feed people.
	foodNeeded := pop
	if !s.RemoveResource(ResFood, foodNeeded) {
		// Famine!
		// s.Resources[ResFood] = 0
		// TODO: handle famine (kill population)
	}

	// Fuel (wood).
	fuelNeeded := pop / 2
	if !s.TakeNOfType(geo.ResourceTypeWood, fuelNeeded) {
		// Cold!
	}
}
