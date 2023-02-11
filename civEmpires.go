package genworldvoronoi

import (
	"container/heap"
	"fmt"
	"log"
	"sort"

	"github.com/Flokey82/go_gens/genlanguage"
)

func (m *Civ) GetEmpire(id int) *Empire {
	if m.RegionToEmpire[id] < 0 {
		return nil
	}
	for _, e := range m.Empires {
		if e.ID == m.RegionToEmpire[id] {
			return e
		}
	}
	return nil
}

func (m *Civ) regPlaceNEmpires(n int) {
	// NOTE: This is not very thought through.
	// This will need quite a bit of tweaking.
	//
	// Instead of assigning territories to regions,  we could instead just
	// keep track of which city states are part of which empire.
	// This would also be way less painful to modify later if, for example,
	// empires collapse, merge, or split.

	numEmpires := n
	if numEmpires > m.NumCityStates {
		numEmpires = m.NumCityStates
	}
	sortCities := make([]*City, m.NumCityStates)
	copy(sortCities, m.Cities)

	// TODO: Use city states with high expansionism and high score.
	sort.Slice(sortCities, func(i, j int) bool {
		return m.getCityScoreForexp(sortCities[i]) > m.getCityScoreForexp(sortCities[j])
	})

	// Truncate the list of cities to the number of empires we want to create.
	sortCities = sortCities[:numEmpires]
	// Start off with the city states with the highest expansionism score.
	for _, c := range sortCities {
		m.placeEmpireAt(c.ID, c)
	}
	m.expandEmpires()
}

func (m *Civ) expandEmpires() {
	var queue ascPriorityQueue
	heap.Init(&queue)

	terr := initRegionSlice(len(m.Cities))
	cityIDToIndex := make(map[int]int)
	cityIDToCity := make(map[int]*City)
	for i, c := range m.Cities {
		cityIDToIndex[c.ID] = i
		cityIDToCity[c.ID] = c
	}

	// Start off with the city states with the highest expansionism score.
	for _, c := range m.Empires {
		terr[cityIDToIndex[c.ID]] = c.ID
		cityScore := m.getCityScoreForMartial(c.Capital)
		for _, r := range m.getTerritoryNeighbors(c.ID, m.RegionToCityState) {
			newdist := m.getCityScoreForMartial(cityIDToCity[r])
			if newdist > cityScore {
				continue // We can't expand to a city with a higher score.
			}
			heap.Push(&queue, &queueEntry{
				score:       newdist,
				origin:      c.ID,
				destination: r,
			})
		}

		log.Printf("City %s has score %f", c.Name, c.Capital.Score)
	}

	// Extend territories until the queue is empty.
	for queue.Len() > 0 {
		u := heap.Pop(&queue).(*queueEntry)
		if terr[cityIDToIndex[u.destination]] >= 0 {
			continue
		}
		terr[cityIDToIndex[u.destination]] = u.origin
		for _, v := range m.getTerritoryNeighbors(u.destination, m.RegionToCityState) {
			if terr[cityIDToIndex[v]] >= 0 {
				continue
			}
			newdist := m.getCityScoreForMartial(cityIDToCity[v])
			cityScore := m.getCityScoreForMartial(cityIDToCity[u.origin])
			if newdist < 0 || newdist > cityScore {
				continue // We can't expand to a city with a higher score.
			}
			heap.Push(&queue, &queueEntry{
				score:       newdist + u.score,
				origin:      u.origin,
				destination: v,
			})
		}
	}

	// Now overwrite the territories with the new territories.
	// For this we will have to copy the city states and
	// set new territories.

	m.RegionToEmpire = initRegionSlice(m.mesh.numRegions)
	for i, t := range m.RegionToCityState {
		cIdx, ok := cityIDToIndex[t]
		if !ok {
			continue
		}
		if tn := terr[cIdx]; tn >= 0 {
			m.RegionToEmpire[i] = tn
		}
	}

	// Now update the empire territories.
	for _, e := range m.Empires {
		// Loop through all cities and gather all that
		// are within the current territory.
		for _, c := range m.Cities {
			if m.RegionToEmpire[c.ID] == e.ID {
				// TODO: If we just conquered this city, there might be a chance
				// we rename it using the primary language of the empire.
				e.Cities = append(e.Cities, c)
			}
		}

		// Collect all regions that are part of the
		// current territory.
		for r, terr := range m.RegionToEmpire {
			if terr == e.ID {
				e.Regions = append(e.Regions, r)
			}
		}
		e.Stats = m.getStats(e.Regions)
		e.Log()
	}
}

func (m *Civ) getCityScoreForexp(c *City) float64 {
	cc := m.GetCulture(c.ID)
	if cc == nil {
		// If there is no culture, we assume a base expansionism of 1.0.
		return c.Score * float64(len(c.Culture.Regions))
	}
	// Use the culture's expansionism as an indicator of
	// how much the city state wants to expand.
	return c.Score * cc.Expansionism * float64(len(c.Culture.Regions))
}

func (m *Civ) getCityScoreForMartial(c *City) float64 {
	cc := m.GetCulture(c.ID)
	if cc == nil {
		// If there is no culture, we assume a base martialism of 1.0.
		return c.Score * float64(len(c.Culture.Regions))
	}
	// Use the culture's martialism as an indicator of
	// how well the city can defend itself or its offensive
	// capabilities.
	return c.Score * cc.Martialism * float64(len(c.Culture.Regions))
}

// Empire contains information about a territory with the given ID.
// TODO: Maybe drop the regions since we can get that info
// relatively cheaply.
type Empire struct {
	ID       int     // Region where the empire originates (capital)
	Name     string  // Name of the empire
	Emperor  string  // Name of the ruler
	Capital  *City   // Capital city
	Cities   []*City // Cities within the territory
	Culture  *Culture
	Language *genlanguage.Language

	// TODO: DO NOT CACHE THIS!
	Regions []int // Regions that are part of the empire
	*Stats
}

func (e *Empire) String() string {
	return fmt.Sprintf("Empire %s", e.Name)
}

func (e *Empire) Log() {
	log.Printf("The Empire of %s: %d cities, %d regions, capital: %s", e.Name, len(e.Cities), len(e.Regions), e.Capital.Name)
	log.Printf("Emperor: %s", e.Emperor)
	e.Stats.Log()
}

func (m *Civ) placeEmpireAt(r int, c *City) *Empire {
	var lang *genlanguage.Language
	if c := m.GetCulture(c.ID); c != nil && c.Language != nil {
		lang = c.Language
	} else {
		lang = GenLanguage(m.Seed + int64(r))
	}
	e := &Empire{
		ID:       r,
		Name:     lang.MakeName(),
		Emperor:  lang.MakeFirstName() + " " + lang.MakeLastName(),
		Capital:  c,
		Culture:  c.Culture,
		Language: lang,
	}
	m.Empires = append(m.Empires, e)
	return e
}

func (m *Civ) GetEmpires() []*Empire {
	// TODO: Deduplicate with GetCityStates.
	return m.Empires
}
