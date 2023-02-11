package genworldvoronoi

import "log"

// CivStats contains statistics about civilization related aspects present given a list of
// regions. This will allow us for example to determine if the primary culture or religion
// of the empire is different from the capital, which might be destabilizing, etc.
type CivStats struct {
	Cultures   map[*Culture]int   // Number of regions per culture.
	Religions  map[*Religion]int  // Number of regions per religion.
	CityStates map[*CityState]int // Number of regions per city state.
	Empires    map[*Empire]int    // Number of regions per empire.
}

func (s *CivStats) Log() {
	log.Println("Cultures:")
	for c, n := range s.Cultures {
		log.Printf("  %s: %d regions", c.Name, n)
	}
	log.Println("Religions:")
	for r, n := range s.Religions {
		log.Printf("  %s: %d regions", r.Name, n)
	}
	log.Println("City states:")
	for cs, n := range s.CityStates {
		log.Printf("  %s: %d regions", cs.Capital.Name, n)
	}
	log.Println("Empires:")
	for e, n := range s.Empires {
		log.Printf("  %s: %d regions", e.Name, n)
	}
}

func (m *Civ) getCivStats(regions []int) *CivStats {
	res := &CivStats{
		Cultures:   make(map[*Culture]int),
		Religions:  make(map[*Religion]int),
		CityStates: make(map[*CityState]int),
		Empires:    make(map[*Empire]int),
	}
	for _, r := range regions {
		if c := m.GetCulture(r); c != nil {
			res.Cultures[c]++
		}
		if r := m.GetReligion(r); r != nil {
			res.Religions[r]++
		}
		if cs := m.GetCityState(r); cs != nil {
			res.CityStates[cs]++
		}
		if e := m.GetEmpire(r); e != nil {
			res.Empires[e]++
		}
	}
	return res
}
