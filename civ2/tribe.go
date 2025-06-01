package civ2

import (
	"fmt"
	"math/rand"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/go_gens/genlanguage"
)

var tribeID = 0

func nextTribeID() int {
	tribeID++
	return tribeID
}

// Tribe represents a tribe in the simulation.
type Tribe struct {
	ID         int
	RegionID   int
	Population int
	Aggressive bool
	Path       *civ.Path
	Culture    *Culture
	Settling   bool
	Origin     *City // TODO: Improve this.
	*Storage
}

func (m *Civ) NewTribe(region, population int) *Tribe {
	tribeID := nextTribeID()
	return &Tribe{
		ID:         tribeID,
		RegionID:   region,
		Population: population,
		Aggressive: rand.Float64() < 0.5,
		Culture:    m.newCulture(region, genlanguage.GenLanguage(int64(tribeID)), m.cultureFunc(region)),
		Storage:    NewStorage(),
	}
}

// GetID returns the ID of the tribe.
func (t Tribe) GetID() int {
	return t.ID
}

// Name returns the name of the tribe.
func (t Tribe) Name() string {
	if t.Origin != nil {
		return fmt.Sprintf("T %d (%s)", t.ID, t.Origin.Name)
	}
	return fmt.Sprintf("T %d", t.ID)
}

// ToSettlement converts the tribe to a settlement.
func (t *Tribe) ToSettlement(n int) *Settlement {
	n = min(n, t.Population)
	// Update culture based on region.
	s := &Settlement{
		ID:         t.RegionID,
		Name:       t.Culture.Language.MakeCityName(),
		Population: n,
		Culture:    t.Culture, // TODO: Switch to sedentary culture?
		Storage:    t.Storage,
	}
	// TODO: Add the settlement to the region.

	// Abandon the tribe.
	t.Population = max(0, t.Population-n)

	return s
}

// Grow the population.
func (t *Tribe) Grow(nDays int) {
	const growthRate = 0.001 // 0.1% growth per year.
	if growth := calcPopulationGrowth(t.Population, growthRate, nDays); growth >= 1 {
		t.Population += int(growth)
	} else if rand.Float64() < growth {
		t.Population++
	}
}

// Fight another tribe. Returns true if the tribe won the fight.
func (t *Tribe) Fight(other *Tribe) bool {
	lossesT, lossesO := calcCombat(t.Population, other.Population)
	t.Population -= lossesT
	other.Population -= lossesO

	// Return true if we have more population left.
	return t.Population > other.Population
}

// Split the tribe into two tribes.
func (t *Tribe) Split(m *Civ, pop int) *Tribe {
	pop = min(pop, t.Population)
	t.Population -= pop
	c := t.Culture // Fork the language? Change the culture?
	if rand.Float64() < 0.5 {
		// Randomly change the culture.
		if c = m.GetCulture(t.RegionID); c == nil {
			c = m.newCulture(t.RegionID, genlanguage.GenLanguage(int64(t.ID)), m.cultureFunc(t.RegionID))
		}
	}
	return &Tribe{
		ID:         nextTribeID(),
		RegionID:   t.RegionID,
		Population: pop,
		Culture:    c,
		Aggressive: t.Aggressive,
		Storage:    NewStorage(),
	}
}

func (t *Tribe) SetPath(path *civ.Path) {
	t.Path = path
	if path != nil && t.Path.Peek() == t.RegionID {
		t.Path.Next() // Skip the current region.
	}
}

var rnbsMoveTribeToRegion = make([]int, 0, 8)

func (m *Civ) navigateOneStep(t *Tribe, region int, navCache *civ.NavCache) bool {
	// Plan the path to the neighbor region.
	newPath := &civ.Path{
		From:  t.RegionID,
		To:    region,
		Steps: []int{region, t.RegionID},
		Idx:   0,
	}
	t.SetPath(newPath)
	return true
}

func (m *Civ) navigateMultiStep(t *Tribe, region int, navCache *civ.NavCache) bool {
	// Plan the path to the neighbor region.
	newPath, found := civ.PlanPath(navCache.GetTile(t.RegionID), navCache.GetTile(region))
	if found {
		t.SetPath(newPath)
	}
	return found
}

// pickNCradlesOfCivilization returns the n most suitable regions
// for the cradle of civilization (of the given biome).
func (m *Civ) pickNCradlesOfCivilization(n int, useBiome bool) []int {
	// Get elevation values.
	elevs := m.Elevation.GetValues()

	// This function will return true if the region is valid for the current cradle.
	var validRegion func(r, n int) bool
	if useBiome {
		bf := m.GetRegWhittakerModBiomeFunc()
		validRegion = func(r, n int) bool {
			return bf(r) == genbiome.WhittakerModBiomeTemperateGrassland
		}
	} else {
		validRegion = func(r, n int) bool {
			return m.cultureFunc(r) == civ.CultureType(max(n%int(civ.CultureTypeCount), 1))
		}
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
		scores := m.CalcCityScoreWithDistanceField(func(r int) float64 {
			if elevs[r] > 0 && validRegion(r, i) {
				return float64(m.calcMaxPopPerRegion(r))
			}
			return 0.0
		}, regDistanceC)

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
			if useBiome {
				panic(fmt.Sprintf("no suitable region found for cradle of civilization %d", i))
			} else {
				panic(fmt.Sprintf("no suitable region found for cradle of civilization %d with culture %d", i, max(i%int(civ.CultureTypeCount), 1)))
			}
		}
		// Place a cradle at the region with the highest fitness score.
		bestRegions = append(bestRegions, bestRegion)

		// Update the distance field.
		regDistanceC = m.UpdateDistanceField(regDistanceC, bestRegions, stopRegions)
	}
	return bestRegions
}
