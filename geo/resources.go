package geo

import (
	"log"
	"math"

	"github.com/Flokey82/genbiome"
)

type Resource struct {
	Name  string
	Type  ResourceType
	Value int
}

type ResourceLocations struct {
	NumRegs   int
	Location  map[*Resource][]bool // This could be byte too since it uses the same amount of memory.
	Resources []*Resource
}

// NewResourceLocation returns a new ResourceLocation.
func NewResourceLocations(numRegs int) *ResourceLocations {
	return &ResourceLocations{
		NumRegs:   numRegs,
		Location:  make(map[*Resource][]bool),
		Resources: make([]*Resource, 0),
	}
}

// AddResource adds a resource to the ResourceLocation.
func (rl *ResourceLocations) AddResource(r *Resource) {
	rl.Resources = append(rl.Resources, r)
	rl.Location[r] = make([]bool, rl.NumRegs)
}

// SetResource sets the resource value for the region.
func (rl *ResourceLocations) SetResource(r *Resource, reg int, value bool) {
	rl.Location[r][reg] = value
}

// GetResources returns all available resources for the region.
func (rl *ResourceLocations) GetResources(reg int) []*Resource {
	resources := make([]*Resource, 0, len(rl.Resources))
	for _, res := range rl.Resources {
		if rl.Location[res][reg] {
			resources = append(resources, res)
		}
	}
	return resources
}

// HasType returns true if the region has the specified resource type.
func (rl *ResourceLocations) HasType(reg int, rType ResourceType) bool {
	for _, res := range rl.Resources {
		if res.Type == rType && rl.Location[res][reg] {
			return true
		}
	}
	return false
}

// GetResourceTotal returns the total value of all resources.
func (rl *ResourceLocations) GetResourceTotal() int {
	total := 0
	for _, res := range rl.Resources {
		total += res.Value
	}
	return total
}

// SumValueOfRegion returns the sum of the resource values in the region.
func (rl *ResourceLocations) SumValueOfRegion(r int) int {
	sum := 0
	for _, res := range rl.Resources {
		if rl.Location[res][r] {
			sum += res.Value
		}
	}
	return sum
}

type ResourceType int

// DEDUPE WITH DUMB STORAGE TYPES
const (
	ResourceTypeMetal ResourceType = iota
	ResourceTypeGem
	ResourceTypeStone
	ResourceTypeWood
	ResourceTypeVarious
	ResourceTypeAnimal
	ResourceTypePlant        // Includes fungi and algae.
	ResourceTypeManufactured // Includes tools, weapons, and other goods.
	ResourceTypeMax
)

func (m *Geo) resourceFitness() []float64 {
	fitness := make([]float64, m.SphereMesh.NumRegions)
	f := m.GetFitnessSteepMountains()
	for r := range fitness {
		fitness[r] = f(r)
	}
	return fitness
}

func (m *Geo) placeResources() {
	// NOTE: This currently sucks.
	// TODO: Use fitness function instead or in addition.

	// Place metals.
	// Metals can be found mainly in mountains, so steepness
	// will be an indicator along with the distance from the
	// mountain seed points.
	m.placeMetals()

	// Place gemstones.
	// Gemstones can be found mainly in inland valleys, so
	// distance from the coastlines, mountains, and oceans
	// will be an indicator.
	m.placeGems()

	// Place forests.
	// Forests can be found mainly in valleys, so steepness
	// will be an indicator along with the distance from the
	// valley's center.
	m.placeForests()

	// Place potential quarry sites.
	// Potential quarry sites can be found mainly in mountains,
	m.placeStones()

	// Place energy sources and other resources.
	// Oil, coal, and natural gas, as well as geothermal energy
	// and magical handwavium... and clay, and salt, and stuff.
	m.placeVarious()

	// Place arable land.
	// Arable land can be found mainly in valleys, so steepness
	// will be an indicator along with the distance from the
	// valley's center.
}

