package civ2

import (
	"log"
	"math/rand"
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

		// Collect resources.
		// m.harvestResources(s.Storage, s.ID, nDays)

		// Spend resources.

		// Find settlements within a certain distance.
		// We can trade with them.

		// Find cities within a certain distance.
		// We can associate with them to gain some benefits.

		// There is a chance that the settlement will be abandoned.
		// TODO: This should be triggered by a series of unfortunate events.
		if rand.Intn(500)*nDays < 5 {
			t := s.ToTribe(s.Population)
			m.Tribes.PlaceObjectAt(t, t.RegionID)
			log.Printf("Settlement %d has been abandoned and tribe %d has been created", s.ID, t.ID)

			// Rename the settlement.
			s.Name += " (abandoned)"
		}

		// If we are large and wealthy enough, we will transform into a city.
		if s.Population > 1000 {
			m.foundCity(s)
		}
	}
}

func (m *Civ) tickDiplomacySettlement(s *Settlement, nDays int) {
	// TODO: Implement diplomacy.
}

func (m *Civ) foundCity(s *Settlement) {
	c := &City{
		ID:         s.ID,
		Name:       s.Name,
		Population: s.Population,
		Culture:    s.Culture,
		Founded:    m.Geo.Calendar.GetYear(),
		Storage:    s.Storage,
	}
	m.Cities.PlaceObjectAt(c, c.ID)
	log.Printf("Settlement %d has become a city", s.ID)

	// Remove the settlement.
	m.Settlements.RemoveObject(s)
}
