package civ2

import "github.com/Flokey82/genworldvoronoi/civ"

// CityState represents a territory governed by a single city.
type CityState struct {
	ID      int      // Region where the city state originates
	Capital *City    // Capital city
	Culture *Culture // Culture of the city state
	Founded int64    // Year when the city state was founded
	*Storage
}

func (c CityState) GetID() int {
	return c.ID
}

// Ref returns the object reference of the city state.
func (c *CityState) Ref() civ.ObjectReference {
	return civ.ObjectReference{
		ID:   c.ID,
		Type: civ.ObjectTypeCityState,
	}
}

func (m *Civ) GetCityState(id int) *CityState {
	return m.CityStates.Get(id)
}