var (
	ResMetalIron = &Resource{
		Name:  "Iron",
		Type:  ResourceTypeMetal,
		Value: 3,
	}
	ResMetalCopper = &Resource{
		Name:  "Copper",
		Type:  ResourceTypeMetal,
		Value: 6,
	}
	ResMetalLead = &Resource{
		Name:  "Lead",
		Type:  ResourceTypeMetal,
		Value: 8,
	}
	ResMetalTin = &Resource{
		Name:  "Tin",
		Type:  ResourceTypeMetal,
		Value: 10,
	}
	ResMetalSilver = &Resource{
		Name:  "Silver",
		Type:  ResourceTypeMetal,
		Value: 16,
	}
	ResMetalGold = &Resource{
		Name:  "Gold",
		Type:  ResourceTypeMetal,
		Value: 32,
	}
	ResMetalPlatinum = &Resource{
		Name:  "Platinum",
		Type:  ResourceTypeMetal,
		Value: 64,
	}
)

func (m *Geo) placeMetals() {
	// distMountains, _, _, _ := m.findCollisions()

	// https://www.reddit.com/r/worldbuilding/comments/kbmnd6/a_guide_to_placing_resources_on_fictional_worlds/
	const (
		chancePlatinum = 0.005
		chanceGold     = chancePlatinum + 0.020
		chanceSilver   = chanceGold + 0.040
		chanceCopper   = chanceSilver + 0.06
		chanceLead     = chanceCopper + 0.07
		chanceTin      = chanceLead + 0.1
		chanceIron     = chanceTin + 0.4
	)
	fn := m.fbmNoiseCustom(2, 1, 2, 2, 2, 0, 0, 0)
	fm := m.GetFitnessSteepMountains()

	// NOTE: By encoding the resources as bit flags, we can easily
	// determine the value of a region given the assumption that
	// each resource is twice (or half) as valuable as the previous
	// resource. This will be handy for fitness functions and such.
	//
	// I feel pretty clever about this one, but it's not realistic.
	m.ResetRand()

	// Initialize resource locations for metals.
	resLocs := m.ResourceLocations
	resLocs.AddResource(ResMetalIron)
	resLocs.AddResource(ResMetalCopper)
	resLocs.AddResource(ResMetalLead)
	resLocs.AddResource(ResMetalTin)
	resLocs.AddResource(ResMetalSilver)
	resLocs.AddResource(ResMetalGold)
	resLocs.AddResource(ResMetalPlatinum)

	// Commonly co-localized metals:
	// - gold and copper
	// - lead and silver
	// - copper and zinc
	// - tin and copper
	// - iron and nickel
	// - aluminum and bauxite

	// TODO: Iron should be more common in general.

	// Get elevation values.
	elevs := m.Elevation.GetValues()

	// TODO: Use noise intersection instead of rand.
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		if elevs[r] <= 0.0 {
			continue
		}

		// Place some extra iron.
		if fmVal := fm(r); fmVal > 0.01 {
			switch rv := math.Abs(m.Rand.NormFloat64() * fn(r)); {
			case rv < chancePlatinum:
				resLocs.Location[ResMetalPlatinum][r] = true
			case rv < chanceGold:
				resLocs.Location[ResMetalGold][r] = true
				if m.Rand.NormFloat64() < 0.5 {
					resLocs.Location[ResMetalCopper][r] = true
				}
			case rv < chanceSilver:
				resLocs.Location[ResMetalSilver][r] = true
				if m.Rand.NormFloat64() < 0.5 {
					resLocs.Location[ResMetalLead][r] = true
				}
			case rv < chanceCopper:
				resLocs.Location[ResMetalCopper][r] = true
				if m.Rand.NormFloat64() < 0.5 {
					resLocs.Location[ResMetalTin][r] = true
				}
			case rv < chanceLead:
				resLocs.Location[ResMetalLead][r] = true
			case rv < chanceTin:
				resLocs.Location[ResMetalTin][r] = true
			case rv < chanceIron:
				resLocs.Location[ResMetalIron][r] = true
			}
		}
	}

	// This attempts some weird variation of:
	// https://www.redblobgames.com/x/1736-resource-placement/
	/*
		nA := m.fbm_noise2(5, 0.5, 5, 5, 5, 0, 0, 0)
		nB := m.fbm_noise2(7, 0.5, 5, 5, 5, 0, 0, 0)
		resources := make([]byte, len(steepness))
		for r := range steepness {
			noiseVal := (nA(r) + nB(r) + m.r_elevation[r]) / 3
			if m.getIntersection(noiseVal, 0.75, 0.01) {
				resources[r] |= ResMetPlatinum
			}
			//chance /= float64(distMountains[r])
		}

		nC := m.fbm_noise2(2, 0.5, 5, 5, 5, 0, 0, 0)
		nD := m.fbm_noise2(7, 0.5, 5, 5, 5, 0, 0, 0)
		for r := range steepness {
			noiseVal := (nC(r) + nD(r) + m.r_elevation[r]) / 3
			if m.getIntersection(noiseVal, 0.75, 0.02) {
				resources[r] |= ResMetGold
			}
			//chance /= float64(distMountains[r])
		}

		nC = m.fbm_noise2(2, 0.5, 1, 1, 1, 0, 0, 0)
		nD = m.fbm_noise2(5, 0.1, 1, 1, 1, 0, 0, 0)
		for r := range steepness {
			noiseVal := (-1*(nC(r)+nD(r)) + m.r_elevation[r]) / 3
			if m.getIntersection(noiseVal, 0.52, 0.07) {
				resources[r] |= ResMetIron
			}
			//chance /= float64(distMountains[r])
		}
	*/

	//m.r_metals = resources
}

