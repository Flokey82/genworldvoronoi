package genworldvoronoi

import (
	"log"
	"sort"

	"github.com/Flokey82/genworldvoronoi/geo"
)

const (
	SkillSurvival  = "survival"
	SkillGeneric   = "generic"
	SkillRiverNav  = "river navigation"
	SkillSeafaring = "seafaring"
	SkillFishery   = "fishery"
	SkillHunting   = "hunting"
	SkillNomadic   = "nomadic"
	SkillClimbing  = "climbing"

	// SkillMining     = "mining"
	// SkillSmithing   = "smithing"
	// SkillMasonry    = "masonry"
	// SkillWoodwork   = "woodwork"
	// SkillGemcutting = "gemcutting"
)

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

	skillMap := make(map[*Culture][]string)

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
			skillMap[c] = append(skillMap[c], SkillSurvival)
		case CultureTypeGeneric:
			skillMap[c] = append(skillMap[c], SkillGeneric)
		case CultureTypeRiver:
			// Other possible specialities or bonuses:
			// - hydro power
			// - trading via rivers (?)
			skillMap[c] = append(skillMap[c], SkillRiverNav)
		case CultureTypeLake:
			skillMap[c] = append(skillMap[c], SkillFishery)
		case CultureTypeNaval:
			// Other possible specialities or bonuses:
			// - trade via sea
			skillMap[c] = append(skillMap[c], SkillSeafaring)
		case CultureTypeNomadic:
			// Other possible specialities or bonuses:
			// - survival
			// - domestication / cattle breeding?
			skillMap[c] = append(skillMap[c], SkillNomadic)
		case CultureTypeHunting:
			// Other possible specialities or bonuses:
			// - riding
			skillMap[c] = append(skillMap[c], SkillHunting)
		case CultureTypeHighland:
			// Other possible specialities or bonuses:
			// - lower penalty for crossing mountains
			// - mining (?)
			skillMap[c] = append(skillMap[c], SkillClimbing)
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
			skillMap[c] = append(skillMap[c], geo.MetalToString(res))
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
			skillMap[c] = append(skillMap[c], geo.GemToString(res))
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
			skillMap[c] = append(skillMap[c], geo.StoneToString(res))
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
			skillMap[c] = append(skillMap[c], geo.WoodToString(res))
		}
	}

	// Log all skills per culture.
	for _, c := range m.Cultures {
		log.Println(c.Name, "specialties:", skillMap[c])
	}
}
