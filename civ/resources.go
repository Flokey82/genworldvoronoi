package civ

import (
	"fmt"
	"log"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
)

// Various additional resource types.
var (
	ResourceTypeManufactured geo.ResourceType = geo.ResourceTypeMax
)

// Various manufactured resources.
var (
	ResLeather = &geo.Resource{
		Type: ResourceTypeManufactured,
		Name: "Leather",
	}
	ResFood = &geo.Resource{
		Type: ResourceTypeManufactured,
		Name: "Food",
	}
	ResTools = &geo.Resource{
		Type: ResourceTypeManufactured,
		Name: "Tools",
	}
	// Housing shouldn't be a resource.
	ResHousing = &geo.Resource{
		Type: ResourceTypeManufactured,
		Name: "Housing",
	}
)

// TODO: Move this to a separate file.
func (m *Civ) handleResources(r int, storage *ComboStorage, incNeighbors bool, pop int, origin fmt.Stringer) {

	// Here is what we could do:
	// - Feed our people.
	// - Produce weapons for hunting.
	// - Produce tools for crafting.

	// TODO:
	// - Calculate suitability for hunting, gathering, herding, farming, etc.

	biome := m.Geo.GetRegWhittakerModBiome(r)
	precipitation := m.Geo.Rainfall.GetValues()
	steepness := m.Geo.Steepness.GetValues()

	getGatheringScore := func(r int) float64 {
		regSteepness := steepness[r]
		regPrecipitation := precipitation[r] * geo.MaxPrecipitation * 100
		regTemperature := m.GetRegTemperature(r)

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
		regSteepness := steepness[r]
		regPrecipitation := precipitation[r] * geo.MaxPrecipitation * 100
		regTemperature := m.GetRegTemperature(r)
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

	regHunting := getHuntingScore(r)
	possibleFoodHuntingPerPersonMax := 5.0
	possibleFoodHuntingPerPersonMin := 1.0

	regGathering := getGatheringScore(r)
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
	availablePopulationGatherer := float64(pop) * 0.9

	// The ratio of the scores will determine how many are spent on hunting and how many on gathering.
	// The rest will be spent on farming.
	// huntingToGatheringRatio := regHunting / regGathering

	// TODO: Also weigh the composition by the preference of the tribe.
	popHunting := float64(pop) / (foodHuntingPerPerson + availablePopulationGatherer - foodGatheringPerPerson)
	popGathering := availablePopulationGatherer - popHunting
	foodHunting := popHunting * foodHuntingPerPerson
	foodGathering := popGathering * foodGatheringPerPerson

	// Calculate the food production for gathering.

	log.Printf("!!!%s is producing food: %d total; %d from hunting (%.2f, %d people), %d from gathering  %.2f, %d people).", origin.String(), int(foodHunting+foodGathering), int(foodHunting), regHunting, int(popHunting), int(foodGathering), regGathering, int(popGathering))

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
	res := m.getResources(r, incNeighbors)

	// Add the resources to the storage.
	storage.ResourceStorage.Add(res)
	storage.AddResource(ResFood, int(foodHunting+foodGathering))
	storage.AddResource(ResLeather, int(foodHunting))
	storage.AddResource(geo.ResWoodsShrub, int(foodGathering))

}

func (m *Civ) spendResources(r int, storage *ComboStorage, pop int, origin fmt.Stringer) {
	// Calculate how much food and firewood we need.
	const requiredFoodPerPerson = 1.0
	const requiredFirewoodPerPerson = 0.5 // This should depend on the climate

	foodNeeded := pop * requiredFoodPerPerson
	storage.RemoveResource(ResFood, foodNeeded)
	if storage.Resources[ResFood] < 0 {
		log.Printf("%s needs more food. %d/%d", origin, storage.Resources[ResFood], foodNeeded)
		storage.Resources[ResFood] = 0 // TODO: Kill off some people
	}

	firewoodNeeded := int(float64(pop) * requiredFirewoodPerPerson)
	storage.TakeNOfType(geo.ResourceTypeWood, firewoodNeeded)
	if !storage.HasAnyOfType(geo.ResourceTypeWood) {
		log.Printf("%s needs more firewood. %d/%d", origin, 0, firewoodNeeded)
		// TODO: Kill off some people
	}
}

// NOTE: This is only about resources.
func (m *Civ) compareResources(src, dst int) (exp, imp LocalResouces) {
	// Determine what resources we have and what resources we need.
	srcRes := m.getResources(src, true)
	dstRes := m.getResources(dst, true)
	imp = dstRes.Remove(srcRes)
	exp = srcRes.Remove(dstRes)
	return
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
		Wood:    m.Geo.HasType(r, geo.ResourceTypeWood),
		Stones:  m.Geo.HasType(r, geo.ResourceTypeStone),
		Metals:  m.Geo.HasType(r, geo.ResourceTypeMetal),
		Gems:    m.Geo.HasType(r, geo.ResourceTypeGem),
		various: m.Geo.HasType(r, geo.ResourceTypeVarious),
	}
}

// LocalResouces represents the resources available in a region.
type LocalResouces struct {
	Resources []*geo.Resource
}

// HasAny returns true if there are any resources available.
func (r LocalResouces) HasAny() bool {
	return len(r.Resources) > 0
}

func (r LocalResouces) String() string {
	var s string
	for _, res := range r.Resources {
		s += res.Name + ", "
	}
	return s
}

func (r LocalResouces) Add(other LocalResouces) LocalResouces {
	seenResources := make(map[*geo.Resource]bool)
	for _, res := range r.Resources {
		seenResources[res] = true
	}
	for _, res := range other.Resources {
		if !seenResources[res] {
			r.Resources = append(r.Resources, res)
		}
	}
	return r
}

func (r LocalResouces) Remove(other LocalResouces) LocalResouces {
	seenResources := make(map[*geo.Resource]bool)
	for _, res := range other.Resources {
		seenResources[res] = true
	}
	var newResources []*geo.Resource
	for _, res := range r.Resources {
		if !seenResources[res] {
			newResources = append(newResources, res)
		}
	}
	r.Resources = newResources
	return r
}

func (r LocalResouces) Log() {
	for _, res := range r.Resources {
		log.Printf("Resource: %s", res.Name)
	}
}

func (m *Civ) getResources(r int, incNeighbors bool) LocalResouces {
	var rsc LocalResouces

	for _, res := range m.Geo.Resources {
		if m.Geo.Location[res][r] {
			rsc.Resources = append(rsc.Resources, res)
		}
	}

	if incNeighbors {
		seenResources := make(map[*geo.Resource]bool)
		for _, res := range rsc.Resources {
			seenResources[res] = true
		}
		for _, nb := range m.R_circulate_r(rNbs, r) {
			for _, res := range m.Geo.Resources {
				if m.Geo.Location[res][nb] && !seenResources[res] {
					rsc.Resources = append(rsc.Resources, res)
					seenResources[res] = true
				}
			}
		}
	}
	return rsc
}

func (m *Civ) hasResourceAny(r int, category geo.ResourceType) bool {
	for _, res := range m.Resources {
		if res.Type == category && m.Location[res][r] {
			return true
		}
	}
	return false
}