var (
	ResGemsAmethyst = &Resource{
		Name:  "Amethyst",
		Type:  ResourceTypeGem,
		Value: 1,
	}
	ResGemsTopaz = &Resource{
		Name:  "Topaz",
		Type:  ResourceTypeGem,
		Value: 2,
	}
	ResGemsSapphire = &Resource{
		Name:  "Sapphire",
		Type:  ResourceTypeGem,
		Value: 4,
	}
	ResGemsEmerald = &Resource{
		Name:  "Emerald",
		Type:  ResourceTypeGem,
		Value: 8,
	}
	ResGemsRuby = &Resource{
		Name:  "Ruby",
		Type:  ResourceTypeGem,
		Value: 16,
	}
	ResGemsDiamond = &Resource{
		Name:  "Diamond",
		Type:  ResourceTypeGem,
		Value: 32,
	}
)

func (m *Geo) placeGems() {
	elevs := m.Elevation.GetValues()
	steepness := m.GetSteepness()
	const (
		chanceDiamond  = 0.005
		chanceRuby     = chanceDiamond + 0.025
		chanceEmerald  = chanceRuby + 0.04
		chanceSapphire = chanceEmerald + 0.05
		chanceTopaz    = chanceSapphire + 0.06
		chanceAmethyst = chanceTopaz + 0.1
		// chanceQuartz   = 0.75 // Usually goes hand in hand with gold?
		// chanceFlint    = 0.9
	)

	// Initialize resource locations for gems.
	resLocs := m.ResourceLocations
	resLocs.AddResource(ResGemsAmethyst)
	resLocs.AddResource(ResGemsTopaz)
	resLocs.AddResource(ResGemsSapphire)
	resLocs.AddResource(ResGemsEmerald)
	resLocs.AddResource(ResGemsRuby)
	resLocs.AddResource(ResGemsDiamond)

	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		if steepness[r] > 0.9 && elevs[r] > 0.5 {
			switch rv := math.Abs(m.Rand.NormFloat64()); {
			case rv < chanceDiamond:
				resLocs.Location[ResGemsDiamond][r] = true
			case rv < chanceRuby:
				resLocs.Location[ResGemsRuby][r] = true
			case rv < chanceEmerald:
				resLocs.Location[ResGemsEmerald][r] = true
			case rv < chanceSapphire:
				resLocs.Location[ResGemsSapphire][r] = true
			case rv < chanceTopaz:
				resLocs.Location[ResGemsTopaz][r] = true
			case rv < chanceAmethyst:
				resLocs.Location[ResGemsAmethyst][r] = true
				// case rv < chanceQuartz:
				//	gems[r] |= ResGemQuartz
				// case rv < chanceFlint:
				//	gems[r] |= ResGemFlint
			}
		}
	}
}

