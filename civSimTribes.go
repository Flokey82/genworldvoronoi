package genworldvoronoi

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/go_gens/genlanguage"
	"github.com/Flokey82/go_gens/genreligion"
	goastar "github.com/beefsack/go-astar"
)

// combatSuccessSatisfaction is the value that will be added to the satisfaction of the tribes
// after a combat outcome.
const combatSuccessSatisfaction = 0.5

// settlingSuccessSatisfaction is the value that will be added to the satisfaction of the tribes
// after a settling outcome.
const settlingSuccessSatisfaction = 0.5

// starvationSatisfaction is the value that will be added to the satisfaction of the tribes if people are starving.
const starvationSatisfaction = -0.5

// migrationSuccessSatisfaction is the value that will be added to the satisfaction of the tribes if they migrate.
const migrationSuccessSatisfaction = 0.2

// tribeSplitForcedSatisfaction is the value that will be added to the satisfaction of the tribes if they split
// because of overpopulation.
const tribeSplitForcedSatisfaction = -0.5

// tribeSplitVoluntarySatisfaction is the value that will be added to the satisfaction of the nwe tribe if one splits.
const tribeSplitVoluntarySatisfaction = 0.2

// Convert the distance between two regions to kilometers.
const unitDistToKm = 6371.0 // km

func (m *Civ) InitSimTribes() {
	// Set up the suitability of the regions for population growth.
	m.calculateSuitability()

	// Initialize exhaustion of resources.
	// This will keep track of the exhaustion of resources for each region.
	m.SoilExhaustion = make([]float64, m.NumRegions)

	// Find the best place for the cradle of civilization.
	// Since we only have one species for now (humans), we will just start
	// with a 'steppe' region, and then expand from there incrementally.
	// Now we pick a suitable region to start with (steppe/grassland).
	bestRegion := m.pickCradleOfCivilization(genbiome.WhittakerModBiomeTemperateGrassland)
	if bestRegion == -1 {
		panic("no suitable region found")
	}

	// Initial population.
	const initialPopulation = 100

	m.Tribes = make([]*Tribe, 0, 1)

	// Start with one tribe.
	t := NewTribe(m.getNextTribeID(), bestRegion, initialPopulation)
	m.Tribes = append(m.Tribes, t)
}

func (m *Civ) LogTribes() {
	for _, t := range m.Tribes {
		if t.hasPath() {
			log.Printf("%s f%d -> t%d (%d rem)", t.String(), t.Path.From, t.Path.To, t.Path.NumRemaining())
		} else {
			log.Printf("%s", t.String())
		}
		// Log current leadership.
		if t.Leadership != nil {
			log.Printf("  Leader: %s", t.Leadership.String())
		}
		// Log all factions.
		for _, f := range t.Factions {
			log.Printf("  Faction: %s", f.String())
		}
		// Log all skills.
		for sk := range t.Skills {
			log.Printf("  %s", sk.Name)
		}
		if t.Settlement != nil {
			log.Printf("  Settlement: %s", t.Settlement.String())
		}
		if t.CityState != nil {
			log.Printf("  City state: %s", t.CityState.String())
		}
		if t.Empire != nil {
			log.Printf("  Empire: %s", t.Empire.String())
		}
		// Log preferred biome
		preferredBiome, preferredStrength := t.LastBiomes.Preferred()
		log.Printf("  Preferred biome: %s (%.2f)", preferredBiome, preferredStrength)
		// Log current region max population.
		log.Printf("  Current Max population: %d", t.currentRegionMaxPop)
		log.Printf("  Max population: %d", t.regionMaxPop)

		log.Printf(" Resources (local):")
		localRes := m.getResources(t.RegionID, false)
		localRes.Log()
		log.Printf(" Resources (regional):")
		regRes := m.getResources(t.RegionID, true)
		regRes.remove(localRes).Log()
		log.Printf(" Resouces (storage):")
		t.ResourceStorage.Log()
	}

	// Log the city states.
	for _, cs := range m.CityStates {
		cs.Log()
	}

	// Log the empires.
	for _, e := range m.Empires {
		e.Log()
	}
}

type regionProp struct {
	biome        HackyBiome
	riverProx    bool
	lakeProx     bool
	oceanProx    bool
	mountainProx bool
}

func (r *regionProp) Log() {
	log.Printf("  biome: %s", r.biome)
	log.Printf("  river proximity: %v", r.riverProx)
	log.Printf("  lake proximity: %v", r.lakeProx)
	log.Printf("  ocean proximity: %v", r.oceanProx)
	log.Printf("  mountain proximity: %v", r.mountainProx)
}

type ResourceStorage struct {
	maxStorage int
	Wood       [geo.ResMaxWoods]int
	Stone      [geo.ResMaxStones]int
	Metal      [geo.ResMaxMetals]int
	Gem        [geo.ResMaxGems]int
	Various    [geo.ResMaxVarious]int
}

func newResourceStorage(maxStorage int) *ResourceStorage {
	return &ResourceStorage{
		maxStorage: maxStorage,
	}
}

func (rs *ResourceStorage) add(r localResouces) {
	for _, w := range r.getWood() {
		rs.Wood[w]++
		if rs.Wood[w] > rs.maxStorage {
			rs.Wood[w] = rs.maxStorage
		}
	}
	for _, s := range r.getStones() {
		rs.Stone[s]++
		if rs.Stone[s] > rs.maxStorage {
			rs.Stone[s] = rs.maxStorage
		}
	}
	for _, m := range r.getMetals() {
		rs.Metal[m]++
		if rs.Metal[m] > rs.maxStorage {
			rs.Metal[m] = rs.maxStorage
		}
	}
	for _, g := range r.getGems() {
		rs.Gem[g]++
		if rs.Gem[g] > rs.maxStorage {
			rs.Gem[g] = rs.maxStorage
		}
	}
	for _, v := range r.getVarious() {
		rs.Various[v]++
		if rs.Various[v] > rs.maxStorage {
			rs.Various[v] = rs.maxStorage
		}
	}
}

func (rs *ResourceStorage) Log() {
	log.Printf("  Resources:")
	log.Printf("    Wood:")
	for i, w := range rs.Wood {
		if w > 0 {
			log.Printf("      %s: %d", geo.WoodToString(i), w)
		}
	}
	log.Printf("    Stone:")
	for i, s := range rs.Stone {
		if s > 0 {
			log.Printf("      %s: %d", geo.StoneToString(i), s)
		}
	}
	log.Printf("    Metal:")
	for i, m := range rs.Metal {
		if m > 0 {
			log.Printf("      %s: %d", geo.MetalToString(i), m)
		}
	}
	log.Printf("    Gem:")
	for i, g := range rs.Gem {
		if g > 0 {
			log.Printf("      %s: %d", geo.GemToString(i), g)
		}
	}
	log.Printf("    Various:")
	for i, v := range rs.Various {
		if v > 0 {
			log.Printf("      %s: %d", geo.VariousToString(i), v)
		}
	}
}

type localResouces struct {
	wood, stones, metals, gems, various byte
}

func (r localResouces) isAny() bool {
	return r.wood != 0 || r.stones != 0 || r.metals != 0 || r.gems != 0 || r.various != 0
}

func getResources(res byte, max int) []int {
	var resSlice []int
	for i := 0; i < max; i++ {
		if res&(1<<uint(i)) != 0 {
			resSlice = append(resSlice, i)
		}
	}
	return resSlice
}

func (r localResouces) getGems() []int {
	return getResources(r.gems, geo.ResMaxGems)
}

func (r localResouces) getMetals() []int {
	return getResources(r.metals, geo.ResMaxMetals)
}

func (r localResouces) getStones() []int {
	return getResources(r.stones, geo.ResMaxStones)
}

func (r localResouces) getWood() []int {
	return getResources(r.wood, geo.ResMaxWoods)
}

func (r localResouces) getVarious() []int {
	return getResources(r.various, geo.ResMaxVarious)
}

func (r localResouces) String() string {
	var s string

	resToString := func(label string, res byte, max int, strFunc func(int) string) string {
		if res == 0 {
			return ""
		}
		var resStr string
		resStr += label + ": "
		for i := 0; i < max; i++ {
			if res&(1<<uint(i)) != 0 {
				resStr += strFunc(i) + " "
			}
		}
		return resStr
	}
	s += resToString("Wood", r.wood, geo.ResMaxWoods, geo.WoodToString)
	s += resToString("Stone", r.stones, geo.ResMaxStones, geo.StoneToString)
	s += resToString("Metal", r.metals, geo.ResMaxMetals, geo.MetalToString)
	s += resToString("Gem", r.gems, geo.ResMaxGems, geo.GemToString)
	s += resToString("Various", r.various, geo.ResMaxVarious, geo.VariousToString)
	return s
}

func (r localResouces) add(other localResouces) localResouces {
	r.wood |= other.wood
	r.stones |= other.stones
	r.metals |= other.metals
	r.gems |= other.gems
	r.various |= other.various
	return r
}

func (r localResouces) remove(other localResouces) localResouces {
	r.wood &^= other.wood
	r.stones &^= other.stones
	r.metals &^= other.metals
	r.gems &^= other.gems
	r.various &^= other.various
	return r
}

func (r localResouces) Log() {
	logEntry := func(name string, res byte, max int, strFunc func(int) string) {
		for i := 0; i < max; i++ {
			if res&(1<<uint(i)) != 0 {
				log.Printf("  %s: %s", name, strFunc(i))
			}
		}
	}
	logEntry("Wood", r.wood, geo.ResMaxWoods, geo.WoodToString)
	logEntry("Stone", r.stones, geo.ResMaxStones, geo.StoneToString)
	logEntry("Metal", r.metals, geo.ResMaxMetals, geo.MetalToString)
	logEntry("Gem", r.gems, geo.ResMaxGems, geo.GemToString)
	logEntry("Various", r.various, geo.ResMaxVarious, geo.VariousToString)
}

func (m *Civ) getResources(r int, incNeighbors bool) localResouces {
	var rsc localResouces
	rsc.wood = m.Geo.Resources.Wood[r]
	rsc.stones = m.Geo.Resources.Stones[r]
	rsc.metals = m.Geo.Resources.Metals[r]
	rsc.gems = m.Geo.Resources.Gems[r]
	rsc.various = m.Geo.Resources.Various[r]

	if incNeighbors {
		for _, nb := range m.R_circulate_r(rNbs, r) {
			rsc.wood |= m.Geo.Resources.Wood[nb]
			rsc.stones |= m.Geo.Resources.Stones[nb]
			rsc.metals |= m.Geo.Resources.Metals[nb]
			rsc.gems |= m.Geo.Resources.Gems[nb]
			rsc.various |= m.Geo.Resources.Various[nb]
		}
	}
	return rsc
}

// Take note of the cultures, city states, empires that need to be updated.
type simState struct {
	m             *Civ
	newTribes     []*Tribe
	tribeAtRegion []*Tribe
	isCity        map[int]bool
	cities        []*City
	nodeCache     map[int]*MigrationTile
	// visitedPathSeg will store how often path segments (neighboring regions connected by a trade route)
	// have been used
	visitedPathSeg map[[2]int]int
	maxElevation   float64
	steepness      []float64
	biomeFunc      func(int) int
	arableLandFunc func(int) float64
	climateFunc    func(int) float64
}

// wasVisited returns how often the path segment between the two given regions has been used.
func (s *simState) wasVisited(i, j int) int {
	return s.visitedPathSeg[getSegment(i, j)]
}

func (s *simState) getTile(i int) *MigrationTile {
	// Make sure we re-use pre-existing nodes.
	n, ok := s.nodeCache[i]
	if ok {
		return n
	}

	// If we have no cached node for this index,
	// create a new one.
	n = &MigrationTile{
		steepness:     s.steepness,
		r:             s.m,
		index:         i,
		getTile:       s.getTile,
		wasVisited:    s.wasVisited,
		maxElevation:  s.maxElevation,
		isCity:        s.isCity,
		tribeAtRegion: s.tribeAtRegion,
	}
	s.nodeCache[i] = n
	return n
}

// moveTribe moves the tribe to the new region.
func (s *simState) moveTribe(t *Tribe, r int) {
	// If the tribe is at the old region, remove it.
	if s.tribeAtRegion[t.RegionID] == t {
		s.tribeAtRegion[t.RegionID] = nil
	}
	// Move the tribe to the new region.
	t.RegionID = r
	if s.tribeAtRegion[r] != nil && s.tribeAtRegion[r] != t {
		log.Printf("Tribe %s has moved to region %d, but it is already occupied by tribe %s", t.String(), r, s.tribeAtRegion[r].String())
		panic(fmt.Sprintf("region %d is already occupied by tribe %s", r, s.tribeAtRegion[r].String()))
	}
	s.tribeAtRegion[t.RegionID] = t
}

// switchTribes switches the current regions of the two tribes.
func (s *simState) switchTribes(t1, t2 *Tribe) {
	// Switch the regions of the two tribes.
	region1 := t1.RegionID
	region2 := t2.RegionID
	// HACK: Nil the tribe at the old region.
	// This will prevent a panic in moveTribe.
	// We need to debug duplicate tribes at the same region.
	if s.tribeAtRegion[region1] != t1 {
		panic(fmt.Sprintf("tribe %s is not at region %d", t1.String(), region1))
	}
	if s.tribeAtRegion[region2] != t2 {
		panic(fmt.Sprintf("tribe %s is not at region %d", t2.String(), region2))
	}
	s.tribeAtRegion[region1] = nil
	s.tribeAtRegion[region2] = nil
	s.moveTribe(t1, region2)
	s.moveTribe(t2, region1)
}

