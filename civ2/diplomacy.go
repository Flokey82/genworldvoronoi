package civ2

import (
	"fmt"
	"log"

	"github.com/Flokey82/genworldvoronoi/civ"
)

const (
	DiplomacyRelationshipNone = iota
	DiplomacyRelationshipNeutral
	DiplomacyRelationshipFriendly
	DiplomacyRelationshipHostile
	DiplomacyRelationshipWar
)

func (m *Civ) SetRelationship(a, b civ.ObjectReference, rel int) {
	if m.Relationships[a] == nil {
		m.Relationships[a] = make(map[civ.ObjectReference]int)
	}
	m.Relationships[a][b] = rel
	// Mutual for now.
	if m.Relationships[b] == nil {
		m.Relationships[b] = make(map[civ.ObjectReference]int)
	}
	m.Relationships[b][a] = rel
}

func (m *Civ) GetRelationship(a, b civ.ObjectReference) int {
	if m.Relationships[a] == nil {
		return DiplomacyRelationshipNone
	}
	return m.Relationships[a][b]
}

func (m *Civ) evaluateWarDeclaration(a, b peopleThing, opinion float64) {
	rel := m.GetRelationship(a.Ref(), b.Ref())
	if rel == DiplomacyRelationshipWar {
		// Already at war! Trigger combat.
		m.resolveCombat(a, b)
		return
	}

	mil := a.GetMilitary()
	if mil == nil {
		return
	}

	// Declare war if opinion is very low and aggressiveness is high.
	if opinion < -0.8 && mil.Aggressiveness > 0.5 {
		m.SetRelationship(a.Ref(), b.Ref(), DiplomacyRelationshipWar)
		m.History.AddEvent("war", fmt.Sprintf("%s has declared war on %s!", a.String(), b.String()), a.Ref())
	}
}

func (m *Civ) tickDiplomacyCity(c *City) {
	logInfo := false
	nearbyCities := m.getCitiesWithin(c.ID, 1000.0)

	for _, nc := range nearbyCities {
		if nc.ID == c.ID {
			continue
		}
		// Get the opinion of the city.
		cityOpinion := c.compare(m, nc)

		// Get the personal opinion of the leadership.
		var leaderOpinion float64
		if c.Leadership != nil && nc.Leadership != nil && c.Leadership.Leader != nil && nc.Leadership.Leader != nil {
			leaderOpinion = c.Leadership.Leader.compare(nc.Leadership.Leader)
		}

		m.evaluateWarDeclaration(c, nc, (cityOpinion+leaderOpinion)/2)

		if logInfo && (cityOpinion != 0 || leaderOpinion != 0) {
			log.Printf("DIPLOMACY: %s has an opinion of %.2f (%.2f leadership) towards %s", c.String(), cityOpinion, leaderOpinion, nc.String())
		}
	}
}

func (m *Civ) tickDiplomacyCityState(cs *CityState) {
	logInfo := false
	// Evaluate own cities.
	for _, r := range cs.Regions {
		c := m.Cities.GetAt(r)
		if c == nil {
			continue
		}
		cityOpinion := cs.compareToCity(m, c)
		var leaderOpinion float64
		if cs.Leadership != nil && c.Leadership != nil && cs.Leadership.Leader != nil && c.Leadership.Leader != nil {
			leaderOpinion = cs.Leadership.Leader.compare(c.Leadership.Leader)
		}
		if logInfo && (cityOpinion != 0 || leaderOpinion != 0) {
			log.Printf("DIPLOMACY (OWN): %s has an opinion of %.2f (%.2f leadership) towards %s", cs.String(), cityOpinion, leaderOpinion, c.String())
		}
	}

	// Neighbor City States.
	for _, ncs := range m.getCityStateNeighbors(cs) {
		csOpinion := cs.compare(m, ncs)
		var leaderOpinion float64
		if cs.Leadership != nil && ncs.Leadership != nil && cs.Leadership.Leader != nil && ncs.Leadership.Leader != nil {
			leaderOpinion = cs.Leadership.Leader.compare(ncs.Leadership.Leader)
		}

		m.evaluateWarDeclaration(cs, ncs, (csOpinion+leaderOpinion)/2)

		if logInfo {
			log.Printf("DIPLOMACY (Neighbor CS): %s has an opinion of %.2f (%.2f leadership) towards %s", cs.String(), csOpinion, leaderOpinion, ncs.String())
		}
	}

	// Neighbor Empires.
	// For now, we'll just check neighboring regions in Empires map.
	for _, nbID := range m.getTerritoryNeighbors(cs.ID, cs.Regions, m.Empires.Regions) {
		ne := m.GetEmpire(nbID)
		if ne == nil {
			continue
		}
		empOpinion := cs.compareToEmpire(m, ne)
		var leaderOpinion float64
		if cs.Leadership != nil && ne.Leadership != nil && cs.Leadership.Leader != nil && ne.Leadership.Leader != nil {
			leaderOpinion = cs.Leadership.Leader.compare(ne.Leadership.Leader)
		}

		m.evaluateWarDeclaration(cs, ne, (empOpinion+leaderOpinion)/2)

		if logInfo {
			log.Printf("DIPLOMACY (Neighbor Empire): %s has an opinion of %.2f (%.2f leadership) towards %s", cs.String(), empOpinion, leaderOpinion, ne.String())
		}
	}
}

