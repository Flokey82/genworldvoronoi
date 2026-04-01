package civ2

import (
	"log"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/civ"
)

func (m *Civ) tickCities(nDays int) {
	rNbs := make([]int, 0, 8)
	for _, c := range m.Cities.Objects {
		if c.Population == 0 {
			// Remove culture from the region.
			// TODO: Slowly decay the city.
			m.Cultures.SetIDAt(c.ID, -1)
			continue
		}
		m.Cultures.SetIDAt(c.ID, c.Culture.ID)

		// Every day happenings:
		// - Population growth / decline
		// - Collect resources
		// - Spend resources
		// - Produce goods
		// - Consume goods
		// - Trade with other

		// Grow the population.
		c.Grow(nDays)
		
		// Centralized economic tick.
		m.tickEconomyBase(c, nDays)

		// Handle trade discovery periodically.
		if rand.Intn(100) < 10 {
			// Find a nearby city to trade with.
			for _, oc := range m.Cities.Objects {
				if oc.ID != c.ID {
					m.EstablishTradeRoute(c, oc)
					break // Only one attempt per tick
				}
			}
		}

		// Tick diplomatic relations.
		m.tickDiplomacyBase(c, nDays)

		// Tick leadership.
		m.tickLeadership(c, nDays)

		// There is a chance that the city will be abandoned by a part of the population.
		if rand.Intn(2000)*nDays < 5 {
			t := cityToTribe(c, c.Population/10)
			m.Tribes.PlaceObjectAt(t, t.RegionID)
			log.Printf("City %d has been (partially) abandoned and tribe %d has been created", c.ID, t.ID)
		}

		// There is a chance we found a city state.
		// Check if the city is within a city state.
		const threshold = 10000
		if m.CityStates.GetAt(c.ID) == nil && c.Population > threshold {
			m.foundCityState(c, rNbs)
		}

		// At some point, we can create a settlement nearby as an outpost,
		// which will be supplied by the city and in turn returns resources.
		// We will look for something with max resources in range.
		if c.Population > 1000 && rand.Intn(100)*nDays < 10 {
			m.findColonyRegion(c, m.navCache)
		}
	}
}


func (m *Civ) findColonyRegion(c *City, navCache *civ.NavCache) bool {
	// Find a suitable region within a certain radius.
	// We'll do it brute force for now.
	maxRadius := rand.Float64() * 200.0 / unitDistToKm // km

	// Get the best region to settle in.
	bestRegion, ok := m.findSettleRegionFor(c.ID, c.Population/10, maxRadius)
	if !ok {
		return false
	}

	// Plan a path.
	// Create a tribe to settle.
	t := cityToTribe(c, c.Population/10)
	log.Printf("City %d has created tribe %d to settle in region %d", c.ID, t.ID, bestRegion)
	if m.navigateMultiStep(t, bestRegion, m.navCache) {
		// Log the resources at destination.
		for _, r := range m.Resources {
			if m.ResourceLocations.Location[r][bestRegion] {
				log.Printf("Resource %s at %d", r.Name, bestRegion)
			}
		}
		t.Settling = true
		t.Origin = c
		m.Tribes.PlaceObjectAt(t, c.ID)
		return true
	}
	return false
}
