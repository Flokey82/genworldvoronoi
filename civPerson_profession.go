package genworldvoronoi

import (
	"fmt"
	"math/rand"

	"github.com/Flokey82/genetics/geneticshuman"
)

// Profession represents a profession that a person can have.
// TODO:
// - Add a way to customize the professions (specializations, etc).
// - Add risk of death, injury, etc.
// - Add specific actions that the profession can do.
//   - creating artifacts, writing books, etc.
//     either as heirlooms, as presents, for trade, or as monuments.
//   - craftsmen might create artifacts
//   - scholars might write books that encode knowledge (skills, etc)
//   - Add some resource conversion
//     e.g. hunter -> meat, gatherer -> berries, farmer -> grain
//     grain -> bread, meat -> cooked meat, berries -> jam
type Profession struct {
	Name          string
	PerPopulation int
	Tick          func(p *Person, nDays int)
	Actions       []*PersonalAction
}

// The various professions a person can have.
var (
	ProfessionHunter = &Profession{
		Name:          "hunter",
		PerPopulation: 3,
	}
	ProfessionGatherer = &Profession{
		Name:          "gatherer",
		PerPopulation: 3,
	}
	ProfessionFarmer = &Profession{
		Name:          "farmer",
		PerPopulation: 5,
	}
	ProfessionCarpenter = &Profession{
		Name:          "carpenter",
		PerPopulation: 20,
	}
	ProfessionMason = &Profession{
		Name:          "mason",
		PerPopulation: 20,
	}
	ProfessionBlacksmith = &Profession{
		Name:          "blacksmith",
		PerPopulation: 20,
	}
	ProfessionJeweler = &Profession{
		Name:          "jeweler",
		PerPopulation: 20,
	}
	ProfessionFisher = &Profession{
		Name:          "fisher",
		PerPopulation: 10,
	}
	ProfessionHerder = &Profession{
		Name:          "herder",
		PerPopulation: 10,
	}
	ProfessionShipwright = &Profession{
		Name:          "shipwright",
		PerPopulation: 20,
	}
	ProfessionSailor = &Profession{
		Name:          "sailor",
		PerPopulation: 20,
	}
	ProfessionMerchant = &Profession{
		Name:          "merchant",
		PerPopulation: 10,
	}
	ProfessionAdministrator = &Profession{
		Name:          "administrator",
		PerPopulation: 100,
	}
	ProfessionAmbassador = &Profession{
		Name:          "ambassador",
		PerPopulation: 200,
	}
	ProfessionCleric = &Profession{
		Name:          "cleric",
		PerPopulation: 100,
	}
	ProfessionHealer = &Profession{
		Name:          "healer",
		PerPopulation: 100,
	}
	ProfessionScholar = &Profession{
		Name:          "scholar",
		PerPopulation: 100,
	}
	ProfessionSpy = &Profession{
		Name:          "spy",
		PerPopulation: 1000,
	}
	ProfessionArchitect = &Profession{
		Name:          "architect",
		PerPopulation: 100,
	}
	ProfessionMiller = &Profession{
		Name:          "miller",
		PerPopulation: 100,
	}
	ProfessionBaker = &Profession{
		Name:          "baker",
		PerPopulation: 100,
	}
	ProfessionButcher = &Profession{
		Name:          "butcher",
		PerPopulation: 100,
	}

	// The following can be hobbies or professions.
	// TODO: Should we drop the per population count for these?
	ProfessionPainter = &Profession{
		Name:          "painter",
		PerPopulation: 100,
	}
	ProfessionMusician = &Profession{
		Name:          "musician",
		PerPopulation: 100,
	}
	ProfessionActor = &Profession{
		Name:          "actor",
		PerPopulation: 100,
	}
	ProfessionWriter = &Profession{
		Name:          "writer",
		PerPopulation: 100,
	}
	ProfessionPoet = &Profession{
		Name:          "poet",
		PerPopulation: 100,
	}
	ProfessionDancer = &Profession{
		Name:          "dancer",
		PerPopulation: 100,
	}
	ProfessionSculptor = &Profession{
		Name:          "sculptor",
		PerPopulation: 100,
		Actions: []*PersonalAction{
			{
				requires: func(p *Person) bool {
					// We need to be older than 10 to sculpt.
					return p.Age > 10
				},
				probability: func(p *Person) float64 {
					prob := 0.01
					// If we are ambitious, we are more likely to sculpt.
					if p.Traits.HasTrait(geneticshuman.TraitAmbitious) {
						prob += 0.4
					}
					// If we are content, we are less likely to sculpt.
					if p.Traits.HasTrait(geneticshuman.TraitContent) {
						prob -= 0.2
					}
					return max(prob, 0)
				},
				consequences: func(m *Civ, p *Person) {
					// Sculpt something.
					// The higher our skill, the better the sculpture.
					art := &Artifact{
						Name: "sculpture",
					}

					if p.Career.Skill/p.Career.Experience > 0.5 {
						art.Name = "masterful " + art.Name
					}

					// Add the sculpture to the person's inventory.
					p.addArtifact(m, art)

					// Add experience to the sculptor.
					p.Career.Skill += 0.01

					// Add a history event.
					m.History.AddEvent("artifact", fmt.Sprintf("%s sculpted %s", p.Name(), art.NameWithArticle()), p.Ref())

					// TODO: Based on the size of the sculpture, we might need to store it somewhere.
					// Also, we can't lose it like we can lose other items.
				},
			},
		},
	}
)

// String returns the string representation of the profession.
func (p Profession) String() string {
	return p.Name
}

// Career returns a new career for the profession.
func (p *Profession) Career(started int) *Career {
	return &Career{
		Profession: p,
		Started:    started,
	}
}

func (m *Civ) tickCareer(p *Person, nDays int) {
	if p.Career == nil {
		return
	}
	p.Career.Tick(p, nDays)
}

// Career represents a career that a person has.
type Career struct {
	Profession *Profession
	Started    int     // The year the person started the career.
	Experience float64 // The years of experience the person has in the career.
	Skill      float64 // The skill level of the person in the career.
}

// Tick updates the career of the person.
func (c *Career) Tick(p *Person, nDays int) {
	// Experience gained in fractions of a year.
	xp := float64(nDays) / 365.0

	// Update the experience.
	c.Experience += xp

	// Depending on our skill, we might have success or failure.
	// On success, our skill increases, on failure, our skill decreases.
	// There are certain factors that can influence the success or failure.
	luckFactor := 0.2
	if p.Traits.HasTrait(geneticshuman.TraitCareless) {
		luckFactor -= 0.2
	}
	if p.Traits.HasTrait(geneticshuman.TraitCareful) {
		luckFactor += 0.2
	}

	// Calculate the ratio of skill to experience.
	// TODO: Avoid division by zero.
	ratio := c.Skill / c.Experience

	// If a random float is below the ratio, we have success.
	success := rand.NormFloat64() < ratio+luckFactor

	// If we have success, increase the skill.
	if success {
		c.Skill += xp
	} else {
		c.Skill -= 0.1 * xp
	}

	if c.Profession.Tick != nil {
		c.Profession.Tick(p, nDays)
	}
}

// String returns the string representation of the career.
func (c Career) String() string {
	return fmt.Sprintf("%s (%.1f yrs, %.1f skll)", c.Profession, c.Experience, c.Skill)
}
