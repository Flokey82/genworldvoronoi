package genworldvoronoi

import (
	"fmt"
	"log"
	"math"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/go_gens/genlanguage"
)

func (m *Civ) getCultureFunc() func(int) *Culture {
	// Build a map of culture IDs to cultures.
	cultureMap := make(map[int]*Culture)
	for _, c := range m.Cultures {
		cultureMap[c.ID] = c
	}
	return func(r int) *Culture {
		return cultureMap[m.RegionToCulture[r]]
	}
}

// GetCulture returns the culture of the given region (if any).
func (m *Civ) GetCulture(r int) *Culture {
	// NOTE: This sucks. This should be done better.
	if m.RegionToCulture[r] < 0 {
		return nil
	}
	for _, c := range m.Cultures {
		if c.ID == m.RegionToCulture[r] {
			return c
		}
	}
	return nil
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
	// The fitness function, returning a score from
	// 0.0 to 1.0 for a given region.
	var scoreFunc func(int) float64

	// The distance seed point function, returning
	// seed points/regions that we want to be far
	// away from.
	var distSeedFunc func() []int

	regCultureFunc := m.getRegionCultureTypeFunc()
	climateFitness := m.GetFitnessClimate()
	scoreFunc = func(r int) float64 {
		if m.Elevation[r] <= 0 {
			return 0
		}
		return math.Sqrt((climateFitness(r) + 3.0) / 4.0)
	}

	// For now we maximize the distance to other cultures.
	distSeedFunc = func() []int {
		var cultureSeeds []int
		for _, c := range m.Cultures {
			cultureSeeds = append(cultureSeeds, c.ID)
		}
		return cultureSeeds
	}

	// Get the stop regions, i.e. regions that we don't want to place cultures in.
	stopRegions := make(map[int]bool)

	// Place n cultures of any type.
	regDistanceC := m.AssignDistanceField(distSeedFunc(), stopRegions)
	for i := 0; i < n; i++ {
		// We use the city score since it identifies regions that are well suited for
		// settlement or general survival.
		c := m.placeCultureWithScore(regCultureFunc, m.CalcCityScoreWithDistanceField(scoreFunc, regDistanceC), nil)
		log.Printf("placing culture %d: %s", i, c.Name)

		// Update the distance field to ensure we evenly distribute the cultures.
		regDistanceC = m.UpdateDistanceField(regDistanceC, distSeedFunc(), stopRegions)
	}
}

// ExpandCultures expands the cultures on the map based on their expansionism,
// terrain preference, and distance to other cultures.
func (m *Civ) ExpandCultures() {
	seeds := make([]int, 0, len(m.Cultures))
	for _, c := range m.Cultures {
		seeds = append(seeds, c.ID)
	}
	m.expandCultures(false, seeds)
}

