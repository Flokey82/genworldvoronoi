package civ

import (
	"fmt"
	"strings"

	"github.com/Flokey82/genworldvoronoi/geo"
)

// CityStat represents the stats of a city.
type CityStat int

// CityStat constants.
// TODO:
// - Crime
// - Pollution
// - Happiness
// - Health / Sanitation
const (
	CityStatDefense CityStat = iota
	CityStatHousing
	CityStatFood
	CityStatWater
	CityStatMax
)

// String returns the string representation of the city stat.
func (cs CityStat) String() string {
	switch cs {
	case CityStatDefense:
		return "Defense"
	case CityStatHousing:
		return "Housing"
	case CityStatFood:
		return "Food"
	case CityStatWater:
		return "Water"
	}
	return "Unknown"
}

// CityStats represents the stats of a city.
// TODO: Stats should range from 0.0 to 2.0.
// - Defense will influence the success of an attack.
// - Housing will influence the population growth and happiness.
// - Food will influence the population growth and happiness.
// - Water will influence food production and happiness.
type CityStats [CityStatMax]float64

// String returns the string representation of the city stats.
func (cs CityStats) String() string {
	var res []string
	for i, s := range cs {
		res = append(res, fmt.Sprintf("%s: %.2f", CityStat(i), s))
	}
	return strings.Join(res, ", ")
}

// CalcCityStats calculates the stats of a city.
func (m *Civ) CalcCityStats(c *City, propF func(int) geo.RegProperty) CityStats {
	// TODO: Calculate the ideal stats for the city so
	// that we can compare the current stats with the ideal stats
	// and make decisions based on that.

	// Calculate the stats of the city.
	// Reset the stats.
	for i := range c.CityStats {
		c.CityStats[i] = 0
	}

	// Get the region property.
	prop := propF(c.ID)

	// Calculate the base stats.

	// CityStatDefense first.
	// Bonus:
	// - valley
	// - on mountain
	// - near coast
	// - near river
	// - TODO: forest, steep terrain, paninsula (surrounded by water), hostile climate, etc.
	// Malus:
	// - exposed, flat terrain
	// - steppe, desert, etc.
	if prop.IsValley || prop.DistanceToMountain <= 1 {
		c.CityStats[CityStatDefense] += 1
	}
	if prop.DistanceToCoast <= 1 || prop.DistanceToRiver <= 1 {
		c.CityStats[CityStatDefense] += 1
	}

	// Count how many neighbors are higher than the current region,
	// how many are oceanic, and how many are rivers.
	/*
		var countHigher int
		nbs := m.R_circulate_r(nil, c.ID)
		for _, nb := range nbs {
			if m.Elevation.Values[nb] > m.Elevation.Values[c.ID] {
				countHigher++
			}
		}


		// If all are higher, we are in a sink and have a malus.
		if countHigher == len(nbs) {
			c.CityStats[CityStatDefense] -= 1
		} else {
			// If we are the highest, we have a bonus.
			c.CityStats[CityStatDefense] += float64(countHigher) / float64(len(nbs))
		}
	*/

	// CityStatHousing.
	// CityStatFood.
	// There are two sources of food: agriculture and hunting.
	// So we need to use the arable land function and some function to get the
	// potential wildlife in the region.
	// We have the arable land function, but we need to implement the wildlife function.
	// For now take the survivability function and use it as a proxy for wildlife.
	arableFunc := m.GetFitnessArableLand()
	survFunc := m.GetFitnessSurviability()

	// We will use the higher of the two values.
	c.CityStats[CityStatFood] = max(arableFunc(c.ID), survFunc(c.ID)) * 2

	// Bonus:
	// - near river
	// - near coast

	// CityStatWater.
	if prop.DistanceToRiver <= 1 || prop.DistanceToCoast <= 1 {
		c.CityStats[CityStatWater] += 1
	}

	// We just iterate over the city features and calculate the stats.
	for _, f := range c.Features.feats {
		for _, eff := range f.StatEffect {
			switch eff.Mode {
			case StatEffectModeAdd:
				c.CityStats[eff.Stat] += eff.Value
			case StatEffectModeMul:
				c.CityStats[eff.Stat] *= eff.Value
			}
		}
	}
	return c.CityStats
}