// getRegionProp returns the properties of the region.
func (s *simState) getRegionProp(r int) regionProp {
	// TODO: Find better way to determine mountain proximity. 0.3 is very low.
	return regionProp{
		biome:        HackyBiome(s.biomeFunc(r)),
		riverProx:    s.m.IsRegRiver(r),
		lakeProx:     s.m.lakeProxFunc(r),
		oceanProx:    s.m.oceanProxFunc(r),
		mountainProx: s.m.Elevation[r] > 0.3,
	}
}

func (s *simState) printRegionInfo(r int) {
	log.Printf("Region %d:", r)
	rProp := s.getRegionProp(r)
	rProp.Log()

	// Print Gems.
	res := s.m.getResources(r, true)
	res.Log()
}

// getRegMultiplier returns the suitability of the region for the tribe.
// This takes in account the preferences of the tribe.
// A higher overlap between the region properties and the tribe preferences
// will result in a higher multiplier, since their survival chances are higher.
func (s *simState) getRegMultiplier(t *Tribe, r int) float64 {
	return t.compareSuitability(s.getRegionProp(r))
}

// Calculate the economic value of the region.
// TODO: Direct neighbors should also be taken into account.
func (s *simState) getRegEconomicMultiplier(r int) float64 {
	multiplier := 0.0

	// Take into account arable land, which allows for agriculture.
	multiplier = max(multiplier, s.arableLandFunc(r))

	// Alternative we should allow for livestock to be a viable option.
	// TODO: Create a fitness function for livestock in general.
	multiplier = max(multiplier, s.climateFunc(r))
	multiplier = max(multiplier, 0.5) // Minimum value.

	res := s.m.getResources(r, true)

	// Add a bonus for non-essential resources.
	// NOTE: These optional resources should give a base bonus if present.
	if res.metals != 0 {
		val := float64(res.metals) / (1 << geo.ResMaxMetals)
		val = math.Sqrt(val) // Sqrt to make it less linear.
		val += 0.2           // Base bonus.
		val = min(val, 1.0)  // Limit to 1.0.
		multiplier = max(multiplier, val)
	}
	if res.gems != 0 {
		val := float64(res.gems) / (1 << geo.ResMaxGems)
		val = math.Sqrt(val) // Sqrt to make it less linear.
		val += 0.2           // Base bonus.
		val = min(val, 1.0)  // Limit to 1.0.
		multiplier = max(multiplier, val)
	}
	if res.various != 0 {
		val := float64(res.various) / (1 << geo.ResMaxVarious)
		val = math.Sqrt(val) // Sqrt to make it less linear.
		val += 0.2           // Base bonus.
		val = min(val, 1.0)  // Limit to 1.0.
		multiplier = max(multiplier, val)
	}

	// Penalize if the region doesn't have resources the tribe needs.
	// We need wood for fuel, construction, and tools.
	if res.wood == 0 {
		multiplier -= 0.2
	}
	// We need stones for construction and tools.
	if res.stones == 0 {
		multiplier -= 0.2
	}
	return multiplier
}

// Calculate the maximum sustainable population for the region.
//
// This depends on:
// - max population for the region
// - soil exhaustion
// - preferences (via multiplier)
//
// TODO: This should also depend on ...
// - skills
// - culture
// - settlement (nomadic, settled, etc.)
func (s *simState) calcMaxPopPerRegion(t *Tribe, r int) int {
	baseVal := float64(s.m.maxPopReg(r) - int(s.m.SoilExhaustion[r]))
	if baseVal <= 0 {
		return 0
	}
	multiplier := s.getRegMultiplier(t, r)
	return int(baseVal * multiplier)
}

// calcTheoreticalMaxPopPerRegion calculates the theoretical maximum population for the region.
// This is the same as calcMaxPopPerRegion, but without the soil exhaustion.
func (s *simState) calcTheoreticalMaxPopPerRegion(t *Tribe, r int) int {
	baseVal := float64(s.m.maxPopReg(r))
	if baseVal <= 0 {
		return 0
	}
	multiplier := s.getRegMultiplier(t, r)
	return int(baseVal * multiplier)
}

func (s *simState) handleTrade(t *Tribe) {
	// TODO: Figure out what we actually need and what we have in excess.
	const tradeRadius = 900.0 // km

	// Create a list of cities that are close enough to trade with.
	type cityTrade struct {
		city *City
		dist float64
	}
	var tradeCities []*cityTrade
	for _, c := range s.cities {
		if c.ID == t.RegionID {
			continue
		}
		dist := s.m.Geo.GetDistance(t.RegionID, c.ID) * unitDistToKm
		if dist < tradeRadius {
			tradeCities = append(tradeCities, &cityTrade{city: c, dist: dist})
		}
	}

	// Sort the trade cities by distance.
	sort.Slice(tradeCities, func(i, j int) bool {
		return tradeCities[i].dist < tradeCities[j].dist
	})

	// NOTE: This is only about resources.
	type cityTradeProposal struct {
		city *City
		dist float64       // Distance to the city.
		exp  localResouces // Resources that are subject to trade.
		imp  localResouces // Resources that are needed.
	}

	compareResources := func(a, b int) (exp, imp localResouces) {
		// Determine what resources we have and what resources we need.
		aRes := s.m.getResources(a, true)
		bRes := s.m.getResources(b, true)
		imp = bRes.remove(aRes)
		exp = aRes.remove(bRes)
		return
	}

	var tradeProposals []*cityTradeProposal

	// Loop through all the trade cities and propose trades.
	// If we have resources that the other city needs, we propose a trade for export.
	// If the other city has resources that we need, we propose a trade for import.
	for _, tc := range tradeCities {
		// Determine what resources we have and what resources we need.
		exp, imp := compareResources(t.RegionID, tc.city.ID)
		tradeProposals = append(tradeProposals, &cityTradeProposal{
			city: tc.city,
			exp:  exp,
			imp:  imp,
			dist: tc.dist,
		})
	}

	// Log the trade proposals.
	for _, tp := range tradeProposals {
		log.Printf("!!!%s has proposed a trade with %s, dist %.2f", t.String(), tp.city.String(), tp.dist)
		if tp.exp.isAny() {
			log.Printf("  Export: %s", tp.exp.String())
		}
		if tp.imp.isAny() {
			log.Printf("  Import: %s", tp.imp.String())
		}

		score := t.Settlement.compare(tp.city)
		log.Printf("  Score: %.2f", score)
	}
}

func (s *simState) handleResources(t *Tribe) {
	// Here is what we could do:
	// - Feed our people.
	// - Produce weapons for hunting.
	// - Produce tools for crafting.

	// TODO:
	// - Calculate suitability for hunting, gathering, herding, farming, etc.

	biome := HackyBiome(s.biomeFunc(t.RegionID))
	precipitation := s.m.Geo.Rainfall
	_, maxElev := minMax(s.m.Elevation)

	getGatheringScore := func(r int) float64 {
		regSteepness := s.steepness[r]
		regPrecipitation := precipitation[r] * geo.MaxPrecipitation * 100
		regTemperature := s.m.GetRegTemperature(r, maxElev)

		// Gathering is scored from 0 to 1.0.
		// Less steepness is better for gathering.
		// - at 1.0 steepness, the gathering score will be the minimum.
		// - at 0.0 steepness, the gathering score will be (up to) the maximum.
		gatheringSteepnessVal := 1.0 - regSteepness
		// Gathering gets worse with lower precipitation.
		// - at 0mm precipitation, the gathering score will be the minimum.
		// - at 1000mm (and above), the gathering score will be (possibly) the maximum.
		gatheringPrecipitationVal := min(regPrecipitation, 1000) / 1000

		// Gathering gets worse with lower temperature.
		// - at 0°C, the gathering score will be the minimum.
		// - at 30°C, the gathering score will be (possibly) the maximum.
		// - above 30°C, the gathering score will reduce again.
		gatheringTemperatureVal := max(regTemperature, 0) / 30
		if regTemperature > 30 {
			gatheringTemperatureVal = 1.0 - min((regTemperature-30)/30, 1.0)
		}
		// The base value is 0.2.
		regGathering := 0.2 + 0.8*gatheringSteepnessVal*gatheringPrecipitationVal*gatheringTemperatureVal

		// There is a disadvantage in snow, desert, cold desert.
		if biome == genbiome.WhittakerModBiomeSnow || biome == genbiome.WhittakerModBiomeSubtropicalDesert || biome == genbiome.WhittakerModBiomeColdDesert {
			regGathering *= 0.5
		}
		return regGathering
	}

	getHuntingScore := func(r int) float64 {
		regSteepness := s.steepness[r]
		regPrecipitation := precipitation[r] * geo.MaxPrecipitation * 100
		regTemperature := s.m.GetRegTemperature(r, maxElev)
		// Hunting is scored from 0 to 1.0.
		// Less steepness is better for hunting, but the minimum is higher than for gathering.
		// - at 1.0 steepness, the hunting score will be the minimum.
		// - at 0.0 steepness, the hunting score will be (up to) the maximum.
		huntingSteepnessVal := 1.0 - regSteepness
		// Hunting gets a little worse with lower precipitation.
		// - at 0mm precipitation, the hunting score will be a little lower (40%).
		// - at 1000mm, the hunting score will be (possibly) the maximum.
		// - above 1000mm, the hunting score will reduce again.
		huntingPrecepVal := 1.0
		if regPrecipitation <= 1000 {
			huntingPrecepVal = (regPrecipitation / 1000)
		} else {
			huntingPrecepVal = 1.0 - min(((regPrecipitation-1000)/1000), 1.0)
		}
		// Hunting gets a little worse with lower temperature.
		// - at 0°C, the hunting score will be a little lower (40%).
		// - at 30°C, the hunting score will be (possibly) the maximum.
		// - above 30°C, the hunting score will reduce again.
		huntingTempVal := 1.0
		if regTemperature <= 30 {
			huntingTempVal = (max(regTemperature, 0) / 30)
		} else {
			huntingTempVal = 1.0 - min(((regTemperature-30)/30), 1.0)
		}
		// The base value is 0.4.
		regHunting := 0.4 + 0.6*huntingSteepnessVal*huntingPrecepVal*huntingTempVal

		// There is a disadvantage in (dense) forests.
		if biome == genbiome.WhittakerModBiomeTemperateRainforest || biome == genbiome.WhittakerModBiomeTropicalRainforest {
			regHunting *= 0.5
		}
		return regHunting
	}

	regHunting := getHuntingScore(t.RegionID)
	possibleFoodHuntingPerPersonMax := 5.0
	possibleFoodHuntingPerPersonMin := 1.0

	regGathering := getGatheringScore(t.RegionID)
	possibleFoodGatheringPerPersonMax := 3.0
	possibleFoodGatheringPerPersonMin := 1.0

	// Calculate the food production for hunting and gathering.
	foodHuntingPerPerson := possibleFoodHuntingPerPersonMin + (possibleFoodHuntingPerPersonMax-possibleFoodHuntingPerPersonMin)*regHunting
	foodGatheringPerPerson := possibleFoodGatheringPerPersonMin + (possibleFoodGatheringPerPersonMax-possibleFoodGatheringPerPersonMin)*regGathering

	log.Printf("!!!%s is hunting: %.2f, gathering: %.2f.", t.String(), regHunting, regGathering)

	// Calculate how much food the tribe can produce.
	// Half of the population is available for hunting.
	// The rest might be too young, too old.
	//availablePopulationHunter := float64(t.Population) * 0.5

	// 90% of the population is available for gathering since
	// gathering is less dangerous or physically demanding.
	availablePopulationGatherer := float64(t.Population) * 0.9

	// The ratio of the scores will determine how many are spent on hunting and how many on gathering.
	// The rest will be spent on farming.
	// huntingToGatheringRatio := regHunting / regGathering

	// TODO: Also weigh the composition by the preference of the tribe.
	popHunting := float64(t.Population) / (foodHuntingPerPerson + availablePopulationGatherer - foodGatheringPerPerson)
	popGathering := availablePopulationGatherer - popHunting
	foodHunting := popHunting * foodHuntingPerPerson
	foodGathering := popGathering * foodGatheringPerPerson

	// Calculate the food production for gathering.

	log.Printf("!!!%s is producing food: %d total; %d from hunting (%d people), %d from gathering (%d people).", t.String(), int(foodHunting+foodGathering), int(foodHunting), int(popHunting), int(foodGathering), int(popGathering))

	// Produce food, mainly to feed our population, but also keep some in storage.
	// m.Suitability[t.RegionID]

	// Check if we have enough resources to build what we want to build.

	// How do we know what we need?
	// - Ongoing consumption of goods and resources?
	// - What do we need to produce?
	// - What do we need to trade?

	// TODO: We also need a resource sink. What do we spend resources on?
	// - Food
	// - Tools
	// - Weapons
	// - Buildings
	// - Trade
	// - Diplomacy

	// Produce goods?
	// - Tools
	// - Weapons
	// - Buildings

	// Generate resources based on the local resources.
	// If we aren't settled, we only can gather resources from the region we are in.
	if t.Type == TribeTypeNomadic {
		res := s.m.getResources(t.RegionID, false)
		t.ResourceStorage.add(res)
		return
	}
	// TODO: Introduce storage for settlements, city states, and empires.
	res := s.m.getResources(t.RegionID, true)
	t.ResourceStorage.add(res)
}

