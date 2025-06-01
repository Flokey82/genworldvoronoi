package civ

import (
	"github.com/Flokey82/genworldvoronoi/geo"
)

// StatEffect represents the effect of a city feature on a city stat.
type StatEffect struct {
	Stat  CityStat
	Value float64
	Mode  StatEffectMode
}

// StatEffectMode represents the mode of a stat effect.
type StatEffectMode int

const (
	StatEffectModeAdd StatEffectMode = iota
	StatEffectModeMul
)

// TODO:
// - One-time cost for building
// - Maintenance cost
// - Add a cost function to the CityFeature struct which will consume resources from the city.
// - Add upgrade levels to the city features.
//   - Each level will introduce new effects and require more resources.
//   - If not enough resources are available, the feature will either operate
//     at a lower level or not at all.
//   - Find a way to quantify the value of a city feature so we
//     can compare the value of different features and have the AI
//     make decisions based on that.
type CityFeature struct {
	*Construction
	Requires   func(c *City, m *Civ) bool
	Effect     func(c *City, m *Civ)
	StatEffect []StatEffect

	// TODO:
	// - Consumes (upkeep)
	// - Produces (output w. optional consumption)
	// - Effects (e.g. increase in defense, decrease in crime, etc)
	// TODO: Add a cost function?
}

// Tick runs the construction for one turn.
func (c *CityFeature) Tick(city *City, m *Civ) bool {
	if !c.Construction.Tick(city.ComboStorage) {
		return false
	}
	c.Effect(city, m)
	return true
}