var (
	ResStoneSandstone = &Resource{
		Name:  "Sandstone",
		Type:  ResourceTypeStone,
		Value: 1,
	}
	ResStoneLimestone = &Resource{
		Name:  "Limestone",
		Type:  ResourceTypeStone,
		Value: 2,
	}
	ResStoneChalk = &Resource{
		Name:  "Chalk",
		Type:  ResourceTypeStone,
		Value: 4,
	}
	ResStoneSlate = &Resource{
		Name:  "Slate",
		Type:  ResourceTypeStone,
		Value: 8,
	}
	ResStoneMarble = &Resource{
		Name:  "Marble",
		Type:  ResourceTypeStone,
		Value: 16,
	}
	ResStoneGranite = &Resource{
		Name:  "Granite",
		Type:  ResourceTypeStone,
		Value: 32,
	}
	ResStoneBasalt = &Resource{
		Name:  "Basalt",
		Type:  ResourceTypeStone,
		Value: 64,
	}
	ResStoneObsidian = &Resource{
		Name:  "Obsidian",
		Type:  ResourceTypeStone,
		Value: 128,
	}
)

func (m *Geo) placeStones() {
	log.Println("placing stones is not implemented")

	// Chalk:
	// Ancient Chalk beds formed on the floor of ancient seas.
	//
	// Limestone:
	// The Chalk later solidifies into Limestone. Can be placed where hill
	// meet grasslands in non wet areas.
	//
	// Flint:
	// Flint (also called Chert) forms as lumps between layers and in cavities
	// left in the sea floor in these Chalk beds.
	//
	// Marble:
	// Marble is formed from Limestone that has been subjected to intense heat
	// and pressure. Marble will be placed near mountain ranges.
	//
	// Obsidian:
	// Obsidian is formed when water flows over volcanic lava to cool it rapidly.
	// Placed near volcanic plate boundaries that no longer have large amounts of
	// water. Water breaks down obsidian over time.
	//
	// Granite:
	// Granite is formed when molten rock is slowly cooled. It forms the bottom
	// layer of all land continents. Placed along two land type convergent boundaries
	// on the uplifted side where it is raised to the surface, making quarrying easy.
	//
	// Sandstone:
	// Sandstone is formed when sand is deposited in large quantities and under goes
	// large amounts of pressure, heat, and drainage causing the sand and other
	// minerals to "cement" together. Placed near ancient drainage basins that deposited
	// sand from deserts or beaches, or alternatively where hills or mountains meet a
	// dry desert.
	//
	// Basalt:
	// Basalt is formed when lava cools quickly. Placed near volcanic plate boundaries
	// that have large amounts of water. Water breaks down basalt over time.
	//
	// Slate:
	// Slate is formed when shale is subjected to intense heat and pressure. Slate
	// will be placed near mountain ranges.

	biomeFunc := m.GetRegWhittakerModBiomeFunc()
	elevs := m.Elevation.GetValues()
	steepness := m.GetSteepness()

	// Generate a distance field for volcanoes, mountains, and faultlines.
	var volcanoes, mountains, faultlines []int
	stopSea := make(map[int]bool)
	isBeach := make(map[int]bool)

	out_r := make([]int, 0, 8)
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		if m.RegionIsVolcano[r] {
			volcanoes = append(volcanoes, r)
		}
		if m.RegionIsMountain[r] {
			mountains = append(mountains, r)
		}
		if math.Abs(m.RegionCompression[r]) > 0.1 {
			faultlines = append(faultlines, r)
		}
		if elevs[r] <= 0.0 {
			stopSea[r] = true
		} else {
			// Check if the region is a beach.
			for _, n := range m.R_circulate_r(out_r, r) {
				if elevs[n] <= 0.0 {
					isBeach[r] = true
					break
				}
			}
		}
	}

	distVolcanoes := m.AssignDistanceField(volcanoes, stopSea)
	distMountains := m.AssignDistanceField(mountains, stopSea)
	distFaultlines := m.AssignDistanceField(faultlines, stopSea)

	// Get current rainfall values.
	rainfall := m.Rainfall.GetValues()

	// Initialize resource locations for stones.
	resLocs := m.ResourceLocations
	resLocs.AddResource(ResStoneSandstone)
	resLocs.AddResource(ResStoneLimestone)
	resLocs.AddResource(ResStoneChalk)
	resLocs.AddResource(ResStoneSlate)
	resLocs.AddResource(ResStoneMarble)
	resLocs.AddResource(ResStoneGranite)
	resLocs.AddResource(ResStoneBasalt)
	resLocs.AddResource(ResStoneObsidian)

	// Loop through all the regions and place stones based on the region's
	// properties.
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		// Skip water regions.
		if elevs[r] <= 0.0 {
			continue
		}

		// Get the region's biome.
		biome := biomeFunc(r)

		// Check if we have sandstone (beach, or desert).
		if biome == genbiome.WhittakerModBiomeSubtropicalDesert || isBeach[r] {
			resLocs.Location[ResStoneSandstone][r] = true
		}

		// Chalk and limestone.
		if biome == genbiome.WhittakerModBiomeTemperateGrassland && steepness[r] > 0.1 {
			// If we are close to mountains, we have marble.
			if distMountains[r] < 2 {
				resLocs.Location[ResStoneMarble][r] = true
			} else if !m.IsRegRiver(r) && !m.IsRegLakeOrWaterBody(r) {
				// Check if we have limestone (dryer, hilly grassland)
				resLocs.Location[ResStoneLimestone][r] = true
			} else if rainfall[r] > 0.5 {
				// Check if we have chalk (wetter, hilly grassland)
				resLocs.Location[ResStoneChalk][r] = true
			}
		}

		// Obsidian, and basalt.
		// For these stones, we need to check if we are near a volcano or faultline.
		if distVolcanoes[r] < 2 || distFaultlines[r] < 2 {
			// Check if we have obsidian (near a volcano).
			if distVolcanoes[r] < 2 {
				resLocs.Location[ResStoneObsidian][r] = true
			}

			// Check if we have basalt (near a faultline).
			if distFaultlines[r] < 2 {
				resLocs.Location[ResStoneBasalt][r] = true
			}
		}

		// Check if we have granite (near a mountain and faultline).
		if distMountains[r] < 3 && distFaultlines[r] < 2 {
			resLocs.Location[ResStoneGranite][r] = true
		} else if steepness[r] > 0.2 && distMountains[r] > 2 && distMountains[r] < 5 {
			// Slate.
			// For slate, we need to check if we are near a mountain range or if the region
			// is steep.
			resLocs.Location[ResStoneSlate][r] = true
		}
	}
}

