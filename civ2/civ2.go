package civ2

import (
	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/genworldvoronoi/geo"
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
		t := m.NewTribe(bestRegion, initialPopulation)
		m.Tribes.PlaceObjectAt(t, bestRegion)
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

	// Update the cultures.
	m.tickCultures(nDays)
}
