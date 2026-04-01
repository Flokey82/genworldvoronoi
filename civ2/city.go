package civ2

import (
	"github.com/Flokey82/genworldvoronoi/civ"
)

// City represents a city in the world.
type City struct {
	BaseEntity
	MaxPopulation int   // Maximum population of the city
	Founded       int64 // Year when the city was founded
}

// Grow the population.
func (c *City) Grow(nDays int) {
	const growthRate = 0.005 // 0.5% growth per year.
	c.BaseEntity.Grow(nDays, growthRate)
}

func (m *Civ) GetCity(id int) *City {
	return m.Cities.Get(id)
}

func cityToTribe(c *City, n int) *Tribe {
	n = min(n, c.Population)
	t := &Tribe{
		BaseEntity: BaseEntity{
			ID:              nextTribeID(),
			Population:      n,
			Culture:         c.Culture,
			Type:            civ.ObjectTypeTribe,
			Storage:         c.Storage,
			GoverningPeople: c.GoverningPeople,
			Infrastructure:    NewInfrastructure(),
			ConstructionQueue: NewConstructionQueue(),
			Military:          c.Military,
		},
		RegionID: c.ID,
	}
	t.Storage = NewStorage()

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


func (c *City) compare(m *Civ, other *City) float64 {
	if c == nil || other == nil {
		return -1.0
	}
	if c == other {
		return 1.0
	}

	cultureValue := c.Culture.compare(other.Culture)
	religionValue := m.GetReligion(c.ID).compare(m.GetReligion(other.ID))

	return (cultureValue + religionValue) / 2
}
