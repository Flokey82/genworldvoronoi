package civ2

import (
	"log"

	"github.com/Flokey82/genworldvoronoi/civ"
)

// CityState represents a territory governed by a single city.
type CityState struct {
	BaseEntity
	Capital *City // Capital city
	Founded int64 // Year when the city state was founded
}
func (m *Civ) foundCityState(c *City, rNbs []int) {
	// ... logic to check neighbors ...
	hasNbSettlement := make([]int, 0, 8)
	for _, nb := range m.R_circulate_r(rNbs, c.ID) {
		if s := m.Settlements.GetAt(nb); s != nil && m.CityStates.GetAt(nb) == nil {
			hasNbSettlement = append(hasNbSettlement, nb)
		}
	}
	if len(hasNbSettlement) > 0 {
		cs := &CityState{
			BaseEntity: BaseEntity{
				ID:              c.ID,
				Name:            "City State of " + c.Name,
				Culture:         c.Culture,
				Type:            civ.ObjectTypeCityState,
				Storage:         NewStorage(),
				GoverningPeople: newGoverningPeople(),
				Infrastructure:    NewInfrastructure(),
				ConstructionQueue: NewConstructionQueue(),
				Military:          c.Military,
			},
			Capital: c,
			Founded: m.Geo.Calendar.GetYear(),
		}
		m.CityStates.PlaceObjectAt(cs, cs.ID)
		cs.AddRegion(cs.ID)
		for _, nb := range hasNbSettlement {
			m.CityStates.PlaceObjectAt(cs, nb)
			cs.AddRegion(nb)
		}
		log.Printf("City %d has become a city state", c.ID)
	}
}

func (m *Civ) GetCityState(id int) *CityState {
	return m.CityStates.Get(id)
}

func (m *Civ) getCityStateNeighborIDs(c *CityState) []int {
	return m.getTerritoryNeighbors(c.ID, c.Regions, m.CityStates.Regions)
}

func (m *Civ) getCityStateNeighbors(c *CityState) []*CityState {
	var neighbors []*CityState
	for _, nbID := range m.getCityStateNeighborIDs(c) {
		if nb := m.GetCityState(nbID); nb != nil {
			neighbors = append(neighbors, nb)
		}
	}
	return neighbors
}

func (cs *CityState) compare(m *Civ, other *CityState) float64 {
	if cs == nil || other == nil {
		return -1.0
	}
	if cs == other {
		return 1.0
	}

	cultureValue := cs.Culture.compare(other.Culture)
	religionValue := m.GetReligion(cs.Capital.ID).compare(m.GetReligion(other.Capital.ID))

	return (cultureValue + religionValue) / 2
}

func (cs *CityState) compareToCity(m *Civ, other *City) float64 {
	if cs == nil || other == nil {
		return -1.0
	}
	if cs.Capital == other {
		return 1.0
	}

	cultureValue := cs.Culture.compare(other.Culture)
	religionValue := m.GetReligion(cs.Capital.ID).compare(m.GetReligion(other.ID))

	return (cultureValue + religionValue) / 2
}

func (cs *CityState) compareToEmpire(m *Civ, other *Empire) float64 {
	if cs == nil || other == nil {
		return -1.0
	}

	cultureValue := cs.Culture.compare(other.Culture)
	religionValue := m.GetReligion(cs.Capital.ID).compare(m.GetReligion(other.Capital.ID))

	return (cultureValue + religionValue) / 2
}
