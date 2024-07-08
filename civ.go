package genworldvoronoi

import (
	"container/heap"
	"log"
	"time"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genideas/genlandmarknames"
	"github.com/Flokey82/genworldvoronoi/geo"
)

type Civ struct {
	*CivConfig
	*geo.Geo
	*History
	altGen            bool
	nextPersonID      int
	People            []*Person    // People in the world
	Empires           []*Empire    // (political) Empires
	RegionToEmpire    []int        // (political) Point / region mapping to territory / empire
	CityStates        []*CityState // (political) City states
	RegionToCityState []int        // (political) Point / region mapping to city / city state
	Cities            []*City      // (political) City seed points / regions
	RegionToCulture   []int        // (cultural) Point / region mapping to culture
	Cultures          []*Culture   // (cultural) Culture seed points / regions
	RegionToReligion  []int        // (cultural) Point / region mapping to religion
	Religions         []*Religion  // (cultural) Religion seed points / regions
	Settled           []int64      // (cultural) Time of settlement per region
	// SettledBySpecies []int // (cultural) Which species settled the region first
	NameGen     *genlandmarknames.NameGenerators
	TradeRoutes [][]int

	// Temp population per region.
	Population     []int
	SoilExhaustion []float64
	Suitability    []float64
	Tribes         []*Tribe
}

func NewCiv(g *geo.Geo, cfg *CivConfig) *Civ {
	if cfg == nil {
		cfg = NewCivConfig()
	}
	return &Civ{
		CivConfig:         cfg,
		Geo:               g,
		History:           NewHistory(g.Calendar),
		altGen:            true,
		RegionToEmpire:    initRegionSlice(g.SphereMesh.NumRegions),
		RegionToCityState: initRegionSlice(g.SphereMesh.NumRegions),
		RegionToCulture:   initRegionSlice(g.SphereMesh.NumRegions),
		RegionToReligion:  initRegionSlice(g.SphereMesh.NumRegions),
		Settled:           initTimeSlice(g.SphereMesh.NumRegions),
		NameGen:           genlandmarknames.NewNameGenerators(g.Seed),
		Population:        initRegionSlice(g.SphereMesh.NumRegions),
		SoilExhaustion:    make([]float64, g.SphereMesh.NumRegions),
		Suitability:       make([]float64, g.SphereMesh.NumRegions),
	}
}

func (m *Civ) GenerateCivilization() {
	if m.altGen {
		// Do the settlement sim.
		m.InitSim()
	} else {
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
		m.PlaceNCities(m.NumCities, TownTypeDefault)
		m.PlaceNCities(m.NumFarmingTowns, TownTypeFarming)
		m.PlaceNCities(m.NumDesertOasis, TownTypeDesertOasis)
		m.PlaceNCities(m.NumMiningTowns, TownTypeMining)
		m.PlaceNCities(m.NumMiningGemsTowns, TownTypeMiningGems)
		m.PlaceNCities(m.NumQuarryTowns, TownTypeQuarry)
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
			m.PlaceNCities(m.NumTradingTowns, TownTypeTrading)
			log.Println("Done trade cities in ", time.Since(start).String())
		}

		_, maxSettled := minMax64(m.Settled)
		m.Geo.Calendar.SetYear(maxSettled)

		start = time.Now()
		m.calculateAgriculturalPotential(m.Cities)
		log.Println("Done calculating agricultural potential in ", time.Since(start).String())

		start = time.Now()
		m.calculateAttractiveness(m.Cities)
		log.Println("Done calculating attractiveness in ", time.Since(start).String())

		start = time.Now()
		m.calculateResourcePotential(m.Cities)
		log.Println("Done calculating resource potential in ", time.Since(start).String())

		start = time.Now()
		m.calculateEconomicPotential()
		log.Println("Done calculating economic potential in ", time.Since(start).String())

		// Age cities as they are founded, like good cheese.
		// TODO: We should also introduce some kind of "aging" of city states or empires
		// to generate some history.
		if m.EnableCityAging {
			start = time.Now()
			m.ageCities()
			log.Println("Done aging cities in ", time.Since(start).String())
		}

		// Organized religions.
		if m.EnableOrganizedReligions {
			m.PlaceNOrganizedReligions(m.NumOrganizedReligions)
			for _, r := range m.Religions {
				log.Println(r.String())
			}
		}
	}
}

var rNbs []int = make([]int, 0, 6)

func (m *Civ) Tick() {
	if m.altGen {
		// Do the settlement sim.
		m.TickSim()
	} else {
		// Update cities.
		//
		// TODO:
		// 1: Update production.
		// 2: Update consumption.
		// 3: Update trade.
		// 4: Update population. (births, deaths, migration)
		// 5: Found new settlements?
		geoDisasterChanceFunc := m.Geo.GetGeoDisasterFunc()
		cultureFunc := m.getCultureFunc()
		for _, c := range m.Cities {
			m.tickCityDays(c, geoDisasterChanceFunc, cultureFunc, 365)
		}
		// Update attractiveness, agricultural potential, and resource potential
		// TODO: Only update new regions until we have climate change?
		m.calculateCitiesStats(m.Cities)

		// Recalculate economic potential.
		m.calculateEconomicPotential()
		// Advance year.
		m.Geo.Calendar.TickYear()
		log.Printf("Aged cities to %d\n", m.Geo.Calendar.GetYear())

		// TODO:
		// Update city states.
		// 1: Update wealth / taxation.
		// 2: Update trade.
		// 3: Update politics.
		// (Alliances, wars, taxes, laws, etc.)
		// 4: Update population sentiment.
		// Update empires.
		// (Similar as city states.)
		// Update cultures.
		// 1: Expansion, assimilation, etc.
		// 2: Update culture sentiments.
		// Update religions.
		// (same as cultures)

		// NOTE: In theory we can partially de-duplicate code relating
		// to city states and empires, since they function similarly.
		// We can also de-duplicate cultures and religions.

		// TODO: We should also introduce some kind of "aging" or "ticking" of
		// city states or empires...
	}
}

