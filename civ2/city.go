package civ2

import (
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/civ"
)

// City represents a city in the world.
type City struct {
	ID            int      // Region where the city is located
	Population    int      // Current population of the city
	MaxPopulation int      // Maximum population of the city
	Culture       *Culture // Culture of the city region
	Name          string   // Name of the city
	Founded       int64    // Year when the city was founded
	*Storage
}

// GetID returns the ID of the city.
func (c City) GetID() int {
	return c.ID
}

// Ref returns the object reference of the city.
func (c *City) Ref() civ.ObjectReference {
	return civ.ObjectReference{
		ID:   c.ID,
		Type: civ.ObjectTypeCity,
	}
}

// Grow the population.
func (c *City) Grow(nDays int) {
	const growthRate = 0.005 // 0.5% growth per year.
	if growth := calcPopulationGrowth(c.Population, growthRate, nDays); growth > 1 {
		c.Population += int(growth)
	} else if rand.Float64() < growth {
		c.Population++
	}
}

func (m *Civ) GetCity(id int) *City {
	return m.Cities.Get(id)
}

func cityToTribe(c *City, n int) *Tribe {
	n = min(n, c.Population)
	t := &Tribe{
		ID:         nextTribeID(),
		RegionID:   c.ID,
		Population: n,
		Culture:    c.Culture,
	}

	if n == c.Population {
		t.Storage = c.Storage
	} else {
		t.Storage = NewStorage()
	}

	// Abandon the city.
	c.Population = max(0, c.Population-n)

	return t
}

// getCitiesWithin returns cities within a certain distance of a region.
func (m *Civ) getCitiesWithin(r int, distance float64) []*City {
	distanceUnitsphere := distance / unitDistToKm

	cities := make([]*City, 0, 10)
	for _, c := range m.Cities.Objects {
		if m.GetDistance(r, c.ID) < distanceUnitsphere {
			cities = append(cities, c)
		}
	}
	return cities
}
