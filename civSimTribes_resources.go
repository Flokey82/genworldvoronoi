package genworldvoronoi

import (
	"log"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
)

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

	log.Printf("!!!%s is producing food: %d total; %d from hunting (%.2f, %d people), %d from gathering  %.2f, %d people).", t.String(), int(foodHunting+foodGathering), int(foodHunting), regHunting, int(popHunting), int(foodGathering), regGathering, int(popGathering))

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

	// Nomadic tribes will just hunt and gather.
	// They can produce some simple tools and simple weapons that will increase their efficiency
	// but they cannot build buildings or produce goods that require a settlement.

	// Settled tribes will produce food, tools, weapons, and buildings.
	// Buildings:
	// - Require goods and resources to be constructed.
	// - Provide certain benefits to the tribe.
	// - They might consume resources or goods each turn and potentially provide resources or goods.
	// - They might need maintenance.
	// Goods:
	// - Are produced from resources or other goods.
	// - They might be consumed by the tribe, or traded.

	// TODO:
	// - Calculate how much food and firewood we need.
	// - Calculate need for buildings (housing, etc.)
	// - Calculate need for maintenance resources (wood, stone, metal, etc.)
	// Determine what we have and what we need.
	// - Calculate the resources we have.
	// - Calculate the resources we need.

	// Generate resources based on the local resources.
	// If we aren't settled, we only can gather resources from the region we are in.
	// TODO: Introduce separate resource storage for settlements, city states, and empires.
	var res LocalResouces
	var storage *ComboStorage
	if t.Type <= TribeTypeSettling {
		res = s.m.getResources(t.RegionID, false)
		storage = t.ComboStorage
	} else {
		res = s.m.getResources(t.RegionID, true)
		storage = t.Settlement.ComboStorage
	}

	// Add the resources to the storage.
	storage.ResourceStorage.Add(res)
	storage.resources[StorageWood] = sumResource(storage.Wood[:])
	storage.resources[StorageFood] += int(foodHunting + foodGathering)
	storage.resources[StorageLeather] += int(foodHunting)
	storage.resources[StorageWood] += int(foodGathering)

	// TODO: Move food and firewood consumption to a separate function.
	const requiredFoodPerPerson = 1.0
	const requiredFirewoodPerPerson = 0.5 // This should depend on the climate

	foodNeeded := t.Population * requiredFoodPerPerson
	storage.resources[StorageFood] -= foodNeeded
	if storage.resources[StorageFood] < 0 {
		log.Printf("Tribe %s needs more food. %d/%d", t.String(), storage.resources[StorageFood], foodNeeded)
		storage.resources[StorageFood] = 0 // TODO: Kill off some people
	}

	firewoodNeeded := int(float64(t.Population) * requiredFirewoodPerPerson)
	storage.resources[StorageWood] -= firewoodNeeded
	if storage.resources[StorageWood] < 0 {
		log.Printf("Tribe %s needs more firewood. %d/%d", t.String(), storage.resources[StorageWood], firewoodNeeded)
		storage.resources[StorageWood] = 0 // TODO: Kill off some people
	}

	// If we lack food or firewood, we need to find a strategy to get more.
	// - We can trade for it.
	// - We can produce more (more efficiently, or just more of it)

	// Further we should check here how much housing we need and how much we have.

	// Build stuff.
	s.buildThings(t)

	// TODO: Maintain stuff.
	// s.maintainThings(t)

	// TODO: We have a resource budget.
	// - We produce resources
	// - We consume resources
	// - We need resources to build stuff as one-time costs.

	// So first we need to establish our current resource situation.
	// ... then we figure out what we need to build and maintain.
}

const (
	ResCategoryWood = iota
	ResCategoryStone
	ResCategoryMetal
	ResCategoryGem
	ResCategoryVarious
)

const ResourceTypeAny = -1

// NOTE: This is only about resources.
func (m *Civ) compareResources(src, dst int) (exp, imp LocalResouces) {
	// Determine what resources we have and what resources we need.
	srcRes := m.getResources(src, true)
	dstRes := m.getResources(dst, true)
	imp = dstRes.Remove(srcRes)
	exp = srcRes.Remove(dstRes)
	return
}

func sumResource(res []int) int {
	var sum int
	for _, r := range res {
		sum += r
	}
	return sum
}