var (
	ResVariousClay = &Resource{
		Name:  "Clay",
		Type:  ResourceTypeVarious,
		Value: 1,
	}
	ResVariousSulfur = &Resource{
		Name:  "Sulfur",
		Type:  ResourceTypeVarious,
		Value: 2,
	}
	ResVariousSalt = &Resource{
		Name:  "Salt",
		Type:  ResourceTypeVarious,
		Value: 4,
	}
	ResVariousCoal = &Resource{
		Name:  "Coal",
		Type:  ResourceTypeVarious,
		Value: 8,
	}
	ResVariousOil = &Resource{
		Name:  "Oil",
		Type:  ResourceTypeVarious,
		Value: 10,
	}
	ResVariousGas = &Resource{
		Name:  "Gas",
		Type:  ResourceTypeVarious,
		Value: 10,
	}
)

func (m *Geo) placeVarious() {
	biomeFunc := m.GetRegWhittakerModBiomeFunc()
	elevs := m.Elevation.GetValues()
	steepness := m.GetSteepness()

	// Initialize resource locations for various resources.
	resLocs := m.ResourceLocations
	resLocs.AddResource(ResVariousClay)
	resLocs.AddResource(ResVariousSulfur)
	resLocs.AddResource(ResVariousSalt)
	resLocs.AddResource(ResVariousCoal)
	resLocs.AddResource(ResVariousOil)
	resLocs.AddResource(ResVariousGas)

	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		if elevs[r] <= 0.0 {
			continue
		}
		biome := biomeFunc(r)

		if m.RegionIsVolcano[r] {
			resLocs.Location[ResVariousSulfur][r] = true
		}

		if m.RegionIsMountain[r] {
			resLocs.Location[ResVariousCoal][r] = true
		}

		if m.IsRegRiver(r) && steepness[r] > 0.1 && steepness[r] < 0.3 {
			resLocs.Location[ResVariousClay][r] = true
		}

		if biome == genbiome.WhittakerModBiomeHotSwamp {
			resLocs.Location[ResVariousGas][r] = true
		}

		// TODO: Salt, oil, coal.
	}
}

