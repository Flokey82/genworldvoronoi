package civ

import (
	"fmt"
	"log"

	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/go_gens/gengovernment"
	"github.com/Flokey82/go_gens/genlanguage"
)

func (m *Civ) GetCityState(r int) *CityState {
	return m.CityStates.GetAt(r)
}

func (m *Civ) PlaceNCityStates(n int) {
	m.ResetRand()
	for i, c := range m.Cities.Objects {
		if i >= n {
			break
		}
		m.PlaceCityStateAt(c.ID, c, nil)
		log.Printf("CityState %d: %s", i, c.Name)
	}
	m.ExpandCityStates()
}

// CityState represents a territory governed by a single city.
type CityState struct {
	ID               int      // Region where the city state originates
	Capital          *City    // Capital city
	Culture          *Culture // Culture of the city state
	Cities           []*City  // Cities within the city state
	Founded          int64    // Year when the city state was founded
	*GoverningPeople          // People governing the city state

	// TODO: DO NOT CACHE THIS!
	Regions []int
	*geo.Stats
	*ComboStorage // Resources the city state has.
}

func (m *Civ) PlaceCityStateAt(r int, c *City, leadership *Faction) *CityState {
	cs := &CityState{
		ID:              r,
		Capital:         c,
		Culture:         m.GetCulture(r),
		Founded:         c.Founded,            // TODO: Use current year.
		Cities:          []*City{c},           // TODO: ??? Remove this?
		Regions:         []int{r},             // TODO: ??? Remove this?
		Stats:           m.GetStats([]int{r}), // TODO: ??? Remove this?
		GoverningPeople: newGoverningPeople(),
		ComboStorage:    newComboStorage(10000), // TODO: Storage should be upgradable.
	}

	// If there is no known culture, generate a new one.
	if cs.Culture == nil {
		cs.Culture = m.PlaceCultureAt(r, false, nil) // TODO: Grow this culture.
	}
	// TODO: Name? Language?

	// If there is no known leadership, generate a new one.
	if leadership == nil {
		// Randomly generate a new leadership.
		// Pick a random leadership form.
		log.Printf("Generating leadership for %s", c.Name)

		// Generate leadership.
		cs.Leadership = genFaction(cs, m, gengovernment.LeadershipFormCouncil, nil)
	} else {
		cs.Leadership = leadership
	}

	m.CityStates.PlaceObjectAt(cs, r)
	m.CityStates.Regions[r] = -1 // FIXME: This is a hack, because expansion won't work otherwise.

	// Add a new event to the history.
	m.History.AddEvent("Founding (City State)", fmt.Sprintf("City state of %s was founded", cs.Capital.Name), c.Ref())
	return cs
}

func (c CityState) GetID() int {
	return c.ID
}

// Ref returns the object reference of the city state.
func (c *CityState) Ref() ObjectReference {
	return ObjectReference{
		ID:   c.ID,
		Type: ObjectTypeCityState,
	}
}

func (c *CityState) String() string {
	return fmt.Sprintf("CityState %s: %d cities, %d regions", c.Capital.Name, len(c.Cities), len(c.Regions))
}

func (c *CityState) Log() {
	log.Printf("The city state of %s: %d cities, %d regions", c.Capital.Name, len(c.Cities), len(c.Regions))
	c.Stats.Log()
}

func (c *CityState) getLanguage() *genlanguage.Language {
	return c.Culture.Language
}

func (c *CityState) compare(other *CityState) float64 {
	if c == other {
		return 1.0
	}
	if c == nil || other == nil {
		return -1.0
	}

	capitalValue := c.Capital.compare(other.Capital)
	cultureValue := c.Culture.compare(other.Culture)

	return (capitalValue + cultureValue) / 2
}

func (c *CityState) compareToCity(other *City) float64 {
	if c == nil || other == nil {
		return -1.0
	}

	capitalValue := c.Capital.compare(other)
	cultureValue := c.Culture.compare(other.Culture)

	return (capitalValue + cultureValue) / 2
}

