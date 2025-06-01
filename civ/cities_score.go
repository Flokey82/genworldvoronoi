package civ

import (
	"math"

	"github.com/Flokey82/go_gens/utils"
)

type CityScores struct {
	Economic       float64 // Economic potential of the city (DYNAMIC)
	Trade          float64 // Trade value of the city (DYNAMIC)
	Resources      float64 // Resources value of the city (PARTLY DYNAMIC)
	Agricultural   float64 // Agriculture value of the city (STATIC)
	Attractiveness float64 // Attractiveness of the city (STATIC)
}

func (m *Civ) calculateEconomicPotential() {
	// We only consider cities that are founded prior or in the current year.
	cities := m.getExistingCities()

	// Calculate the analog of distance between regions by taking the surface
	// of a sphere with radius 1 and dividing it by the number of regions.
	// The square root will work as a somewhat sensible approximation of
	// distance.
	distRegion := math.Sqrt(4*math.Pi/float64(m.SphereMesh.NumRegions)) * unitDistToKm

	// Calculate the base radius in which we can find trade partners.
	var tradeRadius []float64
	for _, c := range cities {
		// The base radius is dependent on the population.
		// ... allow for at least two regions distance.
		tradeRadius = append(tradeRadius, c.Radius()+2*distRegion)
	}

	economicPotential := make([]float64, len(cities))
	for i, c := range cities {
		economicPotential[i] = c.Resources + c.Agricultural
	}

	// Now we go through all the cities, and see if they might be able to
	// trade with each other. This way they can profit from each other's
	// resources.
	//
	// In the future we make this dependent on geographic features, where
	// mountains or the sea might be a barrier.
	//
	// TODO: This should in particular also take in account what kind of
	// resources are available and which are needed, so we would trade
	// only if we have benefits from it. This would also mean that far
	// away mining towns might profit from trade.
	tradePotential := make([]float64, len(cities))
	for i, c := range cities {
		if c.Population == 0 {
			continue // We don't trade with dead cities.
		}

		// Loop through all cities and check if we can trade with them.
		for j, c2 := range cities {
			if i == j || c2.Population == 0 {
				continue // We don't trade with ourselves or dead cities.
			}

			// The trade radius is the sum of the two cities' radius times their economic potential.
			radius := tradeRadius[i]*(1+economicPotential[i]) + tradeRadius[j]*(1+economicPotential[j])

			// If the distance is within the radius, we can trade.
			if dist := m.GetDistance(c.ID, c2.ID) * unitDistToKm; dist <= radius {
				// If the other city has a higher economic potential, we profit less from trade (15% vs 20%).
				if economicPotential[j] > economicPotential[i] {
					tradePotential[i] += economicPotential[i] * (1 - dist/radius) * 0.15
				} else {
					tradePotential[i] += economicPotential[j] * (1 - dist/radius) * 0.2
				}
			}
		}
	}

	// DEBUG: Count the number of cities in range.
	// Loop through all cities and check if we can trade with them.
	for i, c := range cities {
		var tradePartners []int
		for j, c2 := range cities {
			if i == j {
				continue // We don't trade with ourselves.
			}

			if dist := m.GetDistance(c.ID, c2.ID) * unitDistToKm; dist <= tradeRadius[i] {
				tradePartners = append(tradePartners, c2.ID)
			}
		}
		c.TradePartners = tradePartners
	}

	// Now normalize trade potential.
	if maxTrade := utils.MaxArray(tradePotential); maxTrade > 0 {
		for i := range cities {
			tradePotential[i] /= maxTrade
		}
	}

	// Assign the economic potential.
	for i, c := range cities {
		c.Economic = economicPotential[i] + tradePotential[i]
		c.Trade = tradePotential[i]
	}
}

func (m *Civ) calculateCitiesStats(cities []*City) {
	// Calculate the stats of all cities.
	m.calculateAttractiveness(cities)
	m.calculateAgriculturalPotential(cities)
	m.calculateResourcePotential(cities)
}

func (m *Civ) calculateAttractiveness(cities []*City) {
	// Calculate the attractiveness of the supplied cities.
	attrFunc := m.getAttractivenessFunc()
	for _, c := range cities {
		c.Attractiveness = attrFunc(c.ID)
	}
}

func (m *Civ) calculateAgriculturalPotential(cities []*City) {
	// Calculate the agricultural potential of the supplied cities.
	fitnessArableFunc := m.GetFitnessArableLand()
	for _, c := range cities {
		if agrPotential := fitnessArableFunc(c.ID); agrPotential > 0 {
			c.Agricultural = agrPotential
		}
	}
}

func (m *Civ) calculateResourcePotential(cities []*City) {
	// Reset and recalculate the resource potential.
	for _, c := range cities {
		c.Resources = m.calculateCityResourcePotential(c)
	}
}

func (m *Civ) calculateCityAgriculturalPotential(c *City) float64 {
	// Calculate the agricultural potential of the city.
	return m.GetFitnessArableLand()(c.ID)
}

func (m *Civ) calculateCityResourcePotential(c *City) float64 {
	// Calculate the resource potential of the city.
	return float64(m.SumValueOfRegion(c.ID)) / float64(m.GetResourceTotal())
}

func (m *Civ) getAttractivenessFunc() func(int) float64 {
	// The attractiveness of a region is dependent on the following factors:
	// - Climate and elevation
	// - Distance to water (ocean, river, lake)
	// - Arable land (self-sufficiency)
	climateFitnessFunc := m.GetFitnessClimate()
	arableLandFitnessFunc := m.GetFitnessArableLand()
	proximityToWaterFitnessFunc := m.GetFitnessProximityToWater()

	return func(regionID int) float64 {
		// The attractiveness is the average of the fitness functions.
		return (climateFitnessFunc(regionID) + arableLandFitnessFunc(regionID) + proximityToWaterFitnessFunc(regionID)) / 3
	}
}
