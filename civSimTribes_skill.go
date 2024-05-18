package genworldvoronoi

import (
	"log"
	"math/rand"

	"github.com/Flokey82/genbiome"
)

type Skill struct {
	Name           string
	MinScore       float64          // The minimum score required to develop the skill.
	Chance         float64          // The chance of developing the skill.
	Biomes         []int            // The biomes the skill can be developed in.
	ResPres        ResourcePresence // The resources required to develop the skill.
	WaterProximity bool             // If the skill requires water proximity. TODO: Get rid of this.
	RegProx        RegionProximity  // The region proximity required to develop the skill.
	Requires       []*Skill         // Required skills to develop this skill.
	Effect         func(t *Tribe)   // The effect of the skill on the tribe.
}

func (s *Skill) CanDevelopIn(curRegProp *RegionProp, resPres *ResourcePresence) bool {
	waterProximity := curRegProp.River || curRegProp.Lake || curRegProp.Ocean
	// Check required proximities.
	if s.WaterProximity && !waterProximity ||
		s.RegProx.Mountain && !curRegProp.Mountain ||
		s.RegProx.River && !curRegProp.River ||
		s.RegProx.Lake && !curRegProp.Lake ||
		s.RegProx.Ocean && !curRegProp.Ocean ||
		s.ResPres.Wood && !resPres.Wood ||
		s.ResPres.Stones && !resPres.Stones ||
		s.ResPres.Metals && !resPres.Metals ||
		s.ResPres.Gems && !resPres.Gems {
		return false
	}

	// If there is no biome requirement, the skill can be developed anywhere.
	if len(s.Biomes) == 0 {
		return true
	}

	// Check if the biome is suitable for the skill.
	for _, b := range s.Biomes {
		if b == int(curRegProp.Biome) {
			return true
		}
	}
	return false
}

// GetBonusForBiome returns a bonus score derived from the skill based on the biome and water proximity.
func (s *Skill) GetBonusForBiome(curRegProp *RegionProp, resPres *ResourcePresence) float64 {
	if s.CanDevelopIn(curRegProp, resPres) {
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
		ResPres:  ResourcePresence{Wood: true},
		Requires: []*Skill{SSkillGathering},
	}
	SSkillStoneWorking = &Skill{
		Name:     "Stone Working",
		MinScore: 0.5,
		Chance:   0.4,
		ResPres:  ResourcePresence{Stones: true},
		Requires: []*Skill{SSkillGathering},
	}
	SSkillMetalWorking = &Skill{
		Name:     "Metal Working",
		MinScore: 0.5,
		Chance:   0.4,
		ResPres:  ResourcePresence{Metals: true},
		Requires: []*Skill{SSkillStoneWorking},
	}
	SSkillGemWorking = &Skill{
		Name:     "Gem Working",
		MinScore: 0.5,
		Chance:   0.4,
		ResPres:  ResourcePresence{Gems: true},
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
		Name:     "Settling (Highland)",
		MinScore: 0.9,
		Chance:   0.1,
		RegProx: RegionProximity{
			Mountain: true,
		},
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
		Name:     "Sea Faring",
		MinScore: 0.9,
		Chance:   0.1,
		RegProx: RegionProximity{
			Ocean: true,
		},
		Requires: []*Skill{SSkillBoating},
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
