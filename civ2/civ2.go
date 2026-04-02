package civ2

import (
	"log"

	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/go_gens/genlanguage"
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
	*civ.CivConfig
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

func NewCiv(g *geo.Geo, cfg *civ.CivConfig) *Civ {
	if cfg == nil {
		cfg = civ.NewCivConfig()
	}
	c := &Civ{
		Geo:            g,
		CivConfig:      cfg,
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

	useBiomes := false

	// Find the best places for the cradle of civilization.
	// Since we only have one species for now (humans), we will just start
	// with a 'steppe' region, and then expand from there incrementally.
	// Now we pick a suitable region to start with (steppe/grassland).
	bestRegions := m.pickNCradlesOfCivilization(m.NumTribes, useBiomes)

	// Initial population.
	const initialPopulation = 100
	const initialCityPopulation = 1000
	const initialSettlementPopulation = 500

	// Wrap all seeding and floodfill logic in the Legacy toggle.
	if m.SeedEntities {
		// 0. Pre-seed cultures if requested.
		if m.NumCultures > 0 {
			m.placeNCultures(m.NumCultures)
			log.Printf("Placed %d cultures", len(m.Cultures.Objects))
		}


		if m.NumCities > 0 {
			cityRegions := m.pickNCradlesOfCivilization(m.NumCities, useBiomes)
			for _, r := range cityRegions {
				// Find a culture for the city or create a new one.
				culture := m.GetCulture(r)
				if culture == nil {
					lang := genlanguage.GenLanguage(int64(m.getNextFactionID()))
					culture = m.newCulture(r, lang, m.cultureFunc(r))
				}
				m.NewCity(r, initialCityPopulation, culture)
			}
			log.Printf("Placed %d cities", len(m.Cities.Objects))
		}

		// 1b. Pre-seed specialized cities if requested.
		m.placeSpecializedCities(initialCityPopulation)

		// 2. Pre-seed empires if requested.
		if m.NumEmpires > 0 {
			// Pick existing cities as capitals for empires.
			cities := m.Cities.Objects
			for i := 0; i < m.NumEmpires && i < len(cities); i++ {
				m.NewEmpire(cities[i])
			}
			log.Printf("Placed %d empires", len(m.Empires.Objects))
		}

		// 3. Pre-seed city states if requested.
		if m.NumCityStates > 0 {
			// Pick existing cities as capitals for city states.
			cities := m.Cities.Objects
			placed := 0
			for _, c := range cities {
				if placed >= m.NumCityStates {
					break
				}
				if m.Empires.GetIDAt(c.ID) == -1 {
					m.NewCityState(c)
					placed++
				}
			}
			log.Printf("Placed %d city states", len(m.CityStates.Objects))
		}

		// 3b. Pre-seed organized religions if requested.
		if m.EnableOrganizedReligions && m.NumOrganizedReligions > 0 {
			m.placeNOrganizedReligions(m.NumOrganizedReligions)
			log.Printf("Placed %d organized religions", len(m.Religions.Objects))
		}

		// --- Legacy Mode: Flood fill all pre-seeded entities! ---
		log.Printf("Legacy Seeding Mode: Flood-filling territories...")
		
		// Map cultures across regions immediately.
		seeds := make([]int, 0, len(m.Cultures.Objects))
		for _, c := range m.Cultures.Objects {
			seeds = append(seeds, c.ID)
		}
		m.expandCultures(seeds)

		// Iteratively expand city states out to neighbors limit times
		rNbs := make([]int, 0, 10)
		for _, cs := range m.CityStates.Objects {
			m.legacyExpandCityState(cs, rNbs)
		}

		// Force Empire expansion out via sweeping up City States
		m.legacyExpandEmpires()
		log.Printf("Legacy Seeding Mode: Flood-fill completed.")
	}

	// 4. Start with some nomadic tribes.
	bestRegions = m.pickNCradlesOfCivilization(m.NumTribes, useBiomes)
	for _, bestRegion := range bestRegions {
		if m.Cities.GetAt(bestRegion) != nil || m.Settlements.GetAt(bestRegion) != nil {
			continue // Already a city or settlement.
		}
		// Start with one tribe.
		m.NewTribe(bestRegion, initialPopulation)
	}
	log.Printf("Placed %d tribes", len(m.Tribes.Objects))

	// 4. Tick the simulation 200 years to allow the world to evolve organically (Cradle Mode only).
	if !m.SeedEntities {
		log.Printf("Cradle Mode: Ticking simulation 200 years for organic growth...")
		for i := 0; i < 200; i++ {
			m.Tick()
			m.Geo.Calendar.TickYear()
		}
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
