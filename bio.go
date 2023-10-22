package genworldvoronoi

import (
	"log"
	"math/rand"
	"time"
)

// Bio handles the generation of life on the map (plants, animals, etc.).
type Bio struct {
	*BioConfig
	*Geo
	Species                []*Species              // All species on the map.
	SpeciesFamilyToRegions map[SpeciesFamily][]int // Regions where each species is found.
	SpeciesRegions         []int                   // Regions where each species is found.
	GrowthDays             []int                   // Number of days within the growth period for each region.
	GrowthInsolation       []float64               // Average insolation for each region during the growth period.
	rand                   *rand.Rand              // Random number generator.
}

func newBio(geo *Geo, cfg *BioConfig) *Bio {
	if cfg == nil {
		cfg = NewBioConfig()
	}
	return &Bio{
		BioConfig:              cfg,
		Geo:                    geo,
		GrowthDays:             make([]int, geo.SphereMesh.NumRegions),
		GrowthInsolation:       make([]float64, geo.SphereMesh.NumRegions),
		SpeciesRegions:         make([]int, geo.SphereMesh.NumRegions),
		SpeciesFamilyToRegions: make(map[SpeciesFamily][]int),
		rand:                   rand.New(rand.NewSource(geo.Seed)),
	}
}

func (b *Bio) generateBiology() {
	// Calculate the duration of the potential growth period for each region.
	b.calcGrowthPeriod()

	// TODO: Calculate a score for each region that reflects how well
	// suited it is for agriculture during the growth period. This
	// would be based on insolation, temperature, rainfall, steepness,
	// etc.

	// TODO: Calculate a score for each region that reflects how well
	// herbivores would be able to survive there.
	// As long as there are shrubs, etc. then herbivores should be able
	// to survive even in winter.

	// TODO: Calculate a score for each region that reflects how well
	// carnivores would be able to survive there (presence of prey animals,
	// total survivability).

	// Generate the pre-defined species.
	/*
		b.placeAllSpecies(KingdomFauna)
		b.placeAllSpecies(KingdomFlora)
		b.placeAllSpecies(KingdomFungi)
	*/
	b.placeAllSpecies(GenusCereal)

	// Generate the species.
	if b.EnableRandomSpecies {
		b.genNRandomSpecies(b.NumSpecies)
	}
	b.SpeciesRegions = b.expandSpecies()
	b.SpeciesFamilyToRegions = b.expandSpecies2()
}

// calcGrowthPeriod calculates the duration of the potential growth
// period for each region (dormancy can be inferred), which will
// give us the potential for agricultural output (nr of harvests etc).
// Furthermore, it calculates the average insolation for each region
// during the growth period, which will influence the amount of
// agricultural output.
func (b *Bio) calcGrowthPeriod() {
	start := time.Now()
	useGoRoutines := true
	// Use go routines to process a chunk of regions at a time.
	if useGoRoutines {
		kickOffChunkWorkers(b.SphereMesh.NumRegions, b.calcGrowthPeriodChunk)
	} else {
		b.calcGrowthPeriodChunk(0, b.SphereMesh.NumRegions)
	}
	log.Println("calcGrowthPeriod took", time.Since(start))
}

func (b *Bio) calcGrowthPeriodChunk(start, end int) {
	// Calculate the duration of the potential growth period for each region.
	for r := start; r < end; r++ {
		var growthDays int
		var totalInsolation float64
		for i := 0; i < 356; i++ {
			// Calculate daily average temperature.
			min, max := b.getMinMaxTemperatureOfDay(b.LatLon[r][0], i)
			avg := (min + max) / 2

			// TODO: Right now we only count days where the average temperature
			// is above 0. This is not correct, as we should be counting days
			// where the average temperature is above a certain minimum.
			// We should also take in account when there is precipitation.
			if avg > 0 && b.Rainfall[r] > 0 {
				growthDays++
				totalInsolation += calcSolarRadiation(b.LatLon[r][0], i)
			}
		}
		b.GrowthDays[r] = growthDays
		// NOTE: We should take in account the altitude of the region.
		b.GrowthInsolation[r] = totalInsolation / float64(growthDays)
	}
}

func (b *Bio) Tick() {
	// TODO: Tick the species.
}