// HANDLE NOMADIC TRIBES HERE.
//
// Nomadic tribes will move around and settle in different regions, depending on their preferences
// and suitability of the regions (what population can be sustained in the region).
//
// If the most prosperous neighbor region cannot sustain the entire tribe, the tribe will split into
// two or more tribes.
func (s *simState) handleNomadicTribe(t *Tribe) {
	// We look at all neighboring regions and move to the most suitable region.
	// Considerations:
	// - We do not move into a region that is already occupied.
	// - We do not move into a region that is not suitable for the tribe.

	// Calculate the max population for each neighboring region.
	neigbors := s.m.R_circulate_r(rNbs, t.RegionID)
	maxPopPerRegion := make([]int, len(neigbors))
	neighborIndices := make([]int, len(neigbors))
	for i, nb := range neigbors {
		neighborIndices[i] = i
		// We only consider regions that are not ocean regions and are unoccupied.
		if s.m.Elevation[nb] > 0 && s.tribeAtRegion[nb] == nil {
			maxPopPerRegion[i] = s.calcMaxPopPerRegion(t, nb)
		}
	}

	// Sort the neighbor indices by max population.
	sort.Slice(neighborIndices, func(i, j int) bool {
		return maxPopPerRegion[neighborIndices[i]] > maxPopPerRegion[neighborIndices[j]]
	})

	// Start with the most suitable region and check if the entire tribe can move there.
	// If not, we need to split the tribe into two tribes and move the remaining tribe to the next suitable region.
	tCurrent := t
	for _, idx := range neighborIndices {
		nb := neigbors[idx]
		if s.tribeAtRegion[nb] != nil {
			continue // The region is already occupied.
		}

		// Get the max population for the region.
		nbMaxPop := maxPopPerRegion[idx]
		if nbMaxPop <= 0 {
			break // We ran out of suitable regions.
		}

		// Check if the entire tribe can move to the new region.
		if nbMaxPop >= tCurrent.Population {
			// Move the tribe here and be done.
			s.moveTribe(tCurrent, nb)
			// TODO: The satisfaction should depend on the prosperity of the region
			// copared to the current region, etc.
			tCurrent.changeSatisfaction(migrationSuccessSatisfaction)
			s.newTribes = append(s.newTribes, tCurrent)
			tCurrent = nil
			break
		}

		// The max population for the region is less than the population of the tribe.
		// Get the number of people that we leave behind (a minimum of 50 people, if the tribe is large enough).
		diff := min(max(50, tCurrent.Population-nbMaxPop), tCurrent.Population/2)
		log.Println("Tribe ", tCurrent.String(), "is splitting into two tribes. Population:", tCurrent.Population, "Max population:", nbMaxPop, "Diff:", diff)

		// Split the tribe into two tribes. The new tribe will be placed in the original region.
		newTribe := tCurrent.Split(s.m.getNextTribeID(), diff)

		// Move the remaining (the original) tribe to the new region
		// and the new tribe to the original region.
		currentReg := tCurrent.RegionID
		s.moveTribe(tCurrent, nb)
		s.moveTribe(newTribe, currentReg)

		// TODO: Make these constants.
		newTribe.changeSatisfaction(tribeSplitForcedSatisfaction)
		tCurrent.changeSatisfaction(tribeSplitForcedSatisfaction)

		s.newTribes = append(s.newTribes, tCurrent)

		// Set the new tribe as the current tribe.
		tCurrent = newTribe
	}

	// If there is still remaining population, check if they can survive in the current region.
	if tCurrent != nil {
		log.Println("Tribe ", tCurrent.String(), "is trying to survive in the current region. Population:", tCurrent.Population, "Max population:", t.currentRegionMaxPop)

		// Check if the tribe can survive in the current region (or at least part of it can survive).
		if maxPop := s.calcMaxPopPerRegion(t, tCurrent.RegionID); maxPop <= 0 {
			log.Println("Tribe ", tCurrent.String(), "has died out.")

			// TODO: Optionally attack other tribes to move into their regions.
			if s.tribeAtRegion[tCurrent.RegionID] == tCurrent {
				s.tribeAtRegion[tCurrent.RegionID] = nil
			} else {
				// DEBUG: Check if the tribe is in the right region.
				log.Println("Tribe ", tCurrent.String(), "is in the wrong region?????!!!")
			}
		} else {
			// Check if the tribe lost some of its population.
			if maxPop < tCurrent.Population {
				t.changeSatisfaction(starvationSatisfaction)
				log.Println("Tribe ", tCurrent.String(), " lost some of its population.", tCurrent.Population-maxPop, "people died.", maxPop, "people survived.")
				tCurrent.Population = maxPop
			} else {
				t.changeSatisfaction(migrationSuccessSatisfaction)
			}

			// DEBUG: Check if the tribe is in the right region.
			if s.tribeAtRegion[tCurrent.RegionID] != tCurrent {
				log.Println("Tribe", tCurrent.String(), "is in the wrong region.")
			}

			// Re-assign the tribe to the region.
			s.moveTribe(tCurrent, tCurrent.RegionID)

			// We can survive in the current region, retain the tribe.
			s.newTribes = append(s.newTribes, tCurrent)
		}
	}
}

// findNewPath will find a new path to the destination region and assign it to the tribe.
func (s *simState) findNewPath(t *Tribe, destination int) bool {
	newPath, found := planPath(s.getTile(t.RegionID), s.getTile(destination))
	if found {
		if len(newPath.Steps) == 1 {
			panic("path length is 1")
		}
		t.SetPath(newPath)
	}
	return found
}

// findNewRegionToSettle will find a new region to settle in and set a new path to the region.
func (s *simState) findNewRegionToSettle(t *Tribe, avoidRegs []int) bool {
	// Find the best region to settle in.
	settleRegionScore := s.getSettleScoreFunc(t, t.gotVision)
	bestRegion, score := s.findBestRegion(t, t.RegionID, settleRegionScore, avoidRegs)
	if bestRegion == -1 {
		log.Println("!!!Tribe", t.ID, "couldn't find a suitable region to settle in.")
		return false // Couldn't find a suitable region to settle in.
	}
	log.Println("!!!Tribe", t.ID, "has found a region to settle in:", bestRegion, "with a score of", score)

	// We have found a new region to settle in, so plot a path to the region
	// and return the outcome of the pathfinding.
	s.printRegionInfo(bestRegion)
	t.printTribePreferences()
	if !s.findNewPath(t, bestRegion) {
		log.Println("!!!Tribe", t.ID, "couldn't find a path to the new region.")
		return false
	}
	return true
}

func (s *simState) handleSettlingTribe(t *Tribe) {
	// Check if the tribe has a path set. If not, we need to find a suitable region to settle in.
	if !t.hasPath() {
		t.gotVision = rand.Intn(100) < 99
		// Check if we could find a suitable region to settle in.
		if !s.findNewRegionToSettle(t, nil) {
			log.Println("Tribe", t.ID, "couldn't find a new region to settle or a path to the new region.")
		}
	}

	// Couldn't find a path to a new region, so we need to try again next turn.
	if !t.hasPath() {
		return
	}

	// We have a path set, so we need to move the tribe to the next region in the path.
	// In general, if we want to move to the next region, check if the region is already occupied.
	//
	// So we just "peek" at the next region to see if it's occupied and either:
	// - choose combat
	// - find a way around
	//
	// If the next region would be our destination and it is occupied we can:
	// - choose combat
	// - merge with the other tribe
	// - find a new region
	nextRegion := t.Path.Peek()

	// fightForRegion will execute a fight between the attacking and defending tribe.
	// If the attacking tribe wins, the function will return true.
	fightForRegion := func(attacking, defending *Tribe) bool {
		// TODO:
		// - The defending tribe should have a higher chance of inflicting losses on the attacking tribe.
		// - The losses should depend on the population of the opponent.
		// - The losses should depend on the culture of the tribes.
		// - The losses should depend on the skills of the tribes.

		// Calculate the losses for each tribe.
		lossesDefendingTribe := rand.Intn(defending.Population)
		defending.Population -= lossesDefendingTribe

		lossesAttackingTribe := rand.Intn(attacking.Population)
		attacking.Population -= lossesAttackingTribe

		// The tribe with the largest remaining population wins.
		return attacking.Population > defending.Population
	}

	// chooseCombat will determine if the tribe will choose to fight for the region.
	chooseCombat := func(isDestination bool, attacking, defending *Tribe) bool {
		// Determine if the tribe will sack the settlement at the destination
		// if there is a tribe settled there and it is our destination.
		disallowSacking := false

		// For now we don't allow sacking of capitals.
		if defending.CityState != nil && defending.CityState.Capital == defending.Settlement ||
			defending.Empire != nil && defending.Empire.Capital == defending.Settlement {
			log.Printf("Tribe %d is trying to sack the capital of tribe %d.", attacking.ID, defending.ID)
			return false
		}

		// Check if the defending tribe has settled there:
		if defending.Type > TribeTypeSettling && defending.doneSettling && (disallowSacking || !isDestination) {
			return false
		}

		// Depending on our aggressiveness, the strength of the other tribe,
		// and if this is our destination, we might choose to fight for the region.
		multiplier := 1.0

		// This is where we want to settle.
		if isDestination {
			multiplier += 0.1
		}

		// Religion might be a powerful motivator for combat.
		// TODO: Some religions might be peaceful, while others might be more aggressive.
		if attacking.gotVision {
			multiplier += 0.1
		}
		// Consider culture.
		// TODO: Consider expansionism, aggressiveness, etc.

		// We multiply our own strength by the multiplier.
		confidence := float64(attacking.Population) * multiplier
		return confidence > float64(defending.Population)
	}

	// Check if the next region is already occupied and if the occupying tribe is not us.
	if occupier := s.tribeAtRegion[nextRegion]; occupier != nil && occupier != t {
		var success bool

		// Check if the next region is the destination.
		isDestination := t.Path.PeekDone()

		// Check if occupier has settled in the region.
		isSettled := occupier.Settlement != nil && occupier.Settlement.ID == nextRegion

		// We might want to fight for the region.
		// If this is the destination, we might be more inclined to fight for the region.
		if chooseCombat(isDestination, t, occupier) {
			// Try to fight for the region.
			// TODO:
			// - Check if either tribe has died out after the fight.
			// - If we have won, we need to dislodge the other tribe,
			//   if it still exists.
			// - Store new opinion of the tribes about each other.
			if success = fightForRegion(t, occupier); success {
				// If we have succeeded, the tribe at the destination will be moved to our current one
				// and we will move to the destination.
				log.Println("$$$$$$$$$$$$$$Tribe", t.ID, "has fought for region", nextRegion, "and won against tribe", occupier.ID)

				// Change satisfaction of the tribes.
				t.changeSatisfaction(combatSuccessSatisfaction)
				occupier.changeSatisfaction(-combatSuccessSatisfaction)

				// Move the other tribe to the current region.
				s.switchTribes(t, occupier)

				// We have defeated the other tribe, so we can settle in the region and
				// take over the settlement if there is one.
				if (isDestination && occupier.Settlement != nil) != isSettled {
					panic("isDestination and isSettled are not the same")
				}
				if isDestination && isSettled {
					// If the other tribe has settled here at our destination, we will take over the settlement.
					// TODO:
					// - Depopulate the settlement (with the occupiers being moved out of the region)
					// - Optionally rename the settlement.
					// - Instead of assigning it here, we should check for an existing settlement
					//   in the region further below where we would establish a new settlement.
					t.Settlement = occupier.Settlement
					t.Settlement.Name += " (captured)"
					t.Settlement.Culture = t.Culture
					t.Settlement.Religion = t.Religion
					t.Settlement.Population = t.Population

					// TODO: What if the city that we've captured is the capital of a city state or empire?
					if occupier.CityState != nil || occupier.Empire != nil {
						panic("captured city is the capital of a city state or empire")
					}

					// The other tribe has lost its settlement.
					occupier.Settlement = nil
					occupier.doneSettling = false
					occupier.Type = TribeTypeSettling
					log.Printf("Tribe %d has sacked the settlement of tribe %d.", t.ID, occupier.ID)

					// Try to find a new home for the other tribe.
					log.Println("TODO: avoid moving back to the region where we were just kicked out of")
					if !s.findNewRegionToSettle(occupier, []int{t.RegionID, occupier.RegionID, nextRegion}) {
						// If the other tribe couldn't find a new region to settle in, it will become nomadic.
						occupier.makeNomadic()
					}
				} else if occupier.hasPath() {
					// If the other tribe has a path, we need to find a new path for it.
					if !s.findNewPath(occupier, occupier.Path.To) {
						// If the other tribe couldn't find a new path to the destination, it will become nomadic.
						occupier.makeNomadic()
					}
				}
			} else {
				log.Println("$$$$$$$$$$$$$$Tribe", t.ID, "has fought for region", nextRegion, "and lost against tribe", occupier.ID)
				// We have lost the fight for the region.
				t.changeSatisfaction(-combatSuccessSatisfaction)
				occupier.changeSatisfaction(combatSuccessSatisfaction)
			}
		}

		if !success {
			// If we didn't succeed in fighting for the region (or didn't want to fight for it),
			// we need to find a new region to settle in, or find a way around the region that
			// is already occupied.
			if isDestination {
				log.Println("no success... trying to find a new region to settle in")
				// The occupied region is our destination, so we will have to find a new one.
				success = s.findNewRegionToSettle(t, []int{nextRegion, t.RegionID, occupier.RegionID})
				if success {
					log.Println("$$$$$$$$$$$$$$Tribe", t.ID, "has found a new region to settle in as tribe", occupier.ID, "has occupied the region", nextRegion, ".")
				} else {
					log.Println("$$$$$$$$$$$$$$Tribe", t.ID, "failed to find a new region to settle in as tribe", occupier.ID, "has occupied the region", nextRegion, ".")
				}
			} else {
				log.Println("no success... trying to find a new path to settle")
				// Find a way around the region, or find a new region.
				// TODO: The pathfinding should avoid regions that are occupied by other tribes.
				// ... but if no path avoiding other tribes can be found, we might need to fight for the region.
				success = s.findNewPath(t, t.Path.To)
				log.Println("$$$$$$$$$$$$$$Tribe", t.ID, "has found a new path to region", t.Path.To, "in", len(t.Path.Steps), "steps as tribe", occupier.ID, "has occupied the region", nextRegion, ".")
			}
		}

		// If we didn't succeed in dislodging the other tribe or in finding a new region to settle in.
		// What to do? Stay in the current region?
		if !success {
			log.Println("$$$$$$$$$$$$$$Tribe", t.ID, "failed to move to region", nextRegion, " and will stay in region", t.RegionID, " as tribe", occupier.ID, "has occupied the region", nextRegion, ".")
			return
		}
	}

	// Move the tribe to the next region in the path.
	nextRegion = t.Path.Next()
	if s.tribeAtRegion[nextRegion] != nil && s.tribeAtRegion[nextRegion] != t {
		log.Println("NOTE: Tribe", t.ID, "wants to move to region", nextRegion, "but it's already occupied by tribe", s.tribeAtRegion[nextRegion].ID, ".")
		panic("tribe wants to move to an occupied region")
	}

	// If the path is complete (we arrived at the destination),
	// check if there is still someone in the region.
	if t.Path.Done() && s.tribeAtRegion[nextRegion] != nil && s.tribeAtRegion[nextRegion] != t {
		// TODO: This value should depend on the tribe's preferences.
		// (if it is a spiritual calling, the satisfaction should be higher, etc.)
		t.changeSatisfaction(-settlingSuccessSatisfaction)

		// The region is already occupied, so we reset the path and try finding a different region.
		// TODO:
		// - Add option to fight for the region.
		// - Add option to merge with the other tribe.
		// - Make the tribe unhappy, etc.
		t.SetPath(nil)
		log.Println("Tribe", t.ID, "couldn't settle in region", nextRegion, "because it's already occupied by", s.tribeAtRegion[nextRegion].ID, ".")
	} else {
		// What if there is already a tribe in the region that we want to move through?
		if s.tribeAtRegion[nextRegion] != nil && s.tribeAtRegion[nextRegion] != t {
			// TODO: We'd need to either find a way around, or fight for the region.
			log.Println("Tribe", t.ID, "at", t.RegionID, "is moving through region", nextRegion, "which is already occupied by tribe", s.tribeAtRegion[nextRegion].ID, ".")
		}
		s.moveTribe(t, nextRegion)
		log.Println("Tribe", t.ID, "is moving to region", nextRegion, "in order to settle.")

		// Check if the tribe has arrived at the new region.
		if t.Path.Done() {
			// TODO: This value should depend on the tribe's preferences.
			// (if it is a spiritual calling, the satisfaction should be higher, etc.)
			t.changeSatisfaction(settlingSuccessSatisfaction)

			// Get the culture type that we should have in this region.
			ctf := s.m.getRegionCultureTypeFunc() // <- VERY COSTLY!!!
			regionType := ctf(t.RegionID)

			// Get the existing culture of the region.
			existingCulture := s.m.GetCulture(t.RegionID)

			// If the tribe did not have yet its own culture, we either adopt the existing culture
			// of the region, or we create a new culture based on the culture type of the region.
			if t.Culture == nil {
				if existingCulture == nil || existingCulture.Type != regionType {
					// No existing culture in the region or the existing culture is not of the right type
					// for the current region.
					t.Culture = s.m.PlaceCultureAt(t.RegionID, false, t.Language)
				} else {
					// We adopt the existing culture of the region.
					log.Println("Tribe", t.ID, "is adopting the culture of the region", t.RegionID, ".!!!!!!!!!!!!!!!!!!!!1")
					t.Culture = existingCulture
				}
			} else if existingCulture != t.Culture {
				// We are settling in a region that has a different culture than our own,
				// so we retain our own, and fork the culture that we brought with us,
				// which is the culture of the parent tribe.

				// Fork culture.
				// TODO:
				// - Make this by random chance and distance to the original culture.
				// - Make sure that there is no culture that has this ID.
				t.Culture = s.m.AddCultureAt(t.RegionID, t.Culture.Fork(t.RegionID), false)
				t.Culture.SetNewType(regionType) // Set the new type of the culture.
			} else {
				// We are settling in a region that has the same culture as our own.
				log.Println("Tribe", t.ID, "is adopting the culture of the region", t.RegionID, ".!!!!!!!!!!!!!!!!!!!!!!2")
			}

			// Update the type of the new culture based on the preferences of the tribe.
			// TODO: Update all preferences to match the new location.
			log.Println("Tribe", t.ID, "with a culture of", t.GetCultureType(), "should have a culture of", regionType)

			// Check if there is already a settlement in the region and either update it or create a new one.
			if t.Settlement == nil {
				t.Settlement = s.m.placeCityAt(t.RegionID, -1, s.m.getRegCityType(t.RegionID), t.Population, 1.0)
			} else {
				t.Settlement.Culture = t.Culture
				t.Settlement.Religion = t.Religion
				t.Settlement.Population = t.Population
			}

			// Set the tribe as a settled tribe.
			if t.CityState != nil || t.Empire != nil {
				panic("Tribe is a city state or empire")
			}
			t.doneSettling = true
			t.Type = TribeTypeCity

			// If we found this region through the gods, we establish a new religion, either
			// as a variant of the original religion or as a new, independent religion.
			if t.gotVision {
				// This will generate a religion based on the culture and the original religion of the "parent tribe"
				// and update the religion of the tribe's culture.
				// TODO: We should create a new religion as soon as we receive a vision from the gods
				// not only when we settle.
				group := genreligion.GroupFolk
				if t.Religion != nil {
					group = t.Religion.Group
				}
				t.Religion = s.m.placeReligionAt(t.RegionID, -1, group, t.Culture, t.Culture.Language, t.Religion)
				t.Culture.Religion = t.Religion
				log.Println("Tribe", t.ID, "has received a vision from the gods and settled in region", t.RegionID, "and follows the religion of", t.Religion.String())
			} else {
				log.Println("Tribe", t.ID, "has settled in region", t.RegionID, "with a population of", t.Population, "and a culture of", t.Culture.Type)
			}
		}
	}
}

