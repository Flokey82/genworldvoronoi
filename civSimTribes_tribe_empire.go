package genworldvoronoi

import "log"

func (s *simState) handleEmpire(t *Tribe) {
	log.Printf("!!!%s has become an empire.", t.String())
	// Find neighboring empires.
	for _, nbEmpireID := range s.m.getEmpireNeighbors(t.Empire) {
		if nbEmpire := s.m.GetEmpire(nbEmpireID); nbEmpire != nil {
			log.Printf("!!!%s has a neighboring empire: %s (score %.2f)", t.String(), nbEmpire.String(), t.Empire.compare(nbEmpire))
		} else {
			log.Printf("!!!%s has a neighboring empire with ID %d and it could not be found", t.String(), nbEmpireID)
		}
	}

	// Find all neighboring city states that are not part of another empire.
	for _, nbState := range s.m.getEmpireCityStateNeighbors(t.Empire) {
		if s.m.getCityStateEmpire(nbState) == -1 {
			log.Printf("!!!%s has a neighboring city state: %s.", t.String(), nbState.String())
		}
	}
}
