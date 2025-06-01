package civ

import (
	"log"
	"time"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genideas/genlandmarknames"
	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/go_gens/utils"
)

type Civ struct {
	*CivConfig
	*geo.Geo
	*History
	nextPersonID int
	People       []*Person                  // People in the world
	Cities       *geo.SoloReferences[City]  // (political) City seed points / regions
	Empires      *geo.References[Empire]    // (political) Empires
	CityStates   *geo.References[CityState] // (political) City states
	Cultures     *geo.References[Culture]   // (cultural) Cultures
	Religions    *geo.References[Religion]  // (cultural) Religion seed points / regions
	Settled      []int64                    // (cultural) Time of settlement per region
	// SettledBySpecies []int // (cultural) Which species settled the region first
	NameGen     *genlandmarknames.NameGenerators
	TradeRoutes [][]int

	// Temp population per region.
	Population     []int
	SoilExhaustion []float64
	Suitability    []float64
	PeopleToRegion *geo.PSlice[[]*Person]
}

func NewCiv(g *geo.Geo, cfg *CivConfig) *Civ {
	if cfg == nil {
		cfg = NewCivConfig()
	}
	c := &Civ{
		CivConfig:      cfg,
		Geo:            g,
		History:        NewHistory(g.Calendar),
		Cities:         geo.NewSoloReferences[City](g.SphereMesh.NumRegions),
		Empires:        geo.NewReferences[Empire](g.SphereMesh.NumRegions),
		CityStates:     geo.NewReferences[CityState](g.SphereMesh.NumRegions),
		Cultures:       geo.NewReferences[Culture](g.SphereMesh.NumRegions),
		Religions:      geo.NewReferences[Religion](g.SphereMesh.NumRegions),
		Settled:        initTimeSlice(g.SphereMesh.NumRegions),
		NameGen:        genlandmarknames.NewNameGenerators(g.Seed),
		Population:     initRegionSlice(g.SphereMesh.NumRegions),
		SoilExhaustion: make([]float64, g.SphereMesh.NumRegions),
		Suitability:    make([]float64, g.SphereMesh.NumRegions),
	}
	c.PeopleToRegion = geo.NewPSlice(g.NumRegions, c.getPeopleAtRegionsNoCache)
	return c
}

func (m *Civ) GenerateCivilization() {
	// TODO: The generation should happen somewhat like this...
	// 0. Calculate time of settlement per region through flood fill.
	// This will allow us to determine the founding date of the cities and
	// settlements.
	m.GenerateTimeOfSettlement()

	// 1. Generate (species and) cultures.
	// 2. Spread cultures.
	// 3. Generate settlements.
	// 4. Grow settlements.
	// 5. Create organized religions.
	// 6. Spread religions.
	// 7. Select capital cities.
	// 8. Generate city states.
	// 9. Generate empires.

	// Place cultures (and folk religions).
	start := time.Now()
	m.PlaceNCultures(m.NumCultures)
	log.Println("Done cultures in ", time.Since(start).String())

	// Place / expand folk religions.
	start = time.Now()
	m.PlaceNFolkReligions(m.NumCultures)
	log.Println("Done expanding religions in ", time.Since(start).String())

	// Place cities and territories in regions.
	// TODO: Smaller towns should be found in the vicinity of larger cities.
	start = time.Now()
	m.PlaceNCities(m.NumCities, CityTypeDefault)
	m.PlaceNCities(m.NumFarmingTowns, CityTypeFarming)
	m.PlaceNCities(m.NumDesertOasis, CityTypeDesertOasis)
	m.PlaceNCities(m.NumMiningTowns, CityTypeMining)
	m.PlaceNCities(m.NumMiningGemsTowns, CityTypeMiningGems)
	m.PlaceNCities(m.NumQuarryTowns, CityTypeQuarry)
	log.Println("Done cities in ", time.Since(start).String())

	start = time.Now()
	m.PlaceNCityStates(m.NumCityStates)
	log.Println("Done city states in ", time.Since(start).String())

	start = time.Now()
	m.PlaceNEmpires(m.NumEmpires)
	log.Println("Done empires in ", time.Since(start).String())

	// Once we have established the territories, we can add trade towns
	// (we need the territories for the trade routes).
	// We should probably establish the trade routes now, so we ensure
	// that the trade towns will still be placed on the nexus points
	// where trade routes meet.
	if m.NumTradingTowns > 0 {
		start = time.Now()
		m.PlaceNCities(m.NumTradingTowns, CityTypeTrading)
		log.Println("Done trade cities in ", time.Since(start).String())
	}

	m.Geo.Calendar.SetYear(utils.MaxArray(m.Settled))

	start = time.Now()
	m.calculateAgriculturalPotential(m.Cities.Objects)
	log.Println("Done calculating agricultural potential in ", time.Since(start).String())

	start = time.Now()
	m.calculateAttractiveness(m.Cities.Objects)
	log.Println("Done calculating attractiveness in ", time.Since(start).String())

	start = time.Now()
	m.calculateResourcePotential(m.Cities.Objects)
	log.Println("Done calculating resource potential in ", time.Since(start).String())

	start = time.Now()
	m.calculateEconomicPotential()
	log.Println("Done calculating economic potential in ", time.Since(start).String())

	// Age cities as they are founded, like good cheese.
	if m.EnableCityAging {
		start = time.Now()
		m.ageCities()
		log.Println("Done aging cities in ", time.Since(start).String())
	}

	// TODO: We should also introduce some kind of "aging" of city states or empires
	// to generate some history, including ticking diplomatic relations, wars, etc.

	// Organized religions.
	if m.EnableOrganizedReligions {
		m.PlaceNOrganizedReligions(m.NumOrganizedReligions)
		for _, r := range m.Religions.Objects {
			log.Println(r.String())
		}
	}
}

