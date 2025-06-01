package civ

import (
	"log"
	"math/rand"

	"github.com/Flokey82/go_gens/gengovernment"
)

func (m *Civ) handleLeadershipCityState(c *CityState) {
	// Update the factions
	// This will trigger election of new leaders, etc.
	c.Leadership.TickCityState(c, m)
	for _, f := range c.Factions {
		f.TickCityState(c, m)
	}
	c.GoverningPeople.handleLeadership(c, m)
}

func (f *Faction) TickCityState(c *CityState, m *Civ) {
	f.Tick(c, m)

	if rand.Float64() < 0.1 {
		action := f.pickAction(c, c.GoverningPeople, m)
		if action != nil {
			action.Execute(c, m, f)
		}
	}
}

func (c *CityState) getFactionActions(m *Civ, f *Faction) []*FactionAction {
	log.Printf("!!!%s is considering actions.", c.String())
	return defaultFactionActions
}

// getPreferredLeadershipForm returns the preferred leadership form of the city state (capital).
func (c *CityState) getPreferredLeadershipForm() gengovernment.LeadershipForm {
	if len(c.Capital.People) < 100 {
		return gengovernment.LeadershipFormChiefdom
	}
	return gengovernment.LeadershipFormCouncil
}

// getPossibleLeadershipForms returns the possible leadership forms of the city state (capital).
func (c *CityState) getPossibleLeadershipForms() []gengovernment.LeadershipForm {
	return []gengovernment.LeadershipForm{gengovernment.LeadershipFormChiefdom, gengovernment.LeadershipFormCouncil}
}

func (c *CityState) findNaturalProgression() gengovernment.LeadershipForm {
	// TODO: Pick from the preferred leadership forms.
	var currentInfluence gengovernment.LeadershipInfluence
	var fallbackForm gengovernment.LeadershipForm
	switch {
	case c.Capital.Population < 1000:
		currentInfluence = gengovernment.LeadershipInfluenceSettlement
		fallbackForm = gengovernment.LeadershipFormChiefdom
	default:
		currentInfluence = gengovernment.LeadershipInfluenceCityState
		fallbackForm = gengovernment.LeadershipFormCouncil
	}

	// Check if we need to change the leadership form.
	// There is a chance we just retain the current form.
	curMin, curMax := c.Leadership.Form.RangeInfluence()
	if curMin <= currentInfluence && curMax >= currentInfluence && rand.Intn(100) < 20 {
		return c.Leadership.Form
	}

	natProg := c.Leadership.Form.NaturalProgression()
	if len(natProg) == 0 {
		log.Printf("!!!%s has no natural progression for %s", c.String(), c.Leadership.Form)
		return fallbackForm
	}

	for _, i := range rand.Perm(len(natProg)) {
		frm := natProg[i]
		minInf, maxInf := frm.RangeInfluence()
		if minInf <= currentInfluence && maxInf >= currentInfluence {
			return frm
		}
	}
	return c.Leadership.Form
}

// findCoupProgression returns the possible progression of the leadership form through a coup.
func (c *CityState) findCoupProgression() gengovernment.LeadershipForm {
	// TODO: Pick from the preferred leadership forms.
	var currentInfluence gengovernment.LeadershipInfluence
	var fallbackForm gengovernment.LeadershipForm
	switch {
	case c.Capital.Population < 1000:
		currentInfluence = gengovernment.LeadershipInfluenceSettlement
		fallbackForm = gengovernment.LeadershipFormChiefdom
	default:
		currentInfluence = gengovernment.LeadershipInfluenceCityState
		fallbackForm = gengovernment.LeadershipFormCouncil
	}

	// Check if we need to change the leadership form.
	// There is a chance we just retain the current form.
	curMin, curMax := c.Leadership.Form.RangeInfluence()
	if curMin <= currentInfluence && curMax >= currentInfluence && rand.Intn(100) < 20 {
		return c.Leadership.Form
	}

	natProg := c.Leadership.Form.CoupProgression()
	if len(natProg) == 0 {
		log.Printf("!!!%s has no natural progression for %s", c.String(), c.Leadership.Form)
		return fallbackForm
	}

	for _, i := range rand.Perm(len(natProg)) {
		frm := natProg[i]
		minInf, maxInf := frm.RangeInfluence()
		if minInf <= currentInfluence && maxInf >= currentInfluence {
			return frm
		}
	}
	return c.Leadership.Form
}
