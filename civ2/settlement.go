package civ2

import (
	"github.com/Flokey82/genworldvoronoi/civ"
)

type Settlement struct {
	BaseEntity
}

// Grow the population.
func (s *Settlement) Grow(nDays int) {
	const growthRate = 0.005 // 0.5% growth per year.
	s.BaseEntity.Grow(nDays, growthRate)
}

// ToTribe causes the settlement to be (partially) abandoned and a new tribe to be created.
func (s *Settlement) ToTribe(n int) *Tribe {
	n = min(n, s.Population)
	t := &Tribe{
		BaseEntity: BaseEntity{
			ID:              nextTribeID(),
			Population:      n,
			Culture:         s.Culture, // Switch to nomadic?
			Type:            civ.ObjectTypeTribe,
			Storage:         s.Storage, // TODO: We can't take everything with us.
			GoverningPeople: s.GoverningPeople,
			Infrastructure:    NewInfrastructure(),
			ConstructionQueue: NewConstructionQueue(),
			Military:          s.Military,
		},
		RegionID: s.ID,
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