func (s *simState) handleCity(t *Tribe) {
	log.Printf("!!!%s has settled in region %d.", t.String(), t.RegionID)

	// Do city stuff.
	// Diplomacy, trade, defense, etc.
	// - We create proposals and send them to other cities, city states, etc.
	//
	// Types of proposals:
	// - Trade proposals (resources, etc.)
	//   - Determine what resources we have and what resources we need.
	// - Diplomatic proposals (alliances, etc.)
	//   - We can propose alliances, non-aggression pacts, etc.
	//   - We can ask the nest higher level of government to protect us.
	//   - We can ask other cities for alliances, etc. (How do we decide who to ask?)
	// - Military proposals (defense, etc.)
	//   - We can propose to other cities to join us in a war.
	//   - We can propose to other cities to defend us.
	s.handleTrade(t)

	if t.Culture == nil {
		log.Printf("!!!Tribe %d has no culture.", t.ID)
	}

	// Now, if we are big and prosperous enough, we can establish a city state.
	if t.Population > 1000 {
		// Promote to a city state.
		t.Type = TribeTypeCityState
		log.Println("!!!Tribe", t.ID, "has become a city state.")
		// Set up a city state.
		t.CityState = s.m.PlaceCityStateAt(t.RegionID, t.Settlement)
	}
}

func (s *simState) handleCityState(t *Tribe) {
	log.Printf("!!!%s is a city state in region %d.", t.String(), t.RegionID)
	// TODO:
	// If we are big and prosperous, try to negotiate protection, trade, etc. with other city states.
	// We can either try to join an empire that already exists, or try to form a new empire by
	// negotiating other city states to join us.
	s.handleTrade(t)

	// TODO: Do trade, diplomacy, etc.
	// - We create proposals and send them to other city states.
	// - Proposals are resolved in the next iteration.
	// - Handle proposals from the previous iteration.
	for _, c := range t.CityState.Cities {
		if c == t.CityState.Capital {
			continue
		}
		log.Printf("!!!%s has a city: %s.", t.String(), c.String())
	}

	// Get neighboring city states.
	// - We can propose alliances, trade, etc.
	// - Potentially attack other city states.
	for _, nbStateID := range s.m.getCityStateNeighbors(t.CityState) {
		log.Printf("!!!%s has a neighboring city state: %d.", t.String(), nbStateID)
		nbState := s.m.GetCityState(nbStateID)
		if nbState == nil {
			continue
		}
		log.Printf("!!!%s has a neighboring city state: %s (score %.2f)", t.String(), nbState.String(), t.CityState.compare(nbState))
	}

	log.Println("!!!Tribe", t.ID, "is a city state.", t.CityState.Capital.String())

	if t.Population > 2000 {
		// Promote to an empire.
		t.Type = TribeTypeEmpire
		log.Println("!!!Tribe", t.ID, "has become an empire.")
		t.Empire = s.m.placeEmpireAt(t.RegionID, t.Settlement)
	}
}

func (s *simState) handleEmpire(t *Tribe) {
	log.Printf("!!!%s has become an empire.", t.String())
	// Find neighboring empires.
	for _, nbEmpireID := range s.m.getEmpireNeighbors(t.Empire) {
		log.Printf("!!!%s has a neighboring empire: %d.", t.String(), nbEmpireID)
		nbEmpire := s.m.GetEmpire(nbEmpireID)
		if nbEmpire == nil {
			continue
		}
		log.Printf("!!!%s has a neighboring empire: %s (score %.2f)", t.String(), nbEmpire.String(), t.Empire.compare(nbEmpire))
	}

	// Find all neighboring city states that are not part of another empire.
	for _, nbState := range s.m.getEmpireCityStateNeighbors(t.Empire) {
		if s.m.getCityStateEmpire(nbState) == -1 {
			log.Printf("!!!%s has a neighboring city state: %s.", t.String(), nbState.String())
		}
	}
}

// Outline:
// - No path set:
//   - Find a suitable region to settle in and set a path to the region.
// - Path set:
//   - Peek at the next region in the path and see if it is unoccupied.
//   - If it is unoccupied:
//     - Move to the region.
//   - If it is occupied:
//     - We might fight the tribe for the region.
//     - If it is the destination and we can't defeat the tribe or decide not to fight:
//       - Find a new region to settle
//       - Set a path to the new region.
//     - If it is not the destination:
//       - Find a new path to the destination.
// - If we can move to the next region in the path and it is the destination:
//   - Move to the region.
//   - Settle in the region.

// If the tribe has no path set, we need to scout for the best region
// to settle in within a radius and set a path to the new region.
func (s *simState) getSettleScoreFunc(t *Tribe, spiritualCalling bool) func(r int) (float64, bool) {
	const (
		radiusScout       = 300.0  // km
		radiusGods        = 1000.0 // km
		radiusMaxCityProx = 300.0  // km
	)

	// If the region is determined by the gods, we increase the radius to find
	// the most suitable region (TODO: which should stay unoccupied for a while).
	radius := radiusScout
	if spiritualCalling {
		radius = radiusGods
	}
	log.Println("!!!Tribe", t.ID, "is scouting for a new region to settle in within a radius of", radius, "km.")

	settleRegionScore := func(r int) (float64, bool) {
		// Make sure we don't rate the current region too highly.
		if r == t.RegionID {
			return 0, true
		}

		// Check if there is already a settlement in the region.
		/*
			for _, c := range cities {
				if c.ID == r {
					// DEBUG: Encourage settling in regions with cities so we can test the combat system.
					if c.ID != t.RegionID {
						return 100000, true
					}
					return 0, true
				}
			}
		*/

		// Get the distance to the current region in km.
		dist := s.m.Geo.GetDistance(t.RegionID, r) * unitDistToKm // TODO: Make this a method of the Geo struct.
		if dist > radius {
			log.Printf("Tribe %d has scouted a region %d, but it's too far away (%.2f km).", t.ID, r, dist)
			return 0, false
		}
		log.Printf("Tribe %d has scouted a region %d, and the distance is %.2f km.", t.ID, r, dist)

		// Determine the distance to the closest city.
		// NOTE: In theory we could use a distance field to determine the distance to the closest tribe / city
		// and encourage settling in new regions.
		minDist := math.Inf(1)
		for _, c := range s.cities {
			if dist := s.m.Geo.GetDistance(c.ID, r); dist < minDist {
				minDist = dist
			}
		}
		minDist = min(radiusMaxCityProx, minDist*unitDistToKm) // Cap the distance to 300 km (for now)
		log.Printf("Tribe %d has scouted a region %d, and the distance to the closest city is %.2f km.", t.ID, r, minDist)

		// Calculate the max population for the region and see if it's the best region to settle in
		// and add bonus if the region has similar attributes to the tribe's preferences.
		// Also cap the maxPopPerRegion to twice the current population of the tribe.
		maxPopPerReg := float64(min(s.calcMaxPopPerRegion(t, r), 2*t.Population))
		regMul := s.getRegMultiplier(t, r)

		// TODO: The economic score should depend on the skills of the tribe.
		// For example, anything immediately useful to the tribe should be rated higher,
		// while luxury goods should be secondary to the economic value, especially if
		// the tribe doesn't have the skills to extract the resources.
		ecoMul := s.getRegEconomicMultiplier(r)
		val := maxPopPerReg * regMul * ecoMul * minDist / radiusMaxCityProx

		// Disincentivize settling in regions that are already settled by a culture.
		if s.m.RegionToCulture[r] != -1 {
			val /= 1.2
		}

		// Check if there is already a settlement in the region.
		for _, c := range s.cities {
			if c.ID == r {
				val /= 2
				break
			}
		}

		log.Printf("Tribe %d has scouted a region %d, and the score is %.2f. regMul %.2f", t.ID, r, val, regMul)
		return val, true
	}
	return settleRegionScore
}

