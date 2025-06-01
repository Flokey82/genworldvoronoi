package civ

import (
	"fmt"
	"log"
	"math"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/go_gens/genlanguage"
)

// GetCulture returns the culture of the given region (if any).
func (m *Civ) GetCulture(r int) *Culture {
	return m.Cultures.GetAt(r)
}

// PlaceNCultures places n cultures on the map.
// This code is based on:
// https://github.com/Azgaar/Fantasy-Map-Generator/blob/master/modules/cultures-generator.js
func (m *Civ) PlaceNCultures(n int) {
	m.ResetRand()
	m.placeNCultures(n)
	m.ExpandCultures()
}

// placeNCultures places n cultures on the map.
func (m *Civ) placeNCultures(n int) {
	regCultureFunc := GetRegionCultureTypeFunc(m.Geo)
	climateFitness := m.GetFitnessClimate()

	// Get elevation values.
	elevs := m.Elevation.GetValues()

	// The fitness function, returning a score from 0.0 to 1.0 for a given region.
	scoreFunc := func(r int) float64 {
		if elevs[r] <= 0 {
			return 0
		}
		return math.Sqrt((climateFitness(r) + 3.0) / 4.0)
	}

	// For now we maximize the distance to other cultures.
	distSeedFunc := func() []int {
		cultureSeeds := make([]int, 0, len(m.Cultures.Objects))
		for _, c := range m.Cultures.Objects {
			cultureSeeds = append(cultureSeeds, c.ID)
		}
		return cultureSeeds
	}

	// Get the stop regions, i.e. regions that we don't want to place cultures in.
	stopRegions := make(map[int]bool)

	// Place n cultures of any type.
	regDistanceC := m.AssignDistanceField(distSeedFunc(), stopRegions)
	for i := 0; i < n; i++ {
		// We use the city score since it identifies regions that are well suited for settlement or general survival.
		c := m.placeCultureWithScore(regCultureFunc, m.CalcCityScoreWithDistanceField(scoreFunc, regDistanceC), nil)
		log.Printf("placing culture %d: %s", i, c.Name)

		// Update the distance field to ensure we evenly distribute the cultures.
		regDistanceC = m.UpdateDistanceField(regDistanceC, distSeedFunc(), stopRegions)
	}
}

// PlaceCulture places another culture on the map at the region with the highest fitness score.
func (m *Civ) PlaceCulture(regCultureFunc func(int) CultureType, scoreFunc func(int) float64, distSeedFunc func() []int, lang *genlanguage.Language) *Culture {
	return m.placeCultureWithScore(regCultureFunc, m.CalcCityScore(scoreFunc, distSeedFunc), lang)
}

// placeCultureWithScore places a culture at the given region using the computed scores.
func (m *Civ) placeCultureWithScore(regCultureFunc func(int) CultureType, scores []float64, lang *genlanguage.Language) *Culture {
	// Score all regions, pick highest score.
	var newculture int
	lastMax := math.Inf(-1)
	for i, val := range scores {
		if val > lastMax {
			newculture = i
			lastMax = val
		}
	}
	c := m.newCulture(newculture, regCultureFunc(newculture), lang)
	m.Cultures.PlaceObjectAt(c, newculture)
	m.Cultures.Regions[newculture] = -1 // FIXME: This is a hack, because expansion won't work otherwise.
	return c
}

// PlaceCultureAt places a culture at the given region.
// TODO: Allow specifying the culture type?
func (m *Civ) PlaceCultureAt(r int, grow bool, lang *genlanguage.Language) *Culture {
	c := m.newCulture(r, GetRegionCultureTypeFunc(m.Geo)(r), lang)
	return m.AddCultureAt(r, c, grow)
}

// AddCultureAt adds a culture at the given region.
func (m *Civ) AddCultureAt(r int, c *Culture, grow bool) *Culture {
	c.Regions = append(c.Regions, r)
	c.Stats = m.GetStats(c.Regions)
	m.Cultures.PlaceObjectAt(c, r)
	m.Cultures.Regions[r] = -1 // FIXME: This is a hack, because expansion won't work otherwise.
	// NOTE: This might be quite expensive, so we might want to
	// avoid this calling here, or at least limit the regions
	// we process to the ones that are close to the new culture.
	if grow {
		m.ExpandCultures()
	}

	// Add a new event to the history.
	m.History.AddEvent("Founding (Culture)", fmt.Sprintf("Culture of %s has been placed (?)", c.Name), c.Ref())
	return c
}

// Culture represents a culture.
type Culture struct {
	ID           int         // Region where the culture originates
	Name         string      // Name of the culture
	Type         CultureType // Type of the culture
	Expansionism float64     // Expansionism of the culture
	Martialism   float64     // Martial skills of the culture
	Spirituality float64     // Spirituality of the culture (religion, superstition, etc.)
	Openness     float64     // Openness to other cultures or new ideas
	// Sophistication float64
	// Extremism      float64 ?
	// Parent    *Culture
	// Children  []*Culture
	// Extinct   bool
	Language *genlanguage.Language // Language of the culture
	Religion *Religion             // Religion of the culture

	// TODO: DO NOT CACHE THIS!
	Regions []int
	*geo.Stats
}

func (m *Civ) newCulture(r int, cultureType CultureType, lang *genlanguage.Language) *Culture {
	if lang == nil {
		lang = GenLanguage(m.Seed + int64(r))
	}
	c := &Culture{
		ID:       r,
		Name:     lang.MakeName(),
		Language: lang,
	}
	c.SetNewType(cultureType)
	return c
}

