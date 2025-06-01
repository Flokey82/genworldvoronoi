package civ

import (
	"container/heap"
	"fmt"
	"log"
	"sort"

	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/go_gens/genempire"
	"github.com/Flokey82/go_gens/gengovernment"
	"github.com/Flokey82/go_gens/genlanguage"
	"github.com/Flokey82/go_gens/genstory"
)

func (m *Civ) GetEmpire(r int) *Empire {
	return m.Empires.GetAt(r)
}

func (m *Civ) PlaceNEmpires(n int) {
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

	// Copy all cities that are city state capitals.
	sortCities := make([]*City, m.NumCityStates)
	copy(sortCities, m.Cities.Objects)

	// Sort city states by high expansionism and high score. E.g. the city states that
	// want to expand the most and have the highest score will be the first to be placed.
	sort.Slice(sortCities, func(i, j int) bool {
		return m.getCityScoreForExpansion(sortCities[i]) > m.getCityScoreForExpansion(sortCities[j])
	})

	// Truncate the sorted list of cities to the number of empires we want to create.
	sortCities = sortCities[:numEmpires]

	// Start off with placing the empires (city states) with the highest expansionism score.
	for _, c := range sortCities {
		m.placeEmpireAt(c.ID, c, nil)
	}

	// Now expand the empires.
	m.expandEmpires()
}

// Empire contains information about a territory with the given ID.
// TODO: Maybe drop the regions since we can get that info
// relatively cheaply.
type Empire struct {
	ID               int      // Region where the empire originates (capital)
	Name             string   // Name of the empire
	Capital          *City    // Capital city
	Cities           []*City  // Cities within the territory
	Culture          *Culture // Primary culture of the empire
	*GoverningPeople          // People governing the empire
	*ComboStorage             // Resources the empire has.

	// TODO: DO NOT CACHE THIS!
	Regions []int // Regions that are part of the empire
	*geo.Stats
}

func (m *Civ) placeEmpireAt(r int, c *City, leadership *Faction) *Empire {
	e := &Empire{
		ID:              r,
		Capital:         c,
		Culture:         c.Culture,
		GoverningPeople: newGoverningPeople(),
		ComboStorage:    newComboStorage(100000), // TODO: Storage should be upgradable.
	}

	// If there is no known leadership, generate a new one.
	if leadership == nil {
		// Generate leadership.
		e.Leadership = genFaction(e, m, gengovernment.LeadershipFormChiefdom, nil)
	} else {
		e.Leadership = leadership
	}

	// Generate a name for the empire.
	e.Name = m.GenerateEmpireName(e)

	m.Empires.PlaceObjectAt(e, r)

	// Add a new event to the history.
	m.History.AddEvent("Founding (Empire)", fmt.Sprintf("The Empire of %s was founded", e.Name), c.Ref())
	return e
}

func (m *Civ) GenerateEmpireName(e *Empire) string {
	tokenPlace := genstory.TokenReplacement{
		Token:       genempire.TokenPlace,
		Replacement: e.Capital.Name,
	}

	tokenFoundingFigure := genstory.TokenReplacement{
		Token:       genempire.TokenFoundingFigure,
		Replacement: e.Leadership.Leader.LastName,
	}

	tokenRandom := genstory.TokenReplacement{
		Token:       genempire.TokenRandom,
		Replacement: e.Culture.Language.MakeName(),
	}

	tokens := []genstory.TokenReplacement{tokenPlace, tokenFoundingFigure, tokenRandom}

	// Generate the name and method.
	g := genempire.NewGenerator(int64(e.ID), e.Culture.Language)
	gen, err := g.GenEmpireName(tokens)
	if err != nil {
		log.Printf("Error generating empire name: %v", err)
		return ""
	}
	return gen.Text
}

func (e Empire) GetID() int {
	return e.ID
}

// Ref returns the object reference of the empire.
func (e *Empire) Ref() ObjectReference {
	return ObjectReference{
		ID:   e.ID,
		Type: ObjectTypeEmpire,
	}
}

