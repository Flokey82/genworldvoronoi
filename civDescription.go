package genworldvoronoi

import (
	"log"

	"github.com/Flokey82/genworldvoronoi/geo"
)

func (m *Civ) generateCitiesFlavorText() {
	rpFunc := m.GetRegPropertyFunc()
	for _, c := range m.Cities {
		flvTxt := m.generateCityFlavorText(c, rpFunc(c.ID))
		log.Println(c.Name, flvTxt)
	}
}

const (
	minPopCity    = 10000
	minPopTown    = 1000
	minPopVillage = 100
)

// generateCityFlavorText generates a flavor text for a city.
func (m *Civ) generateCityFlavorText(c *City, p geo.RegProperty) string {
	str := c.Name + " is a "
	if c.Population == 0 {
		str += "deserted "
		if c.MaxPopulation > minPopCity {
			str += "city"
		} else if c.MaxPopulation > minPopTown {
			str += "town"
		} else {
			str += "village"
		}
	} else if c.Population < minPopVillage {
		if c.MaxPopulation > minPopTown {
			str += "desolate town"
		} else {
			str += "small village"
		}
	} else if c.Population < minPopTown {
		str += "small town"
	} else if c.Population < minPopCity {
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

	// Generate some flavor text describing the region.
	str += m.GenerateRegPropertyDescription(p)

	// ... and finally add some flavor text for the biome.
	return str + geo.GenerateFlavorTextForBiome(int64(c.ID), p.Biome)
}