func (m *Civ) TickN(n int) {
	for i := 0; i < n; i++ {
		m.Tick()
	}
}

// getRegName attempts to generate a name for the given region.
func (m *Civ) getRegName(r int) string {
	switch m.GetRegWhittakerModBiomeFunc()(r) {
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

func (m *Civ) GenerateTimeOfSettlement() {
	// First we pick a "suitable" region where the cradle of civilization
	// will be located.
	// Since we only have one species for now (humans), we will just start
	// with a 'steppe' region, and then expand from there incrementally.
	// Now we pick a suitable region to start with (steppe/grassland).
	bestRegion := m.pickCradleOfCivilization(genbiome.WhittakerModBiomeTemperateGrassland)
	if bestRegion == -1 {
		panic("no suitable region found")
	}

	// How long it takes for the civilization to expand to a region is
	// determined by the characteristics of the region and if there are
	// more suitable regions nearby. So we will use a priority queue
	// to determine the next region to expand to.
	var queue geo.AscPriorityQueue
	heap.Init(&queue)

	// 'settleTime' is the time when a region was settled.
	settleTime := initTimeSlice(m.SphereMesh.NumRegions)

	// We will start with a settlement time of 0.
	settleTime[bestRegion] = 0

	// terrainDifficulty returns high scores for difficult terrain.
	terrainDifficulty := m.getTerritoryWeightFunc()

	// terrainArable returns high scores if the terrain is arable.
	//terrainArable := m.getFitnessArableLand()

	weight := func(o, u, v int) float64 {
		// Terrain weight.
		// TODO: We should use a slightly different weight function
		// that doesn't treat up- and downhill differently.
		// Also, the penalty should be way higher for "impassable"
		// terrain.
		terrDifficulty := terrainDifficulty(bestRegion, u, v)

		// TODO: The duration that it takes to settle a region should
		// depend on how many regions there are in total (the size of
		// the regions).
		const (
			baseDifficultyTime = 2000 // Time to settle land per difficulty.

			// Time to cross land and sea.
			timeToCrossToLand = 1   // Crossing from sea to land.
			timeToCrossSea    = 20  // Crossing from sea to sea.
			timeToCrossToSea  = 200 // Crossing from land to sea (time to build a boat).
		)

		// If the terrain weight is positive (or zero), the destination region is land.
		if terrDifficulty >= 0 {
			// Settlement on land takes a fraction of 2000 years per (unit) region.
			// 'terrWeight' already takes the actual distance between the regions
			// into account.
			return float64(settleTime[u]) + baseDifficultyTime*terrDifficulty // * (1-terrainArable(v))
		}

		// If the terrain weight is negative, the source- and/or destination region is ocean.
		// This means, we need boats to get there, which will require more time.
		// TODO: For crossing the ocean, we need to wait for boats to be invented?
		var timeReqired float64
		if m.Elevation[v] > 0 {
			// If we were at sea and arrive at land, we only need a year to disembark.
			timeReqired = timeToCrossToLand
		} else if m.Elevation[v] <= 0 && m.Elevation[u] <= 0 {
			// Once we are traveling at sea, we travel at a speed of 20 years
			// per (unit) region.
			timeReqired = timeToCrossSea
		} else {
			// We were on land, but the destination is at sea,
			// it takes us 200 years to build a boat.
			timeReqired = timeToCrossToSea
		}

		// Calculate the actual distance between the two regions,
		// so we are independent of the mesh resolution.
		actualDist := m.GetDistance(u, v)
		return float64(settleTime[u]) + timeReqired*actualDist
	}

	// Now add the region neighbors to the queue.
	out_r := make([]int, 0, 8)
	for _, n := range m.R_circulate_r(out_r, bestRegion) {
		heap.Push(&queue, &geo.QueueEntry{
			Origin:      bestRegion,
			Score:       weight(bestRegion, bestRegion, n),
			Destination: n,
		})
	}

	// Expand settlements until we have settled all regions.
	for queue.Len() > 0 {
		u := heap.Pop(&queue).(*geo.QueueEntry)

		// Check if the region has already been settled.
		if settleTime[u.Destination] >= 0 {
			continue
		}

		// The higher the score, the more difficult it is to settle there,
		// and the longer it took to settle there.
		settleTime[u.Destination] = int64(u.Score)
		for _, v := range m.SphereMesh.R_circulate_r(out_r, u.Destination) {
			// Check if the region has already been settled.
			if settleTime[v] >= 0 {
				continue
			}
			newdist := weight(u.Origin, u.Destination, v)
			if newdist < 0 {
				continue
			}
			heap.Push(&queue, &geo.QueueEntry{
				Score:       newdist,
				Origin:      u.Destination,
				Destination: v,
			})
		}
	}

	m.Settled = settleTime
}