// findBestRegion will find the best region according to the score function.
func (s *simState) findBestRegion(t *Tribe, r int, scoreFunc func(r int) (float64, bool), avoidRegs []int) (int, float64) {
	log.Println("!!!Tribe", t.ID, "is scouting for a new region to settle in.")

	bestRegion := -1
	bestScore := math.Inf(-1)
	var visitNeigbors func(int)
	seenNeigbors := make(map[int]bool)
	avoid := make(map[int]bool)
	for _, r := range avoidRegs {
		avoid[r] = true
	}
	visitNeigbors = func(r int) {
		if seenNeigbors[r] {
			return
		}
		seenNeigbors[r] = true
		if !avoid[r] {
			score, ok := scoreFunc(r)
			if !ok {
				return
			}
			if s.m.Elevation[r] > 0 && score > bestScore {
				bestScore = score
				bestRegion = r
			}
		}
		// NOTE: Do not re-use rNbs since the memory is shared and the subsequent calls to
		// visitNeigbors will overwrite the contents of rNbs.
		for _, nb := range s.m.R_circulate_r(nil, r) {
			if !seenNeigbors[nb] {
				visitNeigbors(nb)
			}
		}
	}
	visitNeigbors(r)
	log.Println("!!!Tribe", t.ID, "has found a region to settle in:", bestRegion, "with score", bestScore)
	if bestScore < 0 || bestRegion == -1 {
		log.Println("!!!Tribe", t.ID, "has not found a suitable region to settle in.")
		panic("no suitable region found")
	}
	return bestRegion, bestScore
}

func (s *simState) checkRandomSplit(t *Tribe) {
	// TODO: Evaluate the happiness of the tribe and if it's unhappy, it might
	// split and a new tribe might try to settle in a different region.
	if rand.Intn(1000) < 2 && t.Population > 100 {
		// Make sure the new tribe will be picked up in the next iteration.
		newTribe := t.RandomSplit(s.m.getNextTribeID())

		// Move the new tribe to a new region.
		var foundNewRegion bool
		for _, nb := range s.m.R_circulate_r(rNbs, t.RegionID) {
			// Do not place them in oceans or lakes.
			if s.m.IsRegLakeOrWaterBody(nb) || s.m.Elevation[nb] <= 0 {
				continue
			}
			if s.tribeAtRegion[nb] == nil {
				s.moveTribe(newTribe, nb)
				foundNewRegion = true
				break
			}
		}
		if !foundNewRegion {
			panic("no region found for new tribe")
		}
		s.newTribes = append(s.newTribes, newTribe)
	}
}

// Fake the lake proximity function.
func (m *Civ) lakeProxFunc(r int) bool {
	for _, nb := range m.R_circulate_r(rNbs, r) {
		if m.IsRegLakeOrWaterBody(nb) && m.WaterbodySize[nb] > 5 {
			return true
		}
	}
	return false
}

// Fake the ocean proximity function.
func (m *Civ) oceanProxFunc(r int) bool {
	for _, nb := range m.R_circulate_r(rNbs, r) {
		if m.Elevation[nb] <= 0 && m.WaterbodySize[nb] > 5 {
			return true
		}
	}
	return false
}

func (m *Civ) tickSimTribes() {
	_, maxElevation := minMax(m.Elevation)
	s := simState{
		m:              m,
		newTribes:      make([]*Tribe, 0, len(m.Tribes)),
		tribeAtRegion:  make([]*Tribe, m.NumRegions),
		isCity:         make(map[int]bool),
		cities:         make([]*City, 0, len(m.Cities)),
		nodeCache:      make(map[int]*MigrationTile),
		visitedPathSeg: make(map[[2]int]int),
		steepness:      m.GetSteepness(),
		maxElevation:   maxElevation,
		biomeFunc:      m.Geo.GetRegWhittakerModBiomeFunc(),
		arableLandFunc: m.GetFitnessArableLand(),
		climateFunc:    m.GetFitnessClimate(),
	}

	// Assign the tribes to their regions.
	for _, t := range m.Tribes {
		// Make sure that two tribes are not in the same region.
		if s.tribeAtRegion[t.RegionID] != nil {
			occupiers := s.tribeAtRegion[t.RegionID]
			log.Printf("Tribe %d (pop %d) is already in region %d.", occupiers.ID, occupiers.Population, t.RegionID)
			log.Printf("Tribe %d (pop %d) is trying to settle in region %d.", t.ID, t.Population, t.RegionID)
			panic("tribe already in region")
		}
		s.tribeAtRegion[t.RegionID] = t
	}

	// Take note of the regions that are already cities.
	for _, c := range m.Cities {
		s.cities = append(s.cities, c)
		s.isCity[c.ID] = true
	}

	// Regenerate the land.
	for i, ex := range m.SoilExhaustion {
		if ex < 0.9 {
			m.SoilExhaustion[i] = 0
		} else if ex != 0 {
			m.SoilExhaustion[i] *= 0.75
		}
	}

	// TODO:
	// - Calculate growth of all tribes.
	// (The rate might not be the same for all tribes.)
	// - Sort the tribes by population (descending).
	// - We need to calculate soil exhaustion before we start moving the tribes around.

	// Sort the tribes by population (descending).
	sort.Slice(m.Tribes, func(i, j int) bool {
		return m.Tribes[i].Population > m.Tribes[j].Population
	})

	// Grow the population of each tribe.
	for _, t := range m.Tribes {
		if t.Population == 0 {
			// The tribe has died out.
			s.tribeAtRegion[t.RegionID] = nil
			continue
		}

		// Grow the population of the tribe.
		// TODO: Change growth rate based on suitability of the region,
		// and the tribe's preferences.
		t.Grow()

		// Add the region properties to the tribe's experiences.
		curRegProp := s.getRegionProp(t.RegionID)
		t.addRegionProp(&curRegProp)

		// TODO: Add function that evaluates the economic value of the region.

		// TODO: Update the culture of the tribe when it has developed enough.
		//
		// Set the current culture of the tribe.
		// t.LastCultures.add(ctf(t.RegionID))
		//
		// Get the preferred culture of the tribe.
		// t.LastCultures.preferred()

		// TODO: There is a chance to develop a more specific culture.
		// if t.Culture == nil {
		// 	// Also check if there is already enough preference for a specific culture.
		// }

		// TODO: There is a chance to develop a religion.
		// if t.Religion == nil {
		//	// Also check if there is already enough preference for a specific religion.
		// }

		// Get the resources of the region.
		// We need this to determine potential new skills of the tribe
		// and to determine the sustainability of the tribe in the region.
		wood := m.Geo.Resources.Wood[t.RegionID] != 0
		stone := m.Geo.Resources.Stones[t.RegionID] != 0
		metal := m.Geo.Resources.Metals[t.RegionID] != 0
		gems := m.Geo.Resources.Gems[t.RegionID] != 0

		// Develop the skills of the tribe given the current region.
		t.DevelopSkills(&curRegProp, wood, stone, metal, gems)

		// Exhaust the resources of the region.
		m.SoilExhaustion[t.RegionID] += float64(t.Population) / t.SustainabilityFactor(&curRegProp, wood, stone, metal, gems)

		// Calculate the max population for the region.
		maxRegionPop := s.calcMaxPopPerRegion(t, t.RegionID)
		t.regionMaxPop = s.calcTheoreticalMaxPopPerRegion(t, t.RegionID)
		t.currentRegionMaxPop = maxRegionPop

		// Random events.
		{
			// Random bad things can happen.
			// TODO: Maybe introduce some modifiers that are gained by these events.
			// Good events:
			// - Good harvest could increase the population growth rate or improve soil fertility / reduce soil exhaustion.
			// - Good weather could give a bonus to positve satisfaction changes and reduce negative satisfaction changes.
			// - Good fortune could improve defense or offense of the tribe, or some magical or religious event.
			//   Maybe they gain a relic or a new skill.
			// Bad events:
			// - Bad events could reduce economic output or destroy resources.
			// - Floods could reduce population growth or destroy resources but also increase soil fertility.
			// TODO: The magnitude of change should depend on the severity of the bad or good thing.
			const (
				goodThingSatChange = 0.1
				badThingSatChange  = -0.1
			)
			if rand.Intn(1000) < 2 {
				// Random good things can happen, increase the satisfaction of the tribe.
				goodThings := []string{
					"good harvest",
					"good weather",
					"good fortune",
				}
				goodThing := goodThings[rand.Intn(len(goodThings))]
				t.changeSatisfaction(goodThingSatChange)
				log.Printf("Good thing happened to tribe %d: %s", t.ID, goodThing)
			} else if rand.Intn(1000) < 2 {
				// Random bad things can happen, reduce the satisfaction of the tribe.
				possibleBadThings := []string{
					"bad harvest",
					"bad weather",
					"bad fortune",
					"illness",
				}
				if curRegProp.mountainProx {
					possibleBadThings = append(possibleBadThings, "earthquake", "rockslide")
				}
				if curRegProp.riverProx {
					possibleBadThings = append(possibleBadThings, "flood")
				}
				if curRegProp.oceanProx {
					possibleBadThings = append(possibleBadThings, "malaria")
				}
				if curRegProp.lakeProx {
					possibleBadThings = append(possibleBadThings, "tsunami")
				}
				badThing := possibleBadThings[rand.Intn(len(possibleBadThings))]
				t.changeSatisfaction(badThingSatChange)
				log.Printf("Bad thing happened to tribe %d: %s", t.ID, badThing)
			}
		}

		// Get the multiplier for the region, which will double as the satisfaction of the tribe.
		// This will be used to determine if the tribe is happy or not with the current region.
		t.changeSatisfaction(0.5 * (s.getRegMultiplier(t, t.RegionID) - float64(t.Satisfaction)))
		if t.Satisfaction < 0.7 {
			s.printRegionInfo(t.RegionID)
			t.printTribePreferences()
		}

		// Update tribe actions based on satisfaction.
		{
			// Update existing factions.
			// - If a faction has a popularity of 0, it should be removed.
			// - Either exile, kill, or merge with another faction.
			// Factions should have independent actions, etc.
			// - Factions with high popularity unlock new actions, etc.
			// - Available action depends on sponsorship, popularity, etc.
			// - We will need to keep track of notable sponsors and members.
			// Example actions:
			// - Charitable actions (help the poor, etc.)
			// - Userper actions (try to take over the tribe)
			// - Religious actions (try to convert the tribe to a new religion)
			// - Intrigue actions (try to kill the leadership, etc.)

			// The lower the satisfaction, the higher the chance that a faction will form.
			if t.Satisfaction < 0.75 && rand.Float64() > float64(t.Satisfaction) {
				log.Printf("Tribe %d is unhappy. A new faction might form.", t.ID)
				if len(t.Factions) == 0 || rand.Float64() < 0.1/float64(len(t.Factions)) {
					f := genFaction(t.Language)
					t.Factions = append(t.Factions, f)
					log.Printf("Tribe %d has formed a new faction %s.", t.ID, f.String())
				} else if len(t.Factions) > 0 && rand.Float64() > float64(t.Leadership.Popularity) {
					// There is a chance that a faction, more popular than the leadership, will try to take over.
					// Sort the factions by popularity.
					sort.Slice(t.Factions, func(i, j int) bool {
						return t.Factions[i].Popularity > t.Factions[j].Popularity
					})

					if topFaction := t.Factions[0]; topFaction.Popularity > t.Leadership.Popularity {
						// The top faction will take over.
						// Depending on chance and popularity, it might kill the leadership, or simply replace it.
						//
						// TODO:
						// - Take note of this event.
						// - Change opinion of factions, etc.
						if oldLeadership := t.Leadership; rand.Float64() > float64(oldLeadership.Popularity) {
							// The more unpopular the leadership, the higher the chance that leadership will be killed.
							if t.Population > len(oldLeadership.Leaders)+len(topFaction.Leaders) {
								t.Population -= len(oldLeadership.Leaders)
							} else {
								log.Printf("%s the population is less than the number of leaders!", t.String())
							}
							log.Printf("%s: Leadership %s eliminated by %s", t.String(), oldLeadership.String(), topFaction.String())
							t.Leadership = topFaction
							t.Factions = t.Factions[1:]
						} else {
							// The faction will simply replace the current leadership.
							log.Printf("%s: Leadership %s replaced by %s", t.String(), oldLeadership.String(), topFaction.String())
							t.Leadership, t.Factions[0] = topFaction, oldLeadership
						}
					}
				}
			}
		}

		// TODO: Move this to the individual tribe handling functions.
		s.handleResources(t)

		// TODO: If we have a city, city state, or empire set, we need to handle all of these.

		switch t.Type {
		case TribeTypeNomadic:
			// The tribe is still nomadic, so we migrate the tribe to the most suitable region.
			// If the tribe is too large for the region, we split the tribe into two or more tribes.
			// The tribe might die out if it cannot find a suitable region to settle in.
			s.handleNomadicTribe(t)
		case TribeTypeSettling:
			// Check if the tribe has settled in a region.
			if t.doneSettling {
				panic("Tribe is already settled, this should not happen")
			}

			// If the tribe is settled, we need to find the best region to settle in,
			// create a settlement, and convert the tribe into a culture.

			// There is a chance that part of the tribe will split off and form a new tribe to
			// follow some other vision or to settle in a different region.
			s.checkRandomSplit(t)
			// The tribe is still looking for a region to settle in or moving to the region
			// where it wants to settle.
			s.handleSettlingTribe(t)

			// Update the tribe on the next turn.
			s.newTribes = append(s.newTribes, t)
		case TribeTypeCity:
			// There is a chance that part of the tribe will split off and form a new tribe to
			// follow some other vision or to settle in a different region.
			s.checkRandomSplit(t)

			// The tribe has successfully settled in a region.
			s.handleCity(t)

			// Update the tribe on the next turn.
			s.newTribes = append(s.newTribes, t)
		case TribeTypeCityState:
			// There is a chance that part of the tribe will split off and form a new tribe to
			// follow some other vision or to settle in a different region.
			s.checkRandomSplit(t)

			// Check if we can add some cities to our city state.
			s.handleCityState(t)

			// Update the tribe on the next turn.
			s.newTribes = append(s.newTribes, t)
		case TribeTypeEmpire:
			s.handleEmpire(t)

			// Update the tribe on the next turn.
			s.newTribes = append(s.newTribes, t)
		}
	}
	// TODO: Update satisfaction / happiness of the tribes.

	// Check if we need to update the cultures.
	// TODO: Make expansion dependent on prosperity of the cultures.
	var cultureSeeds []int
	for _, c := range s.newTribes {
		if c.Culture != nil && c.Population > 0 {
			cultureSeeds = append(cultureSeeds, c.Culture.ID)
		}
	}
	for _, c := range s.m.Cultures {
		// TODO: Check if culture is extinct.
		cultureSeeds = append(cultureSeeds, c.ID)
	}
	if len(cultureSeeds) > 0 {
		m.expandCultures(true, dedupInts(cultureSeeds))
	}

	// Check if we need to update the city states.
	// TODO: Make expansion dependent on prosperity of the city states.
	var cityStateSeeds []int
	for _, c := range s.newTribes {
		if c.CityState != nil && c.Population > 0 {
			cityStateSeeds = append(cityStateSeeds, c.CityState.ID)
		}
	}
	for _, c := range s.m.CityStates {
		if c.Capital == nil || c.Capital.Population == 0 {
			panic("City state capital has no population or no capital, should collapse")
		}
		cityStateSeeds = append(cityStateSeeds, c.ID)
	}
	if len(cityStateSeeds) > 0 {
		m.expandCityStates(true, dedupInts(cityStateSeeds))
	}

	// Check if we need to update the empires.
	var empireSeeds []int
	for _, c := range s.newTribes {
		if c.Empire != nil && c.Population > 0 {
			empireSeeds = append(empireSeeds, c.Empire.ID)
		}
	}
	for _, c := range s.m.Empires {
		if c.Capital == nil || c.Capital.Population == 0 {
			panic("Empire capital has no population or no capital, should collapse")
		}
		empireSeeds = append(empireSeeds, c.ID)
	}
	if len(empireSeeds) > 0 {
		m.expandEmpires()

		// TODO: Align with other implementations.
	}
	m.Tribes = s.newTribes
}

