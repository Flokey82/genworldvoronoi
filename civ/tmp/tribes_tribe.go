package civ

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"

	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/genworldvoronoi/geo"
	"github.com/Flokey82/go_gens/gengovernment"
	"github.com/Flokey82/go_gens/genlanguage"
)

var tribeID int

func getNextTribeID() int {
	id := tribeID
	tribeID++
	return id
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
	ID               int       // Unique ID for the tribe.
	Population       int       // Population of the tribe.
	MaxPopulation    int       // Max population of the tribe.
	Culture          *Culture  // Culture that the tribe might have developed.
	Religion         *Religion // Religion that the tribe might have developed.
	*GoverningPeople           // People governing the tribe.
	People           []*Person // People of the tribe.
	*ComboStorage              // Resources the tribe has.

	Type       TribeType             // The type of the tribe (nomadic, settled, city state, empire, etc.)
	Parent     *Tribe                // The parent tribe of the tribe.
	Language   *genlanguage.Language // The language of the tribe.
	Settlement *City                 // The city the tribe has settled in.
	CityState  *CityState            // City state that the tribe might have developed.
	Empire     *Empire               // Empire that the tribe might have developed.
	Skills     map[*Skill]bool       // Skills of the tribe.

	// Preferences of the tribe.
	TribePreference

	RegionID     int  // The region the tribe is currently in.
	*Path             // Pathfinding for migrating tribes.
	gotVision    bool // If the tribe got their destination to settle from a vision from the gods.
	doneSettling bool // If the tribe has settled in a region.

	regionMaxPop        int // The max pop of the region the tribe is in.
	currentRegionMaxPop int // The current max pop of the region the tribe is in.
}

// NewTribe returns a new tribe with the given population.
func NewTribe(regionID, population int, m *Civ, c *Culture) *Tribe {
	var lang *genlanguage.Language
	if c != nil {
		lang = c.Language
	} else {
		lang = GenLanguage(int64(regionID))
		// Get local culture or place a new culture if none is present.
		if c = m.GetCulture(regionID); c == nil {
			if c = m.PlaceCultureAt(regionID, false, lang); c == nil {
				panic("Culture could not be placed.")
			}
		}
	}

	// Initialize the last 10 biomes with the current as -1.
	var last100Biomes [100]int
	for i := range last100Biomes {
		last100Biomes[i] = -1
	}
	t := &Tribe{
		ID:              getNextTribeID(),
		RegionID:        regionID,
		TribePreference: newTribePreference(),
		Skills:          make(map[*Skill]bool),
		Type:            TribeTypeNomadic,
		Population:      population,
		GoverningPeople: newGoverningPeople(),
		Language:        lang,
		Culture:         c,
		ComboStorage:    newComboStorage(10),
	}
	// Generate leadership.
	t.Leadership = genFaction(t, m, gengovernment.LeadershipFormChiefdom, nil)
	return t
}

func (s *simState) placeTribeAt(r, pop int, parent *Tribe, randomize bool) *Tribe {
	if s.tribeAtRegion[r] != nil {
		panic(fmt.Sprintf("Tribe already at region %d", r))
	}
	var t *Tribe
	if parent != nil {
		// We split off a new tribe from the parent tribe.
		t = parent.Split(pop, randomize, s.m)
		historyMsg := fmt.Sprintf("Tribe %d (%d) split off tribe %d (%d) under the leadership of %s", t.ID, t.Population, parent.ID, parent.Population, t.Leadership.Name)
		if randomize {
			historyMsg += " (randomly)"
		}
		s.m.History.AddEvent("Founding (Tribe)", historyMsg, t.Ref())
	} else {
		// Since there is no parent, we found a new tribe.
		t = NewTribe(r, pop, s.m, nil)
		// TODO: Seed the tribe with the population.
		t.People = s.m.placePopulationAt(r, pop, func(r int) *Culture {
			if t.Culture == nil {
				panic("Culture not found.")
			}
			return t.Culture
		})
		historyMsg := fmt.Sprintf("Tribe %d (%d) was founded by %s", t.ID, t.Population, t.Leadership.Name)
		s.m.History.AddEvent("Founding (Tribe)", historyMsg, t.Ref())
	}
	s.moveTribe(t, r)
	s.newTribes = append(s.newTribes, t)
	log.Println("TODO: Generate culture.")
	return t
}

