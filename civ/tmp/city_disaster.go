package civ

import (
	"fmt"
	"log"
	"math"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/geo"
)

func (m *Civ) tickCityDisaster(c *City, gDisFunc func(int) geo.GeoDisasterChance, days int) {
	// Enable / disable migration of population when a disaster strikes.
	enableDisasterMigration := true

	// There is a chance of some form of disaster.
	geoDisasters := gDisFunc(c.ID).GetDisasters() // Disasters that relate to the region.
	cityDisasters := m.getCityDisasters(c)        // Disasters that relate to the city.

	// Pick a random disaster from the list of disasters.
	dis := geo.RandDisaster(append(geoDisasters, cityDisasters...))
	if dis == geo.DisNone {
		panic("no disaster picked")
	}

	// Calculate the population loss.
	// If towns are heavily affected, they might be destroyed or abandoned.
	popLoss := dis.PopulationLoss * (2 + rand.Float64()) / 3
	dead := int(math.Ceil(float64(c.Population) * popLoss))

	// HACK: Kill the people that died in the disaster.
	// c.People = m.killNPeople(c.People, dead)

	// Add an event to the calendar.
	m.AddEvent(dis.Name, fmt.Sprintf("%s died", numPeopleStr(dead)), c.Ref())

	// Reduce the population.
	c.Population -= dead
	if c.Population <= 0 {
		c.Population = 0
		return
	}

	// Log the disaster, what type, how many people died and where.
	year := m.Geo.Calendar.GetYear()
	log.Printf("Year %d: %s, %s died in %s", year, dis.Name, numPeopleStr(dead), c.Name)

	// Since there was a disaster, depending on the number of people that
	// died, some people might leave the city and move to other cities
	// or found a new settlement.
	if enableDisasterMigration && rand.Float64() < popLoss {
		leave := int(float64(c.Population) * (popLoss * rand.Float64()))
		m.relocateFromCity(c, leave)
	}
}

// getCityDisasters returns the disasters that may occur specifically in
// this city, depending on the type and size of the city.
func (m *Civ) getCityDisasters(c *City) []geo.Disaster {
	if c.Population == 0 {
		return nil // No disasters for deserted cities.
	}

	// Get the disasters that may occur in this city, depending on the
	// type and size of the city.
	//
	// TODO: Industry specific disasters
	// - coal mine: fire; mine: cave in, flooding (low elevation), etc.
	// - farms: drought, blight, etc.
	var ds []geo.Disaster
	switch c.Type {
	case CityTypeQuarry, CityTypeMining, CityTypeMiningGems:
		ds = append(ds, geo.DisRockslide, geo.DisCaveIn)
	case CityTypeDesertOasis:
		ds = append(ds, geo.DisSandstorm)
	}
	ds = append(ds, geo.DisDrought, geo.DisFamine)

	// With increasing population, the city is be more prone to famine or disease.
	// TODO: Improve this with some metrics like population density, sanitation, etc.
	if c.Population > 1000 {
		ds = append(ds, geo.DisDisease)
	}
	if c.Population > 10000 {
		ds = append(ds, geo.DisPlague)
	}
	return ds
}