var (
	ResWoodsOak = &Resource{
		Name:  "Oak",
		Type:  ResourceTypeWood,
		Value: 1,
	}
	ResWoodsBirch = &Resource{
		Name:  "Birch",
		Type:  ResourceTypeWood,
		Value: 2,
	}
	ResWoodsPine = &Resource{
		Name:  "Pine",
		Type:  ResourceTypeWood,
		Value: 1,
	}
	ResWoodsSpruce = &Resource{
		Name:  "Spruce",
		Type:  ResourceTypeWood,
		Value: 2,
	}
	ResWoodsCedar = &Resource{
		Name:  "Cedar",
		Type:  ResourceTypeWood,
		Value: 2,
	}
	ResWoodsShrub = &Resource{
		Name:  "Shrub",
		Type:  ResourceTypeWood,
		Value: 1,
	}
	ResWoodsFir = &Resource{
		Name:  "Fir",
		Type:  ResourceTypeWood,
		Value: 2,
	}
	ResWoodsPalm = &Resource{
		Name:  "Palm",
		Type:  ResourceTypeWood,
		Value: 3,
	}
)

func (m *Geo) placeForests() {
	// Get all biomes that are forested.
	// Place trees in those biomes based on the biome's tree type(s).
	// Of course it can't be too steep.
	biomeFunc := m.GetRegWhittakerModBiomeFunc()
	elevs := m.Elevation.GetValues()
	//steepness := m.GetSteepness()

	// Initialize resource locations for wood.
	resLocs := m.ResourceLocations
	resLocs.AddResource(ResWoodsOak)
	resLocs.AddResource(ResWoodsBirch)
	resLocs.AddResource(ResWoodsPine)
	resLocs.AddResource(ResWoodsSpruce)
	resLocs.AddResource(ResWoodsCedar)
	resLocs.AddResource(ResWoodsShrub)
	resLocs.AddResource(ResWoodsFir)
	resLocs.AddResource(ResWoodsPalm)

	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		if elevs[r] <= 0.0 {
			continue
		}

		// NOTE: This is absolute garbage. It's just a quick hack to get some forests
		// in the world.
		biome := biomeFunc(r)
		if biome == genbiome.WhittakerModBiomeTemperateRainforest {
			resLocs.Location[ResWoodsOak][r] = true
		} else if biome == genbiome.WhittakerModBiomeTemperateSeasonalForest {
			resLocs.Location[ResWoodsOak][r] = true
			resLocs.Location[ResWoodsBirch][r] = true
		} else if biome == genbiome.WhittakerModBiomeTropicalRainforest {
			resLocs.Location[ResWoodsOak][r] = true
			resLocs.Location[ResWoodsPalm][r] = true
		} else if biome == genbiome.WhittakerModBiomeTropicalSeasonalForest {
			resLocs.Location[ResWoodsOak][r] = true
			resLocs.Location[ResWoodsPalm][r] = true
			resLocs.Location[ResWoodsBirch][r] = true
		} else if biome == genbiome.WhittakerModBiomeBorealForestTaiga {
			resLocs.Location[ResWoodsPine][r] = true
			resLocs.Location[ResWoodsSpruce][r] = true
			resLocs.Location[ResWoodsCedar][r] = true
		} else if biome == genbiome.WhittakerModBiomeTundra {
			resLocs.Location[ResWoodsSpruce][r] = true
			resLocs.Location[ResWoodsCedar][r] = true
			resLocs.Location[ResWoodsFir][r] = true
			resLocs.Location[ResWoodsShrub][r] = true
		} else if biome == genbiome.WhittakerModBiomeWetlands {
			resLocs.Location[ResWoodsShrub][r] = true
			resLocs.Location[ResWoodsFir][r] = true
			resLocs.Location[ResWoodsCedar][r] = true
			resLocs.Location[ResWoodsOak][r] = true
			resLocs.Location[ResWoodsBirch][r] = true
		} else if biome == genbiome.WhittakerModBiomeWoodlandShrubland {
			resLocs.Location[ResWoodsShrub][r] = true
			resLocs.Location[ResWoodsFir][r] = true
			resLocs.Location[ResWoodsCedar][r] = true
			resLocs.Location[ResWoodsOak][r] = true
			resLocs.Location[ResWoodsBirch][r] = true
		}
	}
}
