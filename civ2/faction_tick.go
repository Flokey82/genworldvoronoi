package civ2

import (
	"fmt"
	"math/rand"
	"sort"
)

func (m *Civ) tickLeadership(t peopleThing, nDays int) {
	g := t.GetGoverningPeople()
	if g == nil {
		return
	}

	// Tick the main leadership.
	if g.Leadership != nil {
		m.tickFaction(t, g.Leadership, nDays)
	}

	// Tick the secondary factions.
	for _, f := range g.Factions {
		m.tickFaction(t, f, nDays)
	}

	// Handle leadership changes, coups, etc.
	m.handleGoverningBody(t, nDays)
}

func (m *Civ) tickFaction(t peopleThing, f *Faction, nDays int) {
	// Check if the leader(s) are still alive.
	for _, leader := range f.Leaders() {
		if leader.Dead() {
			m.History.AddEvent(HistoryEventLeadership, fmt.Sprintf("%s %s of faction %s died.", f.GetTitleForPerson(leader), leader.Name(), f.Name), f.Ref())
			f.ChooseNewLeader(t, f, m)
		}
	}

	// Maybe perform an action.
	if rand.Intn(100) < 5 {
		if action := m.pickFactionAction(t, f); action != nil {
			action.Execute(t, m, f)
		}
	}

	// Update faction popularity based on events or random drift.
	f.Popularity.Add((rand.Float64() - 0.5) * 0.01)
}

func (m *Civ) pickFactionAction(t peopleThing, f *Faction) *FactionAction {
	actions := t.getFactionActions(m, f)
	var filtered []*FactionAction
	for _, a := range actions {
		if (a.requires == nil || a.requires(f)) && rand.Float64() < a.probability(f) {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered[rand.Intn(len(filtered))]
}

func (m *Civ) handleGoverningBody(t peopleThing, nDays int) {
	g := t.GetGoverningPeople()
	if g == nil {
		return
	}

	// If satisfaction is low, a new faction might form.
	if g.Satisfaction < 0.7 && rand.Float64() < 0.01 {
		m.formNewFaction(t)
	}

	// If a faction is much more popular than the current leadership, it might take over.
	if len(g.Factions) > 0 {
		sort.Slice(g.Factions, func(i, j int) bool {
			return g.Factions[i].Popularity > g.Factions[j].Popularity
		})
		topFaction := g.Factions[0]
		if topFaction.Popularity > g.Leadership.Popularity+0.3 && rand.Float64() < 0.05 {
			m.executeTakeover(t, topFaction)
		}
	}
}

func (m *Civ) formNewFaction(t peopleThing) {
	g := t.GetGoverningPeople()
	form := t.findCoupProgression()

	// Find the most different person to lead the new faction.
	origLeader := g.Leadership.Leader
	var leader *Person
	if origLeader != nil {
		mostDifferent := 1.0
		for _, p := range t.GetPeople() {
			if p.Dead() || p.Title != "" || p.Age < ageOfAdulthood {
				continue
			}
			similarity := p.compare(origLeader)
			if similarity < mostDifferent {
				mostDifferent = similarity
				leader = p
			}
		}
	}

	f := m.genFaction(t, form, leader)
	f.Popularity = 0.5
	g.Factions = append(g.Factions, f)

	m.History.AddEvent(HistoryEventFounding, fmt.Sprintf("A new faction %s has formed in %s.", f.Name, t.String()), t.Ref())
}

func (m *Civ) executeTakeover(t peopleThing, f *Faction) {
	g := t.GetGoverningPeople()
	oldLeadership := g.Leadership

	// The lower the popularity of the old leadership, the higher the chance of a coup.
	if rand.Float64() > float64(oldLeadership.Popularity) {
		m.executeCoup(t, f)
	} else {
		m.executeTransition(t, f)
	}
}

func (m *Civ) executeCoup(t peopleThing, newLeadership *Faction) {
	g := t.GetGoverningPeople()
	oldLeadership := g.Leadership
	newForm := t.findCoupProgression()

	// Swap leadership.
	g.Leadership = newLeadership
	newLeadership.ChangeType(FactionTypeCivil, newForm, m.History)

	// Execute or exile old leaders.
	for _, l := range oldLeadership.Leaders() {
		newLeadership.ExecutePerson(l, "political purge", m)
	}

	m.History.AddEvent(HistoryEventUprising, fmt.Sprintf("A coup by %s has overthrown the leadership of %s!", newLeadership.Name, t.String()), t.Ref())
}

func (m *Civ) executeTransition(t peopleThing, newLeadership *Faction) {
	g := t.GetGoverningPeople()
	oldLeadership := g.Leadership
	newForm := t.findNaturalProgression()

	// Swap leadership.
	g.Leadership = newLeadership
	newLeadership.ChangeType(FactionTypeCivil, newForm, m.History)

	// Old leadership remains as a faction if it wasn't already.
	found := false
	for _, f := range g.Factions {
		if f == oldLeadership {
			found = true
			break
		}
	}
	if !found && oldLeadership != nil {
		g.Factions = append(g.Factions, oldLeadership)
	}

	m.History.AddEvent(HistoryEventElection, fmt.Sprintf("A peaceful transition of power has occurred in %s. %s is now in control.", t.String(), newLeadership.Name), t.Ref())
}