func (c *CityState) compareToEmpire(e *Empire) float64 {
	if c == nil || e == nil {
		return -1.0
	}

	capitalValue := c.Capital.compare(e.Capital)
	cultureValue := c.Culture.compare(e.Culture)

	return (capitalValue + cultureValue) / 2
}

// getGovernedPeople returns the people governed by the city state (capital).
func (c *CityState) getGovernedPeople() *GoverningPeople {
	return c.GoverningPeople
}

// NewRandomPerson returns a new random person which is part of the city state (capital).
func (c *CityState) NewRandomPerson(m *Civ, gender geneticshuman.Gender) *Person {
	return c.Capital.NewRandomPerson(m, gender)
}

// NewRandomChild returns a new random child which is part of the city state (capital) and
// is a child of the given person.
func (c *CityState) NewRandomChild(m *Civ, gender geneticshuman.Gender, parent *Person) *Person {
	return c.Capital.NewRandomChild(m, gender, parent)
}

// GetPeople returns the people living in the city state.
func (c *CityState) GetPeople() []*Person {
	return c.Capital.People // fix this
}

// GetGoverningPeople returns the governing people of the city state.
func (c *CityState) GetGoverningPeople() *GoverningPeople {
	return c.GoverningPeople
}

func (c *CityState) GetPopulation() int {
	var pop int
	for _, city := range c.Cities {
		pop += city.Population
	}
	return pop
}

// ExpandCityStates expands the city states on the map based on their culture,
// terrain preference, and other factors.
func (m *Civ) ExpandCityStates() {
	// Territories are based on cities acting as their capital.
	// Since the algorithm places the cities with the highes scores
	// first, we use the top 'n' cities as the capitals for the
	// territories.
	seedCities := make([]int, 0, len(m.CityStates.Objects))
	for _, c := range m.CityStates.Objects {
		seedCities = append(seedCities, c.ID)
	}
	m.expandCityStates(false, seedCities)
}

func (m *Civ) expandCityStates(aggressive bool, seedCities []int) {
	weight := m.getTerritoryWeightFunc()
	biomeWeight := m.getTerritoryBiomeWeightFunc()
	cultureWeight := m.getTerritoryCultureWeightFunc()

	placeFunc := m.regPlaceNTerritoriesCustom
	if aggressive {
		placeFunc = m.expandTerritoriesAggressive
	}

	m.CityStates.Regions = placeFunc(m.CityStates.Regions, seedCities, func(o, u, v int) float64 {
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
	for _, c := range m.CityStates.Objects {
		// Loop through all cities and gather all that are within the current city state.
		c.Cities = c.Cities[:0]
		for _, ct := range m.Cities.Objects {
			if m.CityStates.Regions[ct.ID] == c.ID {
				c.Cities = append(c.Cities, ct)
			}
		}

		// Collect all regions that are part of the current territory.
		c.Regions = c.Regions[:0]
		for r, terr := range m.CityStates.Regions {
			if terr == c.ID {
				c.Regions = append(c.Regions, r)
			}
		}
		c.Stats = m.GetStats(c.Regions)
		c.Log()
	}
}

// getCityStateNeighborIDs returns all IDs of city states that are neighbors of the
// given city state.
func (m *Civ) getCityStateNeighborIDs(c *CityState) []int {
	return m.getTerritoryNeighbors(c.ID, m.CityStates.Regions)
}

// getCityStateNeighbors returns all neighboring city states of the given city state.
func (m *Civ) getCityStateNeighbors(c *CityState) []*CityState {
	var neighbors []*CityState
	for _, nbID := range m.getCityStateNeighborIDs(c) {
		if nb := m.GetCityState(nbID); nb != nil {
			neighbors = append(neighbors, nb)
		} else {
			log.Printf("!!!%s has a neighboring city state with ID %d and it could not be found", c.String(), nbID)
		}
	}
	return neighbors
}