const numYears = 40000   // Number of years to simulate.
const growthRate = 0.001 // 0.1% growth rate per year
const popDensity = 10    // 10 people per km^2

type HackyBiome int

func (b HackyBiome) String() string {
	return genbiome.WhittakerModBiomeToString(int(b))
}

type TribeType int

const (
	TribeTypeNomadic TribeType = iota
	TribeTypeSettling
	TribeTypeCity
	TribeTypeCityState
	TribeTypeEmpire
)

// We start with one tribe. This tribe will be nomadic, and will move around.
// After a while, the tribe might be big enough to split into two tribes.
//
// A tribe can split due to:
// - Survival reasons (not enough food, etc. in the regions in their proximity).
// - Religious reasons (a new vision from the gods, etc.).
// - TODO: Cultural reasons (a different culture type preference, etc.).
// - TODO: Unhappiness (a string of bad events, deaths, famine, etc.).
//
// When a tribe splits, the new tribe will have a part of the population of the original tribe
// and retain some of the culture, religion, etc. of the original tribe. (In the future, we might
// want to consider that we fork the culture, religion, etc. of the original tribe as soon as the
// new tribe is created, not only when the tribe settles.).
//
// Settled Tribes:
//
// Tribes can either be settled or nomadic. Once a tribe has developed enough
// skills to permanently survive sustainably in a region type, it will look for
// a region to settle in within a certain radius. The radius depends on the
// form of discovery (e.g. a vision from the gods might have a larger radius,
// while a region discovered by the tribe itself through scouting might have a
// smaller radius).
// The tribe will then move to the region and settle there, if it is not already
// occupied by another tribe.
//
// If the region is already occupied, the tribe will either:
// - try to find another region to settle in
// - TODO: fight for the region
// - TODO: merge with the other tribe if the conditions are right
//
// Once the tribe has settled in a region, it will develop a new culture and a religion.
// If the tribe has inherited a religion or culture from the parent tribe, it will fork
// the culture and religion and develop a new one based on the preferences of the tribe.
// The tribe's language will be also become independent from the parent tribe from this point on.
// The tribe will also create a settlement in the region, which will be the cultural and religious
// center of the tribe.
//
// Nomadic Tribes:
//
// Tribes can also be nomadic. Nomadic tribes will move around, always looking for the most
// prosperous / suitable region to settle in. If the most prosperous region cannot sustain the
// entire tribe, the tribe will split into two or more tribes. The new tribes will move to the
// most suitable regions, while the original tribe will try to survive in the current region.
//
// The tribe will gain skills and preferences based on the regions it has been in. The tribe will
// start to prefer the types of regions (biomes, water proximity, mountain proximity, etc.) it has
// been in the most. The tribe will also develop skills based on the regions it has been in.
//
// Skills:
//
// Skills should provide a bonus to the tribe's sustainability in a region. Skills can be developed
// based on the region type, the tribe's preferences, and influence the tribe's culture.
//
// Skills can be developed based on:
// - Biome type and tribe preference for the biome
// - Water proximity
// - Mountain proximity
// - Existing skills of the tribe
// - TODO: Culture of the tribe
// - TODO: Religion of the tribe
// - TODO: Available resources in the region
// - TODO: Available species in the region
//   - Grains (wheat, barley, etc.) for farming, beer, bread, etc.
//   - Animals (cows, buffalos, etc.) for herding, milk, meat, etc.
//   - Fish (salmon, trout, etc.) for fishing, food, etc.
//
// TODO: Happiness:
//
// The happiness of the tribe will depend on various factors:
//
// - Growth rate of the tribe
// - The tribe's preferences (if the tribe is in a region it prefers or not)
// - Deaths (from conflicts, famine, etc.)
// - Randomized events like:
//   - Corrupt leaders
//   - Famine
//   - Disease
//   - Natural disasters
//   - Accidents
//
// TODO:
// - Keep track of the type of regions we are at and at some point adjust
// the culture to prefer certain regions.
// - Keep note of skills and knowledge of the tribe (e.g. farming, mining, etc.)
// Skills can develop based on the biome and region type the tribe is in.
// TODO: Based on the counts, we can evaluate the chances of the tribe
// discovering new skills, knowledge, etc.
// TODO: Keep track of water proximity, etc.
type Tribe struct {
	ID         int                   // Unique ID for the tribe.
	RegionID   int                   // The region the tribe is currently in.
	Population int                   // The population of the tribe.
	Type       TribeType             // The type of the tribe (nomadic, settled, city state, empire, etc.)
	Parent     *Tribe                // The parent tribe of the tribe.
	Language   *genlanguage.Language // The language of the tribe.
	Settlement *City                 // The city the tribe has settled in.
	Culture    *Culture              // Culture that the tribe might have developed.
	Religion   *Religion             // Religion that the tribe might have developed.
	CityState  *CityState            // City state that the tribe might have developed.
	Empire     *Empire               // Empire that the tribe might have developed.

	// TODO: Change this to a running average.
	Satisfaction    ClampedVal           // Happiness of the tribe overall.
	SatisfactionAvg *RunningAverageLimit // Running average of the tribe's satisfaction.
	Leadership      *Faction             // Current leadership of the tribe.
	Factions        []*Faction           // Factions within the tribe / opposition, etc.
	Skills          map[*Skill]bool      // Skills of the tribe.

	// Preferences of the tribe.
	LastBiomes        *Last100[HackyBiome]  // The last 1000 biomes the tribe has been in.
	LastCultures      *Last100[CultureType] // The last 1000 cultures the tribe has been in.
	RiverProximity    *RunningBool
	LakeProximity     *RunningBool
	OceanProximity    *RunningBool
	MountainProximity *RunningBool

	*Path             // Pathfinding for migrating tribes.
	gotVision    bool // If the tribe got their destination to settle from a vision from the gods.
	doneSettling bool // If the tribe has settled in a region.

	regionMaxPop        int // The max pop of the region the tribe is in.
	currentRegionMaxPop int // The current max pop of the region the tribe is in.
	*ResourceStorage        // Resources the tribe has.
}

func (m *Civ) getNextTribeID() int {
	id := m.TribeCount
	m.TribeCount++
	return id
}

func (m *Civ) newTribeAtRegion(r int, pop int) *Tribe {
	t := NewTribe(m.getNextTribeID(), r, pop)
	m.Tribes = append(m.Tribes, t)
	log.Println("TODO: Generate culture.")
	return t
}

// NewTribe returns a new tribe with the given population.
func NewTribe(id, regionID, population int) *Tribe {
	// Initialize the last 10 biomes with the current as -1.
	var last100Biomes [100]int
	for i := range last100Biomes {
		last100Biomes[i] = -1
	}
	t := &Tribe{
		ID:                id,
		RegionID:          regionID,
		Population:        population,
		LastBiomes:        newLast100[HackyBiome]("biome"),
		LastCultures:      newLast100[CultureType]("culture"),
		RiverProximity:    NewRunningBool(),
		LakeProximity:     NewRunningBool(),
		OceanProximity:    NewRunningBool(),
		MountainProximity: NewRunningBool(),
		Skills:            make(map[*Skill]bool),
		Type:              TribeTypeNomadic,
		Satisfaction:      1.0,
		SatisfactionAvg:   NewRunningAverageLimit(100),
		Language:          GenLanguage(int64(regionID)),
		ResourceStorage:   newResourceStorage(10),
	}
	// Generate leadership.
	t.Leadership = genFaction(t.Language)
	return t
}

func (t *Tribe) compare(other *Tribe) float64 {
	// Positive is good, negative is bad.
	// We compare:
	// - Preferences
	// - Skills
	// - Culture
	// - Religion
	// - Language

	// Compare preferences.
	var prefValue float64
	// TODO: Biome
	prefValue -= math.Abs(t.MountainProximity.avg - other.MountainProximity.avg)
	prefValue -= math.Abs(t.RiverProximity.avg - other.RiverProximity.avg)
	prefValue -= math.Abs(t.LakeProximity.avg - other.LakeProximity.avg)
	prefValue -= math.Abs(t.OceanProximity.avg - other.OceanProximity.avg)

	// Compare skills.
	var skillValue float64
	for _, s := range Skills {
		if t.Skills[s] && other.Skills[s] {
			skillValue += 0.1
		} else if t.Skills[s] || other.Skills[s] {
			skillValue -= 0.1
		}
	}
	languageValue := compareLanguage(t.Language, other.Language)
	religionValue := t.Religion.compare(other.Religion)
	cultureValue := t.Culture.compare(other.Culture)

	// Calculate the similarity.
	similarity := prefValue + skillValue + cultureValue + religionValue + languageValue
	return similarity
}

// Ref returns the object reference of the city.
func (t *Tribe) Ref() ObjectReference {
	return ObjectReference{
		ID:   t.ID,
		Type: ObjectTypeTribe,
	}
}

func (t *Tribe) SetPath(path *Path) {
	t.Path = path
	if path != nil && t.Path.Peek() == t.RegionID {
		t.Path.Next() // Skip the current region.
	}
}

func (t *Tribe) hasPath() bool {
	return t.Path != nil
}

// makeNomadic will make the tribe nomadic.
func (t *Tribe) makeNomadic() {
	// TODO: If the tribe is in charge of a city state or empire, we'd need to handle that.
	// Will we let the empire collapse, or do we allow the tribe to control the empire nomadically?
	t.Type = TribeTypeNomadic
	t.SetPath(nil)
	t.doneSettling = false
	delete(t.Skills, SSkillSettling)
	delete(t.Skills, SSkillSettlingHighland)
	delete(t.Skills, SSkillSettlingWetlands)
	log.Println("!!!Tribe", t.ID, "has become nomadic.")
}

func (t *Tribe) changeSatisfaction(diff float64) {
	origSat := t.Satisfaction
	t.Satisfaction.Add(diff)
	t.SatisfactionAvg.Add(float64(t.Satisfaction))
	log.Printf("Tribe %d satisfaction changed from %.2f to %.2f (rnng avg %.2f)", t.ID, origSat, t.Satisfaction, t.SatisfactionAvg.Current())

	// If the satisfaction has decreased, the leadership popularity should decrease,
	// and the faction popularity should increase... and vice versa if the satisfaction
	// has increased.
	t.Leadership.Popularity.Add(diff)

	// We distribute the satisfaction change to the factions based on their influence.
	var sumPop float64
	for _, f := range t.Factions {
		sumPop += float64(f.Leadership.Popularity)
	}

	// Now we distribute the satisfaction change to the factions.
	// TODO:
	// There should be some additional randomness in the distribution,
	// or the popularity should change through random events, etc.
	// Also it should depend on the event type and the individual faction
	// if the faction would benefit from the event or not.
	for _, f := range t.Factions {
		f.Popularity.Add(diff * float64(f.Popularity) / sumPop) // Clamp to [0, 1]
	}
}