func (t Tribe) GetID() int {
	return t.ID
}

// Ref returns the object reference of the city.
func (t *Tribe) Ref() ObjectReference {
	return ObjectReference{
		ID:   t.ID,
		Type: ObjectTypeTribe,
	}
}

func (t *Tribe) String() string {
	// Assemble the preferred proximity string.
	proxStr := t.TribePreference.String()
	var cultureStr string
	if t.Culture != nil {
		cultureStr = " " + t.Culture.Type.String()
	}
	var leaderStr string
	if t.Leadership != nil {
		leaderStr = " " + t.Leadership.String()
	}
	return fmt.Sprintf("Tribe %d (%s) in region %d with population %d; satisfaction %.2f (%q)%s%s", t.ID, proxStr, t.RegionID, t.Population, t.Satisfaction, t.GetCultureType(), cultureStr, leaderStr)
}

func (t *Tribe) getLanguage() *genlanguage.Language {
	return t.Language
}

func (t *Tribe) compare(other *Tribe) float64 {
	// Positive is good, negative is bad.
	// Compare preferences.
	prefValue := t.TribePreference.compare(&other.TribePreference)

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
func (t *Tribe) makeNomadic(h *History) {
	// TODO: If the tribe is in charge of a city state or empire, we'd need to handle that.
	// Will we let the empire collapse, or do we allow the tribe to control the empire nomadically?
	t.Type = TribeTypeNomadic
	t.SetPath(nil)
	t.doneSettling = false
	delete(t.Skills, SSkillSettling)
	delete(t.Skills, SSkillSettlingHighland)
	delete(t.Skills, SSkillSettlingWetlands)
	h.AddEvent("Nomadic", fmt.Sprintf("Tribe %d (%d) has become nomadic.", t.ID, t.Population), t.Ref())
}

func (t *Tribe) changeSatisfaction(diff float64) {
	origSat := t.Satisfaction
	t.Satisfaction.Add(diff)
	t.SatisfactionAvg.Add(float64(t.Satisfaction))
	if origSat != t.Satisfaction {
		log.Printf("Tribe %d satisfaction changed from %.2f to %.2f (rnng avg %.2f)", t.ID, origSat, t.Satisfaction, t.SatisfactionAvg.Current())
	}

	// If the satisfaction has decreased, the leadership popularity should decrease,
	// and the faction popularity should increase... and vice versa if the satisfaction
	// has increased.
	t.Leadership.Popularity.Add(diff)

	// We distribute the satisfaction change to the factions based on their influence.
	var sumPop float64
	for _, f := range t.Factions {
		sumPop += float64(f.Popularity)
	}

	if sumPop == 0 {
		log.Printf("Tribe %d has no factions to distribute the satisfaction change to.", t.ID)
	} else {
		// Now we distribute the satisfaction change to the factions.
		// TODO:
		// There should be some additional randomness in the distribution.
		// Also it should depend on the event type and the individual faction
		// if the faction would benefit from the event or not.
		for _, f := range t.Factions {
			f.Popularity.Add(diff * float64(f.Popularity) / sumPop) // Clamp to [0, 1]
		}
	}
}

// GetCultureType returns the culture type of the tribe.
func (t *Tribe) GetCultureType() CultureType {
	preferredBiome, score := t.LastBiomes.Preferred()
	if !(t.Skills[SSkillSettling] || t.Skills[SSkillSettlingWetlands] || t.Skills[SSkillSettlingHighland]) && isInIntList(nomadicBiomes, int(preferredBiome)) && score > 0.5 {
		return CultureTypeNomadic
	}
	if t.MountainProximity.Current() && t.RiverProximity.Current() {
		if t.MountainProximity.avg > t.RiverProximity.avg {
			return CultureTypeHighland
		}
		return CultureTypeRiver
	}
	if t.MountainProximity.Current() {
		return CultureTypeHighland
	}
	if t.RiverProximity.Current() {
		return CultureTypeRiver
	}
	if t.LakeProximity.Current() {
		return CultureTypeLake
	}
	if t.OceanProximity.Current() {
		return CultureTypeNaval
	}
	if t.Skills[SSkillHunting] && isInIntList(huntingBiomes, int(preferredBiome)) {
		return CultureTypeHunting
	}
	return CultureTypeGeneric
}

// NewRandomPerson returns a new random person which is part of the tribe.
func (t *Tribe) NewRandomPerson(m *Civ, gender geneticshuman.Gender) *Person {
	p := m.newRandomPersonAt(t.RegionID, t.Culture, gender, nil)
	t.People = append(t.People, p)
	return p
}

// NewRandomChild returns a new random child which is part of the tribe and
// is a child of the given person.
func (t *Tribe) NewRandomChild(m *Civ, gender geneticshuman.Gender, parent *Person) *Person {
	p := m.newRandomPersonAt(t.RegionID, t.Culture, gender, parent)
	t.People = append(t.People, p)
	return p
}

// GetPeople returns the people of the tribe.
func (t *Tribe) GetPeople() []*Person {
	return t.People
}

// GetGoverningPeople returns the governing people of the tribe.
func (t *Tribe) GetGoverningPeople() *GoverningPeople {
	return t.GoverningPeople
}

const (
	growthRateSettled = 0.001 // 0.1% growth rate per year
	growthRateNomadic = 0.001 // 0.1% growth rate per year for nomadic tribes
)

// Grow the population of the tribe for one year.
func (t *Tribe) Grow(days int) {
	// Get the growth rate of the tribe.
	// TODO: The growth rate should depend on the region type, the resources, etc.
	gr := growthRateNomadic
	if t.Type > TribeTypeSettling {
		gr = growthRateSettled
	}

	// Calculate the population growth rate for the tribe.
	// Use the exponential growth model.
	newPop := float64(t.Population) * math.Pow(math.E, gr*float64(days)/365)
	if diff := newPop - float64(t.Population); diff < 1 {
		// Use rand to potentially grow the population by one.
		if rand.Float64() < diff*float64(days)/365 {
			t.Population++
		}
	} else {
		t.Population = int(newPop)
	}

	// Update the max population of the tribe.
	if t.Population > t.MaxPopulation {
		t.MaxPopulation = t.Population
	}

	// Update the population of the settlement.
	if t.Settlement != nil {
		t.Settlement.Population = t.Population
	}
}

// Split the tribe into two tribes with a new one with the given population.
func (t *Tribe) Split(newPopulation int, randomize bool, m *Civ) *Tribe {
	t.Population -= newPopulation
	nt := NewTribe(t.RegionID, newPopulation, m, t.Culture) // TODO: Generate a unique ID.
	nt.Type = t.Type
	nt.Parent = t
	nt.Satisfaction = t.Satisfaction // Maybe that should be different?
	t.SatisfactionAvg.CopyTo(nt.SatisfactionAvg)

	// Copy the last biomes and cultures to the new tribe.
	t.TribePreference.CopyTo(&nt.TribePreference)

	// TODO: If we have a culture, we should copy it to the new tribe.
	// We should also track religion and we either mutate the religion or
	// or the culture of the new tribe.
	nt.Language = t.Language
	nt.Religion = t.Religion // For now, we use the same religion.
	nt.Culture = t.Culture   // For now, we use the same culture.

	// Copy the skills to the new tribe.
	for s := range t.Skills {
		nt.Skills[s] = true
	}

	// Generate leadership if no other faction exists.
	if len(t.Factions) == 0 {
		nt.Leadership = genFaction(nt, m, nt.getPreferredLeadershipForm(), nil)
	} else {
		// Promote the most popular faction to the new tribe.
		// TODO: The population of the new tribe should depend on the popularity of the faction.
		sort.Slice(t.Factions, func(i, j int) bool {
			return t.Factions[i].Popularity > t.Factions[j].Popularity
		})
		nt.Leadership = t.Factions[0]
		t.Factions = t.Factions[1:]

		// Make sure to move all people from the faction to the new tribe.
		mustMoveToNT := make([]*Person, 0, newPopulation)
		mustMoveToNT = append(mustMoveToNT, nt.Leadership.Leaders()...)

		// Make sure that the old leadership remains in the old tribe.
		mustRemainInOldT := make([]*Person, 0, len(t.People)-newPopulation)
		mustRemainInOldT = append(mustRemainInOldT, t.Leadership.Leaders()...)

		// Also keep all the remaining faction leaders in the old tribe.
		for _, f := range t.Factions {
			mustRemainInOldT = append(mustRemainInOldT, f.Leaders()...)
		}

		// Pick the people to move to the new tribe.
		// Make sure that spouses and children are moved as well.
		mustMoveToNT = m.PickNPeopleToMigrate(t.People, mustMoveToNT, mustRemainInOldT, newPopulation)

		// Keep track of the people that have been moved to the new tribe.
		ntPeopleSeen := make(map[*Person]bool)
		for _, p := range mustMoveToNT {
			ntPeopleSeen[p] = true
		}

		mustRemainInOldT = make([]*Person, 0, len(t.People)-newPopulation)
		for _, p := range t.People {
			if ntPeopleSeen[p] {
				continue
			}
			mustRemainInOldT = append(mustRemainInOldT, p)
		}
		t.People = mustRemainInOldT
		nt.People = mustMoveToNT
	}

	if randomize {
		// There is a random chance that the tribe will leave due to a vision or a spiritual calling.
		nt.gotVision = rand.Intn(100) < 50

		if rand.Intn(100) < 10 {
			// There is a chance that the new tribe will revert to being nomadic.
			// TODO: Find a way to have a religion that doesn't require a spiritual center.
			// One way to solve this is to have a place of pilgrimage that the tribe can visit
			// but then we'd have to make sure that the religion doesn't really spread from there
			// but rather from the practitioners (tribes, etc.).
			nt.makeNomadic(m.History)
		} else {
			// Make sure the tribe will look for a new region to settle in.
			nt.Type = TribeTypeSettling
			nt.SetPath(nil)
			nt.doneSettling = false
		}

		// There is a chance that the new tribe will have some preferences reset.
		nt.TribePreference.RandomReset()
	}
	return nt
}

// DevelopSkills develops the skills of the tribe based on the biome and region type.
func (t *Tribe) DevelopSkills(curRegProp *geo.RegionProp, resPres *ResourcePresence, h *History) {
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
		if s.CanDevelopIn(curRegProp, resPres) && s.DevelopAt(t.LastBiomes.GetScoreOf(curRegProp.Biome)) {
			t.Skills[s] = true
			s.EffectOnTribe(t)

			// Add a history entry.
			historyMsg := fmt.Sprintf("Tribe %d developed the skill %s", t.ID, s.Name)
			h.AddEvent("Development", historyMsg, t.Ref())
		}
	}
}

func (t *Tribe) printTribePreferences() {
	log.Printf("Tribe %d preferences:", t.ID)
	curTribePref := t.getPreferredRegionProp()
	curTribePref.Log()
}

func (t *Tribe) SustainabilityFactor(curRegProp *geo.RegionProp, resPres *ResourcePresence) float64 {
	// TODO: This should depend on the region (whether it's a desert, etc.)
	// Depending on the skills and knowledge of the tribe, we can calculate
	// the sustainability factor of the tribe.
	factor := 1.0
	for _, s := range Skills {
		if t.Skills[s] && s.CanDevelopIn(curRegProp, resPres) {
			factor += 0.1
		}
	}

	return factor
}
