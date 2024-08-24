package genworldvoronoi

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
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
	t := NewTribe(bestRegion, initialPopulation, m, nil)
	// TODO: Seed the tribe with the population.
	t.People = m.placePopulationAt(bestRegion, initialPopulation, func(r int) *Culture {
		return t.Culture
	})
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
			// Log leadership and their parents (if any).
			for _, p := range t.Leadership.Leaders() {
				log.Printf("    %s", p.String())
				log.Printf("    %s", p.StringGenes())
				if p.Father != nil {
					log.Printf("      Father: %s", p.Father.String())
					log.Printf("      %s", p.Father.StringGenes())
				}
				if p.Mother != nil {
					log.Printf("      Mother: %s", p.Mother.String())
					log.Printf("      %s", p.Mother.StringGenes())
				}
			}
		}
		// Log all factions.
		for _, f := range t.Factions {
			log.Printf("  Faction: %s", f.String())
			// Log leadership and their parents (if any).
			for _, p := range f.Leaders() {
				log.Printf("    %s", p.String())
				log.Printf("    %s", p.StringGenes())
				if p.Father != nil {
					log.Printf("      Father: %s", p.Father.String())
					log.Printf("      %s", p.Father.StringGenes())
				}
				if p.Mother != nil {
					log.Printf("      Mother: %s", p.Mother.String())
					log.Printf("      %s", p.Mother.StringGenes())
				}
			}
		}
		// Log all skills.
		for sk := range t.Skills {
			log.Printf("  %s", sk.Name)
		}
		// Log dumbStorage.
		log.Printf("  %s", t.dumbStorage.String())
		if t.Settlement != nil {
			log.Printf("  Settlement: %s", t.Settlement.String())
			log.Printf("  %s", t.Settlement.dumbStorage.String())
			log.Printf(" Resouces (settlement storage):")
			t.Settlement.ResourceStorage.Log()
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

		// Log the population that is alive.
		log.Printf("  Population: %d", t.Population)
		log.Printf("  Num people: %d", len(t.People))
		for _, p := range t.People {
			log.Printf("    %s", p.String())
			log.Printf("    %s", p.StringGenes())
			if p.Father != nil {
				log.Printf("      Father: %s", p.Father.String())
				log.Printf("      %s", p.Father.StringGenes())
			}
			if p.Mother != nil {
				log.Printf("      Mother: %s", p.Mother.String())
				log.Printf("      %s", p.Mother.StringGenes())
			}
		}

		log.Printf(" Resources (local):")
		localRes := m.getResources(t.RegionID, false)
		localRes.Log()
		log.Printf(" Resources (regional):")
		regRes := m.getResources(t.RegionID, true)
		regRes.Remove(localRes).Log()
		log.Printf(" Resouces (storage):")
		t.ResourceStorage.Log()
		log.Printf("  History:")
		ref := t.Ref()
		for _, h := range m.History.GetEvents(ref.ID, ref.Type) {
			log.Printf("    %d: %s", h.Year, h.String())
		}
	}

	// Log the city states.
	for _, cs := range m.CityStates {
		cs.Log()
		log.Printf("  %s", cs.dumbStorage.String())
		log.Printf(" Resouces (city state storage):")
		cs.ResourceStorage.Log()
	}

	// Log the empires.
	for _, e := range m.Empires {
		e.Log()
		log.Printf("  %s", e.dumbStorage.String())
		log.Printf(" Resouces (empire storage):")
		e.ResourceStorage.Log()
	}

	// Log history.
	log.Println("History:")
	for _, h := range m.History.Events {
		log.Printf("%d: %s", h.Year, h.String())
	}
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

