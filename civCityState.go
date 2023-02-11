package genworldvoronoi

import (
	"log"
)

func (m *Civ) GetCityState(id int) *CityState {
	if m.RegionToCityState[id] < 0 {
		return nil
	}
	for _, cs := range m.CityStates {
		if cs.ID == m.RegionToCityState[id] {
			return cs
		}
	}
	return nil
}

func (m *Civ) rPlaceNCityStates(n int) {
	m.resetRand()
	for i, c := range m.Cities {
		if i >= n {
			break
		}
		m.PlaceCityStateAt(c.ID, c)
		log.Printf("CityState %d: %s", i, c.Name)
	}
	m.expandCityStates()
}

// CityState represents a territory governed by a single city.
type CityState struct {
	ID      int      // Region where the city state originates
	Capital *City    // Capital city
	Culture *Culture // Culture of the city state
	Cities  []*City  // Cities within the city state
	Founded int64    // Year when the city state was founded

	// TODO: DO NOT CACHE THIS!
	Regions []int
	*Stats
}

func (c *CityState) Log() {
	log.Printf("The city state of %s: %d cities, %d regions", c.Capital.Name, len(c.Cities), len(c.Regions))
	c.Stats.Log()
}

func (m *Civ) PlaceCityStateAt(r int, c *City) *CityState {
	cs := &CityState{
		ID:      r,
		Capital: c,
		Culture: m.GetCulture(r),
		Founded: c.Founded,            // TODO: Use current year.
		Cities:  []*City{c},           // TODO: ??? Remove this?
		Regions: []int{r},             // TODO: ??? Remove this?
		Stats:   m.getStats([]int{r}), // TODO: ??? Remove this?
	}

	// If there is no known culture, generate a new one.
	if c.Culture == nil {
		c.Culture = m.PlaceCultureAt(r) // TODO: Grow this culture.
	}

	m.CityStates = append(m.CityStates, cs)
	// TODO: Name? Language?
	return cs
}

func (m *Civ) expandCityStates() {
	// Territories are based on cities acting as their capital.
	// Since the algorithm places the cities with the highes scores
	// first, we use the top 'n' cities as the capitals for the
	// territories.
	var seedCities []int
	for _, c := range m.CityStates {
		seedCities = append(seedCities, c.ID)
	}

	weight := m.getTerritoryWeightFunc()
	biomeWeight := m.getTerritoryBiomeWeightFunc()
	cultureWeight := m.getTerritoryCultureWeightFunc()

	m.RegionToCityState = m.regPlaceNTerritoriesCustom(seedCities, func(o, u, v int) float64 {
		// TODO: Make sure we take in account expansionism, wealth, score, and culture.
		w := weight(o, u, v)
		if w < 0 {
			return -1
		}
		b := biomeWeight(o, u, v)
		if b < 0 {
			return -1
		}
		c := cultureWeight(o, u, v)
		if c < 0 {
			return -1
		}
		return (w + b + c) / 3
	})

	// Before relaxing the territories, we'd need to ensure that we only
	// relax without changing the borders of the empire...
	// So we'd only re-assign IDs that belong to the same territory.
	// m.rRelaxTerritories(m.r_city, 5)

	// Update the city states with the new regions.
	for _, c := range m.CityStates {
		// Loop through all cities and gather all that
		// are within the current city state.
		c.Cities = nil
		for _, ct := range m.Cities {
			if m.RegionToCityState[ct.ID] == c.ID {
				c.Cities = append(c.Cities, ct)
			}
		}

		// Collect all regions that are part of the
		// current territory.
		c.Regions = nil
		for r, terr := range m.RegionToCityState {
			if terr == c.ID {
				c.Regions = append(c.Regions, r)
			}
		}
		c.Stats = m.getStats(c.Regions)
		c.Log()
	}
}

func (m *Civ) GetCityStates() []*CityState {
	// TODO: Deduplicate with GetEmpires.
	return m.CityStates
}

// getCityStateNeighbors returns all city states that are neighbors of the
// given city state.
func (m *Civ) getCityStateNeighbors(c *CityState) []int {
	return m.getTerritoryNeighbors(c.ID, m.RegionToCityState)
}
