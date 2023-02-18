package genworldvoronoi

import (
	"log"
	"math"
)

// GetCity returns the city at the given region / with the given ID.
func (m *Civ) GetCity(r int) *City {
	for _, c := range m.Cities {
		if c.ID == r {
			return c
		}
	}
	return nil
}

func (m *Civ) getExistingCities() []*City {
	var cities []*City
	for _, c := range m.Cities {
		if c.Founded <= m.History.GetYear() {
			cities = append(cities, c)
		}
	}
	return cities
}

func (m *Civ) ageCities() {
	// HACK: Age city populations.
	// TODO: Instead we should spawn the cities from the capitals.
	// Also, the theoretical population should be based on the
	// economic potential of the region, the type of settlement,
	// and the time of settlement.
	cultureFunc := m.getCultureFunc()
	gDisFunc := m.Geo.getGeoDisasterFunc()

	// Get the year when the last region was settled.
	_, maxSettled := minMax64(m.Settled)

	// Reset the year to 0.
	m.Geo.Calendar.SetYear(0)
	knownCities := len(m.Cities)
	for year := int64(0); year < maxSettled; year++ {
		// Age cities that exist at this given year.
		for _, c := range m.getExistingCities() {
			// if c.Founded == year {
			//	// If the city was just founded this year, we generate
			//	// a random population.
			//	m.addNToCity(c, c.Population, cultureFunc)
			// }

			// Age the city.
			m.tickCityDays(c, gDisFunc, cultureFunc, 365)
		}

		// Update attractiveness, agricultural potential, and resource potential
		// for new cities.
		if len(m.Cities) > knownCities {
			// TODO: Only update new regions until we have climate change?
			m.calculateCitiesStats(m.Cities[knownCities:])
			knownCities = len(m.Cities)
		}

		// Recalculate economic potential.
		m.calculateEconomicPotential()

		// var totalPeople int
		// for _, c := range m.getExistingCities() {
		// 	totalPeople += len(c.People)
		// }
		// log.Printf("Total people: %d", totalPeople)

		// Advance year.
		m.Geo.Calendar.TickYear()
		log.Printf("Aged cities to %d/%d\n", year, maxSettled)

		// Age population.
		// TODO: Would it make more sense to age the population
		// per city? Or per region?
		// m.People = m.tickPeople(m.People, 356, cultureFunc)
	}
}

