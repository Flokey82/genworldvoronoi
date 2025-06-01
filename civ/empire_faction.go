package civ

import (
	"log"
	"math/rand"

	"github.com/Flokey82/go_gens/gengovernment"
)

func (m *Civ) handleLeadershipEmpire(e *Empire) {
	// Update the factions
	// This will trigger election of new leaders, etc.
	e.Leadership.TickEmpire(e, m)
	for _, f := range e.Factions {
		f.TickEmpire(e, m)
	}
	e.GoverningPeople.handleLeadership(e, m)
}

func (f *Faction) TickEmpire(e *Empire, m *Civ) {
	f.Tick(e, m)

	if rand.Float64() < 0.1 {
		action := f.pickAction(e, e.GoverningPeople, m)
		if action != nil {
			action.Execute(e, m, f)
		}
	}
}

func (e *Empire) getFactionActions(m *Civ, f *Faction) []*FactionAction {
	log.Printf("!!!%s is considering actions.", e.String())
	return defaultFactionActions
}

// getPreferredLeadershipForm returns the preferred leadership form of the empire.
func (e *Empire) getPreferredLeadershipForm() gengovernment.LeadershipForm {
	// TODO: This should depend on the culture of the population,
	// the size of the empire, etc.
	if e.Capital.Population < 1000 {
		return gengovernment.LeadershipFormMonarchy
	}
	return gengovernment.LeadershipFormDictatorship
}

// getPossibleLeadershipForms returns the possible leadership forms of the empire.
func (e *Empire) getPossibleLeadershipForms() []gengovernment.LeadershipForm {
	return []gengovernment.LeadershipForm{
		gengovernment.LeadershipFormMonarchy,
		gengovernment.LeadershipFormDictatorship,
		gengovernment.LeadershipFormCouncil,
		gengovernment.LeadershipFormRepublic,
		gengovernment.LeadershipFormDemocracy,
	}
}

func (e *Empire) findNaturalProgression() gengovernment.LeadershipForm {

	// TODO: Pick from the preferred leadership forms.
	var currentInfluence gengovernment.LeadershipInfluence
	var fallbackForm gengovernment.LeadershipForm
	switch {
	case e.Capital.Population < 1000:
		currentInfluence = gengovernment.LeadershipInfluenceCityState
		fallbackForm = gengovernment.LeadershipFormMonarchy
	default:
		currentInfluence = gengovernment.LeadershipInfluenceEmpire
		fallbackForm = gengovernment.LeadershipFormDictatorship
	}

	// Check if we need to change the leadership form.
	// There is a chance we just retain the current form.
	curMin, curMax := e.Leadership.Form.RangeInfluence()
	if curMin <= currentInfluence && curMax >= currentInfluence && rand.Intn(100) < 20 {
		return e.Leadership.Form
	}

	natProg := e.Leadership.Form.NaturalProgression()
	if len(natProg) == 0 {
		log.Printf("!!!%s has no natural progression for %s", e.String(), e.Leadership.Form)
		return fallbackForm
	}

	for _, i := range rand.Perm(len(natProg)) {
		frm := natProg[i]
		minInf, maxInf := frm.RangeInfluence()
		if minInf <= currentInfluence && maxInf >= currentInfluence {
			return frm
		}
	}
	return e.Leadership.Form
}

// findCoupProgression returns the possible progression of the leadership form through a coup.
func (e *Empire) findCoupProgression() gengovernment.LeadershipForm {
	// TODO: Pick from the preferred leadership forms.
	var currentInfluence gengovernment.LeadershipInfluence
	var fallbackForm gengovernment.LeadershipForm
	switch {
	case e.Capital.Population < 1000:
		currentInfluence = gengovernment.LeadershipInfluenceCityState
		fallbackForm = gengovernment.LeadershipFormMonarchy
	default:
		currentInfluence = gengovernment.LeadershipInfluenceEmpire
		fallbackForm = gengovernment.LeadershipFormDictatorship
	}

	// Check if we need to change the leadership form.
	// There is a chance we just retain the current form.
	curMin, curMax := e.Leadership.Form.RangeInfluence()
	if curMin <= currentInfluence && curMax >= currentInfluence && rand.Intn(100) < 20 {
		return e.Leadership.Form
	}

	natProg := e.Leadership.Form.CoupProgression()
	if len(natProg) == 0 {
		log.Printf("!!!%s has no natural progression for %s", e.String(), e.Leadership.Form)
		return fallbackForm
	}

	for _, i := range rand.Perm(len(natProg)) {
		frm := natProg[i]
		minInf, maxInf := frm.RangeInfluence()
		if minInf <= currentInfluence && maxInf >= currentInfluence {
			return frm
		}
	}
	return e.Leadership.Form
}
