package civ2

import (
	"log"
	"math"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/civ"
)

func (m *Civ) tickSettlements(nDays int) {
	for _, s := range m.Settlements.Objects {
		if s.Population == 0 {
			// Remove culture from the region.
			// TODO: Slowly decay the settlement.
			m.Cultures.SetIDAt(s.ID, -1)
			continue
		}
		m.Cultures.SetIDAt(s.ID, s.Culture.ID)

		// Grow the population.
		s.Grow(nDays)

		// Log the resources at the location.
		log.Printf("Settlement %d at %d", s.ID, s.ID)
		for _, r := range m.Resources {
			if m.ResourceLocations.Location[r][s.ID] {
				log.Printf("Resource %s at %d", r.Name, s.ID)
			}
		}

		// Centralized economic tick.
		m.tickEconomyBase(s, nDays)

		// Tick diplomatic relations.
		m.tickDiplomacyBase(s, nDays)

		// Tick leadership.
		m.tickLeadership(s, nDays)

		// Handle trade discovery periodically.
		if rand.Intn(100) < 5 {
			// Find a nearby city or settlement to trade with.
			for _, oc := range m.Cities.Objects {
				if oc.ID != s.ID {
					m.EstablishTradeRoute(s, oc)
					break // Only one attempt per tick
				}
			}
		}

		// There is a chance that the settlement will be abandoned.
		// A 0.2% chance per year (at nDays=365).
		if rand.Float64() < (1.0 - math.Pow(1.0-0.002/365.0, float64(nDays))) {
			t := s.ToTribe(s.Population)
			m.Tribes.PlaceObjectAt(t, t.RegionID)
			t.AddRegion(t.RegionID)
			log.Printf("Settlement %d has been abandoned and tribe %d has been created", s.ID, t.ID)

			// Rename the settlement.
			s.Name += " (abandoned)"
			s.RemoveRegion(s.ID)
		}

		// If we are large and wealthy enough, we will transform into a city.
		if s.Population > 1000 {
			m.foundCity(s)
		}
	}
}


func (m *Civ) foundCity(s *Settlement) {
	c := &City{
		BaseEntity: BaseEntity{
			ID:              s.ID,
			Name:            s.Name,
			Population:      s.Population,
			Culture:         s.Culture,
			Type:            civ.ObjectTypeCity,
			Storage:         s.Storage,
			GoverningPeople: newGoverningPeople(),
			Infrastructure:    s.Infrastructure,
			ConstructionQueue: s.ConstructionQueue,
			Military:          s.Military,
		},
		Founded: m.Geo.Calendar.GetYear(),
	}
	m.Cities.PlaceObjectAt(c, c.ID)
	c.AddRegion(c.ID)
	log.Printf("Settlement %d has become a city", s.ID)

	// Remove the settlement.
	m.Settlements.RemoveObject(s)
	s.RemoveRegion(s.ID)
}
