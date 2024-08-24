package genworldvoronoi

import (
	"log"
	"math/rand"
	"sort"
	"strings"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/genworldvoronoi/geo"
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
	Profession     *Profession      // The profession that can be derived from the skill.
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
		Name:       "Hunting",
		MinScore:   0.4,
		Chance:     0.4,
		Biomes:     huntingBiomes,
		Requires:   []*Skill{SSkillGathering},
		Profession: ProfessionHunter,
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
		Profession: ProfessionGatherer,
	}
	SSkillWoodworking = &Skill{
		Name:       "Woodworking",
		MinScore:   0.5,
		Chance:     0.4,
		ResPres:    ResourcePresence{Wood: true},
		Requires:   []*Skill{SSkillGathering},
		Profession: ProfessionCarpenter,
	}
	SSkillStoneWorking = &Skill{
		Name:       "Stone Working",
		MinScore:   0.5,
		Chance:     0.4,
		ResPres:    ResourcePresence{Stones: true},
		Requires:   []*Skill{SSkillGathering},
		Profession: ProfessionMason,
	}
	SSkillMetalWorking = &Skill{
		Name:       "Metal Working",
		MinScore:   0.5,
		Chance:     0.4,
		ResPres:    ResourcePresence{Metals: true},
		Requires:   []*Skill{SSkillStoneWorking},
		Profession: ProfessionBlacksmith,
	}
	SSkillGemWorking = &Skill{
		Name:       "Gem Working",
		MinScore:   0.5,
		Chance:     0.4,
		ResPres:    ResourcePresence{Gems: true},
		Requires:   []*Skill{SSkillStoneWorking},
		Profession: ProfessionJeweler,
	}
	SSkillFishing = &Skill{
		Name:           "Fishing",
		MinScore:       0.9,
		Chance:         0.1,
		Biomes:         nil, // Requires water proximity
		WaterProximity: true,
		Requires:       []*Skill{SSkillGathering},
		Profession:     ProfessionFisher,
	}
	SSkillFarming = &Skill{
		Name:     "Farming",
		MinScore: 0.9,
		Chance:   0.4,
		Biomes: []int{
			genbiome.WhittakerModBiomeTemperateGrassland,
			genbiome.WhittakerModBiomeWetlands,
		},
		Requires:   []*Skill{SSkillGathering},
		Profession: ProfessionFarmer,
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
		Requires:   []*Skill{SSkillHunting},
		Profession: ProfessionHerder,
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
		Profession: ProfessionShipwright,
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
		Profession: ProfessionSailor,
	}

	SSkillSurvival = &Skill{
		Name:     "Survival",
		MinScore: 0.9,
		Chance:   0.1,
		Requires: []*Skill{SSkillHunting, SSkillGathering},
		Effect: func(t *Tribe) {
			// TODO: Have some effect.
			// Survival should be unlocked for tribes that live in difficult
			// regions, like deserts, tundras, wildlands, etc.
		},
	}

	SSkillRiverNav = &Skill{
		Name:     "River Navigation",
		MinScore: 0.9,
		Chance:   0.1,
		RegProx: RegionProximity{
			River: true,
		},
		Requires: []*Skill{SSkillBoating},
		Effect:   func(t *Tribe) {},
	}

	SSkillClimbing = &Skill{
		Name:     "Climbing",
		MinScore: 0.9,
		Chance:   0.1,
		RegProx: RegionProximity{
			Mountain: true,
		},
		Effect: func(t *Tribe) {
		},
	}

	SSkillNomadic = &Skill{
		Name:     "Nomadic",
		MinScore: 0.9,
		Chance:   0.1,
		Biomes:   nomadicBiomes,
		Effect: func(t *Tribe) {
			if t.Type < TribeTypeNomadic {
				t.Type = TribeTypeNomadic
			}
		},
	}
)

var Skills = []*Skill{
	SSkillHunting,
	SSkillGathering,
	SSkillWoodworking,  // Change to carpentry?
	SSkillStoneWorking, // Change to masonry?
	SSkillMetalWorking, // Change to smithing?
	SSkillGemWorking,   // Change to gemcutting?
	SSkillFishing,
	SSkillFarming,
	SSkillHerding,
	SSkillSettling,
	SSkillSettlingWetlands,
	SSkillSettlingHighland,
	SSkillBoating,
	SSkillSeaFaring,

	// TODO:
	// - SkillMining
	// - SkillQuarrying
	// - SkillWeaving
	// - SkillPottery
	// - SkillMountaineering
	// - SkillDungeoneering
	// - SkillCaving

	SSkillSurvival,
	SSkillRiverNav,
	SSkillNomadic,
	SSkillClimbing,
}

