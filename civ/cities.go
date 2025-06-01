package civ

import (
	"log"
	"time"

	"github.com/Flokey82/go_gens/utils"
)

// GetCity returns the city at the given region / with the given ID.
func (m *Civ) GetCity(r int) *City {
	return m.Cities.GetAt(r)
}

func (m *Civ) getExistingCities() []*City {
	var cities []*City
	for _, c := range m.Cities.Objects {
		if c.Founded <= m.History.GetYear() {
			cities = append(cities, c)
		}
	}
	return cities
}

func (m *Civ) ageCities() {
	// Get the year when the last region was settled.
	maxSettled := utils.MaxArray(m.Settled)

	// Reset the year to 0.
	m.Geo.Calendar.SetYear(0)
	knownCities := 0
	existingCities := m.getExistingCities() // Get the cities that exist at year 0.
	gDisFunc := m.Geo.GetGeoDisasterFunc()  // Get the geo disaster function.
	gRegProp := m.Geo.GetRegPropertyFunc()  // Get the region property function.
	for year := int64(0); year < maxSettled; year++ {
		// Age cities that exist at this given year.
		now := time.Now()
		for _, c := range existingCities {
			// if c.Founded == year {
			//	// If the city was just founded this year, we generate a random population.
			//	m.addNToCity(c, c.Population, cultureFunc)
			// }

			// Age the city.
			m.tickCityDays(c, gDisFunc, gRegProp, 365)
		}

		log.Printf("Aged cities to %d/%d in %s (tick)\n", year, maxSettled, time.Since(now))
		now = time.Now()

		// Update attractiveness, agricultural potential, and resource potential
		// for new cities.
		if len(existingCities) > knownCities {
			// TODO: Update new regions until we have climate change?
			m.calculateCitiesStats(existingCities[knownCities:])
			knownCities = len(existingCities)
		}

		log.Printf("Calculated stats for cities in %s (stats)\n", time.Since(now))
		now = time.Now()

		// Recalculate economic potential.
		m.calculateEconomicPotential()
		log.Printf("Calculated economic potential in %s (economic)\n", time.Since(now))

		// Advance year.
		m.Geo.Calendar.TickYear()
		log.Printf("Aged cities to %d/%d\n", year, maxSettled)

		// var totalPeople int
		// for _, c := range m.getExistingCities() {
		// 	totalPeople += len(c.People)
		// }
		// log.Printf("Total people: %d", totalPeople)

		// Age population.
		// m.People = m.tickPeople(m.People, 356, cultureFunc)
		existingCities = m.getExistingCities()
	}
}
