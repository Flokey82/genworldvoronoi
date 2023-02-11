package genworldvoronoi

import (
	"log"
	"math"
	"sort"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/go_gens/genlanguage"
)

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
	m.resetRand()
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
	climateFitness := m.getFitnessClimate()
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
	// Place n cities of the given type.
	for i := 0; i < n; i++ {
		c := m.PlaceCulture(regCultureFunc, scoreFunc, distSeedFunc)
		log.Printf("placing culture %d: %s", i, c.Name)
	}
}

// ExpandCultures expands the cultures on the map based on their expansionism,
// terrain preference, and distance to other cultures.
func (m *Civ) ExpandCultures() {
	// The cultural centers will be the seed points for the expansion.
	var seeds []int
	originToCulture := make(map[int]*Culture)
	for _, c := range m.Cultures {
		seeds = append(seeds, c.ID)
		originToCulture[c.ID] = c
	}

	rCellType := m.getRegCellTypes()
	_, maxElev := minMax(m.Elevation)
	territoryWeightFunc := m.getTerritoryWeightFunc()
	biomeWeight := m.getTerritoryBiomeWeightFunc()
	m.RegionToCulture = m.regPlaceNTerritoriesCustom(seeds, func(o, u, v int) float64 {
		c := originToCulture[o]

		// Get the cost to expand to this biome.
		gotBiome := m.getAzgaarRegionBiome(v, m.Elevation[v]/maxElev, maxElev)
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
		c.Regions = nil
		for r, cu := range m.RegionToCulture {
			if cu == c.ID {
				c.Regions = append(c.Regions, r)
			}
		}
		c.Stats = m.getStats(c.Regions)
	}
	m.genCultureSpecialties()
}

func (m *Civ) genCultureSpecialties() {
	// Now re-evaluate the specialities of each culture, based on the
	// resources they have access to.

	// NOTE: Someone smarter should come up with the rules for this...
	// and maybe this should also be more generalized so it can be
	// evaluated for all other things that occupy multiple regions like
	// religions, monestaries, city-states, etc. !!!!!!!!!!

	// We calculate the ratio of resources to number of regions, then
	// we assign the specialties based on the highest ratio for each
	// resource per culture.
	// For all resource groups and types, then sort the cultures
	// by the ratio of the resource to the number of regions.
	// The top 3 cultures will get the specialty for that resource?

	// Specialities should give some advantage or bonus for the culture.
	// For example, a culture with the seafaring specialty should get
	// a bonus to naval combat, or a bonus to trade with coastal regions,
	// giving them access to exotic goods.
	// A culture with the survival specialty should get a bonus to
	// survival skills, or a bonus to exploration.

	log.Println("re-evaluating culture specialties... (just a placeholder for now)")

	// Copy the cultures to a slice.
	cultureCopy := make([]*Culture, len(m.Cultures))
	copy(cultureCopy, m.Cultures)

	specialtyMap := make(map[*Culture][]string)

	for _, c := range m.Cultures {
		// Add the default speciality/-ies based on the culture type.
		// TODO: Depending on the culture, there should be several options
		// to select from, also based on the statistics of the regions the
		// culture has access to. (randomized?)
		// For example, naval cultures should only get the seafaring
		// specialty if they have access to a wide coastal regions.
		// Highland cultures should be able to get different survival
		// skills based on the climate of the highlands.
		switch c.Type {
		case CultureTypeWildland:
			specialtyMap[c] = append(specialtyMap[c], "survival")
		case CultureTypeGeneric:
			specialtyMap[c] = append(specialtyMap[c], "generic")
		case CultureTypeRiver:
			// Other possible specialities or bonuses:
			// - hydro power
			// - trading via rivers (?)
			specialtyMap[c] = append(specialtyMap[c], "river navigation")
		case CultureTypeLake:
			specialtyMap[c] = append(specialtyMap[c], "fishery")
		case CultureTypeNaval:
			// Other possible specialities or bonuses:
			// - trade via sea
			specialtyMap[c] = append(specialtyMap[c], "seafaring")
		case CultureTypeNomadic:
			// Other possible specialities or bonuses:
			// - survival
			// - domestication / cattle breeding?
			specialtyMap[c] = append(specialtyMap[c], "nomadic")
		case CultureTypeHunting:
			// Other possible specialities or bonuses:
			// - riding
			specialtyMap[c] = append(specialtyMap[c], "hunting")
		case CultureTypeHighland:
			// Other possible specialities or bonuses:
			// - lower penalty for crossing mountains
			// - mining (?)
			specialtyMap[c] = append(specialtyMap[c], "climbing")
		}
	}

	// Metals.
	for res := 0; res < ResMaxMetals; res++ {
		sort.Slice(cultureCopy, func(i, j int) bool {
			return float64(cultureCopy[i].Stats.ResMetal[res])/float64(len(cultureCopy[i].Regions)) > float64(cultureCopy[j].Stats.ResMetal[res])/float64(len(cultureCopy[j].Regions))
		})
		for i, c := range cultureCopy {
			if i >= 3 {
				break
			}
			specialtyMap[c] = append(specialtyMap[c], metalToString(res))
		}
	}

	// Gems.
	for res := 0; res < ResMaxGems; res++ {
		sort.Slice(cultureCopy, func(i, j int) bool {
			return float64(cultureCopy[i].Stats.ResGems[res])/float64(len(cultureCopy[i].Regions)) > float64(cultureCopy[j].Stats.ResGems[res])/float64(len(cultureCopy[j].Regions))
		})
		for i, c := range cultureCopy {
			if i >= 3 {
				break
			}
			specialtyMap[c] = append(specialtyMap[c], gemToString(res))
		}
	}

	// Stones.
	for res := 0; res < ResMaxStones; res++ {
		sort.Slice(cultureCopy, func(i, j int) bool {
			return float64(cultureCopy[i].Stats.ResStones[res])/float64(len(cultureCopy[i].Regions)) > float64(cultureCopy[j].Stats.ResStones[res])/float64(len(cultureCopy[j].Regions))
		})
		for i, c := range cultureCopy {
			if i >= 3 {
				break
			}
			specialtyMap[c] = append(specialtyMap[c], stoneToString(res))
		}
	}

	// Woods.
	for res := 0; res < ResMaxWoods; res++ {
		sort.Slice(cultureCopy, func(i, j int) bool {
			return float64(cultureCopy[i].Stats.ResWood[res])/float64(len(cultureCopy[i].Regions)) > float64(cultureCopy[j].Stats.ResWood[res])/float64(len(cultureCopy[j].Regions))
		})
		for i, c := range cultureCopy {
			if i >= 3 {
				break
			}
			specialtyMap[c] = append(specialtyMap[c], woodToString(res))
		}
	}

	// Log for each culture their specialties.
	for _, c := range m.Cultures {
		log.Println(c.Name, "specialties:", specialtyMap[c])
	}
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
	// Sophistication float64
	// Extremism      float64 ?
	// Openness       float64 ?
	// Parent    *Culture
	// Children  []*Culture
	// Extinct   bool
	Language *genlanguage.Language // Language of the culture
	Religion *Religion             // Religion of the culture

	// TODO: DO NOT CACHE THIS!
	Regions []int
	*Stats
}

func (c *Culture) Log() {
	log.Printf("The Folk of %s (%s): %d regions", c.Name, c.Type.String(), len(c.Regions))
	log.Printf("Followers of %s (%s)", c.Religion.Name, c.Religion.Group)
	c.Stats.Log()
}

func (m *Civ) newCulture(r int, cultureType CultureType) *Culture {
	lang := GenLanguage(m.Seed + int64(r))
	c := &Culture{
		ID:           r,
		Name:         lang.MakeName(),
		Type:         cultureType,
		Expansionism: cultureType.Expansionism(),
		Martialism:   cultureType.Martialism(),
		Language:     lang,
	}
	c.Religion = m.genFolkReligion(c)
	return c
}

// PlaceCulture places another culture on the map at the region with the highest fitness score.
func (m *Civ) PlaceCulture(regCultureFunc func(int) CultureType, scoreFunc func(int) float64, distSeedFunc func() []int) *Culture {
	// Score all regions, pick highest score.
	var newculture int
	lastMax := math.Inf(-1)
	for i, val := range m.CalcCityScore(scoreFunc, distSeedFunc) {
		if val > lastMax {
			newculture = i
			lastMax = val
		}
	}
	c := m.newCulture(newculture, regCultureFunc(newculture))
	m.Cultures = append(m.Cultures, c)
	return c
}

// PlaceCultureAt places a culture at the given region.
// TODO: Allow specifying the culture type?
func (m *Civ) PlaceCultureAt(r int) *Culture {
	c := m.newCulture(r, m.getRegionCultureTypeFunc()(r))
	c.Regions = []int{r}
	c.Stats = m.getStats(c.Regions)
	m.Cultures = append(m.Cultures, c)
	m.RegionToCulture[r] = r
	// NOTE: This might be quite expensive, so we might want to
	// avoid this calling here, or at least limit the regions
	// we process to the ones that are close to the new culture.
	m.ExpandCultures()
	return c
}

// getRegionCutureTypeFunc returns a function that returns the culture type suitable for a given region.
func (m *Civ) getRegionCultureTypeFunc() func(int) CultureType {
	cellType := m.getRegCellTypes()
	getType := m.getRegionFeatureTypeFunc()
	biomeFunc := m.getRegWhittakerModBiomeFunc()
	_, maxElev := minMax(m.Elevation)

	// Return culture type based on culture center region.
	return func(r int) CultureType {
		eleVal := m.Elevation[r] / maxElev
		gotBiome := m.getAzgaarRegionBiome(r, eleVal, maxElev)
		log.Println(gotBiome)
		log.Println(biomeFunc(r))

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
		rHaven, harborSize := m.getRegHaven(r)
		havenType := getType(rHaven) // Get the type of the haven region.
		regionType := getType(r)
		log.Println(havenType, regionType)

		// Ensure only larger lakes will result in the 'lake' culture type.
		if havenType == FeatureTypeLake && m.WaterbodySize[rHaven] > 5 {
			return CultureTypeLake // low water cross penalty and high for growth not along coastline
		}

		// If we have a harbor (more than 1 water neighbor), or are on an island,
		// we are potentially a naval culture.
		if (harborSize > 0 && P(0.1) && havenType != FeatureTypeLake) ||
			(harborSize == 1 && P(0.6)) ||
			(regionType == FeatureTypeIsle && P(0.4)) {
			return CultureTypeNaval // low water cross penalty and high for non-along-coastline growth
		}

		// If we are on a big river (flux > 2*rainfall), we are a river culture.
		if m.isRegBigRiver(r) {
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

		// TODO: Wildlands?
		// TODO: What culture would have originated in seasonal forests?
		log.Println(gotBiome, gotBiome, gotBiome, gotBiome)
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
