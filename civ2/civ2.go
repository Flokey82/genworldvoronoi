package civ2

import (
	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/genworldvoronoi/geo"
)

const (
	HistoryEventConstruction = "construction"
	HistoryEventDisaster     = "disaster"
	HistoryEventLeadership   = "leadership"
	HistoryEventFounding     = "founding"
	HistoryEventTakeover     = "takeover"
	HistoryEventUprising     = "uprising"
	HistoryEventElection     = "election"
	HistoryEventFaction      = "faction"
	HistoryEventDiplomacy    = "diplomacy"
	HistoryEventMilitary     = "military"

	CurrencyCopper   = 0.01
	CurrencySilver   = 0.1
	CurrencyGold     = 1.0
	CurrencyPlatinum = 10.0
)

type Civ struct {
	*geo.Geo
	SoilExhaustion []float64
	arableLandFunc func(int) float64
	climateFunc    func(int) float64
	cultureFunc    func(int) civ.CultureType

	Settlements   *geo.SoloReferences[Settlement] // (political) Settlements
	Tribes        *geo.SoloReferences[Tribe]      // (political) Tribes
	Cities        *geo.SoloReferences[City]       // (political) City seed points / regions
	Empires       *geo.References[Empire]         // (political) Empires
	CityStates    *geo.References[CityState]      // (political) City states
	Cultures      *geo.References[Culture]        // (cultural) Cultures
	Religions     *geo.References[Religion]       // (cultural) Religion seed points / regions
	Population    []int
	Settled       []int64
	Suitability   []float64
	NumEmpires    int // Number of generated territories
	NumCities     int // Number of generated cities (regions)
	NumCityStates int // Number of generated city states
	navCache      *civ.NavCache
	nextPersonID   int
	nextFactionID  int
	nextArtifactID int
	People         []*Person // People in the world
	TradeRoutes   []*TradeRoute
	Relationships map[civ.ObjectReference]map[civ.ObjectReference]int
	disasterFunc  func(int) geo.GeoDisasterChance

	*civ.History
}

func NewCiv(g *geo.Geo) *Civ {
	c := &Civ{
		Geo:            g,
		SoilExhaustion: make([]float64, g.SphereMesh.NumRegions),
		Tribes:         geo.NewSoloReferences[Tribe](g.SphereMesh.NumRegions),
		Settlements:    geo.NewSoloReferences[Settlement](g.SphereMesh.NumRegions),
		Cities:         geo.NewSoloReferences[City](g.SphereMesh.NumRegions),
		Empires:        geo.NewReferences[Empire](g.SphereMesh.NumRegions),
		CityStates:     geo.NewReferences[CityState](g.SphereMesh.NumRegions),
		Cultures:       geo.NewReferences[Culture](g.SphereMesh.NumRegions),
		Religions:      geo.NewReferences[Religion](g.SphereMesh.NumRegions),
		Population:     make([]int, g.SphereMesh.NumRegions),
		Settled:        make([]int64, g.SphereMesh.NumRegions),
		Suitability:    make([]float64, g.SphereMesh.NumRegions),
		History:        civ.NewHistory(g.Calendar),
		TradeRoutes:    []*TradeRoute{},
		Relationships:  make(map[civ.ObjectReference]map[civ.ObjectReference]int),
		disasterFunc:   g.GetGeoDisasterFunc(),
	}
	c.navCache = civ.NewNavCache(g, func(from, to *civ.NavTile, cost float64) (float64, bool) {
		if c.Tribes.GetIDAt(to.ID) != -1 {
			return 0, false
		}
		return cost, true
	})
	return c
}

func (m *Civ) GenerateCivilization() {
	// HACK: Initialize the fitness functions.
	// We need to do that since the cached values are not available yet when NewCiv is called.
	m.arableLandFunc = m.GetFitnessArableLand()
	m.climateFunc = m.GetFitnessClimate()
	m.cultureFunc = civ.GetRegionCultureTypeFunc(m.Geo)

	const numTribes = 10

	useBiomes := false

	// Find the best places for the cradle of civilization.
	// Since we only have one species for now (humans), we will just start
	// with a 'steppe' region, and then expand from there incrementally.
	// Now we pick a suitable region to start with (steppe/grassland).
	bestRegions := m.pickNCradlesOfCivilization(numTribes, useBiomes)

	// Initial population.
	const initialPopulation = 100

	for _, bestRegion := range bestRegions {
		// Start with one tribe.
		m.NewTribe(bestRegion, initialPopulation)
	}
}

func (m *Civ) Tick() {
	nDays := 365

	// Update the tribes.
	m.tickTribes(nDays)

	// Update the settlements.
	m.tickSettlements(nDays)

	// Update the cities.
	m.tickCities(nDays)

	// Update the city states.
	m.tickCityStates(nDays)

	// Update the empires.
	m.tickEmpires(nDays)

	// Update religions.
	m.tickReligions(nDays)

	// Update cultures.
	m.tickCultures(nDays)

	// Update people.
	m.tickPeople(nDays)

	// Update trade.
	m.tickTrade(nDays)

	// Sync the global population map.
	m.syncPopulation()
}

func (m *Civ) syncPopulation() {
	for i := range m.Population {
		m.Population[i] = 0
	}
	for _, t := range m.Tribes.Objects {
		m.Population[t.RegionID] += t.Population
	}
	for _, s := range m.Settlements.Objects {
		m.Population[s.ID] += s.Population
	}
	for _, c := range m.Cities.Objects {
		m.Population[c.ID] += c.Population
	}
}

func (m *Civ) getNextPersonID() int {
	m.nextPersonID++
	return m.nextPersonID
}

func (m *Civ) getNextFactionID() int {
	m.nextFactionID++
	return m.nextFactionID
}

func (m *Civ) getNextArtifactID() int {
	m.nextArtifactID++
	return m.nextArtifactID
}

// getTerritoryNeighbors returns a list of territories neighboring the
// territory with the ID 'terrID' based on the provided slice of len NumRegions
// which maps the index (region id) to their respective territory ID.
// 'regions' is the list of regions controlled by the territory.
func (m *Civ) getTerritoryNeighbors(terrID int, regions []int, r_terr []int) []int {
	var res []int
	seenTerritories := make(map[int]bool)
	outReg := make([]int, 0, 8)
	for _, r := range regions {
		for _, nb := range m.SphereMesh.R_circulate_r(outReg, r) {
			// Determine territory ID.
			nbTerrID := r_terr[nb]
			if nbTerrID < 0 || nbTerrID == terrID || seenTerritories[nbTerrID] {
				continue
			}
			seenTerritories[nbTerrID] = true
			res = append(res, nbTerrID)
		}
	}
	return res
}

func (m *Civ) hasResourceAny(r int, rType geo.ResourceType) bool {
	return m.ResourceLocations.HasType(r, rType)
}
