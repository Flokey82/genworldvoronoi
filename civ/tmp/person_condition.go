package civ

import (
	"math/rand"

	"github.com/Flokey82/genetics/geneticshuman"
)

// ConditionType represents the type of condition.
type ConditionType int

// The various types of conditions.
const (
	ConditionTypeCurse ConditionType = iota
	ConditionTypeBlessing
	ConditionTypeIllness
	ConditionTypeInjury
	ConditionTypeDisability
	ConditionTypeTrauma
)

// Conditions are temporary traits that can be applied to a person.
// They can be removed by a reversal process.
// TODO: Make sure to apply the condition effects to the person.
type Conditions struct {
	Conditions []*Condition
}

// NewConditions creates a new Conditions struct.
func NewConditions() *Conditions {
	return &Conditions{}
}

func (c *Conditions) Add(condition *Condition, m *Civ, p *Person) {
	// Make sure the condition is not already applied.
	for _, cond := range c.Conditions {
		if cond == condition {
			return
		}
	}
	c.Conditions = append(c.Conditions, condition)
	condition.apply(m, p)
}

func (c *Conditions) Remove(condition *Condition, success bool, m *Civ, p *Person) {
	for i, cond := range c.Conditions {
		if cond == condition {
			c.Conditions = append(c.Conditions[:i], c.Conditions[i+1:]...)
			if condition.reverseSuccess != nil && success {
				condition.reverseSuccess(m, p)
			} else if condition.reverseFail != nil && !success {
				condition.reverseFail(m, p)
			}
			return
		}
	}
}

// TODO:
// - Add a way to customize the conditions
// Like for example, name it after a wizard, a god, demon, etc.
// - Also add illness and other conditions.
// - Add a requirement for the Condition to be removed.
// - Add a factor for potential contagion.
// - Store the conditions in the person struct.
// - Add an optional duration.
type Condition struct {
	name           string
	Type           ConditionType
	apply          func(m *Civ, p *Person) // effect of the condition
	reverseSuccess func(m *Civ, p *Person) // on successful reversal
	reverseFail    func(m *Civ, p *Person) // on failed reversal
}

var curseOfCursedFamily = &Condition{
	name: "Curse of the Cursed Family",
	Type: ConditionTypeCurse,
	apply: func(m *Civ, p *Person) {
		// Get all living relatives.
		relatives := pickRelatives(p)

		// Apply the curse to all relatives.
		for _, relative := range relatives {
			// Apply the curse to the relative.
			relative.p.Conditions.Add(curseOfCursedFamilyMember, m, relative.p)
		}
	},
}

var curseOfCursedFamilyMember = &Condition{
	name: "Curse of the Cursed Family (Family Member)",
	Type: ConditionTypeCurse,
	apply: func(m *Civ, p *Person) {
		// Apply the curse to the person.
		p.Traits |= geneticshuman.TraitCruel
		p.Popularity.Add(5.0)
		p.NickName = "the Cursed family"
	},
}

var curseLazinessAndCarelessness = &Condition{
	name: "Curse of Laziness and Carelessness",
	Type: ConditionTypeCurse,
	apply: func(m *Civ, p *Person) {
		p.Traits |= geneticshuman.TraitCareless
		p.Traits &^= geneticshuman.TraitCareful
		p.Traits &^= geneticshuman.TraitAmbitious
		p.Traits &^= geneticshuman.TraitKind
		p.Traits &^= geneticshuman.TraitHonest
		p.Traits &^= geneticshuman.TraitBrave
		p.Traits &^= geneticshuman.TraitAggressive
		p.Popularity.Add(0.5)
		p.NickName = "the Cursed"
	},
}

var curseCrueltyAndDeception = &Condition{
	name: "Curse of Cruelty and Deciet",
	Type: ConditionTypeCurse,
	apply: func(m *Civ, p *Person) {
		p.Traits |= geneticshuman.TraitDeceptive
		p.Traits |= geneticshuman.TraitCruel
		p.Traits &^= geneticshuman.TraitKind
		p.Traits &^= geneticshuman.TraitHonest
		p.Popularity.Add(1.0)
		p.NickName = "the Cursed"
	},
}

var curseAmbitionAndCruelty = &Condition{
	name: "Curse of Ambition and Cruealty",
	Type: ConditionTypeCurse,
	apply: func(m *Civ, p *Person) {
		p.Traits |= geneticshuman.TraitDeceptive
		p.Traits |= geneticshuman.TraitAmbitious
		p.Traits |= geneticshuman.TraitCruel

		if rand.Intn(100) < 50 {
			p.Traits |= geneticshuman.TraitCareful
			p.Traits &^= geneticshuman.TraitCareless
		}

		// Remove positive traits.
		p.Traits &^= geneticshuman.TraitKind
		p.Traits &^= geneticshuman.TraitHonest
		p.Traits &^= geneticshuman.TraitContent
		// p.Traits &^= geneticshuman.TraitBrave // Let's keep this one for now.
		p.Popularity.Add(5.0)
		p.NickName = "the Cursed"
	},
}

var curseParanoiaCrueltyAndDeception = &Condition{
	name: "Curse of Paranoia, Cruelty and Deception",
	Type: ConditionTypeCurse,
	apply: func(m *Civ, p *Person) {
		p.Traits |= geneticshuman.TraitDeceptive
		p.Traits |= geneticshuman.TraitCruel
		p.Traits |= geneticshuman.TraitParanoid
		p.Traits |= geneticshuman.TraitCareful
		p.Traits &^= geneticshuman.TraitHonest
		p.Traits &^= geneticshuman.TraitKind
		p.Traits &^= geneticshuman.TraitTrusting
		p.Traits &^= geneticshuman.TraitCareless
		p.NickName = "the Cursed"
	},
}

var blessingOfTheBrave = &Condition{
	name: "Blessing of the Brave",
	Type: ConditionTypeBlessing,
	apply: func(m *Civ, p *Person) {

		p.Traits |= geneticshuman.TraitKind
		p.Traits |= geneticshuman.TraitBrave
		p.Traits |= geneticshuman.TraitHonest
		p.Traits |= geneticshuman.TraitCareful

		if rand.Intn(100) < 50 {
			p.Traits |= geneticshuman.TraitAmbitious
			p.Traits &^= geneticshuman.TraitContent
		}

		// Remove negative traits.
		p.Traits &^= geneticshuman.TraitCruel
		p.Traits &^= geneticshuman.TraitDeceptive
		p.Traits &^= geneticshuman.TraitCareless
		// p.Traits &^= geneticshuman.TraitAggressive // Let's keep this one for now.
		p.NickName = "the Blessed"
		p.Popularity.Add(1.0)
	},
}

var lessonForTheBrave = &Condition{
	name: "Lesson for the Brave",
	Type: ConditionTypeBlessing,
	apply: func(m *Civ, p *Person) {
		p.Traits |= geneticshuman.TraitBrave
		p.Traits ^= geneticshuman.TraitCowardly
	},
}

var lessonForTheCareful = &Condition{
	name: "Lesson for the Careful",
	Type: ConditionTypeBlessing,
	apply: func(m *Civ, p *Person) {
		p.Traits |= geneticshuman.TraitCareful
		p.Traits ^= geneticshuman.TraitCareless
	},
}

var traumaOfTheCowardly = &Condition{
	name: "Trauma of the Cowardly",
	Type: ConditionTypeTrauma,
	apply: func(m *Civ, p *Person) {
		p.Traits |= geneticshuman.TraitCowardly
		p.Traits ^= geneticshuman.TraitBrave
	},
}
