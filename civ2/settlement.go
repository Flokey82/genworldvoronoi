package civ2

import (
	"math/rand"
)

type Settlement struct {
	ID         int
	Name       string
	Population int
	Culture    *Culture
	*Storage
}

// GetID returns the ID of the settlement.
func (s Settlement) GetID() int {
	return s.ID
}

// Grow the population.
func (s *Settlement) Grow(nDays int) {
	const growthRate = 0.005 // 0.5% growth per year.
	if growth := calcPopulationGrowth(s.Population, growthRate, nDays); growth > 1 {
		s.Population += int(growth)
	} else if rand.Float64() < growth {
		s.Population++
	}
}

// ToTribe causes the settlement to be (partially) abandoned and a new tribe to be created.
func (s *Settlement) ToTribe(n int) *Tribe {
	n = min(n, s.Population)
	t := &Tribe{
		ID:         nextTribeID(),
		RegionID:   s.ID,
		Population: n,
		Culture:    s.Culture, // Switch to nomadic?
		Storage:    s.Storage, // TODO: We can't take everything with us.
	}

	// Abandon the settlement.
	s.Storage = NewStorage()
	s.Population = max(0, s.Population-n)

	return t
}

// getSettlementsWithin returns all settlements within a certain distance of the given region.
func (m *Civ) getSettlementsWithin(regionID int, distance float64) []*Settlement {
	distanceUnitsphere := distance / unitDistToKm

	settlements := make([]*Settlement, 0, 8)
	for _, s := range m.Settlements.Objects {
		if m.GetDistance(regionID, s.ID) <= distanceUnitsphere {
			settlements = append(settlements, s)
		}
	}
	return settlements
}