// moveTribe moves the tribe to the new region.
func (s *simState) moveTribe(t *Tribe, r int) {
	// If the tribe is at the old region, remove it.
	if s.tribeAtRegion[t.RegionID] == t {
		s.tribeAtRegion[t.RegionID] = nil
	}
	// Calculate the distance that we have to travel.
	dist := s.m.GetDistance(t.RegionID, r) * unitDistToKm
	log.Printf("Tribe %s is moving from region %d to region %d (%.2f km)", t.String(), t.RegionID, r, dist)

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
	log.Printf("Switching tribes %s and %s", t1.String(), t2.String())
	// Switch the regions of the two tribes.
	region1 := t1.RegionID
	region2 := t2.RegionID
	if t1.RegionID == t2.RegionID {
		panic("tribes are already in the same region")
	}
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

type RegionProp struct {
	Biome HackyBiome
	RegionProximity
}

type RegionProximity struct {
	River    bool
	Lake     bool
	Ocean    bool
	Mountain bool
}

func (r *RegionProp) Log() {
	log.Printf("  biome: %s", r.Biome)
	log.Printf("  river proximity: %v", r.River)
	log.Printf("  lake proximity: %v", r.Lake)
	log.Printf("  ocean proximity: %v", r.Ocean)
	log.Printf("  mountain proximity: %v", r.Mountain)
}

// getRegionProp returns the properties of the region.
func (s *simState) getRegionProp(r int) RegionProp {
	// TODO: Find better way to determine mountain proximity. 0.3 is very low.
	return RegionProp{
		Biome:           HackyBiome(s.biomeFunc(r)),
		RegionProximity: s.getRegionProx(r),
	}
}

// getRegionProx returns the proximity of the region to rivers, lakes, oceans, and mountains.
func (s *simState) getRegionProx(r int) RegionProximity {
	return RegionProximity{
		River:    s.m.IsRegRiver(r),
		Lake:     s.m.LakeProxFunc(r),
		Ocean:    s.m.OceanProxFunc(r),
		Mountain: s.m.Elevation[r] > 0.3,
	}
}

// Fake the lake proximity function.
func (m *Civ) LakeProxFunc(r int) bool {
	for _, nb := range m.R_circulate_r(rNbs, r) {
		if m.IsRegLakeOrWaterBody(nb) && m.WaterbodySize[nb] > 5 {
			return true
		}
	}
	return false
}

// Fake the ocean proximity function.
func (m *Civ) OceanProxFunc(r int) bool {
	for _, nb := range m.R_circulate_r(rNbs, r) {
		if m.Elevation[nb] <= 0 && m.WaterbodySize[nb] > 5 {
			return true
		}
	}
	return false
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
	// However, certain metals, gems, etc. should only give an increased bonus, if
	// there are other cities nearby that can trade with them.
	if res.Metals != 0 {
		val := float64(res.Metals) / (1 << geo.ResMaxMetals)
		val = math.Sqrt(val) // Sqrt to make it less linear.
		val += 0.2           // Base bonus.
		val = min(val, 1.0)  // Limit to 1.0.
		multiplier = max(multiplier, val)
	}
	if res.Gems != 0 {
		val := float64(res.Gems) / (1 << geo.ResMaxGems)
		val = math.Sqrt(val) // Sqrt to make it less linear.
		val += 0.2           // Base bonus.
		val = min(val, 1.0)  // Limit to 1.0.
		multiplier = max(multiplier, val)
	}
	if res.Various != 0 {
		val := float64(res.Various) / (1 << geo.ResMaxVarious)
		val = math.Sqrt(val) // Sqrt to make it less linear.
		val += 0.2           // Base bonus.
		val = min(val, 1.0)  // Limit to 1.0.
		multiplier = max(multiplier, val)
	}

	// Penalize if the region doesn't have resources the tribe needs.
	// We need wood for fuel, construction, and tools.
	if res.Wood == 0 {
		multiplier -= 0.2
	}
	// We need stones for construction and tools.
	if res.Stones == 0 {
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
		// Move the new tribe to a new region.
		var foundNewRegion bool
		// TODO: Sort regions by current max population and bound the new tribe's population to the max population of
		// the most suitable region.
		for _, nb := range s.m.R_circulate_r(rNbs, t.RegionID) {
			// Do not place them in oceans or lakes.
			if s.m.IsRegLakeOrWaterBody(nb) || s.m.Elevation[nb] <= 0 {
				continue
			}
			if s.tribeAtRegion[nb] == nil {
				newTribe := s.placeTribeAt(nb, rand.Intn(t.Population)/2, t, true)

				// If the original tribe satisfaction was low, we increase the satisfaction of the new tribe.
				if t.Satisfaction < 0.3 {
					newTribe.changeSatisfaction(tribeSplitVoluntarySatisfaction)
				}
				foundNewRegion = true
				break
			}
		}
		if !foundNewRegion {
			panic("no region found for new tribe")
		}
	}
}

func (m *Civ) tickSimTribes() {
	_, maxElevation := minMax(m.Elevation)
	s := simState{
		m:              m,
		newTribes:      make([]*Tribe, 0, len(m.Tribes)),
		tribeAtRegion:  make([]*Tribe, m.NumRegions),
		isCity:         make(map[int]bool),
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
		if t.Population == 0 {
			// The tribe has died out.
			continue
		}
		// Make sure that two tribes are not in the same region.
		if s.tribeAtRegion[t.RegionID] != nil {
			occupiers := s.tribeAtRegion[t.RegionID]
			log.Println("Tribe", t.ID, "cannot be in region", t.RegionID)
			log.Printf("Tribe %d (pop %d) is already in region %d.", occupiers.ID, occupiers.Population, t.RegionID)
			log.Printf("Tribe %d (pop %d) is trying to settle in region %d.", t.ID, t.Population, t.RegionID)
			panic("tribe already in region")
		}
		s.tribeAtRegion[t.RegionID] = t
	}

	// Take note of the regions that are already cities.
	for _, c := range m.Cities {
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
		// If the tribe has died out, skip it.
		if t.Population == 0 {
			s.tribeAtRegion[t.RegionID] = nil
			continue
		}
		s.newTribes = append(s.newTribes, t)

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
		resPres := m.getResourcePresence(t.RegionID)

		// Develop the skills of the tribe given the current region.
		t.DevelopSkills(&curRegProp, resPres, m.History)

		// Update the professions of the tribe.
		t.AssignProfessions(m)

		// Exhaust the resources of the region.
		m.SoilExhaustion[t.RegionID] += float64(t.Population) / t.SustainabilityFactor(&curRegProp, resPres)

		// Calculate the max population for the region.
		maxRegionPop := s.calcMaxPopPerRegion(t, t.RegionID)
		t.regionMaxPop = s.calcTheoreticalMaxPopPerRegion(t, t.RegionID)
		t.currentRegionMaxPop = maxRegionPop

		// Get the multiplier for the region, which will double as the satisfaction of the tribe.
		// This will be used to determine if the tribe is happy or not with the current region.
		t.changeSatisfaction(0.5 * (s.getRegMultiplier(t, t.RegionID) - float64(t.Satisfaction)))
		if t.Satisfaction < 0.7 {
			s.printRegionInfo(t.RegionID)
			t.printTribePreferences()
		}

		// TODO: Tick culture and religion.
		if t.Culture != nil {
			// t.Culture.Tick()
			// The culture can develop new specializations, etc. based on the region(s) they are in.
			// If they have access to specific resources or have a above average number of poeple with a
			// specific occupation, they might develop a new specialization or become renowned for a specific
			// skill.
			// - This could also lead to the tribe becoming known for a specific product, etc.
			// - The tribe might also develop a bonus for a specific action, etc.
			// - The tribe might also develop a new skill, etc.
			//
			// A culture would have certain customs that are acceptable or inacceptable like:
			// - Eating habits (cannibalism, vegetarianism, etc.)
			// - Clothing (nudity, specific colors, etc.)
			// - Marriage (polygamy, monogamy, etc.)
			// - Social structure (caste system, etc.)
			// - Slavery (acceptable, inacceptable, etc.)
			//
			// The culture might also have specific rituals, etc.
			// - Birth rituals
			// - Death rituals
			// - Marriage rituals
			// - Coming of age rituals
			// - War rituals
			//
			// Certain events might be more or less important to the culture, etc.
			//
			// Certain behaviors or characteristics might be more or less important.
			// Base stats (physical):
			// - Strength
			// - Intelligence
			// - Dexterity
			// - Resilience

			// The culture might also have certain virtues, etc.
			// - Brave / Craven
			// - Calm / Wrathful
			// - Chaste / Lustful
			// - Content / Ambitious
			// - Diligent / Lazy
			// - Forgiving / Vengeful
			// - Generous / Greedy
			// - Gregarious / Solitary
			// - Honest / Deceitful
			// - Humble / Arrogant
			// - Just / Arbitrary
			// - Patient / Impatient
			// - Temperate / Indulgent
			// - Trusting / Paranoid
			// - Zealous / Cyincal
			// - Compassionate / Callous / Cruel
			// - Fickle / Steadfast / Eccentric

			// There are actions a culture can take which have associated attributes or virtues.
			// Depending on the outcome, the culture might start to value certain virtues more or less.

			// Depending on the environment, and the economy, the culture might value certain stats more or less.
			// - In a harsh environment, resilience might be more important.
			// - A culture depending on hunting might value strength or dexterity more.
			// - A culture depending on trade might value intelligence more.
			// etc.

			// We need to compare the current requirements for the culture with what the tribe values.
			// If the tribe values the same things that are required of them, they will be more satisfied.
			// If there is a large discrepancy, they will be less satisfied and there might be unrest, etc.
		}

		if t.Religion != nil {
			// TODO:
			// - Determine a value representing how "in tune" the tribe is with their religion.
			// - This could be a running value that is calculated based on the actions of the tribe.
			// - If an action is in line with the religion, the value goes up, otherwise it goes down.
			// - If the value is high, good things might happen, if it is low, bad things might happen.

			// There should be values that track the religion:
			// - Piety
			// - Happiness
			// - ...
			//
			// Any action directly related to the religion should increase the piety of the tribe
			// but can either increase or decrease the happiness of the tribe.
			// A malevolent god might cause misfortune and plagues, etc.
		}

		// Random events.
		s.handleEvents(t, curRegProp)

		// Update the leadership and factions.
		s.handleLeadership(t)

		// Handle resource extraction.
		s.handleResources(t)

		// There is a chance that part of the tribe will split off and form a new tribe to
		// follow some other vision or to settle in a different region.
		if t.Type > TribeTypeNomadic {
			s.checkRandomSplit(t)
		}

		if t.Type == TribeTypeNomadic {
			// The tribe is still nomadic, so we migrate the tribe to the most suitable region.
			// If the tribe is too large for the region, we split the tribe into two or more tribes.
			// The tribe might die out if it cannot find a suitable region to move to.
			s.handleNomadicTribe(t)
		} else if t.Type == TribeTypeSettling {
			// The tribe has decided to settle, so we move to the most suitable region within a certain radius.
			s.handleSettlingTribe(t)
		} else {
			// We have settled, so we need to update all the things we are in charge of.
			s.handleSettledTribe(t)
		}

		// Update the tribe's population.
		t.People = s.m.tickPeople(t.People, 365, func(r int) *Culture {
			return t.Culture
		}, t.Population, t.RegionID)
		for _, p := range t.People {
			if p.Dead() {
				continue
			}
			// Update the location of the person.
			m.updatePersonLocation(p, t.RegionID)
		}

		// Check if the settlement has people that are not in the tribe.
		if t.Settlement != nil {
			// Compare the people in the tribe vs the people in the settlement.
			seenTribePeople := make(map[int]bool)
			for _, p := range t.People {
				seenTribePeople[p.ID] = true
			}
			for _, p := range t.Settlement.People {
				if !seenTribePeople[p.ID] {
					log.Println("Person", p.ID, "is in settlement", t.Settlement.ID, "but not in tribe", t.ID)
				}
			}
		}

		log.Printf("Tribe %s has population %d and assigned %d people", t.String(), t.Population, len(t.People))
		if t.Settlement != nil {
			log.Printf("Tribe %s has settlement %s", t.String(), t.Settlement.String())
			log.Printf("Tribe %s has settlement with %d people", t.String(), len(t.Settlement.People))
		}
	}

	// Check if we need to expand the cultures.
	// TODO: Make expansion dependent on prosperity of the cultures.
	// TODO: Check if culture is extinct.
	var cultureSeeds []int
	for _, c := range s.newTribes {
		if c.Culture != nil && c.Population > 0 {
			cultureSeeds = append(cultureSeeds, c.Culture.ID)
		}
	}
	for _, c := range s.m.Cultures {
		cultureSeeds = append(cultureSeeds, c.ID)
	}
	if len(cultureSeeds) > 0 {
		m.expandCultures(true, dedupInts(cultureSeeds))
	}

	// Check if we need to expand the city states.
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

	// Check if we need to expand the empires.
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

const numYears = 40000 // Number of years to simulate.

type HackyBiome int

func (b HackyBiome) String() string {
	return genbiome.WhittakerModBiomeToString(int(b))
}

type ClampedVal float64

func (c *ClampedVal) Add(v float64) {
	*c = ClampedVal(math.Max(0, math.Min(1, float64(*c)+v)))
}
