package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/Flokey82/genworldvoronoi"
	"github.com/Flokey82/genworldvoronoi/cmd2/maptiles"
	"github.com/hajimehoshi/ebiten/v2"
)

var (
	seed                    int64   = 12345
	numPlates               int     = 25
	oceanPlatesFraction     float64 = 0.65
	oceanPlatesAltSelection bool    = true
	numPoints               int     = 400000
	numVolcanoes            int     = 10
	jitter                  float64 = 0.0
	numCities               int     = 150
	numEmpires              int     = 10
	numCityStates           int     = 150
	numTribes               int     = 30
	numMiningTowns          int     = 60
	numMiningGemsTowns      int     = 60
	numQuarryTowns          int     = 60
	numFarmingTowns         int     = 60
	numTradingTowns         int     = 10
	numDesertOasis          int     = 10
	enableCityAging         bool    = true
	enableOrganizedReligions bool    = true
	migrationOverpopulationExcessPopulationFactor float64 = 1.2
	migrationOverpopulationMinPopulationFactor    float64 = 0.1
	migrationToNClosestCities                     int     = 10
	migrationToNewSettlementWithinNRegions        int     = 10
	migrationFatalityChance                       float64 = 0.02
	seedEntities                            bool    = true
)

func init() {
	flag.Int64Var(&seed, "seed", seed, "the world seed")
	flag.IntVar(&numPlates, "num_plates", numPlates, "number of plates")
	flag.IntVar(&numPoints, "num_points", numPoints, "number of points")
	flag.IntVar(&numVolcanoes, "num_volcanoes", numVolcanoes, "number of volcanoes")
	flag.Float64Var(&jitter, "jitter", jitter, "jitter")
	flag.IntVar(&numCities, "num_cities", numCities, "number of cities")
	flag.IntVar(&numEmpires, "num_empires", numEmpires, "number of empires")
	flag.IntVar(&numCityStates, "num_citystates", numCityStates, "number of city states")
	flag.IntVar(&numTribes, "num_tribes", numTribes, "number of tribes")
	flag.IntVar(&numMiningTowns, "num_mining_towns", numMiningTowns, "number of mining towns")
	flag.IntVar(&numMiningGemsTowns, "num_mining_gems_towns", numMiningGemsTowns, "number of mining gems towns")
	flag.IntVar(&numQuarryTowns, "num_quarry_towns", numQuarryTowns, "number of quarry towns")
	flag.IntVar(&numFarmingTowns, "num_farming_towns", numFarmingTowns, "number of farming towns")
	flag.IntVar(&numTradingTowns, "num_trading_towns", numTradingTowns, "number of trading towns")
	flag.IntVar(&numDesertOasis, "num_desert_oasis", numDesertOasis, "number of desert oasis")
	flag.BoolVar(&enableCityAging, "enable_city_aging", enableCityAging, "enable city aging")
	flag.BoolVar(&enableOrganizedReligions, "enable_organized_religions", enableOrganizedReligions, "enable organized religions")
	flag.Float64Var(&migrationOverpopulationExcessPopulationFactor, "migration_excess_factor", migrationOverpopulationExcessPopulationFactor, "migration excess factor")
	flag.Float64Var(&migrationOverpopulationMinPopulationFactor, "migration_min_factor", migrationOverpopulationMinPopulationFactor, "migration min factor")
	flag.IntVar(&migrationToNClosestCities, "migration_n_cities", migrationToNClosestCities, "migration n closest cities")
	flag.IntVar(&migrationToNewSettlementWithinNRegions, "migration_n_regions", migrationToNewSettlementWithinNRegions, "migration n regions")
	flag.Float64Var(&migrationFatalityChance, "migration_fatality_chance", migrationFatalityChance, "migration fatality chance")
	flag.BoolVar(&seedEntities, "seed_entities", seedEntities, "seed entities")
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile to this file")

func main() {
	// Generate a new world.
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	defer func() {
		if *memprofile != "" {
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal(err)
			}
			pprof.WriteHeapProfile(f)
			f.Close()
		}
	}()

	// Initialize the config.
	cfg := genworldvoronoi.NewConfig()
	cfg.GeoConfig.NumPlates = numPlates
	cfg.GeoConfig.OceanPlatesFraction = oceanPlatesFraction
	cfg.GeoConfig.OceanPlatesAltSelection = oceanPlatesAltSelection
	cfg.GeoConfig.NumPoints = numPoints
	cfg.GeoConfig.NumVolcanoes = numVolcanoes
	cfg.GeoConfig.Jitter = jitter
	cfg.CivConfig.NumCities = numCities
	cfg.CivConfig.NumEmpires = numEmpires
	cfg.CivConfig.NumCityStates = numCityStates
	cfg.CivConfig.NumTribes = numTribes
	cfg.CivConfig.NumMiningTowns = numMiningTowns
	cfg.CivConfig.NumMiningGemsTowns = numMiningGemsTowns
	cfg.CivConfig.NumQuarryTowns = numQuarryTowns
	cfg.CivConfig.NumFarmingTowns = numFarmingTowns
	cfg.CivConfig.NumTradingTowns = numTradingTowns
	cfg.CivConfig.NumDesertOasis = numDesertOasis
	cfg.CivConfig.EnableCityAging = enableCityAging
	cfg.CivConfig.EnableOrganizedReligions = enableOrganizedReligions
	cfg.CivConfig.MigrationOverpopulationExcessPopulationFactor = migrationOverpopulationExcessPopulationFactor
	cfg.CivConfig.MigrationOverpopulationMinPopulationFactor = migrationOverpopulationMinPopulationFactor
	cfg.CivConfig.MigrationToNClosestCities = migrationToNClosestCities
	cfg.CivConfig.MigrationToNewSettlementWithinNRegions = migrationToNewSettlementWithinNRegions
	cfg.CivConfig.MigrationFatalityChance = migrationFatalityChance
	cfg.CivConfig.SeedEntities = seedEntities

	sp, err := genworldvoronoi.NewMapFromConfig(seed, cfg)
	if err != nil {
		log.Fatal(err)
	}

	g, err := maptiles.NewGame(sp)
	if err != nil {
		os.Exit(1)
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Printf("error running game: %v", err)
	}
}