// GetID returns the ID of the culture.
func (c Culture) GetID() int {
	return c.ID
}

func (c *Culture) Ref() ObjectReference {
	return ObjectReference{ID: c.ID, Type: ObjectTypeCulture}
}

func (c *Culture) Log() {
	log.Printf("The Folk of %s (%s): %d regions", c.Name, c.Type.String(), len(c.Regions))
	log.Printf("Followers of %s (%s)", c.Religion.Name, c.Religion.Group)
	c.Stats.Log()
}

func (c *Culture) compare(b *Culture) float64 {
	if c == b {
		return 1.0
	}
	if c == nil || b == nil {
		return -1.0
	}

	// If two cultures are high in expansionism, they will probably
	// not get along well.
	// High expansionism: -1.0, low expansionism: 1.0
	expVal := 1.0 - (c.Expansionism+b.Expansionism)/maxExpansionism

	// Similar martialism will lead to respect.
	// The larger the difference, the less compatible the cultures are.
	// Identical martialism: 1.0, max difference martialism: -1.0
	martialVal := 1.0 - 2*math.Abs(c.Martialism-b.Martialism)/maxMartialism

	// Same as martialism, but for spirituality.
	spiritVal := 1.0 - 2*math.Abs(c.Spirituality-b.Spirituality)/maxSpirituality

	// Evaluate the openness of the cultures.
	// The higher the openness, the higher the value.
	openVal := -1 + (c.Openness+b.Openness)/maxOpenness

	// Check if the languages are identical.
	langVal := compareLanguage(c.Language, b.Language)

	// Check if the Religions are identical.
	// The more spiritual the culture is, the more important this score is.
	relVal := c.Religion.compare(b.Religion) * (c.Spirituality + b.Spirituality) / maxSpirituality

	// Check if the types are identical.
	value := expVal + martialVal + spiritVal + openVal + langVal + relVal
	if c.Type == b.Type {
		value += 1
	} else {
		value -= 1
	}

	return value / 7
}

func (c *Culture) Fork(newID int) *Culture {
	if c == nil {
		return nil
	}
	useLangClone := false
	lang := c.Language
	log.Println("TODO: Clone language")
	if useLangClone {
		// TODO: This needs to better randomize subsequent names. It'll generate the same names than the original.
		lang = c.Language.Fork(int64(newID))
	}

	cNew := &Culture{
		ID:       newID,
		Name:     c.Language.MakeName(), // Give the new culture a new name.
		Language: lang,
		Religion: c.Religion,
		Stats:    geo.NewStats(), // TODO: This will need to be regenerated.
	}
	cNew.SetNewType(c.Type)
	return cNew
}

func (c *Culture) SetNewType(t CultureType) {
	c.Type = t
	c.Expansionism = t.Expansionism()
	c.Martialism = t.Martialism()
	c.Spirituality = t.Spirituality()
	c.Openness = t.Openness()
}

// ExpandCultures expands the cultures on the map based on their expansionism,
// terrain preference, and distance to other cultures.
func (m *Civ) ExpandCultures() {
	seeds := make([]int, 0, len(m.Cultures.Objects))
	for _, c := range m.Cultures.Objects {
		seeds = append(seeds, c.ID)
	}
	m.expandCultures(false, seeds)
}

func (m *Civ) expandCultures(aggressive bool, seeds []int) {
	// The cultural centers will be the seed points for the expansion.
	originToCulture := make(map[int]*Culture)
	for _, c := range m.Cultures.Objects {
		originToCulture[c.ID] = c
	}

	rCellType := m.GetRegCellTypes()

	// Get the elevation values.
	elevs := m.Elevation.GetValues()
	maxElev := m.Elevation.Max

	territoryWeightFunc := m.getTerritoryWeightFunc()
	biomeWeight := m.getTerritoryBiomeWeightFunc()

	placeFunc := m.regPlaceNTerritoriesCustom
	if aggressive {
		placeFunc = m.expandTerritoriesAggressive
	}
	m.Cultures.Regions = placeFunc(m.Cultures.Regions, seeds, func(o, u, v int) float64 {
		c := originToCulture[o]
		if c == nil {
			panic(fmt.Sprintf("culture %d not found", o))
		}

		// Get the cost to expand to this biome.
		gotBiome := m.GetAzgaarRegionBiome(v, elevs[v]/maxElev)
		biomePenalty := biomeWeight(o, u, v) * float64(genbiome.AzgaarBiomeMovementCost[gotBiome]) / 100

		// Check if we have a non-native biome, if so we apply an additional penalty.
		// NOTE: This check has been disabled for now.
		// if m.getAzgaarRegionBiome(o, m.Elevation[o]/maxElev, maxElev) != gotBiome {
		biomePenalty *= c.Type.BiomeCost(gotBiome)
		// }

		cellTypePenalty := c.Type.CellTypeCost(rCellType[v])
		return biomePenalty + cellTypePenalty*territoryWeightFunc(o, u, v)/c.Expansionism
	})

	// TODO: There are small islands that do not have a culture...
	// We should (or could) fix that.

	// Update the cultures with the new regions.
	for _, c := range m.Cultures.Objects {
		// Collect all regions that are part of the current culture.
		c.Regions = c.Regions[:0]

		// TODO: Maybe avoid repeatedly iterating over the regions and
		// do it only once for all cultures.
		for r, cu := range m.Cultures.Regions {
			if cu == c.ID {
				c.Regions = append(c.Regions, r)
			}
		}
		c.Stats = m.GetStats(c.Regions)
	}

	// TODO: Move this somewhere else or improve how it is handled.
	// m.genCultureSkills()
}