func (m *Civ) genCultureSkills() {
	// Now re-evaluate the specialities of each culture, based on the
	// resources they have access to.

	// NOTE: Someone smarter should come up with the rules for this...
	// and maybe this should also be more generalized so it can be
	// evaluated for all other things that occupy multiple regions like
	// religions, monestaries, city-states, etc. !!!!!!!!!!

	// We calculate the ratio of resources to number of regions, then
	// we assign the skills based on the highest ratio for each
	// resource per culture.
	// For all resource groups and types, then sort the cultures
	// by the ratio of the resource to the number of regions.
	// The top 3 cultures will get the specialty for that resource?

	// Specialities should give some advantage or bonus for the culture.
	// For example, a culture with the seafaring specialty should get
	// a bonus to naval combat, or a bonus to trade with coastal regions,
	// giving them access to exotic goods.
	// A culture with the survival specialty should get a bonus to
	// survival skills, or a bonus to exploration.

	log.Println("re-evaluating culture skills... (just a placeholder for now)")

	// Copy the cultures to a slice.
	cultureCopy := make([]*Culture, len(m.Cultures))
	copy(cultureCopy, m.Cultures)

	skillMap := make(map[*Culture][]*Skill)

	for _, c := range m.Cultures {
		// Add the default skill/skills based on the culture type.
		// TODO: Depending on the culture, there should be several options
		// to select from, also based on the statistics of the regions the
		// culture has access to. (randomized?)
		// For example, naval cultures should only get the seafaring
		// specialty if they have access to a wide coastal regions.
		// Highland cultures should be able to get different survival
		// skills based on the climate of the highlands.
		switch c.Type {
		case CultureTypeWildland:
			skillMap[c] = append(skillMap[c], SSkillSurvival)
		case CultureTypeGeneric:
		case CultureTypeRiver:
			// Other possible specialities or bonuses:
			// - hydro power
			// - trading via rivers (?)
			skillMap[c] = append(skillMap[c], SSkillRiverNav)
		case CultureTypeLake:
			skillMap[c] = append(skillMap[c], SSkillFishing)
		case CultureTypeNaval:
			// Other possible specialities or bonuses:
			// - trade via sea
			skillMap[c] = append(skillMap[c], SSkillSeaFaring)
		case CultureTypeNomadic:
			// Other possible specialities or bonuses:
			// - survival
			// - domestication / cattle breeding?
			skillMap[c] = append(skillMap[c], SSkillNomadic)
		case CultureTypeHunting:
			// Other possible specialities or bonuses:
			// - riding
			skillMap[c] = append(skillMap[c], SSkillHunting)
		case CultureTypeHighland:
			// Other possible specialities or bonuses:
			// - lower penalty for crossing mountains
			// - mining (?)
			skillMap[c] = append(skillMap[c], SSkillClimbing)
		}
	}

	// Metals.
	// TODO: Change this to "metalwork" with a speciality for each metal type.
	for res := 0; res < geo.ResMaxMetals; res++ {
		sort.Slice(cultureCopy, func(i, j int) bool {
			return float64(cultureCopy[i].Stats.ResMetal[res])/float64(len(cultureCopy[i].Regions)) > float64(cultureCopy[j].Stats.ResMetal[res])/float64(len(cultureCopy[j].Regions))
		})
		for i, c := range cultureCopy {
			if i >= 3 {
				break
			}
			// geo.MetalToString(res)
			skillMap[c] = append(skillMap[c], SSkillMetalWorking)
		}
	}

	// Gems.
	// TODO: Change this to "gemwork" with a speciality for each gem type.
	for res := 0; res < geo.ResMaxGems; res++ {
		sort.Slice(cultureCopy, func(i, j int) bool {
			return float64(cultureCopy[i].Stats.ResGems[res])/float64(len(cultureCopy[i].Regions)) > float64(cultureCopy[j].Stats.ResGems[res])/float64(len(cultureCopy[j].Regions))
		})
		for i, c := range cultureCopy {
			if i >= 3 {
				break
			}
			// geo.GemToString(res)
			skillMap[c] = append(skillMap[c], SSkillGemWorking)
		}
	}

	// Stones.
	// TODO: Change this to "stonework" with a speciality for each stone type.
	for res := 0; res < geo.ResMaxStones; res++ {
		sort.Slice(cultureCopy, func(i, j int) bool {
			return float64(cultureCopy[i].Stats.ResStones[res])/float64(len(cultureCopy[i].Regions)) > float64(cultureCopy[j].Stats.ResStones[res])/float64(len(cultureCopy[j].Regions))
		})
		for i, c := range cultureCopy {
			if i >= 3 {
				break
			}
			// geo.StoneToString(res)
			skillMap[c] = append(skillMap[c], SSkillStoneWorking)
		}
	}

	// Woods.
	// TODO: Change this to "woodwork" with a speciality for each wood type.
	for res := 0; res < geo.ResMaxWoods; res++ {
		sort.Slice(cultureCopy, func(i, j int) bool {
			return float64(cultureCopy[i].Stats.ResWood[res])/float64(len(cultureCopy[i].Regions)) > float64(cultureCopy[j].Stats.ResWood[res])/float64(len(cultureCopy[j].Regions))
		})
		for i, c := range cultureCopy {
			if i >= 3 {
				break
			}
			// geo.WoodToString(res)
			skillMap[c] = append(skillMap[c], SSkillWoodworking)
		}
	}

	// Log all skills per culture.
	for _, c := range m.Cultures {
		var skillNames []string
		for _, s := range skillMap[c] {
			skillNames = append(skillNames, s.Name)
		}
		log.Println(c.Name, "specialties:", strings.Join(skillNames, ", "))
	}
}