func (m *Civ) expandCultures(aggressive bool, seeds []int) {
	// The cultural centers will be the seed points for the expansion.
	originToCulture := make(map[int]*Culture)
	for _, c := range m.Cultures {
		originToCulture[c.ID] = c
	}

	rCellType := m.GetRegCellTypes()
	_, maxElev := minMax(m.Elevation)
	territoryWeightFunc := m.getTerritoryWeightFunc()
	biomeWeight := m.getTerritoryBiomeWeightFunc()

	placeFunc := m.regPlaceNTerritoriesCustom
	if aggressive {
		placeFunc = m.expandTerritoriesAggressive
	}
	m.RegionToCulture = placeFunc(m.RegionToCulture, seeds, func(o, u, v int) float64 {
		c := originToCulture[o]
		if c == nil {
			panic(fmt.Sprintf("culture %d not found", o))
		}

		// Get the cost to expand to this biome.
		gotBiome := m.GetAzgaarRegionBiome(v, m.Elevation[v]/maxElev, maxElev)
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
	for _, c := range m.Cultures {
		// Collect all regions that are part of the current culture.
		c.Regions = c.Regions[:0]

		// TODO: Maybe avoid repeatedly iterating over the regions and
		// do it only once for all cultures.
		for r, cu := range m.RegionToCulture {
			if cu == c.ID {
				c.Regions = append(c.Regions, r)
			}
		}
		c.Stats = m.GetStats(c.Regions)
	}

	// TODO: Move this somewhere else or improve how it is handled.
	// m.genCultureSkills()
}

// Culture represents a culture.
//
// Also see: https://ck3.paradoxwikis.com/Culture
//
// TODO:
//
// # VALUES
//
// The type of culture will also influence their values.
// For example, a nomadic or a hunting culture will have
// higher regart for martial skills and lower regard for
// sophistication.
//
// I would propose to have a point pool for these attributes
// and then randomly assign them to the various skills with
// a certain distribution based on the culture type.
//
// # CRAFTS AND SKILLS
//
// The culture will also have a set of crafts, arts, and skills
// that they are good at. This will be based on their values and
// the environment they live in.
//
// For example, a nomadic culture will be good at hunting and
// tracking. They might be exceptional at crafting leather wares
// (like saddles), and bows.
//
// A river culture will have a ready supply of clay, and will
// be good at pottery, but not at crafin leather wares like
// saddles.
//
// A mountain culture will be good at mining, prospecting, and
// stone carving. If they mine precious metals, they will be
// good at jewelry making.
//
// A naval culture will be good at ship building, and sailing.
// Since they might have sea shells as a ready supply, they
// might be good at jewelry making.
//
// # ARTS
//
// Arts will be based on the environment and the values of the
// culture. For example, a river culture might focus on water
// and rivers, as well as the flora and fauna related to rivers.
//
// A mountain culture might use iconography to represent the
// harshness of the mountains, and the gifts of the mines.
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
	// Sum up the expansionism of both cultures.
	// This will increase negative effects if the cultures are very different.
	expSum := 1.0 + c.Expansionism + b.Expansionism

	// Sum up martialism.
	martialSum := 1.0 + c.Martialism + b.Martialism

	// Sum up spirituality.
	spiritSum := 1.0 + c.Spirituality + b.Spirituality

	// Evaluate the openness of the cultures.
	// The higher the openness, the higher the value.
	openVal := (c.Openness + b.Openness) / 2.0

	// Check if the languages are identical.
	langVal := compareLanguage(c.Language, b.Language)

	// Check if the Religions are identical.
	// The more spiritual the culture is, the more important this score is.
	relVal := c.Religion.compare(b.Religion) * spiritSum

	var value float64

	value += langVal + relVal + openVal

	log.Printf("Comparing %s and %s: langVal: %f, relVal: %f", c.Name, b.Name, langVal, relVal)

	// Check if the types are identical.
	if c.Type == b.Type {
		value += 0.5
	} else {
		value -= 0.5 * expSum * martialSum
	}

	return value
}

func (c *Culture) Fork(newID int) *Culture {
	if c == nil {
		return nil
	}
	useLangClone := false
	lang := c.Language
	log.Println("TODO: Clone language")
	if useLangClone {
		// TODO: This needs to better randomize subsequemt names. It'll generate the same names than the original.
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
	m.Cultures = append(m.Cultures, c)
	return c
}

// PlaceCultureAt places a culture at the given region.
// TODO: Allow specifying the culture type?
func (m *Civ) PlaceCultureAt(r int, grow bool, lang *genlanguage.Language) *Culture {
	c := m.newCulture(r, m.getRegionCultureTypeFunc()(r), lang)
	c.Regions = []int{r}
	c.Stats = m.GetStats(c.Regions)
	m.Cultures = append(m.Cultures, c)
	// NOTE: This might be quite expensive, so we might want to
	// avoid this calling here, or at least limit the regions
	// we process to the ones that are close to the new culture.
	if grow {
		m.ExpandCultures()
	} else {
		m.RegionToCulture[r] = c.ID
	}

	// Add a new event to the history.
	m.History.AddEvent("Founding (Culture)", fmt.Sprintf("Culture of %s has emerged", c.Name), c.Ref())
	return c
}

// AddCultureAt adds a culture at the given region.
func (m *Civ) AddCultureAt(r int, c *Culture, grow bool) *Culture {
	c.Regions = append(c.Regions, r)
	c.Stats = m.GetStats(c.Regions)
	m.Cultures = append(m.Cultures, c)
	// NOTE: This might be quite expensive, so we might want to
	// avoid this calling here, or at least limit the regions
	// we process to the ones that are close to the new culture.
	if grow {
		m.ExpandCultures()
	} else {
		m.RegionToCulture[r] = c.ID
	}
	m.History.AddEvent("Founding (Culture)", fmt.Sprintf("Culture of %s has been placed (?)", c.Name), c.Ref())
	return c
}

// getRegionCutureTypeFunc returns a function that returns the culture type suitable for a given region.
func (m *Civ) getRegionCultureTypeFunc() func(int) CultureType {
	cellType := m.GetRegCellTypes()
	getType := m.GetRegionFeatureTypeFunc()
	//biomeFunc := m.GetRegWhittakerModBiomeFunc()
	_, maxElev := minMax(m.Elevation)
	//log.Println("TODO: Map whittaker to azgaar biomes")

	// Return culture type based on culture center region.
	return func(r int) CultureType {
		eleVal := m.Elevation[r] / maxElev
		gotBiome := m.GetAzgaarRegionBiome(r, eleVal, maxElev)
		//log.Println(gotBiome)
		//log.Println(biomeFunc(r))

		// Desert and grassland means a nomadic culture.
		// BUT: Grassland is extremely well suited for farming... Which is not nomadic.
		if eleVal < 0.7 && (gotBiome == genbiome.AzgaarBiomeHotDesert ||
			gotBiome == genbiome.AzgaarBiomeColdDesert ||
			gotBiome == genbiome.AzgaarBiomeGrassland) {
			return CultureTypeNomadic // high penalty in forest biomes and near coastline
		}

		// Montane cultures in high elevations and hills
		// that aren't deserts or grassland.
		if eleVal > 0.3 {
			return CultureTypeHighland // no penalty for hills and moutains, high for other elevations
		}

		// Get the region (if any) that represents the haven for this region.
		// A haven is the closest neighbor that is a water body.
		// NOTE: harborSize indicates the number of neighbors that are water.
		rHaven, harborSize := m.GetRegHaven(r)
		havenType := getType(rHaven) // Get the haven type of the region.
		regionType := getType(r)     // Get the region type of the region.

		// Ensure only larger lakes will result in the 'lake' culture type.
		if havenType == geo.FeatureTypeLake && m.WaterbodySize[rHaven] > 5 {
			return CultureTypeLake // low water cross penalty and high for growth not along coastline
		}

		// If we have a harbor (more than 1 water neighbor), or are on an island,
		// we are potentially a naval culture.
		if (harborSize > 0 && P(0.1) && havenType != geo.FeatureTypeLake) ||
			(harborSize == 1 && P(0.6)) ||
			(regionType == geo.FeatureTypeIsle && P(0.4)) {
			return CultureTypeNaval // low water cross penalty and high for non-along-coastline growth
		}

		// If we are on a big river (flux > 2*rainfall), we are a river culture.
		if m.IsRegBigRiver(r) {
			return CultureTypeRiver // no River cross penalty, penalty for non-River growth
		}

		// If we are inland (cellType > 2) and in one of the listed biomes,
		// we are a hunting culture.
		if cellType[r] > 2 && (gotBiome == genbiome.AzgaarBiomeSavanna ||
			gotBiome == genbiome.AzgaarBiomeTropicalRainforest ||
			gotBiome == genbiome.AzgaarBiomeTemperateRainforest ||
			gotBiome == genbiome.AzgaarBiomeWetland ||
			gotBiome == genbiome.AzgaarBiomeTaiga ||
			gotBiome == genbiome.AzgaarBiomeTundra || // Tundra is also nomadic?
			gotBiome == genbiome.AzgaarBiomeGlacier) {
			return CultureTypeHunting // high penalty in non-native biomes
		}

		// TODO:
		// - Wildlands?
		// - What culture would have originated in seasonal forests?
		return CultureTypeGeneric
	}
}

/*

func (m *Map) getBiomeCost(cultureCenter int, biome int, cType CultureType) int {
	_, maxElev := minMax(m.r_elevation)
	eleVal := m.r_elevation[cultureCenter] / maxElev
	gotBiome := m.getRBiomeTEMP(cultureCenter, eleVal, maxElev)
	if gotBiome == biome {
		return 10 // tiny penalty for native biome
	}
	if cType == CultureTypeHunting {
		return genbiome.AzgaarBiomeMovementCost[biome] * 5 // non-native biome penalty for hunters
	}
	if cType == CultureTypeNomadic && biome > 4 && biome < 10 {
		return genbiome.AzgaarBiomeMovementCost[biome] * 10 // forest biome penalty for nomads
	}
	return genbiome.AzgaarBiomeMovementCost[biome] * 2 // general non-native biome penalty
}

func (m *Map) getHeightCost(i int, h float64, cType CultureType) int {
	f = pack.features[cells.f[i]]
	a = cells.area[i]
	if cType == CultureTypeLake && f.Type == "lake" {
		return 10 // no lake crossing penalty for Lake cultures
	}
	if cType == CultureTypeNaval && h < 20 {
		return a * 2 // low sea/lake crossing penalty for Naval cultures
	}
	if cType == CultureTypeNomadic && h < 20 {
		return a * 50 // giant sea/lake crossing penalty for Nomads
	}
	if h < 20 {
		return a * 6 // general sea/lake crossing penalty
	}
	if cType == CultureTypeHighland && h < 44 {
		return 3000 // giant penalty for highlanders on lowlands
	}
	if cType == CultureTypeHighland && h < 62 {
		return 200 // giant penalty for highlanders on lowhills
	}
	if cType == CultureTypeHighland {
		return 0 // no penalty for highlanders on highlands
	}
	if h >= 67 {
		return 200 // general mountains crossing penalty
	}
	if h >= 44 {
		return 30 // general hills crossing penalty
	}
	return 0
}

	if r_waterbodies[i] >= 0 {
		if r_waterbody_size[r_waterbodies[i]] > m.mesh.numRegions/25 {
			return "ocean"
		}
		if r_waterbody_size[r_waterbodies[i]] > m.mesh.numRegions/100 {
			return "sea"
		}
		return "gulf"
	}
	  function defineOceanGroup(number) {
	    if (number > grid.cells.i.length / 25) return "ocean";
	    if (number > grid.cells.i.length / 100) return "sea";
	    return "gulf";
	  }

	  function defineIslandGroup(cell, number) {
	    if (cell && features[cells.f[cell - 1]].type === "lake") return "lake_island";
	    if (number > grid.cells.i.length / 10) return "continent";
	    if (number > grid.cells.i.length / 1000) return "island";
	    return "isle";
	  }*/
/*


  // expand cultures across the map (Dijkstra-like algorithm)
  const expand = function () {
    TIME && console.time("expandCultures");
    cells = pack.cells;

    const queue = new PriorityQueue({comparator: (a, b) => a.p - b.p});
    pack.cultures.forEach(function (c) {
      if (!c.i || c.removed) return;
      queue.queue({e: c.center, p: 0, c: c.i});
    });

    const neutral = (cells.i.length / 5000) * 3000 * neutralInput.value; // limit cost for culture growth
    const cost = [];
    while (queue.length) {
      const next = queue.dequeue(),
        n = next.e,
        p = next.p,
        c = next.c;
      const type = pack.cultures[c].type;
      cells.c[n].forEach(function (e) {
        const biome = cells.biome[e];
        const biomeCost = getBiomeCost(c, biome, type);
        const biomeChangeCost = biome === cells.biome[n] ? 0 : 20; // penalty on biome change
        const heightCost = getHeightCost(e, cells.h[e], type);
        const riverCost = getRiverCost(cells.r[e], e, type);
        const typeCost = getTypeCost(cells.t[e], type);
        const totalCost =
          p + (biomeCost + biomeChangeCost + heightCost + riverCost + typeCost) / pack.cultures[c].expansionism;

        if (totalCost > neutral) return;

        if (!cost[e] || totalCost < cost[e]) {
          if (cells.s[e] > 0) cells.culture[e] = c; // assign culture to populated cell
          cost[e] = totalCost;
          queue.queue({e, p: totalCost, c});
        }
      });
    }

    TIME && console.timeEnd("expandCultures");
  };



  function getHeightCost(i, h, type) {
    const f = pack.features[cells.f[i]],
      a = cells.area[i];
    if (type === "Lake" && f.type === "lake") return 10; // no lake crossing penalty for Lake cultures
    if (type === "Naval" && h < 20) return a * 2; // low sea/lake crossing penalty for Naval cultures
    if (type === "Nomadic" && h < 20) return a * 50; // giant sea/lake crossing penalty for Nomads
    if (h < 20) return a * 6; // general sea/lake crossing penalty
    if (type === "Highland" && h < 44) return 3000; // giant penalty for highlanders on lowlands
    if (type === "Highland" && h < 62) return 200; // giant penalty for highlanders on lowhills
    if (type === "Highland") return 0; // no penalty for highlanders on highlands
    if (h >= 67) return 200; // general mountains crossing penalty
    if (h >= 44) return 30; // general hills crossing penalty
    return 0;
  }

  function getRiverCost(r, i, type) {
    if (type === "River") return r ? 0 : 100; // penalty for river cultures
    if (!r) return 0; // no penalty for others if there is no river
    return minmax(cells.fl[i] / 10, 20, 100); // river penalty from 20 to 100 based on flux
  }

  function getTypeCost(t, type) {
    if (t === 1) return type === "Naval" || type === "Lake" ? 0 : type === "Nomadic" ? 60 : 20; // penalty for coastline
    if (t === 2) return type === "Naval" || type === "Nomadic" ? 30 : 0; // low penalty for land level 2 for Navals and nomads
    if (t !== -1) return type === "Naval" || type === "Lake" ? 100 : 0; // penalty for mainland for navals
    return 0;
  }*/
