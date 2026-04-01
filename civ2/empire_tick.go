package civ2

import (
	"log"
)

func (m *Civ) tickEmpires(nDays int) {
	for _, e := range m.Empires.Objects {
		// Empires might collapse if they lose on influence.
		if e.Capital == nil || e.Capital.Population == 0 {
			continue
		}

		log.Printf("Empire %d: %s", e.ID, e.Name)
		
		// Centralized economic tick.
		m.tickEconomyBase(e, nDays)

		// Tick diplomatic relations.
		m.tickDiplomacyBase(e, nDays)

		// Tick leadership.
		m.tickLeadership(e, nDays)

		// We might expand or contract our influence.
		// Look at surrounding settlements and cities that would be willing to join us.
	}
}