var rNbs []int = make([]int, 0, 6)

func (m *Civ) Tick() {
	m.PeopleToRegion.NeedsUpdate = true // Force update of people to region mapping.
	// Update cities.
	geoDisasterChanceFunc := m.Geo.GetGeoDisasterFunc()
	geoRegPropFunc := m.Geo.GetRegPropertyFunc()
	for _, c := range m.Cities.Objects {
		m.tickCityDays(c, geoDisasterChanceFunc, geoRegPropFunc, 365)
	}

	// Update attractiveness, agricultural potential, and resource potential
	m.calculateCitiesStats(m.Cities.Objects)

	// Recalculate economic potential.
	m.calculateEconomicPotential()

	// Tick city states and empires.
	for _, cs := range m.CityStates.Objects {
		m.tickDiplomacyCityState(cs)
	}
	for _, e := range m.Empires.Objects {
		m.tickDiplomacyEmpire(e)
	}

	// Advance year.
	m.Geo.Calendar.TickYear()
	log.Printf("Aged civilization to %d\n", m.Geo.Calendar.GetYear())
}

func (m *Civ) TickN(n int) {
	for i := 0; i < n; i++ {
		m.Tick()
	}
}

// getRegName attempts to generate a name for the given region.
func (m *Civ) getRegName(r int) string {
	switch m.GetRegWhittakerModBiome(r) {
	case genbiome.WhittakerModBiomeBorealForestTaiga,
		genbiome.WhittakerModBiomeTemperateRainforest,
		genbiome.WhittakerModBiomeTemperateSeasonalForest,
		genbiome.WhittakerModBiomeTropicalRainforest,
		genbiome.WhittakerModBiomeTropicalSeasonalForest:
		return m.NameGen.Forest.Generate(int64(r), r%2 == 0)
	case genbiome.WhittakerModBiomeHotSwamp,
		genbiome.WhittakerModBiomeWetlands:
		return m.NameGen.Swamp.Generate(int64(r), r%2 == 0)
	}
	return ""
}

// pickCradleOfCivilization returns the region that is most suitable
// for the cradle of civilization (of the given biome).
func (m *Civ) pickCradleOfCivilization(wantBiome int) int {
	// First we pick a "suitable" region where the cradle of civilization
	// will be located.
	// There are some theories where, if we put the origin of civilization
	// in a less suitable region, we will expand to more suitable regions.
	// See: https://forhinhexes.blogspot.com/2019/08/history-xvii-cradle-of-civilizations.html?m=1
	// I feel like this will only work for migration to the most suitable
	// regions, but we know that people will also migrate to less suitable
	// regions, if they have to, or if they are forced to, or if they
	// are just too stubborn to give up.
	// We will use the climate fitness function and filter by biome.
	bestRegion := -1
	bestFitness := 0.0
	fa := m.GetFitnessClimate()
	bf := m.GetRegWhittakerModBiomeFunc()
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		if bf(r) == wantBiome {
			fitness := fa(r)
			if fitness > bestFitness {
				bestFitness = fitness
				bestRegion = r
			}
		}
	}
	return bestRegion
}

// pickNCradlesOfCivilization returns the n most suitable regions
// for the cradle of civilization (of the given biome).
func (m *Civ) pickNCradlesOfCivilization(wantBiome, n int) []int {
	fa := m.GetFitnessClimate()
	bf := m.GetRegWhittakerModBiomeFunc()

	// Get elevation values.
	elevs := m.Elevation.GetValues()

	// The fitness function, returning a score from 0.0 to 1.0 for a given region.
	// Select the fitness function based on the city type.
	scoreFunc := func(r int) float64 {
		if elevs[r] > 0 && bf(r) == wantBiome {
			return fa(r)
		}
		return 0.0
	}

	// The distance seed point function, returning seed points/regions that we
	// want to be far away from.
	// For now we just maximize the distance to the other cradles of civilization.
	bestRegions := make([]int, 0, n)

	// Get the stop regions, i.e. regions that we don't want to place cradles in.
	stopRegions := make(map[int]bool)

	// Place n cradles of the given type.
	regDistanceC := m.AssignDistanceField(bestRegions, stopRegions)
	for i := 0; i < n; i++ {
		scores := m.CalcCityScoreWithDistanceField(scoreFunc, regDistanceC)

		// Find the region with the highest fitness score.
		bestScore := 0.0
		bestRegion := -1
		for r, s := range scores {
			if s > bestScore {
				bestScore = s
				bestRegion = r
			}
		}
		if bestRegion == -1 {
			panic("no suitable region found")
		}
		// Place a cradle at the region with the highest fitness score.
		bestRegions = append(bestRegions, bestRegion)

		// Update the distance field.
		regDistanceC = m.UpdateDistanceField(regDistanceC, bestRegions, stopRegions)
	}
	return bestRegions
}
