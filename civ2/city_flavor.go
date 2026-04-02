package civ2

import (
	"github.com/Flokey82/genworldvoronoi/geo"
)

const (
	minPopCity    = 10000
	minPopTown    = 1000
	minPopVillage = 100
)

// GenerateCityFlavorText generates a flavor text for a city or settlement.
func (m *Civ) GenerateCityFlavorText(c peopleThing) string {
	id := c.GetID()
	p := m.GetRegPropertyFunc()(id)
	pop := c.GetPopulation()
	
	str := c.String() + " is a "
	if pop == 0 {
		str += "deserted "
		str += "settlement"
	} else if pop < minPopVillage {
		str += "small village"
	} else if pop < minPopTown {
		str += "small town"
	} else if pop < minPopCity {
		str += "large town"
	} else {
		str += "large city"
	}
	
	if p.IsValley && p.DistanceToCoast > 3 {
		str += " in a valley"
	} else if p.Steepness > 0.5 {
		if p.Elevation > 0.5 {
			str += " on a mountain"
		} else if p.DistanceToCoast <= 1 {
			str += " on a coastal cliff"
		} else {
			str += " on a hillside"
		}
	} else if p.DistanceToCoast <= 1 {
		str += " on the coast"
	}
	str += ".\n"

	// Generate some flavor text describing the region from geo.
	str += m.GenerateRegPropertyDescription(p)

	// ... and finally add some flavor text for the biome.
	return str + geo.GenerateFlavorTextForBiome(int64(id), p.Biome)
}
