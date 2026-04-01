package civ2

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/go_gens/gameconstants"
	"github.com/Flokey82/go_gens/genlanguage"
)

func (m *Civ) tickPeople(nDays int) {
	var newBorns []*Person
	for _, p := range m.People {
		if child := m.tickPerson(p, nDays); child != nil {
			newBorns = append(newBorns, child)
		}
	}
	// Add newborns to the world population.
	// NOTE: They are already added in tickPersonPregnancy, but we might want to
	// handle them here if we change the architecture.
}

func (m *Civ) tickPerson(p *Person, nDays int) *Person {
	if !m.doesPersonExist(p) {
		return nil
	}
	// Calculate age.
	m.tickPersonAge(p, nDays)

	// Advance pregnancy.
	var child *Person
	if p.Prengancy != nil {
		child = m.tickPersonPregnancy(p, nDays)
	}

	// Check if person dies of natural causes.
	m.tickPersonDeath(p, nDays)

	// If the person is dead, we don't need to do anything else.
	if p.Dead() {
		return nil
	}

	// Check existing relationships.
	{
		// Check who we love or hate in our family.
		var hated, loved []relation
		for _, c := range pickRelatives(p) {
			// Check if we hate the person.
			if p.Opinions.GetOpinion(c.p) < -0.5 {
				hated = append(hated, c)
			} else if p.Opinions.GetOpinion(c.p) > 0.5 {
				loved = append(loved, c)
			}
		}

		// Add an entry to the history.
		if len(hated) > 0 {
			var str string
			for _, h := range hated {
				str += fmt.Sprintf("%s (%s %.2f), ", h.p.String(), h.relationOfPerson, p.Opinions.GetOpinion(h.p))
			}
			str = str[:len(str)-2]
			m.AddEvent("Hate", fmt.Sprintf("%s hates %s", p.String(), str), p.Ref())
		}
		if len(loved) > 0 {
			var str string
			for _, h := range loved {
				str += fmt.Sprintf("%s (%s %.2f), ", h.p.String(), h.relationOfPerson, p.Opinions.GetOpinion(h.p))
			}
			str = str[:len(str)-2]
			m.AddEvent("Love", fmt.Sprintf("%s loves %s", p.String(), str), p.Ref())
		}

		// Check if we have a nemesis.
		nemesis := p.Opinions.GetNemesis()
		if nemesis != nil {
			m.AddEvent("Nemesis", fmt.Sprintf("%s has a nemesis: %s (%.2f)", p.String(), nemesis.String(), p.Opinions.GetOpinion(nemesis)), p.Ref())
		}
	}

	// TODO: tickFamily
	return child
}

func (m *Civ) tickPersonAge(p *Person, nDays int) {
	// Calculate current age.
	if m.History.GetDayOfYear() < p.Birth.Day {
		p.Age = int(m.History.GetYear()) - p.Birth.Year - 1
	} else {
		p.Age = int(m.History.GetYear()) - p.Birth.Year
	}
}

func (m *Civ) tickPersonDeath(p *Person, nDays int) {
	// Check if person dies of natural causes.
	if !gameconstants.DiesAtAgeWithinNDays(p.Age, nDays) || p.Dead() {
		return
	}
	options := []string{
		"illness", "an accident", "a mysterious cause", "a heart attack",
		"a stroke", "a fall", "a lightning strike", "a snake bite",
		"a wild animal attack", "a drowning", "a fire", "a poisoning",
	}

	if p.Age > 14 {
		options = append(options, "old age")
		if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
			options = append(options, "a duel", "a fight", "challenging a wild animal")
		}
		if p.Traits.HasTrait(geneticshuman.TraitCareless) {
			options = append(options, "an accident", "eating spoiled food", "a fall")
		}
		if p.Traits.HasTrait(geneticshuman.TraitTrusting) {
			options = append(options, "being deceived", "being robbed", "being poisoned")
		}
	}
	m.killPerson(p, options[rand.Intn(len(options))])
}

func (m *Civ) killPerson(p *Person, reason string) *civ.Event {
	p.Death.Day = int(m.History.GetDayOfYear())
	p.Death.Year = int(m.History.GetYear())
	p.Death.Region = p.Region

	if p.Spouse != nil {
		p.Spouse.Spouse = nil
	}

	if p.Prengancy != nil {
		m.killPerson(p.Prengancy, fmt.Sprintf("mother %s dying due to %s", p.Name(), reason))
	}

	// handleInheritance
	m.handleInheritance(p)

	var deathStr string
	name := p.Name()
	if name == "" {
		if p.Gender() == GenderFemale {
			name = "unborn girl"
		} else {
			name = "unborn boy"
		}
	}
	if reason == "" {
		deathStr = fmt.Sprintf("%s died at age %d", name, p.Age)
	} else {
		deathStr = fmt.Sprintf("%s died at age %d due to %s", name, p.Age, reason)
	}
	return m.AddEvent("Death", deathStr, p.Ref())
}

func (m *Civ) handleInheritance(p *Person) {
	relatives := pickRelatives(p)
	sort.Slice(relatives, func(i, j int) bool {
		return p.Opinions.GetOpinion(relatives[i].p) > p.Opinions.GetOpinion(relatives[j].p)
	})
	// TODO: Transfer artifacts and gold.
}

func (m *Civ) tickPersonPregnancy(p *Person, nDays int) *Person {
	if p.Prengancy == nil {
		return nil
	}

	p.PregnancyCounter -= nDays
	if p.PregnancyCounter > 0 {
		return nil
	}

	wasBornNDaysAgo := -p.PregnancyCounter
	child := p.Prengancy

	p.Prengancy = nil
	p.PregnancyCounter = 0

	var lang *genlanguage.Language
	if p.Spouse != nil && rand.Intn(100) < 50 {
		lang = p.Spouse.Culture.Language
	} else {
		lang = p.Culture.Language
	}

	var firstName string
	if poolSize := lang.GetFirstNamePoolSize(); poolSize > 100 && rand.Intn(poolSize) > 10 {
		firstName = lang.GetFirstName()
	}
	if firstName == "" {
		firstName = lang.MakeFirstName()
	}
	child.FirstName = firstName
	p.Children = append(p.Children, child)

	if p.Spouse != nil {
		p.Spouse.Children = append(p.Spouse.Children, child)
	} else if p.Father != nil {
		p.Father.Children = append(p.Father.Children, child)
	}

	child.LastName = p.LastName
	child.Birth.Region = p.Region
	child.Birth.Year = int(m.History.GetYear())
	child.Birth.Day = m.History.GetDayOfYear() - wasBornNDaysAgo
	if child.Birth.Day < 0 {
		child.Birth.Year--
		child.Birth.Day += 365
		m.tickPerson(child, wasBornNDaysAgo)
	}

	child.City = p.City
	child.Culture = p.Culture
	child.Region = p.Region

	m.People = append(m.People, child)
	return child
}

func (m *Civ) doesPersonExist(p *Person) bool {
	return !p.Dead() && p.Birth.IsSet() && p.Birth.Year <= int(m.History.GetYear())
}