func (m *Civ) calculateEconomicPotential() {
	// TODO: Cities should have several values
	// Some are static, some are dynamic.
	//
	// Static:
	//
	// Static values are based on the region and are not affected by
	// the population.
	//
	// - Local resources
	//   metals, food, etc.
	// - Arable land score
	//   how much land is arable
	// - Climate
	//   how attractive is the climate for settlement
	// - Access to water
	//
	// Dynamic:
	//
	// This is based on the population, which directly impacts the
	// maximum distance from which resources can be gathered,
	// and the number of cities we can trade with.
	//
	// - Trade with nearby cities
	// - Nearby resources
	//
	// Other interesting values to consider:
	//   culture, is capital, etc.

	// We only consider cities that are founded prior or in the current year.
	cities := m.getExistingCities()

	// Calculate the analog of distance between regions by taking the surface
	// of a sphere with radius 1 and dividing it by the number of regions.
	// The square root will work as a somewhat sensible approximation of
	// distance.
	distRegion := math.Sqrt(4 * math.Pi / float64(m.mesh.numRegions))

	// Calculate the base radius in which we can find trade partners.
	var tradeRadius []float64
	for _, c := range cities {
		// The base radius is dependent on the population.
		// ... allow for at least two regions distance.
		radius := c.radius() + 2*distRegion
		tradeRadius = append(tradeRadius, radius)
	}

	economicPotential := make([]float64, len(cities))
	for i, c := range cities {
		economicPotential[i] = c.Resources + c.Agriculture
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
		// Calculate the distance field of all cities to the current city.
		if c.Population == 0 {
			continue
		}

		// Loop through all cities and check if we can trade with them.
		for j, c2 := range cities {
			// We don't trade with ourselves.
			if i == j || c2.Population == 0 {
				continue
			}
			// The trade radius is the sum of the two cities' radius times their economic potential.
			radius := tradeRadius[i]*(1+economicPotential[i]) + tradeRadius[j]*(1+economicPotential[j])

			// If the distance is within the radius, we can trade.
			// However, if the other city has a higher economic potential,
			// we profit less from the trade.
			// TODO: Switch this to population size?
			// the closer we are, the more economic potential we have (up to 20%).
			dist := m.GetDistance(c.ID, c2.ID)
			if dist <= radius {
				if economicPotential[j] > economicPotential[i] {
					// We don't profit as much from the trade (up to 15%).
					tradePotential[i] += economicPotential[i] * (1 - dist/radius) * 0.15
				} else {
					// We profit more from the trade (up to 20%).
					tradePotential[i] += economicPotential[j] * (1 - dist/radius) * 0.2
				}
			}
		}
	}

	// DEBUG: Count the number of cities in range.
	// Loop through all cities and check if we can trade with them.
	for i, c := range cities {
		var count int
		for j, c2 := range cities {
			if i == j {
				continue // We don't trade with ourselves.
			}
			dist := m.GetDistance(c.ID, c2.ID)
			if dist <= tradeRadius[i] {
				count++
			}
		}
		c.TradePartners = count
	}

	// Now normalize trade potential.
	if _, maxTrade := minMax(tradePotential); maxTrade > 0 {
		for i := range cities {
			tradePotential[i] /= maxTrade
		}
	}

	// Assign the economic potential.
	for i, c := range cities {
		c.EconomicPotential = economicPotential[i] + tradePotential[i]
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
	fitnessArableFunc := m.getFitnessArableLand()
	for _, c := range cities {
		if agrPotential := fitnessArableFunc(c.ID); agrPotential > 0 {
			c.Agriculture = agrPotential
		}
	}
}

func (m *Civ) calculateResourcePotential(cities []*City) {
	// Now get the resource potential of all cities.
	calcResourceValues := func(res []byte) {
		for _, c := range cities {
			// Sum up the normalized resource values.
			c.Resources += float64(sumResources(res[c.ID])) / 36 // 36 is the maximum value.
		}
	}

	// Reset the resource potential.
	for _, c := range cities {
		c.Resources = 0
	}

	// Calculate the resource potential for each resource.
	calcResourceValues(m.Metals)
	calcResourceValues(m.Gems)
	calcResourceValues(m.Stones)
	calcResourceValues(m.Wood)
	calcResourceValues(m.Various)
}

func (m *Civ) getAttractivenessFunc() func(int) float64 {
	// The attractiveness of a region is dependent on the following factors:
	// - Climate and elevation
	// - Distance to water (ocean, river, lake)
	// - Arable land (self-sufficiency)
	climateFitnessFunc := m.getFitnessClimate()
	arableLandFitnessFunc := m.getFitnessArableLand()
	proximityToWaterFitnessFunc := m.getFitnessProximityToWater()

	return func(regionID int) float64 {
		// The attractiveness is the average of the fitness functions.
		return (climateFitnessFunc(regionID) + arableLandFitnessFunc(regionID) + proximityToWaterFitnessFunc(regionID)) / 3
	}
}

// CalcCityScore calculates the fitness value for settlements for all regions.
//
// 'sf': Fitness function for scoring a region.
// 'distSeedFunc': Returns a number of regions from which we maximize the distance.
func (m *Civ) CalcCityScore(sf func(int) float64, distSeedFunc func() []int) []float64 {
	sfCity := func(r int) float64 {
		// If we are below (or at) sea level, or we are in a pool of water,
		// assign lowest score and continue.
		if m.Elevation[r] <= 0 || m.Waterpool[r] > 0 {
			return -1.0
		}
		return sf(r)
	}
	return m.CalcFitnessScore(sfCity, distSeedFunc)
}

func (m *Civ) getFitnessTradingTowns() func(int) float64 {
	// TODO: Fix this.
	// I think this function should avoid the penalty wrt.
	// proximity to towns of other types.
	_, connecting := m.getTradeRoutes()
	return func(r int) float64 {
		return float64(len(connecting[r]))
	}
}

func (m *Civ) getFitnessCityDefault() func(int) float64 {
	_, maxFlux := minMax(m.Flux)
	steepness := m.GetSteepness()

	return func(r int) float64 {
		// If we are below (or at) sea level, or we are in a pool of water,
		// assign lowest score and continue.
		if m.Elevation[r] <= 0 || m.Waterpool[r] > 0 {
			return -1.0
		}

		// Visit all neighbors and modify the score based on their properties.
		var hasWaterBodyBonus bool
		nbs := m.GetRegNeighbors(r)

		// Initialize fitness score with the normalized flux value.
		// This will favor placing cities along (and at the end of)
		// large rivers.
		score := math.Sqrt(m.Flux[r] / maxFlux)
		for _, nb := range nbs {
			// Add bonus if near ocean or lake.
			if m.isRegBelowOrAtSeaLevelOrPool(nb) {
				// We only apply this bonus once.
				if hasWaterBodyBonus {
					continue
				}
				// If a neighbor is below (or at) sea level, or a lake,
				// we increase the fitness value and reduce it by a fraction,
				// depending on the size of the lake or ocean it is part of.
				//
				// TODO: Improve this.

				// If nb is part of a waterbody (ocean) or lake, we reduce the score by a constant factor.
				// The larger the waterbody/lake, the smaller the penalty, which will favor larger waterbodies.
				if wbSize := m.getRegLakeOrWaterBodySize(nb); wbSize > 0 {
					hasWaterBodyBonus = true
					score += 0.55 * (1 - 1/(float64(wbSize)+1e-9))
				}
			} else {
				// If the sourrounding terrain is flat, we get a bonus.
				stp := steepness[nb]
				score += 0.5 * (1.0 - stp*stp) / float64(len(nbs))
			}

			// TODO:
			// - Consider biome
			// - Consider sediment/fertility of land.
			// - Add bonus for mountain proximity (mines, resources)
		}

		// The steeper the terrain, the less likely it is to be settled.
		// TODO: Bonus for trade routes.
		stp := steepness[r]
		score *= 1.0 - (stp * stp)
		return score
	}
}

func (m *Civ) getFitnessProximityToCities(except ...TownType) func(int) float64 {
	var cities []int
	exceptMap := make(map[TownType]bool)
	for _, t := range except {
		exceptMap[t] = true
	}
	for _, c := range m.Cities {
		if !exceptMap[c.Type] {
			cities = append(cities, c.ID)
		}
	}
	distCities := m.assignDistanceField(cities, make(map[int]bool))
	_, maxDist := minMax(distCities)
	if maxDist == 0 {
		maxDist = 1
	}
	return func(r int) float64 {
		if distCities[r] == 0 {
			return 0
		}
		return 1 - float64(distCities[r])/maxDist
	}
}
