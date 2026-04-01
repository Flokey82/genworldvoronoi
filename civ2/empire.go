package civ2

import (
	// No imports needed for now
)

// Empire contains information about a territory with the given ID.
type Empire struct {
	BaseEntity
	Capital *City // Capital city
	Founded int64 // Year when the empire was founded
}


func (m *Civ) GetEmpire(id int) *Empire {
	return m.Empires.Get(id)
}


func (m *Civ) getEmpireNeighborIDs(c *Empire) []int {
	return m.getTerritoryNeighbors(c.ID, c.Regions, m.Empires.Regions)
}

func (m *Civ) getEmpireNeighbors(e *Empire) []*Empire {
	var neighbors []*Empire
	for _, nbID := range m.getEmpireNeighborIDs(e) {
		if nb := m.GetEmpire(nbID); nb != nil {
			neighbors = append(neighbors, nb)
		}
	}
	return neighbors
}

func (m *Civ) getEmpireCityStates(e *Empire) []*CityState {
	var res []*CityState
	seen := make(map[int]bool)
	for _, id := range e.Regions {
		if cs := m.CityStates.GetAt(id); cs != nil && !seen[cs.ID] {
			res = append(res, cs)
			seen[cs.ID] = true
		}
	}
	return res
}

func (m *Civ) getEmpireCityStateNeighbors(e *Empire, onlyIndependent bool) []*CityState {
	seen := make(map[int]bool)
	ours := m.getEmpireCityStates(e)
	var theirs []*CityState
	for _, c := range ours {
		seen[c.ID] = true
	}
	for _, c := range ours {
		for _, r := range m.getCityStateNeighborIDs(c) {
			if seen[r] {
				continue
			}
			seen[r] = true
			if !onlyIndependent || m.Empires.GetIDAt(r) == -1 {
				theirs = append(theirs, m.GetCityState(r))
			}
		}
	}
	return theirs
}

func (e *Empire) compare(m *Civ, other *Empire) float64 {
	if e == nil || other == nil {
		return -1.0
	}
	if e == other {
		return 1.0
	}

	cultureValue := e.Culture.compare(other.Culture)
	religionValue := m.GetReligion(e.Capital.ID).compare(m.GetReligion(other.Capital.ID))

	return (cultureValue + religionValue) / 2
}

func (e *Empire) compareToCity(m *Civ, other *City) float64 {
	if e == nil || other == nil {
		return -1.0
	}
	if e.Capital == other {
		return 1.0
	}

	cultureValue := e.Culture.compare(other.Culture)
	religionValue := m.GetReligion(e.Capital.ID).compare(m.GetReligion(other.ID))

	return (cultureValue + religionValue) / 2
}

func (e *Empire) compareToCityState(m *Civ, other *CityState) float64 {
	if e == nil || other == nil {
		return -1.0
	}

	cultureValue := e.Culture.compare(other.Culture)
	religionValue := m.GetReligion(e.Capital.ID).compare(m.GetReligion(other.Capital.ID))

	return (cultureValue + religionValue) / 2
}
