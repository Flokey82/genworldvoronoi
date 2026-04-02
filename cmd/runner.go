package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/Flokey82/genworldvoronoi"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile to this file")
var seed = flag.Int64("seed", 12345, "the world seed")
var numCities = flag.Int("num_cities", 150, "number of cities")
var numEmpires = flag.Int("num_empires", 10, "number of empires")
var numCityStates = flag.Int("num_citystates", 150, "number of city states")
var numTribes = flag.Int("num_tribes", 30, "number of tribes")
var numMiningTowns = flag.Int("num_mining_towns", 60, "number of mining towns")
var numMiningGemsTowns = flag.Int("num_mining_gems_towns", 60, "number of mining gems towns")
var numQuarryTowns = flag.Int("num_quarry_towns", 60, "number of quarry towns")
var numFarmingTowns = flag.Int("num_farming_towns", 60, "number of farming towns")
var numTradingTowns = flag.Int("num_trading_towns", 10, "number of trading towns")
var numDesertOasis = flag.Int("num_desert_oasis", 10, "number of desert oasis")
var enableCityAging = flag.Bool("enable_city_aging", true, "enable city aging")
var enableOrganizedReligions = flag.Bool("enable_organized_religions", true, "enable organized religions")
var migrationOverpopulationExcessPopulationFactor = flag.Float64("migration_excess_factor", 1.2, "migration excess factor")
var migrationOverpopulationMinPopulationFactor = flag.Float64("migration_min_factor", 0.1, "migration min factor")
var migrationToNClosestCities = flag.Int("migration_n_cities", 10, "migration n closest cities")
var migrationToNewSettlementWithinNRegions = flag.Int("migration_n_regions", 10, "migration n regions")
var migrationFatalityChance = flag.Float64("migration_fatality_chance", 0.02, "migration fatality chance")
var seedEntities = flag.Bool("seed_entities", true, "seed entities")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	cfg := genworldvoronoi.NewConfig()
	cfg.NumPoints = 400000
	cfg.EnableCityAging = false
	cfg.CivConfig.NumCities = *numCities
	cfg.CivConfig.NumEmpires = *numEmpires
	cfg.CivConfig.NumCityStates = *numCityStates
	cfg.CivConfig.NumTribes = *numTribes
	cfg.CivConfig.NumMiningTowns = *numMiningTowns
	cfg.CivConfig.NumMiningGemsTowns = *numMiningGemsTowns
	cfg.CivConfig.NumQuarryTowns = *numQuarryTowns
	cfg.CivConfig.NumFarmingTowns = *numFarmingTowns
	cfg.CivConfig.NumTradingTowns = *numTradingTowns
	cfg.CivConfig.NumDesertOasis = *numDesertOasis
	cfg.CivConfig.EnableCityAging = *enableCityAging
	cfg.CivConfig.EnableOrganizedReligions = *enableOrganizedReligions
	cfg.CivConfig.MigrationOverpopulationExcessPopulationFactor = *migrationOverpopulationExcessPopulationFactor
	cfg.CivConfig.MigrationOverpopulationMinPopulationFactor = *migrationOverpopulationMinPopulationFactor
	cfg.CivConfig.MigrationToNClosestCities = *migrationToNClosestCities
	cfg.CivConfig.MigrationToNewSettlementWithinNRegions = *migrationToNewSettlementWithinNRegions
	cfg.CivConfig.MigrationFatalityChance = *migrationFatalityChance
	cfg.CivConfig.SeedEntities = *seedEntities

	sp, err := genworldvoronoi.NewMapFromConfig(*seed, cfg)
	if err != nil {
		log.Fatal(err)
	}

	exportPNG := true
	exportOBJ := true
	exportSVG := true
	exportWebp := true
	if exportPNG {
		sp.ExportPng("test.png")
	}
	if exportOBJ {
		sp.ExportOBJ("test.obj")
	}
	if exportSVG {
		sp.ExportSVG("test.svg")
	}
	if exportWebp {
		sp.ExportWebp("test.webp")
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}
}