func (t *Tribe) addRegionProp(curRegProp *regionProp) {
	// Set the current biome of the tribe and get the preferred biome of the tribe.
	t.LastBiomes.Add(curRegProp.biome)

	// Set water proximity and get the preferred water proximity.
	t.RiverProximity.Add(curRegProp.riverProx)

	// Set lake proximity and get the preferred lake proximity.
	t.LakeProximity.Add(curRegProp.lakeProx)

	// Set ocean proximity and get the preferred ocean proximity.
	t.OceanProximity.Add(curRegProp.oceanProx)

	// Set mountain proximity and get the preferred mountain proximity.
	t.MountainProximity.Add(curRegProp.mountainProx)
}

func (t *Tribe) getPreferredRegionProp() *regionProp {
	preferredBiome, _ := t.LastBiomes.Preferred()
	return &regionProp{
		biome:        preferredBiome,
		riverProx:    t.RiverProximity.Current(),
		lakeProx:     t.LakeProximity.Current(),
		oceanProx:    t.OceanProximity.Current(),
		mountainProx: t.MountainProximity.Current(),
	}
}

func (t *Tribe) printTribePreferences() {
	log.Printf("Tribe %d preferences:", t.ID)
	curTribePref := t.getPreferredRegionProp()
	curTribePref.Log()
}

// Calculate the "attractiveness" multiplier for the region and the tribe.
// This will return a multiplier signifying how well the tribe can extract resources
// or value from the region.
func (t *Tribe) compareSuitability(gotProp regionProp) float64 {
	wantProp := t.getPreferredRegionProp()
	multiplier := 1.0

	// Penalize if the biome doesn't match the preferred biome.
	if wantProp.biome != gotProp.biome {
		multiplier -= 0.2 * (1 - t.LastBiomes.GetScoreOf(gotProp.biome))
	}

	// If there is a specific preference for the region, we penalize
	// if the region doesn't match the preference.
	if wantProp.riverProx && !gotProp.riverProx {
		multiplier -= 0.2 * t.RiverProximity.avg
	}
	if wantProp.lakeProx && !gotProp.lakeProx {
		multiplier -= 0.2 * t.LakeProximity.avg
	}
	if wantProp.oceanProx && !gotProp.oceanProx {
		multiplier -= 0.2 * t.OceanProximity.avg
	}
	if wantProp.mountainProx && !gotProp.mountainProx {
		multiplier -= 0.2
	}
	return multiplier
}

func (t *Tribe) SustainabilityFactor(curRegProp *regionProp, wood, stone, metal, gems bool) float64 {
	// TODO: This should depend on the region (whether it's a desert, etc.)
	// Depending on the skills and knowledge of the tribe, we can calculate
	// the sustainability factor of the tribe.
	factor := 1.0
	for _, s := range Skills {
		if !t.Skills[s] {
			continue
		}
		if s.CanDevelopIn(curRegProp, wood, stone, metal, gems) {
			factor += 0.1
		}
	}

	return factor
}

func (t *Tribe) String() string {
	// Assemble the preferred proximity string.
	proxStr := ""
	if t.RiverProximity.Current() {
		proxStr += "R"
	} else {
		proxStr += "-"
	}
	if t.LakeProximity.Current() {
		proxStr += "L"
	} else {
		proxStr += "-"
	}
	if t.OceanProximity.Current() {
		proxStr += "O"
	} else {
		proxStr += "-"
	}
	if t.MountainProximity.Current() {
		proxStr += "M"
	} else {
		proxStr += "-"
	}
	var cultureStr string
	if t.Culture != nil {
		cultureStr = " " + t.Culture.Type.String()
	}
	var leaderStr string
	if t.Leadership != nil {
		leaderStr = " " + t.Leadership.String()
	}
	return fmt.Sprintf("Tribe %d (%s) in region %d with population %d; satisfaction %f (%q)%s%s", t.ID, proxStr, t.RegionID, t.Population, t.Satisfaction, t.GetCultureType(), cultureStr, leaderStr)
}

// Grow the population of the tribe for one year.
func (t *Tribe) Grow() {
	// Calculate the population growth rate for the tribe.
	// Use the exponential growth model.
	newPop := float64(t.Population) * math.Pow(math.E, growthRate)
	if diff := newPop - float64(t.Population); diff < 1 {
		// Use rand to potentially grow the population by one.
		if rand.Float64() < diff {
			t.Population++
		}
	} else {
		t.Population = int(newPop)
	}
}

// GetCultureType returns the culture type of the tribe.
func (t *Tribe) GetCultureType() CultureType {
	preferredBiome, score := t.LastBiomes.Preferred()
	if !(t.Skills[SSkillSettling] || t.Skills[SSkillSettlingWetlands] || t.Skills[SSkillSettlingHighland]) && isAnyOf(int(preferredBiome), nomadicBiomes) && score > 0.5 {
		return CultureTypeNomadic
	}

	if t.MountainProximity.Current() && t.RiverProximity.Current() {
		if t.MountainProximity.avg > t.RiverProximity.avg {
			return CultureTypeHighland
		}
		return CultureTypeRiver
	} else if t.MountainProximity.Current() {
		return CultureTypeHighland
	} else if t.RiverProximity.Current() {
		return CultureTypeRiver
	}

	// TODO:
	// - Culture lake
	if t.LakeProximity.Current() {
		return CultureTypeLake
	}

	// - Culture naval
	if t.OceanProximity.Current() {
		return CultureTypeNaval
	}

	// Check if we have hunters.
	if t.Skills[SSkillHunting] && isAnyOf(int(preferredBiome), huntingBiomes) {
		return CultureTypeHunting
	}

	return CultureTypeGeneric
}

func isAnyOf(b int, biomes []int) bool {
	for _, bb := range biomes {
		if b == bb {
			return true
		}
	}
	return false
}

// Split the tribe into two tribes with a new one with the given population.
func (t *Tribe) Split(id int, newPopulation int) *Tribe {
	t.Population -= newPopulation
	nt := NewTribe(id, t.RegionID, newPopulation) // TODO: Generate a unique ID.
	// Copy the last biomes and cultures to the new tribe.
	t.LastBiomes.CopyTo(nt.LastBiomes)
	t.LastCultures.CopyTo(nt.LastCultures)
	nt.Type = t.Type
	nt.Parent = t
	nt.Satisfaction = t.Satisfaction // Maybe that should be different?
	t.SatisfactionAvg.CopyTo(nt.SatisfactionAvg)
	t.RiverProximity.CopyTo(nt.RiverProximity)
	t.LakeProximity.CopyTo(nt.LakeProximity)
	t.OceanProximity.CopyTo(nt.OceanProximity)
	t.MountainProximity.CopyTo(nt.MountainProximity)

	// TODO: If we have a culture, we should copy it to the new tribe.
	// We should also track religion and we either mutate the religion or
	// or the culture of the new tribe.
	log.Println("TODO: Mutate religion or culture and find a better way to fork language")
	nt.Language = t.Language
	nt.Religion = t.Religion // For now, we use the same religion.
	nt.Culture = t.Culture   // For now, we use the same culture.

	// Copy the skills to the new tribe.
	for s := range t.Skills {
		nt.Skills[s] = true
	}

	// Generate leadership if no other faction exists.
	if len(t.Factions) == 0 {
		nt.Leadership = genFaction(nt.Language)
	} else {
		// Promote the most popular faction to the new tribe.
		// TODO: The population of the new tribe should depend on the popularity of the faction.
		sort.Slice(t.Factions, func(i, j int) bool {
			return t.Factions[i].Leadership.Popularity > t.Factions[j].Leadership.Popularity
		})
		nt.Leadership = t.Factions[0]
		t.Factions = t.Factions[1:]
	}
	return nt
}

func (t *Tribe) RandomSplit(nextID int) *Tribe {
	// The tribe is unhappy and might split.
	nt := t.Split(nextID, rand.Intn(t.Population)/2)

	// If the original tribe satisfaction was low, we increase the satisfaction of the new tribe.
	if t.Satisfaction < 0.3 {
		nt.changeSatisfaction(tribeSplitVoluntarySatisfaction)
	}

	// There is a random chance that the tribe will leave due to a vision or a spiritual calling.
	nt.gotVision = rand.Intn(100) < 50

	if rand.Intn(100) < 10 {
		// There is a chance that the new tribe will revert to being nomadic.
		// TODO: Find a way to have a religion that doesn't require a spiritual center.
		// One way to solve this is to have a place of pilgrimage that the tribe can visit
		// but then we'd have to make sure that the religion doesn't really spread from there
		// but rather from the practitioners (tribes, etc.).
		nt.makeNomadic()
	} else {
		// Make sure the tribe will look for a new region to settle in.
		nt.Type = TribeTypeSettling
		nt.SetPath(nil)
		nt.doneSettling = false
	}

	// There is a chance that the new tribe will have some preferences reset.
	if rand.Intn(100) < 50 {
		nt.LastBiomes.Reset()
	}
	if rand.Intn(100) < 50 {
		nt.RiverProximity.Reset()
	}
	if rand.Intn(100) < 50 {
		nt.LakeProximity.Reset()
	}
	if rand.Intn(100) < 50 {
		nt.OceanProximity.Reset()
	}
	if rand.Intn(100) < 50 {
		nt.MountainProximity.Reset()
	}
	log.Printf("!!!%s has split into %s and %s.", t.String(), t.String(), nt.String())
	return nt
}

// DevelopSkills develops the skills of the tribe based on the biome and region type.
func (t *Tribe) DevelopSkills(curRegProp *regionProp, wood, stone, metal, gems bool) {
	// Check all possible skills and check if we can develop them.
SkillLoop:
	for _, s := range Skills {
		if t.Skills[s] {
			continue // The tribe already has this skill.
		}
		// Check if we have the required skills to develop this skill.
		for _, r := range s.Requires {
			if !t.Skills[r] {
				continue SkillLoop
			}
		}
		if s.CanDevelopIn(curRegProp, wood, stone, metal, gems) && s.DevelopAt(t.LastBiomes.GetScoreOf(curRegProp.biome)) {
			t.Skills[s] = true
			s.EffectOnTribe(t)
			log.Printf("Tribe %d developed the skill %s", t.ID, s.Name)
		}
	}
}

type ClampedVal float64

func (c *ClampedVal) Add(v float64) {
	*c = ClampedVal(math.Max(0, math.Min(1, float64(*c)+v)))
}

// Leadership represents the leadership of a tribe or faction.
type Leadership struct {
	// TODO: Add leadership properties.
	// Type (e.g. democratic, autocratic, etc.
	// Titles (e.g. king, council, etc.)
	// Leader(s) (e.g. council, king, etc.)
	Leaders []string
	// Influence  ClampedVal // Influence or popularity will determine the power of the leadership.
	Popularity ClampedVal // Popularity will determine the happiness of the tribe.
}

// TODO: Also add culture maybe?
func genLeadership(lang *genlanguage.Language) *Leadership {
	leadership := &Leadership{
		Leaders: []string{lang.MakeFirstName() + " " + lang.MakeLastName()},
		// Influence:  1.0,
		Popularity: 1.0,
	}
	return leadership
}

// String returns a string representation of the leadership.
func (l *Leadership) String() string {
	return fmt.Sprintf("%s (%f)", strings.Join(l.Leaders, ", "), l.Popularity)
}

// Faction represents a faction within a tribe or a city state.
type Faction struct {
	Name string // Name of the faction.
	*Leadership
	// TODO: Add faction properties.
	// Type (e.g. religious, military, etc.)
}

// String returns a string representation of the faction.
func (f *Faction) String() string {
	return fmt.Sprintf("%s (%s)", f.Name, f.Leadership.String())
}

// genFaction generates a new faction with the given name.
func genFaction(lang *genlanguage.Language) *Faction {
	// TODO: Migrate the name generation to a dedicated function.
	var name string
	if rand.Float64() < 0.5 {
		name = "The "
	}
	prefixes := []string{"Brothers", "Sisters", "Warriors", "Mystics", "Hunters", "Fishers", "Farmers", "Craftsmen", "Artisans"}
	name += prefixes[rand.Intn(len(prefixes))]

	if rand.Float64() < 0.5 {
		name += " of "
		if rand.Float64() < 0.5 {
			suffixes := []string{"the Sun", "the Moon", "the Stars", "the Earth", "the Sea", "the Mountains", "the Forest", "the River", "the Lake", "the Ocean"}
			name += suffixes[rand.Intn(len(suffixes))]
		} else {
			// TODO: Like religion, allow place names, etc.
			name += lang.MakeName()
		}
	}

	f := &Faction{
		Name:       name,
		Leadership: genLeadership(lang),
	}
	return f
}

type Skill struct {
	Name              string
	MinScore          float64        // The minimum score required to develop the skill.
	Chance            float64        // The chance of developing the skill.
	Biomes            []int          // The biomes the skill can be developed in.
	Wood              bool           // If the skill requires wood.
	Stone             bool           // If the skill requires stone.
	Metal             bool           // If the skill requires metal.
	Gems              bool           // If the skill requires gems.
	WaterProximity    bool           // If the skill requires water proximity.
	RiverProximity    bool           // If the skill requires river proximity.
	LakeProximity     bool           // If the skill requires lake proximity.
	OceanProximity    bool           // If the skill requires ocean proximity.
	MountainProximity bool           // If the skill requires mountain proximity. (e.g. mining)
	Requires          []*Skill       // Required skills to develop this skill.
	Effect            func(t *Tribe) // The effect of the skill on the tribe.
}

