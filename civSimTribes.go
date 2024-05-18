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
		regRes.Remove(localRes).Log()
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
		Lake:     s.m.lakeProxFunc(r),
		Ocean:    s.m.oceanProxFunc(r),
		Mountain: s.m.Elevation[r] > 0.3,
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

type cityDist struct {
	city *City
	dist float64
}

// getNearbyCities returns the cities that are within the given radius of the region.
func (s *simState) getNearbyCities(r int, radius float64) []*cityDist {
	var res []*cityDist
	for _, c := range s.cities {
		if c.ID == r {
			continue
		}
		dist := s.m.Geo.GetDistance(r, c.ID) * unitDistToKm
		if dist < radius {
			res = append(res, &cityDist{city: c, dist: dist})
		}
	}

	// Sort the cities by distance.
	sort.Slice(res, func(i, j int) bool {
		return res[i].dist < res[j].dist
	})
	return res
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
		resPres := m.getResourcePresence(t.RegionID)

		// Develop the skills of the tribe given the current region.
		t.DevelopSkills(&curRegProp, resPres)

		// Exhaust the resources of the region.
		m.SoilExhaustion[t.RegionID] += float64(t.Population) / t.SustainabilityFactor(&curRegProp, resPres)

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
				if curRegProp.Mountain {
					possibleBadThings = append(possibleBadThings, "earthquake", "rockslide")
				}
				if curRegProp.River {
					possibleBadThings = append(possibleBadThings, "flood")
				}
				if curRegProp.Ocean {
					possibleBadThings = append(possibleBadThings, "malaria")
				}
				if curRegProp.Lake {
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

type ClampedVal float64

func (c *ClampedVal) Add(v float64) {
	*c = ClampedVal(math.Max(0, math.Min(1, float64(*c)+v)))
}
