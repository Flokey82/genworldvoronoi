package genworldvoronoi

import (
	"log"
	"sort"
)

// HANDLE NOMADIC TRIBES HERE.
//
// Nomadic tribes will move around and settle in different regions, depending on their preferences
// and suitability of the regions (what population can be sustained in the region).
//
// If the most prosperous neighbor region cannot sustain the entire tribe, the tribe will split into
// two or more tribes.
func (s *simState) handleNomadicTribe(t *Tribe) {
	// We look at all neighboring regions and move to the most suitable region.
	// Considerations:
	// - We do not move into a region that is already occupied.
	// - We do not move into a region that is not suitable for the tribe.

	// Calculate the max population for each neighboring region.
	neigbors := s.m.R_circulate_r(rNbs, t.RegionID)
	maxPopPerRegion := make([]int, len(neigbors))
	neighborIndices := make([]int, len(neigbors))
	for i, nb := range neigbors {
		neighborIndices[i] = i
		// We only consider regions that are not ocean regions and are unoccupied.
		if s.m.Elevation[nb] > 0 && s.tribeAtRegion[nb] == nil {
			maxPopPerRegion[i] = s.calcMaxPopPerRegion(t, nb)
		}
	}

	// Sort the neighbor indices by max population.
	sort.Slice(neighborIndices, func(i, j int) bool {
		return maxPopPerRegion[neighborIndices[i]] > maxPopPerRegion[neighborIndices[j]]
	})

	// Start with the most suitable region and check if the entire tribe can move there.
	// If not, we need to split the tribe into two tribes and move the remaining tribe to the next suitable region.
	tCurrent := t
	for _, idx := range neighborIndices {
		nb := neigbors[idx]
		if s.tribeAtRegion[nb] != nil {
			continue // The region is already occupied.
		}

		// Get the max population for the region.
		nbMaxPop := maxPopPerRegion[idx]
		if nbMaxPop <= 0 {
			break // We ran out of suitable regions.
		}

		// Check if the entire tribe can move to the new region.
		if nbMaxPop >= tCurrent.Population {
			// Move the tribe here and be done.
			s.moveTribe(tCurrent, nb)
			// TODO: The satisfaction should depend on the prosperity of the region
			// copared to the current region, etc.
			tCurrent.changeSatisfaction(migrationSuccessSatisfaction)
			tCurrent = nil
			break
		}

		// The max population for the region is less than the population of the tribe.
		// Get the number of people that we leave behind (a minimum of 50 people, if the tribe is large enough).
		diff := min(max(50, tCurrent.Population-nbMaxPop), tCurrent.Population/2)
		log.Println("Tribe ", tCurrent.String(), "is splitting into two tribes. Population:", tCurrent.Population, "Max population:", nbMaxPop, "Diff:", diff)

		// Move the remaining (the original) tribe to the new region
		// and the new tribe to the original region.
		currentReg := tCurrent.RegionID
		s.moveTribe(tCurrent, nb)

		// Split the tribe into two tribes. The new tribe will be placed in the original region.
		newTribe := s.placeTribeAt(currentReg, diff, tCurrent, false)

		/*
			// Split the tribe into two tribes. The new tribe will be placed in the original region.
			newTribe := tCurrent.Split(s.m.getNextTribeID(), diff)

			// Move the remaining (the original) tribe to the new region
			// and the new tribe to the original region.
			currentReg := tCurrent.RegionID
			s.moveTribe(tCurrent, nb)
			s.moveTribe(newTribe, currentReg)

			s.newTribes = append(s.newTribes, tCurrent)
		*/

		// TODO: Make these constants.
		newTribe.changeSatisfaction(tribeSplitForcedSatisfaction)
		tCurrent.changeSatisfaction(tribeSplitForcedSatisfaction)

		// Set the new tribe as the current tribe.
		tCurrent = newTribe
	}

	// If there is still remaining population, check if they can survive in the current region.
	if tCurrent != nil {
		log.Println("Tribe ", tCurrent.String(), "is trying to survive in the current region. Population:", tCurrent.Population, "Max population:", t.currentRegionMaxPop)

		// Check if the tribe can survive in the current region (or at least part of it can survive).
		if maxPop := s.calcMaxPopPerRegion(t, tCurrent.RegionID); maxPop <= 0 {
			log.Println("Tribe ", tCurrent.String(), "has died out.")
			tCurrent.Population = 0

			// TODO: Optionally attack other tribes to move into their regions.
			if s.tribeAtRegion[tCurrent.RegionID] == tCurrent {
				s.tribeAtRegion[tCurrent.RegionID] = nil
			} else {
				// DEBUG: Check if the tribe is in the right region.
				log.Println("Tribe ", tCurrent.String(), "is in the wrong region?????!!!")
			}
		} else {
			// Check if the tribe lost some of its population.
			if maxPop < tCurrent.Population {
				t.changeSatisfaction(starvationSatisfaction)
				log.Println("Tribe ", tCurrent.String(), " lost some of its population.", tCurrent.Population-maxPop, "people died.", maxPop, "people survived.")
				tCurrent.Population = maxPop
			} else {
				t.changeSatisfaction(migrationSuccessSatisfaction)
			}

			// DEBUG: Check if the tribe is in the right region.
			if s.tribeAtRegion[tCurrent.RegionID] != tCurrent {
				log.Println("Tribe", tCurrent.String(), "is in the wrong region.")
			}

			// Re-assign the tribe to the region.
			s.moveTribe(tCurrent, tCurrent.RegionID)
		}
	}
}