func (s *Skill) CanDevelopIn(curRegProp *regionProp, wood, stone, metal, gems bool) bool {
	waterProximity := curRegProp.riverProx || curRegProp.lakeProx || curRegProp.oceanProx
	// Check required proximities.
	if s.WaterProximity && !waterProximity ||
		s.MountainProximity && !curRegProp.mountainProx ||
		s.RiverProximity && !curRegProp.riverProx ||
		s.LakeProximity && !curRegProp.lakeProx ||
		s.OceanProximity && !curRegProp.oceanProx ||
		s.Wood && !wood ||
		s.Stone && !stone ||
		s.Metal && !metal ||
		s.Gems && !gems {
		return false
	}

	// If there is no biome requirement, the skill can be developed anywhere.
	if len(s.Biomes) == 0 {
		return true
	}

	// Check if the biome is suitable for the skill.
	for _, b := range s.Biomes {
		if b == int(curRegProp.biome) {
			return true
		}
	}
	return false
}

// GetBonusForBiome returns a bonus score derived from the skill based on the biome and water proximity.
func (s *Skill) GetBonusForBiome(curRegProp *regionProp, wood, stone, metal, gems bool) float64 {
	if s.CanDevelopIn(curRegProp, wood, stone, metal, gems) {
		return 1
	}
	return 0
}

// DevelopAt returns true if the skill can be developed at the given score.
func (s *Skill) DevelopAt(score float64) bool {
	if score >= s.MinScore {
		return rand.Float64() < s.Chance
	}
	return false
}

// EffectOnTribe applies the effect of the skill on the tribe.
func (s *Skill) EffectOnTribe(t *Tribe) {
	if s.Effect != nil {
		s.Effect(t)
	}
}

var nomadicBiomes = []int{
	genbiome.WhittakerModBiomeSubtropicalDesert,
	genbiome.WhittakerModBiomeColdDesert,
	genbiome.WhittakerModBiomeTemperateGrassland,
}

var huntingBiomes = []int{
	genbiome.WhittakerModBiomeTemperateGrassland,
	genbiome.WhittakerModBiomeTropicalSeasonalForest,
	genbiome.WhittakerModBiomeTemperateSeasonalForest,
	genbiome.WhittakerModBiomeTropicalRainforest,
	genbiome.WhittakerModBiomeTemperateRainforest,
	genbiome.WhittakerModBiomeTundra,
	genbiome.WhittakerModBiomeSavannah,
	genbiome.WhittakerModBiomeBorealForestTaiga,
	genbiome.WhittakerModBiomeSnow,
}

// Skills that can be developed.
// TODO:
// - Skills might depend on other skills.
// - There might be a chance to lose a skill if the tribe is not using it.
//
// Possible skills:
// - Hunting (requires grassland, forest, etc.)
// - Gathering (requires grassland, forest, etc.)
// - Fishing (requires water proximity)
// - Farming (requires grassland)
// - Herding (requires grassland)
// - Quarrying (requires mountains)
// - Mining (requires mountains)
// - Smithing (requires mining)
// - Weaving (requires plants)
// - Pottery (requires water proximity)
// - Carpentry
var (
	SSkillHunting = &Skill{
		Name:     "Hunting",
		MinScore: 0.4,
		Chance:   0.4,
		Biomes:   huntingBiomes,
		Requires: []*Skill{SSkillGathering},
	}
	SSkillGathering = &Skill{
		Name:     "Gathering",
		MinScore: 0.2,
		Chance:   0.4,
		Biomes: []int{
			genbiome.WhittakerModBiomeTemperateGrassland,
			genbiome.WhittakerModBiomeTropicalRainforest,
			genbiome.WhittakerModBiomeTemperateRainforest,
			genbiome.WhittakerModBiomeTemperateSeasonalForest,
			genbiome.WhittakerModBiomeWoodlandShrubland,
			genbiome.WhittakerModBiomeTemperateGrassland,
			genbiome.WhittakerModBiomeBorealForestTaiga,
			genbiome.WhittakerModBiomeTundra,
			genbiome.WhittakerModBiomeWetlands,
			genbiome.WhittakerModBiomeSnow, // Meh, ?
		},
	}
	SSkillWoodworking = &Skill{
		Name:     "Woodworking",
		MinScore: 0.5,
		Chance:   0.4,
		Wood:     true,
		Requires: []*Skill{SSkillGathering},
	}
	SSkillStoneWorking = &Skill{
		Name:     "Stone Working",
		MinScore: 0.5,
		Chance:   0.4,
		Stone:    true,
		Requires: []*Skill{SSkillGathering},
	}
	SSkillMetalWorking = &Skill{
		Name:     "Metal Working",
		MinScore: 0.5,
		Chance:   0.4,
		Metal:    true,
		Requires: []*Skill{SSkillStoneWorking},
	}
	SSkillGemWorking = &Skill{
		Name:     "Gem Working",
		MinScore: 0.5,
		Chance:   0.4,
		Gems:     true,
		Requires: []*Skill{SSkillStoneWorking},
	}
	SSkillFishing = &Skill{
		Name:           "Fishing",
		MinScore:       0.9,
		Chance:         0.1,
		Biomes:         nil, // Requires water proximity
		WaterProximity: true,
		Requires:       []*Skill{SSkillGathering},
	}
	SSkillFarming = &Skill{
		Name:     "Farming",
		MinScore: 0.9,
		Chance:   0.4,
		Biomes: []int{
			genbiome.WhittakerModBiomeTemperateGrassland,
			genbiome.WhittakerModBiomeWetlands,
		},
		Requires: []*Skill{SSkillGathering},
	}
	SSkillHerding = &Skill{
		Name:     "Herding",
		MinScore: 0.5,
		Chance:   0.4,
		Biomes: []int{
			genbiome.WhittakerModBiomeTemperateGrassland,
			genbiome.WhittakerModBiomeTundra,
			genbiome.WhittakerModBiomeSubtropicalDesert,
			genbiome.WhittakerModBiomeSavannah,
			genbiome.WhittakerModBiomeColdDesert,
			genbiome.WhittakerModBiomeSnow,
		},
		Requires: []*Skill{SSkillHunting},
	}
	SSkillSettling = &Skill{
		Name:     "Settling",
		MinScore: 0.9,
		Chance:   0.1,
		Requires: []*Skill{SSkillFarming, SSkillHerding},
		Effect: func(t *Tribe) {
			if t.Type < TribeTypeSettling {
				t.Type = TribeTypeSettling
			}
			log.Printf("Tribe %d has developed a taste for settling down.", t.ID)
		},
	}
	SSkillSettlingWetlands = &Skill{
		Name:     "Settling (Wetlands)",
		MinScore: 0.9,
		Chance:   0.1,
		Biomes: []int{
			genbiome.WhittakerModBiomeWetlands,
		},
		Requires: []*Skill{SSkillFarming, SSkillGathering, SSkillFishing},
		Effect: func(t *Tribe) {
			if t.Type < TribeTypeSettling {
				t.Type = TribeTypeSettling
			}
			log.Printf("Tribe %d has developed a taste for settling down in the wetlands.", t.ID)
		},
	}
	SSkillSettlingHighland = &Skill{
		Name:              "Settling (Highland)",
		MinScore:          0.9,
		Chance:            0.1,
		MountainProximity: true,
		Biomes: []int{
			genbiome.WhittakerModBiomeColdDesert,
			genbiome.WhittakerModBiomeSnow,
		},
		Requires: []*Skill{SSkillGathering, SSkillHerding},
		Effect: func(t *Tribe) {
			if t.Type < TribeTypeSettling {
				t.Type = TribeTypeSettling
			}
			log.Printf("Tribe %d has developed a taste for settling down in the highlands.", t.ID)
		},
	}
	SSkillBoating = &Skill{
		Name:           "Boating",
		MinScore:       0.9,
		Chance:         0.1,
		WaterProximity: true,
		Requires:       []*Skill{SSkillWoodworking, SSkillFishing},
		Effect: func(t *Tribe) {
			// TODO: Have some effect.
			// This should improve trade with cities that:
			// - Are accessible through the same river system.
			// - Are accessible through the same lake.
			// - Are accessible through the same ocean on the same landmass.
		},
	}
	SSkillSeaFaring = &Skill{
		Name:           "Sea Faring",
		MinScore:       0.9,
		Chance:         0.1,
		OceanProximity: true,
		Requires:       []*Skill{SSkillBoating},
		Effect: func(t *Tribe) {
			// TODO: Have some effect.
			// This should allow trade with cities that border on the same sea
			// and are on a different landmass.
		},
	}
)

var Skills = []*Skill{
	SSkillHunting,
	SSkillGathering,
	SSkillWoodworking,
	SSkillStoneWorking,
	SSkillMetalWorking,
	SSkillGemWorking,
	SSkillFishing,
	SSkillFarming,
	SSkillHerding,
	SSkillSettling,
	SSkillSettlingWetlands,
	SSkillSettlingHighland,
	SSkillBoating,
	SSkillSeaFaring,
	// TODO: SkillMining
}

type MigrationTile struct {
	r             *Civ                       // Reference to the Civ object
	getTile       func(i int) *MigrationTile // Fetch tiles from cache
	index         int                        // region index
	used          int                        // number of times this node was used for a trade route
	steepness     []float64                  // cached steepness of all regiones
	wasVisited    func(i, j int) int         // quick lookup if a segment was already visited
	isCity        map[int]bool               // quick lookup if an index is a city
	maxElevation  float64
	tribeAtRegion []*Tribe // mapping of tribes to region
}

func (n *MigrationTile) SetUsed() {
	n.used++
}

// PathNeighbors returns the direct neighboring nodes of this node which
// can be pathed to.
func (n *MigrationTile) PathNeighbors() []goastar.Pather {
	nbs := make([]goastar.Pather, 0, 6)
	for _, i := range n.r.GetRegNeighbors(n.index) {
		nbs = append(nbs, n.getTile(i))
	}
	return nbs
}

// PathNeighborCost calculates the exact movement cost to neighbor nodes.
func (n *MigrationTile) PathNeighborCost(to goastar.Pather) float64 {
	tot := to.(*MigrationTile)

	// Discourage underwater paths and occupied regions.
	// TODO: Make sure that tribeAtRegion is not infinite but just very high cost.
	if n.r.Elevation[n.index] <= 0 || n.r.Elevation[tot.index] <= 0 || n.tribeAtRegion[tot.index] != nil {
		return math.Inf(1)
	}

	// TODO: Fix this... this is highly inefficient.
	nIdx := tot.index

	// Altitude changes come with a cost (downhill is cheaper than uphill)
	cost := 1.0 + (n.r.Elevation[nIdx]-n.r.Elevation[n.index])/n.maxElevation

	// The steeper the terrain, the more expensive.
	cost *= 1.0 + n.steepness[nIdx]*n.steepness[nIdx]

	// Highly incentivize re-using used segments
	if nvis := n.wasVisited(n.index, nIdx); nvis > 0 {
		cost /= 8.0 * float64(nvis) * float64(nvis)
	} else {
		cost *= 8.0
	}

	// Heavily incentivize re-using existing roads.
	if nUsed := tot.used; nUsed > 0 {
		cost /= 8.0 * float64(nUsed) * float64(nUsed)
	} else {
		cost *= 8.0
	}

	// Penalty if the neighbor is a city.
	if n.isCity[nIdx] {
		cost *= 4.0
	}

	// Bonus if along coast.
	for _, nbnb := range n.r.GetRegNeighbors(nIdx) {
		if n.r.Elevation[nbnb] <= 0 {
			cost /= 2.0
			break
		}
	}

	// Bonus for moving along rivers.
	// TODO: This should depend on the culture of the tribe.
	// River cultures should not be penalized for crossing rivers.
	if n.r.isDownstream(n.index, nIdx) || n.r.isUpstream(n.index, nIdx) {
		cost /= 2.0
	} else if n.r.IsRegRiver(n.index) != n.r.IsRegRiver(nIdx) {
		cost *= 1.4 // Cost of crossing rivers.
	}

	// Penalty for crossing into a new territory
	if n.r.RegionToEmpire[n.index] != n.r.RegionToEmpire[nIdx] {
		cost *= 2.0
	}

	// Penalty for crossing into a new culture.
	if n.r.RegionToCulture[n.index] != n.r.RegionToCulture[nIdx] {
		cost *= 2.0
	}

	return cost
}

// PathEstimatedCost is a heuristic method for estimating movement costs
// between non-adjacent nodes.
func (n *MigrationTile) PathEstimatedCost(to goastar.Pather) float64 {
	return n.r.GetDistance(n.index, to.(*MigrationTile).index)
}

type Path struct {
	From, To int
	Steps    []int
	Idx      int
}

func planPath(from, to *MigrationTile) (*Path, bool) {
	path, _, found := goastar.Path(from, to)
	if !found {
		return nil, false
	}
	p := &Path{
		From: from.index,
		To:   to.index,
	}
	for _, n := range path {
		nti := n.(*MigrationTile)
		p.Steps = append(p.Steps, nti.index)
	}
	return p, true
}

// Peek returns the next region in the path without advancing the path.
func (p *Path) Peek() int {
	if p.Idx >= len(p.Steps) {
		return -1
	}
	return p.Steps[len(p.Steps)-p.Idx-1]
}

// Next returns the next region in the path and advances the path.
func (p *Path) Next() int {
	if p.Idx >= len(p.Steps) {
		return -1
	}
	// The path is reversed, so we need to go backwards.
	next := p.Steps[len(p.Steps)-p.Idx-1]
	p.Idx++
	return next
}

// PeekDone returns true if the path is done after the next step.
func (p *Path) PeekDone() bool {
	return p.Idx+1 >= len(p.Steps)
}

// Done returns true if the path is done.
func (p *Path) Done() bool {
	return p.Idx >= len(p.Steps)
}

// NumRemaining returns the number of steps remaining in the path.
func (p *Path) NumRemaining() int {
	return len(p.Steps) - p.Idx
}