func (m *Civ) tickDiplomacyEmpire(e *Empire) {
	logInfo := false
	// Evaluate own cities.
	for _, r := range e.Regions {
		c := m.Cities.GetAt(r)
		if c == nil {
			continue
		}
		cityOpinion := e.compareToCity(m, c)
		var leaderOpinion float64
		if e.Leadership != nil && c.Leadership != nil && e.Leadership.Leader != nil && c.Leadership.Leader != nil {
			leaderOpinion = e.Leadership.Leader.compare(c.Leadership.Leader)
		}
		if logInfo {
			log.Printf("DIPLOMACY (OWN City): %s has an opinion of %.2f (%.2f leadership) towards %s", e.String(), cityOpinion, leaderOpinion, c.String())
		}
	}

	// Own City States.
	for _, cs := range m.getEmpireCityStates(e) {
		csOpinion := e.compareToCityState(m, cs)
		var leaderOpinion float64
		if e.Leadership != nil && cs.Leadership != nil && e.Leadership.Leader != nil && cs.Leadership.Leader != nil {
			leaderOpinion = e.Leadership.Leader.compare(cs.Leadership.Leader)
		}
		if logInfo {
			log.Printf("DIPLOMACY (OWN CS): %s has an opinion of %.2f (%.2f leadership) towards %s", e.String(), csOpinion, leaderOpinion, cs.String())
		}
	}

	// Neighbor Empires.
	for _, ne := range m.getEmpireNeighbors(e) {
		empOpinion := e.compare(m, ne)
		var leaderOpinion float64
		if e.Leadership != nil && ne.Leadership != nil && e.Leadership.Leader != nil && ne.Leadership.Leader != nil {
			leaderOpinion = e.Leadership.Leader.compare(ne.Leadership.Leader)
		}

		m.evaluateWarDeclaration(e, ne, (empOpinion+leaderOpinion)/2)

		if logInfo {
			log.Printf("DIPLOMACY (Neighbor Empire): %s has an opinion of %.2f (%.2f leadership) towards %s", e.String(), empOpinion, leaderOpinion, ne.String())
		}
	}

	// Neighboring City States.
	for _, ncs := range m.getEmpireCityStateNeighbors(e, true) {
		csOpinion := e.compareToCityState(m, ncs)
		var leaderOpinion float64
		if e.Leadership != nil && ncs.Leadership != nil && e.Leadership.Leader != nil && ncs.Leadership.Leader != nil {
			leaderOpinion = e.Leadership.Leader.compare(ncs.Leadership.Leader)
		}

		m.evaluateWarDeclaration(e, ncs, (csOpinion+leaderOpinion)/2)

		if logInfo {
			log.Printf("DIPLOMACY (Neighbor CS): %s has an opinion of %.2f (%.2f leadership) towards %s", e.String(), csOpinion, leaderOpinion, ncs.String())
		}
	}
}