func (e *Empire) String() string {
	return fmt.Sprintf("Empire %s", e.Name)
}

func (e *Empire) Log() {
	log.Printf("%s: %d cities, %d regions, capital: %s", e.Name, len(e.Cities), len(e.Regions), e.Capital.Name)
	if e.Leadership != nil && e.Leadership.Leader != nil {
		log.Printf("Leader: %s", e.Leadership.Leader.Name())
	} else {
		log.Printf("Leader: Ungoverened")
	}
	e.Stats.Log()
}

func (e *Empire) getLanguage() *genlanguage.Language {
	return e.Culture.Language
}

func (e *Empire) compare(other *Empire) float64 {
	if e == other {
		return 1.0
	}
	if e == nil || other == nil {
		return -1.0
	}

	capitalValue := e.Capital.compare(other.Capital)
	cultureValue := e.Culture.compare(other.Culture)

	return (capitalValue + cultureValue) / 2
}

func (e *Empire) compareToCity(other *City) float64 {
	if e == nil || other == nil {
		return -1.0
	}

	capitalValue := e.Capital.compare(other)
	cultureValue := e.Culture.compare(other.Culture)

	return (capitalValue + cultureValue) / 2
}

func (e *Empire) compareToCityState(cs *CityState) float64 {
	if e == nil || cs == nil {
		return -1.0
	}

	capitalValue := e.Capital.compare(cs.Capital)
	cultureValue := e.Culture.compare(cs.Culture)

	return (capitalValue + cultureValue) / 2
}

// getGovernedPeople returns the people governed by the Empire.
func (e *Empire) getGovernedPeople() *GoverningPeople {
	return e.GoverningPeople
}

// NewRandomPerson returns a new random person which is part of the empire (capital).
func (e *Empire) NewRandomPerson(m *Civ, gender geneticshuman.Gender) *Person {
	return e.Capital.NewRandomPerson(m, gender)
}

// NewRandomChild returns a new random child which is part of the empire (capital) and
// is a child of the given person.
func (e *Empire) NewRandomChild(m *Civ, gender geneticshuman.Gender, parent *Person) *Person {
	return e.Capital.NewRandomChild(m, gender, parent)
}

// GetPeople returns the people living in the empire (capital).
func (e *Empire) GetPeople() []*Person {
	return e.Capital.GetPeople()
}

// GetGoverningPeople returns the governing people of the empire.
func (e *Empire) GetGoverningPeople() *GoverningPeople {
	return e.GoverningPeople
}

