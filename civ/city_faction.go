package civ

import (
	"log"
	"math/rand"

	"github.com/Flokey82/go_gens/gengovernment"
)

func (m *Civ) handleLeadershipCity(c *City) {
	// Update the factions
	// This will trigger election of new leaders, etc.
	c.Leadership.TickCity(c, m)
	for _, f := range c.Factions {
		f.TickCity(c, m)
	}
	c.GoverningPeople.handleLeadership(c, m)
}

func (f *Faction) TickCity(c *City, m *Civ) {
	f.Tick(c, m)

	if rand.Float64() < 0.1 {
		action := f.pickAction(c, c.GoverningPeople, m)
		if action != nil {
			action.Execute(c, m, f)
		}
	}
}

func (c *City) getFactionActions(m *Civ, f *Faction) []*FactionAction {
	log.Printf("!!!%s is considering actions.", c.String())
	return defaultFactionActions
}

// getPreferredLeadershipForm returns the preferred leadership form of the tribe.
func (t *City) getPreferredLeadershipForm() gengovernment.LeadershipForm {
	if len(t.People) < 100 {
		return gengovernment.LeadershipFormChiefdom
	}
	return gengovernment.LeadershipFormCouncil
}

// getPossibleLeadershipForms returns the possible leadership forms of the tribe.
func (t *City) getPossibleLeadershipForms() []gengovernment.LeadershipForm {
	return []gengovernment.LeadershipForm{gengovernment.LeadershipFormChiefdom, gengovernment.LeadershipFormCouncil}
}

func (t *City) findNaturalProgression() gengovernment.LeadershipForm {
	// TODO: Pick from the preferred leadership forms.
	var currentInfluence gengovernment.LeadershipInfluence
	var fallbackForm gengovernment.LeadershipForm
	switch {
	case t.Population < 100:
		currentInfluence = gengovernment.LeadershipInfluenceTribe
		fallbackForm = gengovernment.LeadershipFormChiefdom
	default:
		currentInfluence = gengovernment.LeadershipInfluenceSettlement
		fallbackForm = gengovernment.LeadershipFormChiefdom
	}

	// Check if we need to change the leadership form.
	// There is a chance we just retain the current form.
	curMin, curMax := t.Leadership.Form.RangeInfluence()
	if curMin <= currentInfluence && curMax >= currentInfluence && rand.Intn(100) < 20 {
		return t.Leadership.Form
	}

	natProg := t.Leadership.Form.NaturalProgression()
	if len(natProg) == 0 {
		log.Printf("!!!%s has no natural progression for %s", t.String(), t.Leadership.Form)
		return fallbackForm
	}

	for _, i := range rand.Perm(len(natProg)) {
		frm := natProg[i]
		minInf, maxInf := frm.RangeInfluence()
		if minInf <= currentInfluence && maxInf >= currentInfluence {
			return frm
		}
	}
	return t.Leadership.Form
}

// findCoupProgression returns the possible progression of the leadership form through a coup.
func (t *City) findCoupProgression() gengovernment.LeadershipForm {
	// TODO: Pick from the preferred leadership forms.
	var currentInfluence gengovernment.LeadershipInfluence
	var fallbackForm gengovernment.LeadershipForm
	switch {
	case t.Population < 100:
		currentInfluence = gengovernment.LeadershipInfluenceTribe
		fallbackForm = gengovernment.LeadershipFormChiefdom
	default:
		currentInfluence = gengovernment.LeadershipInfluenceSettlement
		fallbackForm = gengovernment.LeadershipFormChiefdom
	}

	// Check if we need to change the leadership form.
	// There is a chance we just retain the current form.
	curMin, curMax := t.Leadership.Form.RangeInfluence()
	if curMin <= currentInfluence && curMax >= currentInfluence && rand.Intn(100) < 20 {
		return t.Leadership.Form
	}

	natProg := t.Leadership.Form.CoupProgression()
	if len(natProg) == 0 {
		log.Printf("!!!%s has no natural progression for %s", t.String(), t.Leadership.Form)
		return fallbackForm
	}

	for _, i := range rand.Perm(len(natProg)) {
		frm := natProg[i]
		minInf, maxInf := frm.RangeInfluence()
		if minInf <= currentInfluence && maxInf >= currentInfluence {
			return frm
		}
	}
	return t.Leadership.Form
}
