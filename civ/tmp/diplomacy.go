package civ

import "log"

const (
	DiplomacyRelationshipNone = iota
	DiplomacyRelationshipNeutral
	DiplomacyRelationshipFriendly
	DiplomacyRelationshipHostile
)

func (m *Civ) tickDiplomacyCity(c *City) {
	logInfo := true
	// TODO: Implement this.
	// Get all nearby cities.
	// Evaluate the relationships with the nearby cities.
	// If the relationship has changed, update the relationship and change agreements accordingly.
	// Agreements include trade agreements, military alliances, non-aggression pacts, etc.

	// Get all nearby cities.
	nearbyCities := m.getNearbyCities(c.ID, 1000.0)

	// TODO: Get our historic values for each city.
	opinions := make(map[int]float64)
	for _, nc := range nearbyCities {
		// Get the opinion of the city.
		cityOpinion := c.compare(nc.city)

		// Get the personal opinion of the leadership.
		var leaderOpinion float64
		if c.Leadership != nil && nc.city.Leadership != nil {
			leaderOpinion = c.Leadership.compare(nc.city.Leadership)
		}
		opinions[nc.city.ID] = cityOpinion

		if logInfo {
			log.Printf("!!!DIPLOMACY Compat %s has an opinion of %.2f (%.2f leadership) towards %s", c.String(), cityOpinion, leaderOpinion, nc.city.String())
		}
	}
}

func (m *Civ) tickDiplomacyCityState(cs *CityState) {
	logInfo := true
	// Get all cities that are part of the city state.
	cities := cs.Cities
	opinionsCities := make(map[int]float64)
	for _, c := range cities {
		// Get the opinion of the city.
		cityOpinion := cs.compareToCity(c)

		// Get the personal opinion of the leadership.
		var leaderOpinion float64
		if cs.Leadership != nil && c.Leadership != nil {
			leaderOpinion = cs.Leadership.compare(c.Leadership)
		}
		opinionsCities[c.ID] = cityOpinion

		if logInfo {
			log.Printf("!(OWN)!!%s has an opinion of %.2f (%.2f leadership) towards %s", cs.String(), cityOpinion, leaderOpinion, c.String())
		}
	}

	// Get all nearby city states.
	var nearbyCityStates []*CityState

	// Get all nearby empires.
	var nearbyEmpires []*Empire
	for _, csn := range m.getCityStateNeighbors(cs) {
		// Check if there is an empire at the location.
		if emp := m.Empires.GetAt(csn.ID); emp != nil {
			nearbyEmpires = append(nearbyEmpires, emp)
		} else {
			nearbyCityStates = append(nearbyCityStates, csn)
		}
	}

	// Evaluate the relationships with the nearby city states.
	opinions := make(map[int]float64)
	for _, ncs := range nearbyCityStates {
		// Get the opinion of the city state.
		cityStateOpinion := cs.compare(ncs)

		// Get the personal opinion of the leadership.
		var leaderOpinion float64
		if cs.Leadership != nil && ncs.Leadership != nil {
			leaderOpinion = cs.Leadership.compare(ncs.Leadership)
		}
		opinions[ncs.ID] = cityStateOpinion

		if logInfo {
			log.Printf("!!!DIPLOMACY Compare%s has an opinion of %.2f (%.2f leadership) towards %s", cs.String(), cityStateOpinion, leaderOpinion, ncs.String())
		}
	}

	// Evaluate the relationships with the nearby empires.
	opinionsEmpires := make(map[int]float64)
	for _, ne := range nearbyEmpires {
		// Get the opinion of the empire.
		empireOpinion := cs.compareToEmpire(ne)

		// Get the personal opinion of the leadership.
		var leaderOpinion float64
		if cs.Leadership != nil && ne.Leadership != nil {
			leaderOpinion = cs.Leadership.compare(ne.Leadership)
		}
		opinionsEmpires[ne.ID] = empireOpinion

		if logInfo {
			log.Printf("!!!DIPLOMACY%s has an opinion of %.2f (%.2f leadership) towards %s", cs.String(), empireOpinion, leaderOpinion, ne.String())
		}
	}
}

func (m *Civ) tickDiplomacyEmpire(e *Empire) {
	logInfo := true
	// Get all cities that are part of the empire.
	cities := e.Cities
	opinionsCities := make(map[int]float64)
	for _, c := range cities {
		// Get the opinion of the city.
		cityOpinion := e.compareToCity(c)

		// Get the personal opinion of the leadership.
		var leaderOpinion float64
		if e.Leadership != nil && c.Leadership != nil {
			leaderOpinion = e.Leadership.compare(c.Leadership)
		}
		opinionsCities[c.ID] = cityOpinion

		if logInfo {
			log.Printf("!(OWN)!!%s has an opinion of %.2f (%.2f leadership) towards %s", e.String(), cityOpinion, leaderOpinion, c.String())
		}
	}

	// Get all city states that are part of the empire.
	cityStates := m.getEmpireCityStates(e)
	opinionsOwnCityStates := make(map[int]float64)
	for _, cs := range cityStates {
		// Get the opinion of the city state.
		cityStateOpinion := e.compareToCityState(cs)

		// Get the personal opinion of the leadership.
		var leaderOpinion float64
		if e.Leadership != nil && cs.Leadership != nil {
			leaderOpinion = e.Leadership.compare(cs.Leadership)
		}
		opinionsOwnCityStates[cs.ID] = cityStateOpinion

		if logInfo {
			log.Printf("!(OWN)!!%s has an opinion of %.2f (%.2f leadership) towards %s", e.String(), cityStateOpinion, leaderOpinion, cs.String())
		}
	}

	// Get all nearby empires.
	nearbyEmpires := m.getEmpireNeighbors(e)

	// Evaluate the relationships with the nearby empires.
	opinions := make(map[int]float64)
	for _, ne := range nearbyEmpires {
		// Get the opinion of the empire.
		empireOpinion := e.compare(ne)

		// Get the personal opinion of the leadership.
		var leaderOpinion float64
		if e.Leadership != nil && ne.Leadership != nil {
			leaderOpinion = e.Leadership.compare(ne.Leadership)
		}
		opinions[ne.ID] = empireOpinion

		if logInfo {
			log.Printf("!!!DIPLOMACY%s has an opinion of %.2f (%.2f leadership) towards %s", e.String(), empireOpinion, leaderOpinion, ne.String())
		}
	}

	// Get all nearby city states.
	nearbyCityStates := m.getEmpireCityStateNeighbors(e, true)

	opinionsCityStates := make(map[int]float64)
	for _, ncs := range nearbyCityStates {
		// Get the opinion of the city state.
		cityStateOpinion := e.compareToCityState(ncs)

		// Get the personal opinion of the leadership.
		var leaderOpinion float64
		if e.Leadership != nil && ncs.Leadership != nil {
			leaderOpinion = e.Leadership.compare(ncs.Leadership)
		}
		opinionsCityStates[ncs.ID] = cityStateOpinion

		if logInfo {
			log.Printf("!!!DIPLOMACY%s has an opinion of %.2f (%.2f leadership) towards %s", e.String(), cityStateOpinion, leaderOpinion, ncs.String())
		}
	}
}
