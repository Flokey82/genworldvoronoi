package civ2

import (
	"fmt"
	"github.com/Flokey82/genworldvoronoi/geo"
)

// ConstructionBlueprint defines the static properties of a building or feature.
type ConstructionBlueprint struct {
	Name        string
	Description string
	Cost        *ConstructionCost
	Upkeep      *ConstructionCost
	Produces    []ResAmount
	Requires    func(e Constructible, m *Civ) bool
	Effects     func(e Constructible, m *Civ)
}

// ConstructionCost represents the specific or categorical resource requirements.
type ConstructionCost struct {
	ResourceTypes []ResTypeAmount
	Resources     []ResAmount
}

type ResTypeAmount struct {
	Type   geo.ResourceType
	Amount int
}

type ResAmount struct {
	Res    *geo.Resource
	Amount int
}

// Project represents an active construction effort.
type Project struct {
	Blueprint *ConstructionBlueprint
	Progress  float64 // 0.0 to 1.0
}

// Infrastructure tracks completed buildings and modifiers.
type Infrastructure struct {
	Buildings []*ConstructionBlueprint
}

func NewInfrastructure() *Infrastructure {
	return &Infrastructure{
		Buildings: make([]*ConstructionBlueprint, 0),
	}
}

// ConstructionQueue manages active projects.
type ConstructionQueue struct {
	Current *Project
	Pending []*Project
}

func NewConstructionQueue() *ConstructionQueue {
	return &ConstructionQueue{
		Pending: make([]*Project, 0),
	}
}

// Blueprints
var (
	BlueprintHut = &ConstructionBlueprint{
		Name: "Hut",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 10},
			},
		},
		Produces: []ResAmount{
			{Res: ResHousing, Amount: 10},
		},
	}

	BlueprintGranary = &ConstructionBlueprint{
		Name: "Granary",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 50},
				{Type: geo.ResourceTypeStone, Amount: 20},
			},
		},
		Description: "Reduces food spoilage and increases max population limit.",
	}

	BlueprintWalls = &ConstructionBlueprint{
		Name: "Walls",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeStone, Amount: 100},
				{Type: geo.ResourceTypeWood, Amount: 20},
			},
		},
		Requires: func(e Constructible, m *Civ) bool {
			// Needs at least a size of 1000.
			if e.GetPopulation() < 1000 {
				return false
			}
			// Needs either stone or wood in the region.
			return m.hasResourceAny(e.GetID(), geo.ResourceTypeStone) || m.hasResourceAny(e.GetID(), geo.ResourceTypeWood)
		},
		Effects: func(e Constructible, m *Civ) {
			// Increase defense significantly.
			// TODO: Implement a proper defense rating in BaseEntity.
		},
		Description: "Increases defense significantly.",
	}

	BlueprintAquaeduct = &ConstructionBlueprint{
		Name: "Aquaeduct",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeStone, Amount: 100},
				{Type: geo.ResourceTypeWood, Amount: 20},
			},
		},
		Requires: func(e Constructible, m *Civ) bool {
			// The city should have access to running water (near a river, lake, etc).
			r := e.GetID()
			if m.Elevation.GetValues()[r] <= 0 || m.Waterpool[r] > 0 {
				return false
			}
			if m.Flux.GetValues()[r] > 0.01 { // Simple river check
				return true
			}
			// If any neighbor region has a river or is oceanic or a lake, then it's fine.
			for _, nb := range m.SphereMesh.R_circulate_r(nil, r) {
				if m.Flux.GetValues()[nb] > 0.01 || m.Elevation.GetValues()[nb] <= 0 || m.Waterpool[nb] > 0 {
					return true
				}
			}
			return false
		},
		Effects: func(e Constructible, m *Civ) {
			// Provides water to the city. Increases growth and fertility.
			// TODO: Implement a growth modifier in BaseEntity.
		},
		Description: "Provides water to the city. Increases growth and fertility.",
	}

	BlueprintQuarry = &ConstructionBlueprint{
		Name: "Quarry",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 20},
			},
		},
		Requires: func(e Constructible, m *Civ) bool {
			return m.hasResourceAny(e.GetID(), geo.ResourceTypeStone)
		},
		Produces: []ResAmount{
			{Res: geo.ResStoneMarble, Amount: 1}, // Default to Marble for now.
		},
		Description: "Extracts stone from the region.",
	}

	BlueprintMine = &ConstructionBlueprint{
		Name: "Mine",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 30},
				{Type: geo.ResourceTypeStone, Amount: 10},
			},
		},
		Requires: func(e Constructible, m *Civ) bool {
			return m.hasResourceAny(e.GetID(), geo.ResourceTypeMetal)
		},
		Produces: []ResAmount{
			{Res: geo.ResMetalIron, Amount: 1}, // Default to Iron for now.
		},
		Description: "Extracts metal ores from the region.",
	}

	BlueprintLumberyard = &ConstructionBlueprint{
		Name: "Lumberyard",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 10},
			},
		},
		Requires: func(e Constructible, m *Civ) bool {
			return m.hasResourceAny(e.GetID(), geo.ResourceTypeWood)
		},
		Produces: []ResAmount{
			{Res: geo.ResWoodsOak, Amount: 1}, // Default to Oak for now.
		},
		Description: "Processes wood from the region.",
	}

	BlueprintGemMine = &ConstructionBlueprint{
		Name: "Gem Mine",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 40},
				{Type: geo.ResourceTypeStone, Amount: 20},
			},
		},
		Requires: func(e Constructible, m *Civ) bool {
			return m.hasResourceAny(e.GetID(), geo.ResourceTypeGem)
		},
		Produces: []ResAmount{
			{Res: geo.ResGemsAmethyst, Amount: 1}, // Default to Amethyst for now.
		},
		Description: "Extracts precious gems from the region.",
	}

	BlueprintMarket = &ConstructionBlueprint{
		Name: "Market",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 50},
			},
		},
		Effects: func(e Constructible, m *Civ) {
			// Increases trade gains and wealth generation.
		},
		Description: "Increases trade gains and wealth generation.",
	}

	BlueprintLibrary = &ConstructionBlueprint{
		Name: "Library",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 30},
				{Type: geo.ResourceTypeStone, Amount: 10},
			},
		},
		Effects: func(e Constructible, m *Civ) {
			// Increases research and knowledge accumulation.
		},
		Description: "Increases research and knowledge accumulation.",
	}

	AllBlueprints = []*ConstructionBlueprint{
		BlueprintHut,
		BlueprintGranary,
		BlueprintWalls,
		BlueprintAquaeduct,
		BlueprintQuarry,
		BlueprintMine,
		BlueprintLumberyard,
		BlueprintGemMine,
		BlueprintMarket,
		BlueprintLibrary,
	}
)

func (p *Project) String() string {
	return fmt.Sprintf("%s (%.1f%%)", p.Blueprint.Name, p.Progress*100)
}
