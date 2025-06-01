package civ2

import (
	"log"

	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/genworldvoronoi/geo"
)

// Skill represents a skill developed by a culture.
//
// A skill can be developed if certain conditions are met.
// - The region has the required resources or terrain.
// - The culture has the required skills (if any).
// - The culture has the required culture type (some skills are inherent to certain culture types).
//
// Skills unlock the ability to
// - Harvest specific resources
// - Craft specific items
// - Build specific structures
// - Perform specific actions
// - Develop other skills
type Skill struct {
	ID                   int
	Name                 string
	DependsOn            []*Skill
	RequiresResourceType []geo.ResourceType
	RequiresWater        bool
}

// Skills developed by a culture.
var (
	SkillGathering = &Skill{
		ID:   1,
		Name: "Gathering",
	}
	SkillHunting = &Skill{
		ID:   2,
		Name: "Hunting",
	}
	SkillFishing = &Skill{
		ID:            3,
		Name:          "Fishing",
		RequiresWater: true,
	}
	SkillHerding = &Skill{
		ID:        4,
		Name:      "Herding",
		DependsOn: []*Skill{SkillHunting},
	}
	SkillAgriculture = &Skill{
		ID:        5,
		Name:      "Agriculture",
		DependsOn: []*Skill{SkillGathering},
	}
	SkillLumbering = &Skill{
		ID:                   6,
		Name:                 "Lumbering",
		RequiresResourceType: []geo.ResourceType{geo.ResourceTypeWood},
	}
	SkillMining = &Skill{
		ID:                   7,
		Name:                 "Mining",
		RequiresResourceType: []geo.ResourceType{geo.ResourceTypeStone, geo.ResourceTypeMetal},
	}
	SkillCarpentry = &Skill{
		ID:                   8,
		Name:                 "Carpentry",
		DependsOn:            []*Skill{SkillLumbering},
		RequiresResourceType: []geo.ResourceType{geo.ResourceTypeWood},
	}
	SkillMasonry = &Skill{
		ID:        9,
		Name:      "Masonry",
		DependsOn: []*Skill{SkillMining},
	}
	SkillForging = &Skill{
		ID:                   10,
		Name:                 "Forging",
		DependsOn:            []*Skill{SkillMining},
		RequiresResourceType: []geo.ResourceType{geo.ResourceTypeMetal},
	}
)

var allSkills = []*Skill{
	SkillGathering,
	SkillHunting,
	SkillFishing,
	SkillHerding,
	SkillAgriculture,
	SkillLumbering,
	SkillMining,
	SkillCarpentry,
	SkillMasonry,
	SkillForging,
}

func cultureTypeToStartingSkills(t civ.CultureType) []*Skill {
	switch t {
	case civ.CultureTypeWildland, civ.CultureTypeGeneric:
		return []*Skill{SkillGathering, SkillHunting}
	case civ.CultureTypeRiver, civ.CultureTypeLake, civ.CultureTypeNaval:
		return []*Skill{SkillGathering, SkillFishing}
	case civ.CultureTypeNomadic:
		return []*Skill{SkillHunting, SkillHerding}
	case civ.CultureTypeHunting:
		return []*Skill{SkillGathering, SkillHunting}
	case civ.CultureTypeHighland:
		return []*Skill{SkillGathering, SkillHunting, SkillHerding}
	}
	return nil
}

func (m *Civ) developSkills(c *Culture, stats *regionStats, nDays int) {
	// Log the stats.
	log.Printf("Culture %d has %d regions, %d rivers, %d lakes, %d coastal regions",
		c.ID, stats.numRegions, stats.numRiver, stats.numLake, stats.numCoastal)

	// If we have no regions, we skip the culture.
	if stats.numRegions == 0 {
		return
	}

Bla:
	for _, s := range allSkills {
		if c.HasSkill(s) {
			continue
		}

		for _, d := range s.DependsOn {
			if !c.HasSkill(d) {
				continue Bla
			}
		}
		for _, rt := range s.RequiresResourceType {
			if m.Rand.Intn(stats.numRegions) >= stats.numResourceType[rt] {
				continue Bla
			}
		}
		if s.RequiresWater && m.Rand.Intn(stats.numRegions) >= stats.numRiver+stats.numLake+stats.numCoastal {
			continue Bla
		}
		if m.Rand.Intn(365)*2 < nDays {
			log.Printf("Culture %d has developed skill %s", c.ID, s.Name)
			c.Skills = append(c.Skills, s)
			// We only develop one skill per tick.
			return
		}
	}
}

// HasSkill returns true if the culture has the skill.
func (c *Culture) HasSkill(s *Skill) bool {
	for _, sk := range c.Skills {
		if sk == s {
			return true
		}
	}
	return false
}

func (m *Civ) getRegionSkills(r int) []*Skill {
	// Get the skills that are useful in the region.
	var skills []*Skill
	if m.IsRegRiver(r) || m.IsRegLakeOrWaterBody(r) || m.RegCellTypes.Values[r] == geo.CellTypeCoastalLand {
		skills = append(skills, SkillFishing)
	}

	// Get the culture type for the region.
	cultureType := m.cultureFunc(r)
	skills = append(skills, cultureTypeToStartingSkills(cultureType)...)

	// Check if there is wood.
	if m.ResourceLocations.HasType(r, geo.ResourceTypeWood) {
		skills = append(skills, SkillLumbering)
	}

	// Check if there is stone.
	if m.ResourceLocations.HasType(r, geo.ResourceTypeStone) {
		skills = append(skills, SkillMasonry)
	}

	return nil
}
