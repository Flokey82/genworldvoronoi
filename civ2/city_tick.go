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

		// Collect resources.
		// m.harvestResources(c.Storage, c.ID, nDays)
		// Calculate what resources we need to consume.
		// If we have a deficit, we need to import it, so we need to find a city that produces it.

		// Do we produce trade goods upfront or do we take
		// orders and produce them for the next tick?

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

func (m *Civ) foundCityState(c *City, rNbs []int) {
	// We might create our own city state.
	// Conditions:
	// - Population > threshold
	// - At least one neighbour that is not part of a city state
	//   and has a settlement (TODO: Diplomacy, cities as well)
	hasNbSettlement := make([]int, 0, 8)
	for _, nb := range m.R_circulate_r(rNbs, c.ID) {
		if s := m.Settlements.GetAt(nb); s != nil && m.CityStates.GetAt(nb) == nil {
			hasNbSettlement = append(hasNbSettlement, nb)
		}
	}
	if len(hasNbSettlement) > 0 {
		cs := &CityState{
			ID:      c.ID,
			Capital: c,
			Culture: c.Culture,
			Founded: m.Geo.Calendar.GetYear(),
		}
		m.CityStates.PlaceObjectAt(cs, cs.ID)
		for _, nb := range hasNbSettlement {
			m.CityStates.PlaceObjectAt(cs, nb)
		}
		log.Printf("City %d has become a city state", c.ID)
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