func (m *Civ) expandEmpires() {
	// Empires do not expand region by region, but rather occupy entire city states.
	var queue geo.AscPriorityQueue
	heap.Init(&queue)

	cityIndexToEmpireID := initRegionSlice(len(m.Cities.Objects))
	cityIDToIndex := make(map[int]int)
	cityIDToCity := make(map[int]*City)
	for i, c := range m.Cities.Objects {
		cityIDToIndex[c.ID] = i
		cityIDToCity[c.ID] = c
	}

	// Start with the city states that are the core of the empires.
	for _, empire := range m.Empires.Objects {
		cityIndexToEmpireID[cityIDToIndex[empire.ID]] = empire.ID

		// Get the martial score of the empire / city state we are expanding from.
		// and check if there are any neighbors that we can expand to.
		empireScore := m.getCityScoreForMartial(empire.Capital)
		for _, r := range m.getTerritoryNeighbors(empire.ID, m.CityStates.Regions) {
			// We only expand to city states that have a lower martial score than the empire.
			if destScore := m.getCityScoreForMartial(cityIDToCity[r]); destScore <= empireScore {
				heap.Push(&queue, &geo.QueueEntry{
					Score:       destScore,
					Origin:      empire.ID,
					Destination: r,
				})
			}
		}
		log.Printf("City %s has score %f", empire.Name, empire.Capital.Score)
	}

	// Extend territories until the queue is empty.
	for queue.Len() > 0 {
		u := heap.Pop(&queue).(*geo.QueueEntry)
		if cityIndexToEmpireID[cityIDToIndex[u.Destination]] >= 0 {
			continue
		}
		cityIndexToEmpireID[cityIDToIndex[u.Destination]] = u.Origin

		// Get the martial score of the city state we are expanding from.
		originScore := m.getCityScoreForMartial(cityIDToCity[u.Origin])

		// Check if there are any neighbors that we can expand to.
		for _, v := range m.getTerritoryNeighbors(u.Destination, m.CityStates.Regions) {
			if cityIndexToEmpireID[cityIDToIndex[v]] >= 0 {
				continue
			}

			// Get the martial score of the city state we want to expand to.
			destScore := m.getCityScoreForMartial(cityIDToCity[v])

			// If the destination score is higher than the origin score, we can't expand
			// to this city state since they would resist our expansion successfully.
			if destScore >= 0 && destScore < originScore {
				heap.Push(&queue, &geo.QueueEntry{
					Score:       destScore + u.Score, // The further away, the higher the score, the lower the rank in the queue.
					Origin:      u.Origin,
					Destination: v,
				})
			}
		}
	}

	// Now overwrite the empire territories the territories of the city states they occupy.
	m.Empires.ResetRegions()
	for i, t := range m.CityStates.Regions {
		if cIdx, ok := cityIDToIndex[t]; ok {
			m.Empires.SetIDAt(i, cityIndexToEmpireID[cIdx])
		}
	}

	// Clear the cities and regions of the empires.
	for _, e := range m.Empires.Objects {
		e.Cities = e.Cities[:0]
		e.Regions = e.Regions[:0]
	}

	// Collect all cities that are part of each empire.
	for _, c := range m.Cities.Objects {
		if emp := m.Empires.GetAt(c.ID); emp != nil {
			emp.Cities = append(emp.Cities, c)
		}
	}

	// Collect all regions that are part of each empire.
	for r, terr := range m.Empires.Regions {
		if emp := m.Empires.Get(terr); emp != nil {
			emp.Regions = append(emp.Regions, r)
		}
	}

	// Now update the empire territories stats.
	for _, e := range m.Empires.Objects {
		e.Stats = m.GetStats(e.Regions)
		e.Log()
	}
}

// getEmpireCityStates returns all city states that are part of the
// given empire.
func (m *Civ) getEmpireCityStates(e *Empire) []*CityState {
	var cityStates []*CityState
	for _, c := range m.CityStates.Objects {
		if m.Empires.Regions[c.ID] == e.ID {
			cityStates = append(cityStates, c)
		}
	}
	return cityStates
}

// getEmpireNeighborIDs returns all empires that are neighbors of the given empire.
func (m *Civ) getEmpireNeighborIDs(c *Empire) []int {
	return m.getTerritoryNeighbors(c.ID, m.Empires.Regions)
}

// getEmpireNeighbors returns all neighboring empires of the given empire.
func (m *Civ) getEmpireNeighbors(e *Empire) []*Empire {
	var neighbors []*Empire
	for _, nbID := range m.getEmpireNeighborIDs(e) {
		if nb := m.GetEmpire(nbID); nb != nil {
			neighbors = append(neighbors, nb)
		} else {
			log.Printf("!!!%s has a neighboring empire with ID %d and it could not be found", e.String(), nbID)
		}
	}
	return neighbors
}

// getEmpireCityStateNeighbors returns all city states that are neighbors of the given empire.
// If onlyIndependent is true, city states that are part of another empire will not be returned.
func (m *Civ) getEmpireCityStateNeighbors(e *Empire, onlyIndependent bool) []*CityState {
	seen := make(map[int]bool)
	ours := m.getEmpireCityStates(e)
	var theirs []*CityState
	for _, c := range ours {
		seen[c.ID] = true
	}
	for _, c := range ours {
		for _, r := range m.getTerritoryNeighbors(c.ID, m.CityStates.Regions) {
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