type ComboStorage struct {
	*ResourceStorage
	*dumbStorage
}

func newComboStorage(maxStorage int) *ComboStorage {
	return &ComboStorage{
		ResourceStorage: newResourceStorage(maxStorage),
		dumbStorage:     newDumbStorage(),
	}
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

func hasAmount(resource []int, rType, amount int) bool {
	if rType == ResourceTypeAny {
		// Sum up the available resources.
		var available int
		for _, a := range resource {
			available += a
		}
		return available >= amount
	}
	return resource[rType] >= amount
}

func (rs *ResourceStorage) Has(category, rType, amount int) bool {
	switch category {
	case ResCategoryWood:
		return hasAmount(rs.Wood[:], rType, amount)
	case ResCategoryStone:
		return hasAmount(rs.Stone[:], rType, amount)
	case ResCategoryMetal:
		return hasAmount(rs.Metal[:], rType, amount)
	case ResCategoryGem:
		return hasAmount(rs.Gem[:], rType, amount)
	case ResCategoryVarious:
		return hasAmount(rs.Various[:], rType, amount)
	}
	return false
}

func (rs *ResourceStorage) HasHowMuchMissing(category, rType, amount int) int {
	howManyMissing := func(resource []int, amount int) int {
		missing := amount
		for _, r := range resource {
			if r > 0 {
				missing -= r
			}
			if missing <= 0 {
				return 0
			}
		}
		return missing
	}
	var resource []int
	switch category {
	case ResCategoryWood:
		resource = rs.Wood[:]
	case ResCategoryStone:
		resource = rs.Stone[:]
	case ResCategoryMetal:
		resource = rs.Metal[:]
	case ResCategoryGem:
		resource = rs.Gem[:]
	case ResCategoryVarious:
		resource = rs.Various[:]
	default:
		return amount
	}

	if rType == ResourceTypeAny {
		return howManyMissing(resource, amount)
	}
	return amount - resource[rType]
}

func takeAmount(resource []int, rType, amount int) {
	if rType == ResourceTypeAny {
		// Take the resources from the first available resource.
		remaining := amount
		for i := range resource {
			if resource[i] > 0 {
				toTake := min(remaining, resource[i])
				resource[i] -= toTake
				remaining -= toTake
				if remaining == 0 {
					break
				}
			}
		}
	} else {
		resource[rType] -= amount
	}

	// TODO: add a check if we have enough resources.
}

func (rs *ResourceStorage) Take(category, rType, amount int) {
	switch category {
	case ResCategoryWood:
		takeAmount(rs.Wood[:], rType, amount)
	case ResCategoryStone:
		takeAmount(rs.Stone[:], rType, amount)
	case ResCategoryMetal:
		takeAmount(rs.Metal[:], rType, amount)
	case ResCategoryGem:
		takeAmount(rs.Gem[:], rType, amount)
	case ResCategoryVarious:
		takeAmount(rs.Various[:], rType, amount)
	}
}

func (rs *ResourceStorage) AddResource(category, rType, amount int) {
	switch category {
	case ResCategoryWood:
		rs.Wood[rType] += amount
	case ResCategoryStone:
		rs.Stone[rType] += amount
	case ResCategoryMetal:
		rs.Metal[rType] += amount
	case ResCategoryGem:
		rs.Gem[rType] += amount
	case ResCategoryVarious:
		rs.Various[rType] += amount
	}
}

func (rs *ResourceStorage) Add(r LocalResouces) {
	for _, w := range r.GetWood() {
		rs.Wood[w]++
		if rs.Wood[w] > rs.maxStorage {
			rs.Wood[w] = rs.maxStorage
		}
	}
	for _, s := range r.GetStones() {
		rs.Stone[s]++
		if rs.Stone[s] > rs.maxStorage {
			rs.Stone[s] = rs.maxStorage
		}
	}
	for _, m := range r.GetMetals() {
		rs.Metal[m]++
		if rs.Metal[m] > rs.maxStorage {
			rs.Metal[m] = rs.maxStorage
		}
	}
	for _, g := range r.GetGems() {
		rs.Gem[g]++
		if rs.Gem[g] > rs.maxStorage {
			rs.Gem[g] = rs.maxStorage
		}
	}
	for _, v := range r.GetVarious() {
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

// LocalResouces represents the resources available in a region.
type LocalResouces struct {
	Wood    byte // Available wood types.
	Stones  byte // Available stone types.
	Metals  byte // Available metal types.
	Gems    byte // Available gem types.
	Various byte // Various resources.
}

// HasAny returns true if there are any resources available.
func (r LocalResouces) HasAny() bool {
	return r.Wood != 0 || r.Stones != 0 || r.Metals != 0 || r.Gems != 0 || r.Various != 0
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

func (r LocalResouces) GetGems() []int {
	return getResources(r.Gems, geo.ResMaxGems)
}

func (r LocalResouces) GetMetals() []int {
	return getResources(r.Metals, geo.ResMaxMetals)
}

func (r LocalResouces) GetStones() []int {
	return getResources(r.Stones, geo.ResMaxStones)
}

func (r LocalResouces) GetWood() []int {
	return getResources(r.Wood, geo.ResMaxWoods)
}

func (r LocalResouces) GetVarious() []int {
	return getResources(r.Various, geo.ResMaxVarious)
}

func (r LocalResouces) String() string {
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
	s += resToString("Wood", r.Wood, geo.ResMaxWoods, geo.WoodToString)
	s += resToString("Stone", r.Stones, geo.ResMaxStones, geo.StoneToString)
	s += resToString("Metal", r.Metals, geo.ResMaxMetals, geo.MetalToString)
	s += resToString("Gem", r.Gems, geo.ResMaxGems, geo.GemToString)
	s += resToString("Various", r.Various, geo.ResMaxVarious, geo.VariousToString)
	return s
}

func (r LocalResouces) Add(other LocalResouces) LocalResouces {
	r.Wood |= other.Wood
	r.Stones |= other.Stones
	r.Metals |= other.Metals
	r.Gems |= other.Gems
	r.Various |= other.Various
	return r
}

func (r LocalResouces) Remove(other LocalResouces) LocalResouces {
	r.Wood &^= other.Wood
	r.Stones &^= other.Stones
	r.Metals &^= other.Metals
	r.Gems &^= other.Gems
	r.Various &^= other.Various
	return r
}

func (r LocalResouces) Log() {
	logEntry := func(name string, res byte, max int, strFunc func(int) string) {
		for i := 0; i < max; i++ {
			if res&(1<<uint(i)) != 0 {
				log.Printf("  %s: %s", name, strFunc(i))
			}
		}
	}
	logEntry("Wood", r.Wood, geo.ResMaxWoods, geo.WoodToString)
	logEntry("Stone", r.Stones, geo.ResMaxStones, geo.StoneToString)
	logEntry("Metal", r.Metals, geo.ResMaxMetals, geo.MetalToString)
	logEntry("Gem", r.Gems, geo.ResMaxGems, geo.GemToString)
	logEntry("Various", r.Various, geo.ResMaxVarious, geo.VariousToString)
}

func (m *Civ) getResources(r int, incNeighbors bool) LocalResouces {
	var rsc LocalResouces
	rsc.Wood = m.Geo.Resources.Wood[r]
	rsc.Stones = m.Geo.Resources.Stones[r]
	rsc.Metals = m.Geo.Resources.Metals[r]
	rsc.Gems = m.Geo.Resources.Gems[r]
	rsc.Various = m.Geo.Resources.Various[r]

	if incNeighbors {
		for _, nb := range m.R_circulate_r(rNbs, r) {
			rsc.Wood |= m.Geo.Resources.Wood[nb]
			rsc.Stones |= m.Geo.Resources.Stones[nb]
			rsc.Metals |= m.Geo.Resources.Metals[nb]
			rsc.Gems |= m.Geo.Resources.Gems[nb]
			rsc.Various |= m.Geo.Resources.Various[nb]
		}
	}
	return rsc
}

type ResourcePresence struct {
	Wood    bool
	Stones  bool
	Metals  bool
	Gems    bool
	various bool
}

func (m *Civ) getResourcePresence(r int) *ResourcePresence {
	return &ResourcePresence{
		Wood:    m.Geo.Resources.Wood[r] > 0,
		Stones:  m.Geo.Resources.Stones[r] > 0,
		Metals:  m.Geo.Resources.Metals[r] > 0,
		Gems:    m.Geo.Resources.Gems[r] > 0,
		various: m.Geo.Resources.Various[r] > 0,
	}
}