var (
	// WATER (irrigation, drinking water, etc)

	// CityFeatureAquaeduct is a city feature that provides water to the city.
	CityFeatureAquaeduct = &CityFeature{
		Construction: &Construction{
			Name: "Aquaeduct",
			Cost: &ConstructionCost{
				ResourceTypes: []ResTypeAmount{
					{Type: geo.ResourceTypeStone, Amount: 100},
					{Type: geo.ResourceTypeWood, Amount: 20},
				},
			},
		},
		Requires: func(c *City, m *Civ) bool {
			// The city should have access to running water (near a river, lake, etc).
			if m.Geo.IsRegRiver(c.ID) {
				return true
			}
			// If any neighbor region has a river or is oceanic or a lake, then it's fine.
			for _, nb := range m.R_circulate_r(nil, c.ID) {
				if m.Geo.IsRegRiver(nb) || m.Geo.IsRegLakeOrWaterBody(nb) {
					return true
				}
			}
			return false
		},
		Effect: func(c *City, m *Civ) {
			// Increase soil fertility etc.
		},
	}

	CityFeatureQuarry = &CityFeature{
		Construction: &Construction{
			Name: "Quarry",
		},
		Requires: func(c *City, m *Civ) bool {
			return m.hasResourceAny(c.ID, geo.ResourceTypeStone)
		},
		Effect: func(c *City, m *Civ) {
			// TODO: Get the specific type.
			c.ComboStorage.AddResource(geo.ResStoneMarble, 1)
		},
	}

	CityFeatureMine = &CityFeature{
		Construction: &Construction{
			Name: "Mine",
		},
		Requires: func(c *City, m *Civ) bool {
			return m.hasResourceAny(c.ID, geo.ResourceTypeMetal)
		},
		Effect: func(c *City, m *Civ) {
			// TODO: Get the specific type.
			c.ComboStorage.AddResource(geo.ResMetalIron, 1)
		},
	}

	CityFeatureLumberyard = &CityFeature{
		Construction: &Construction{
			Name: "Lumberyard",
		},
		Requires: func(c *City, m *Civ) bool {
			return m.hasResourceAny(c.ID, geo.ResourceTypeWood)
		},
		Effect: func(c *City, m *Civ) {
			// TODO: Get the specific type.
			c.ComboStorage.AddResource(geo.ResWoodsOak, 1)
		},
	}

	CityFeatureGemMine = &CityFeature{
		Construction: &Construction{
			Name: "Gem Mine",
		},
		Requires: func(c *City, m *Civ) bool {
			return m.hasResourceAny(c.ID, geo.ResourceTypeGem)
		},
		Effect: func(c *City, m *Civ) {
			// TODO: Get the specific type.
			c.ComboStorage.AddResource(geo.ResGemsAmethyst, 1)
		},
	}

	// CityFeatureWells
	// Basic water source for the city. Reduces risk of drought.
	// Can be improved with wind powered pumps or aqueducts.

	// CityFeatureWindPumps
	// Wind powered pumps for water. Increases water supply.
	// Requires access to wind.

	// FOOD (farming, fishing, etc)

	// CityFeatureGranary
	// Provides a place for storing food. (reduces impact of food shortages)

	// CityFeatureCommunalMill
	// Wind or water powered mill for grinding grain.
	// Will increase food production (reduces per capita food cost).
	// Requires access to wind or water.

	// CityFeatureSaltPans
	// Provides a place for salt production. (increases food preservation)
	// Requires access to salt water.

	// SECURITY (walls, guards, etc)

	// CityFeatureWalls
	// Provides protection and defense. (reduces success rate of raids)
	// TODO: Introduce upgrade levels.
	CityFeatureWalls = &CityFeature{
		Construction: &Construction{
			Name: "Walls",
			Cost: &ConstructionCost{
				ResourceTypes: []ResTypeAmount{
					{Type: geo.ResourceTypeStone, Amount: 100},
					{Type: geo.ResourceTypeWood, Amount: 20},
				},
			},
		},
		Requires: func(c *City, m *Civ) bool {
			// Needs at least a size of 1000.
			if c.Population < 1000 {
				return false
			}

			// Needs either stone or wood.
			return m.hasResourceAny(c.ID, geo.ResourceTypeStone) || m.hasResourceAny(c.ID, geo.ResourceTypeWood)
		},
		Effect: func(c *City, m *Civ) {
			// Increase defense.
		},
	}

	// CityFeatureStronghold
	// Provides protection and defense. (reduces success rate of raids)

	// CityFeaturePrison
	// Provides a place for holding criminals and enemies. (reduces crime rate)

	// MONUMENTS (statues, temples, etc)

	// CityFeatureStatue, CityFeatureObelisk, CityFeatureMonument
	// Provides a place for public art and monuments. MIGHT increase satisfaction and happiness.
	// Depending on whom the thing is dedicated to, it might increase or decrease satisfaction.

	// CityFeatureMausoleum, CityFeatureDungeon, CityFeatureCrypt
	// Provides a place for the dead. MIGHT increase satisfaction and happiness.
	// Can become a destination for pilgrims or tourists and once forgotten, a place for adventurers.

	// CityFeatureTemple
	// Provides a place for worship and religious activities. Will increase satisfaction and happiness.

	// KNOWLEDGE (libraries, schools, etc)

	// CityFeatureLibrary
	// Provides a place for knowledge and learning. Will increase research and education.

	// CityFeatureSchool
	// Provides a place for education and training. Will increase research and education.
	// Will increase wealth generation.

	// CityFeatureUniversity
	// Provides a place for higher education and research. Will increase research and education.
	// Will increase wealth generation and unlock new technologies.

	// CityFeatureObservatory
	// Provides a place for studying the stars and planets. Will increase research and education.
	// Will increase wealth generation and unlock new technologies.

	// CityFeatureLaboratory
	// Provides a place for scientific research and experimentation. Will increase research and education.

	// TRADE (markets, ports, etc)

	// CityFeatureMarket
	// Provides a place for trade and commerce.
	// Will increase trade gains and wealth generation.

	// CityFeaturePavedRoads
	// Provides faster travel and transport. (increases trade gains)

	// CityFeaturePier, CityFeaturePort, CityFeatureWharf
	// Provides fishing, trade, transport, and ship building.
	// Requires access to a body of water or a river.

	// INFRASTRUCTURE (sewers, roads, etc)

	// CityFeatureSewers
	// Provides sanitation and waste management. (lowers disease risk)
	// Requires some water source for operation.

	// CityFeatureHospital
	// Provides medical care and treatment. (lowers disease risk)
)

var CityFeatures = []*CityFeature{
	CityFeatureAquaeduct,
	CityFeatureQuarry,
	CityFeatureMine,
	CityFeatureLumberyard,
	CityFeatureGemMine,
	CityFeatureWalls,
}

type Features struct {
	feats []*CityFeature
}

func NewFeatures() *Features {
	return &Features{}
}

func (f *Features) Tick(c *City, m *Civ) {
	for _, ft := range f.feats {
		ft.Tick(c, m)
	}
}

func (f *Features) Add(ft *CityFeature) {
	if f.Has(ft) {
		return
	}
	f.feats = append(f.feats, ft)
}

func (f *Features) Has(ft *CityFeature) bool {
	for _, f := range f.feats {
		if f == ft {
			return true
		}
	}
	return false
}

func (f *Features) CanBuild(c *City, m *Civ) []*CityFeature {
	var res []*CityFeature
	for _, ft := range f.feats {
		if !f.Has(ft) && ft.Requires(c, m) && c.ComboStorage.Fulfill(ft.Cost) {
			res = append(res, ft)
		}
	}
	return res
}

func (f *Features) String() string {
	var res string
	for _, ft := range f.feats {
		res += ft.Name + ", "
	}
	return res
}
