package civ2

import (
	"log"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/civ"
)

// Convert the distance between two regions to kilometers.
const unitDistToKm = 6371.0 // km

func (m *Civ) LogTribes() {
	for _, t := range m.Tribes.Objects {
		log.Printf("Tribe %d (%s): %d people, region %d", t.ID, t.Culture.Type.String(), t.Population, t.RegionID)
	}

	for _, s := range m.Settlements.Objects {
		log.Printf("Settlement %d (%s): %d people, region %d", s.ID, s.Culture.Type, s.Population, s.ID)
	}

	for _, c := range m.Cities.Objects {
		log.Printf("City %d (%s): %d people, region %d", c.ID, c.Culture.Type, c.Population, c.ID)
	}
}

// TickTribes ticks the tribes in the simulation.
// NOTE: This function is currently implemented to tick 365 days.
//
// Executed steps:
// At the beginning of the year, we decide where to migrate after
// the year has passed. We will try to migrate to the region with
// the highest fitness score (arable land - exhaustion).
func (m *Civ) tickTribes(nDays int) {

	// First sort the tribes by population size.
	m.Tribes.Sort(func(a, b *Tribe) bool {
		return a.Population > b.Population
	})
	log.Printf("Tribes: %d", len(m.Tribes.Objects))

	// Now the year should commence.
	// For each tribe, handle growth, economy, and diplomacy.
	for _, t := range m.Tribes.Objects {
		if t.Population <= 0 {
			continue
		}
		// Grow the population.
		t.Grow(nDays)

		// Centralized economic tick.
		m.tickEconomyBase(t, nDays)

		// Tick diplomatic relations.
		m.tickDiplomacyBase(t, nDays)

		// Tick leadership.
		m.tickLeadership(t, nDays)
	}

	// Update the resource exhaustion for each region that
	// the tribes have caused.
	m.updateResourceExhaustion()

	// Migrate the tribes to a new region.
	m.migrateTribes(nDays)

	// Cleanup tribes with a population of 0.
	m.cleanupTribes()
}

func (m *Civ) calcMaxPopPerRegion(r int) int {
	// Get the area of the region (which is the area on a unit sphere).
	regArea := m.GetRegArea(r, nil) * unitDistToKm * unitDistToKm
	// Calculate the maximum population for each region.
	// NOTE: Survival is normally not dependent on arable land!
	return int(regArea * m.arableLandFunc(r) * 10) // 10 people per km^2
}

// cleanupTribes removes tribes with a population of 0 and
// sorts them by descending population size.
func (m *Civ) cleanupTribes() {
	for _, t := range m.Tribes.Objects {
		if t.Population <= 0 {
			m.Tribes.RemoveObject(t)
			m.Cultures.SetIDAt(t.RegionID, -1)
		} else {
			m.Cultures.SetIDAt(t.RegionID, t.Culture.ID)
		}
	}
}


// Update the resource exhaustion for each region.
func (m *Civ) updateResourceExhaustion() {
	calcMaxPopPerRegion := func(r int) int {
		regArea := m.GetRegArea(r, nil) * unitDistToKm * unitDistToKm
		return int(regArea * m.arableLandFunc(r) * 10) // 10 people per km^2
	}

	for r := range m.SoilExhaustion {
		if tar := m.Tribes.GetAt(r); tar != nil {
			maxPop := float64(calcMaxPopPerRegion(r))
			m.SoilExhaustion[r] += float64(tar.Population) / maxPop
			if m.SoilExhaustion[r] > 1.0 {
				m.SoilExhaustion[r] = 1.0
			}
		} else if m.SoilExhaustion[r] > 0 {
			m.SoilExhaustion[r] *= 0.5
			if m.SoilExhaustion[r] < 0.05 {
				m.SoilExhaustion[r] = 0.0
			}
		}
	}
}

// checkIfSplit returns true if the tribe should split.
func (m *Civ) checkIfSplit(t *Tribe, nDays int) bool {
	return t.Population > 150 && rand.Intn(1000)*nDays < t.Population
}

// checkIfSettle returns true if the tribe should settle.
func (m *Civ) checkIfSettle(t *Tribe, nDays int) bool {
	return t.Settling || t.Population > 200 && rand.Intn(100)*nDays < 10
}

func (m *Civ) completeSettle(t *Tribe) {
	// Settle.
	// Check if there is a city or a settlement in the region.
	if s := m.Settlements.GetAt(t.RegionID); s != nil {
		// There is already a settlement in the region.
		// We will merge the tribe with the settlement.
		// If the settlement is abandoned, we will rename it.
		if s.Population == 0 {
			s.Name = t.Culture.Language.MakeCityName() + " (resettled)"
			s.Culture = t.Culture
		}
		s.Population += t.Population
		t.Population = 0
		log.Printf("tribe %d has settled in region %d", t.ID, t.RegionID)

		// Remove the tribe.
		m.Tribes.RemoveObject(t)
	} else if c := m.GetCity(t.RegionID); c != nil {
		// There is already a city in the region.
		// We will merge the tribe with the city.
		// If the city is abandoned, we will rename it.
		if c.Population == 0 {
			c.Name = t.Culture.Language.MakeCityName() + " (resettled)"
			c.Culture = t.Culture
		}
		c.Population += t.Population
		t.Population = 0
		log.Printf("tribe %d has settled in region %d", t.ID, t.RegionID)

		// Remove the tribe.
		m.Tribes.RemoveObject(t)
	} else {
		settlement := t.ToSettlement(t.Population)
		settlement.GoverningPeople = newGoverningPeople()
		m.Settlements.PlaceObjectAt(settlement, t.RegionID)
		log.Printf("tribe %d has settled in region %d", t.ID, t.RegionID)

		// Remove the tribe.
		m.Tribes.RemoveObject(t)
	}
}

func (m *Civ) hasPopulatedSettlement(r int) bool {
	return m.Settlements.GetAt(r) != nil && m.Settlements.GetAt(r).Population > 0
}

func (m *Civ) hasPopulatedCity(r int) bool {
	return m.GetCity(r) != nil && m.GetCity(r).Population > 0
}

func (m *Civ) findSettleRegion(t *Tribe, navCache *civ.NavCache) bool {
	// Find a suitable region within a certain radius.
	// We'll do it brute force for now.
	maxRadius := rand.Float64() * 2000.0 / unitDistToKm // km

	// Get the best region to settle in.
	bestRegion, ok := m.findSettleRegionFor(t.RegionID, t.Population, maxRadius, optCultureType(t.Culture.Type), optNearbyCities, optAbandonedSettlements)
	if !ok {
		return false
	}

	// Plan a path.
	if m.navigateMultiStep(t, bestRegion, m.navCache) {
		t.Settling = true
		return true
	}
	return false
}

type findSettleOpt func(m *Civ, r int) (float64, bool)

func optCultureType(cultureType civ.CultureType) findSettleOpt {
	return func(m *Civ, r int) (float64, bool) {
		if m.cultureFunc(r) != cultureType {
			return -1.0, false
		}
		return 0, true
	}
}

func optNearbyCities(m *Civ, r int) (float64, bool) {
	var score float64
	for _, nb := range m.R_circulate_r(nil, r) {
		if m.Cities.GetIDAt(nb) != -1 {
			score = +0.1
		}
	}
	return score, true
}

func optAbandonedSettlements(m *Civ, r int) (float64, bool) {
	if s := m.Settlements.GetAt(r); s != nil && s.Population == 0 {
		return 0.1, true
	}
	if c := m.GetCity(r); c != nil && c.Population == 0 {
		return 0.1, true
	}
	return 0, true
}

// TODO: Introduce options that can be passed to the findSettleRegionFor function.
func (m *Civ) findSettleRegionFor(srcReg, settleSize int, maxRadius float64, opts ...findSettleOpt) (int, bool) {
	scores := m.CalcCityScore(func(r int) float64 {
		// Check distance and avoid other tribes, settlements, cities.
		if dist := m.GetDistance(srcReg, r); dist > maxRadius ||
			m.Tribes.GetIDAt(r) != -1 ||
			m.hasPopulatedSettlement(r) || m.hasPopulatedCity(r) {
			return -1.0
		}

		// Check the options.
		multiplier := 1.0
		for _, opt := range opts {
			score, ok := opt(m, r)
			if !ok {
				return -1.0
			}
			multiplier += score
		}

		maxPop := float64(m.calcMaxPopPerRegion(r)) * (1 - m.SoilExhaustion[r])
		if maxPop < float64(settleSize) {
			return -1.0
		}
		return maxPop * multiplier * float64(m.ResourceLocations.SumValueOfRegion(r))
	}, func() []int { return []int{srcReg} })

	// Find the region with the highest fitness score.
	bestScore := 0.0
	bestRegion := -1
	for r, s := range scores {
		if s > bestScore {
			bestScore = s
			bestRegion = r
		}
	}
	return bestRegion, bestRegion != -1
}

func (m *Civ) migrateTribes(nDays int) {
	// Check random tribe splits.
	// TODO: Move this out.
	for _, t := range m.Tribes.Objects {
		if t.Population <= 0 {
			continue
		}

		// Check if we randomly split the tribe.
		if m.checkIfSplit(t, nDays) {
			// Get an empty neighbor region.
			neighbors := m.R_circulate_r(nil, t.RegionID)
			emptyNeighbors := make([]int, 0, len(neighbors))
			for _, nb := range neighbors {
				if m.Tribes.GetIDAt(nb) == -1 {
					emptyNeighbors = append(emptyNeighbors, nb)
				}
			}
			if len(emptyNeighbors) == 0 {
				// No empty neighbors, so we have to kill the tribe.
				log.Printf("tribe %d has been killed as there are no empty neighbors", t.ID)
				continue
			}
			// Random split.
			newTr := t.Split(m, t.Population/2)
			// Move the new tribe to the empty neighbor region.
			m.Tribes.PlaceObjectAt(newTr, emptyNeighbors[rand.Intn(len(emptyNeighbors))])
			log.Printf("tribe %d has split into two tribes", t.ID)
		}
	}

	// - First, for every tribe, plan where they want to go.
	// - If there are no suitable regions, we need to split the tribe.
	// - Move every tribe where there is no conflict.
	// - Then deconflict the regions and move the remaining tribes.
	// - Loop until there are no conflicts.
	var toRelocate []*Tribe
	for _, t := range m.Tribes.Objects {
		if t.Population <= 0 {
			continue
		}
		// Check if we decide to settle (if we aren't already), and if we can find a suitable region.
		if !t.Settling && m.checkIfSettle(t, nDays) && !m.findSettleRegion(t, m.navCache) {
			log.Printf("tribe %d could not find a suitable region to settle in", t.ID)
		}
		toRelocate = append(toRelocate, t)
	}

	// Keep track of final locations for each tribe that can move without conflict.
	finalLocations := make([]*Tribe, m.Geo.NumRegions)

	var currentTribe *Tribe

	// Create a navigation cache for navigating around other tribes.
	// This will be used to find alternative routes if the direct path is blocked.
	navCache := civ.NewNavCache(m.Geo, func(from, to *civ.NavTile, cost float64) (float64, bool) {
		if dstT := finalLocations[to.ID]; dstT != nil && dstT != currentTribe {
			return 0, false
		}
		return cost, true
	})

	for len(toRelocate) > 0 {
		regionToTribes := make(map[int][]*Tribe)

		// Loop over all tribes that still need to be relocated / placed on the next turn
		// and take note where they would like to go on the next turn.
		var nextRelocate []*Tribe
		for _, t := range toRelocate {
			var nextRegion int

			if t.Settling {
				// If we are settling, we check if the next region in the path is available
				// for us to move to.
				nextRegion = t.Path.Peek()

				// Check if the region is already settled or will be occupied by another tribe
				// on the end of the turn.
				// - If the settlement or city is abandoned, we should be able to move there anyway.
				// - If the tribe is aggressive, we should be able to fight for the region.
				if m.hasPopulatedSettlement(nextRegion) || m.hasPopulatedCity(nextRegion) ||
					finalLocations[nextRegion] != nil {
					if t.Path.PeekDone() {
						// This is our destination, but we can't settle here,
						// so we have to find a new region to settle in.
						if m.findSettleRegion(t, m.navCache) {
							nextRegion = t.Path.Peek()
						} else {
							log.Printf("tribe %d could not find a suitable region to settle in", t.ID)
						}
					} else {
						// This transit region is already occupied, so we have to find a new path.
						currentTribe = t // This is a hack and should be fixed.
						if m.navigateMultiStep(t, t.Path.To, navCache) {
							nextRegion = t.Path.Peek()
						}
					}
				}
			} else {
				// Still nomadic, so we look at direct neighbors for a suitable region.
				nextRegion, _ = m.FindBestNeighbor(t.RegionID, func(r int) float64 {
					if finalLocations[r] != nil {
						return -1.0
					}
					multiplier := 1.0
					// Check if the region would accomodate the tribe culturally.
					if m.cultureFunc(r) != t.Culture.Type {
						multiplier -= 0.1
					}
					return float64(m.calcMaxPopPerRegion(r)) * (1 - m.SoilExhaustion[r]) * multiplier
				})
				if nextRegion == -1 {
					// No suitable region found, so we have to split the tribe.
					// TODO: split the tribe. For now, we just stay in the same region.
					nextRegion = t.RegionID
					log.Printf("tribe %d has no suitable region to move to", t.ID)
				}
			}
			regionToTribes[nextRegion] = append(regionToTribes[nextRegion], t)
		}

		// Move all tribes that can move without conflict.
		for r, tribes := range regionToTribes {
			if len(tribes) == 1 {
				finalLocations[r] = tribes[0]
				continue
			}

			// Now we have to deconflict the regions.
			// We loop through all the tribes and non-aggressive tribes will
			// try to move to another region. If there are two or more aggressive
			// tribes attempting to move into the same region, we will have to
			// resolve the conflict.
			aggressiveTribes := make([]*Tribe, 0, len(tribes))
			otherTribes := make([]*Tribe, 0, len(tribes))
			for _, t := range tribes {
				if t.Aggressive {
					aggressiveTribes = append(aggressiveTribes, t)
				} else {
					otherTribes = append(otherTribes, t)
				}
			}

			if len(aggressiveTribes) == 1 {
				// If there is one aggressive tribe, it will move to the region
				// and all other tribes will be added to the list of tribes to relocate.
				finalLocations[r] = aggressiveTribes[0]
				nextRelocate = append(nextRelocate, otherTribes...)
			} else if len(aggressiveTribes) > 1 {
				// We let the aggressive tribes fight it out.
				var winner *Tribe
				for _, at := range aggressiveTribes {
					if winner == nil {
						winner = at
					} else if m.resolveCombat(at, winner) {
						// The winner becomes the new winner, the previous winner
						// will be relocated.
						nextRelocate = append(nextRelocate, winner)
						winner = at
					} else {
						// The loser will be relocated.
						nextRelocate = append(nextRelocate, at)
					}
				}
				// The last winner will move to the region and the other tribes will
				// be added to the list of tribes to relocate.
				finalLocations[r] = winner
				nextRelocate = append(nextRelocate, otherTribes...)
			} else {
				// If there are no aggressive tribes, one can move to the region.
				// We use the first one, since it is the largest tribe.
				// The other tribes will be added to the list of tribes to relocate.
				finalLocations[r] = otherTribes[0]
				nextRelocate = append(nextRelocate, otherTribes[1:]...)
			}
		}

		if false {
			// TODO: Do a pass to check if the destination regions are suitable for
			// the tribes. If not, we have to split the tribe and add the new tribes
			// to the list of tribes to relocate.
			for r, t := range finalLocations {
				if t == nil {
					continue
				}

				// Check if the new region can sustain the tribe.
				// If not, we have to split the tribe.
				if maxPop := m.calcMaxPopPerRegion(r); maxPop < t.Population {
					// Split the tribe!
					diff := t.Population - maxPop
					newTribe := t.Split(m, diff)
					nextRelocate = append(nextRelocate, newTribe)
					log.Printf("tribe %d has split into two tribes. Population: %d, Max population: %d, Diff: %d", t.ID, t.Population, maxPop, diff)
					// Add the tribe to the object map? If we don't do this, we will run into a nil pointer.
				}
			}
		}

		// Continue with the tribes that need to be relocated.
		toRelocate = nextRelocate
	}

	// Now place the tribes in their final locations.
	for r, t := range finalLocations {
		if t == nil {
			continue
		}

		// Log the culture of the tribe.
		log.Printf("Tribe %d: %s", t.ID, t.Culture.Type)

		// Check if the new region can sustain the tribe.
		// If not, we have to split the tribe. For now we kill part of the population.
		if maxPop := float64(m.calcMaxPopPerRegion(r)); maxPop < float64(t.Population) {
			deaths := t.Population - int(maxPop)
			t.Population -= deaths
			log.Printf("tribe %d has suffered from famine, losing %d people (%d remaining)", t.ID, deaths, t.Population)
		}

		// Move the culture to the new region.
		m.Cultures.SetIDAt(r, t.Culture.ID)
		m.Cultures.SetIDAt(t.RegionID, -1)

		// Move the tribe to the new region.
		m.Tribes.PlaceObjectAt(t, r)
		t.RegionID = r

		// Advance the path and if we have reached the destination, we will settle.
		if t.Path != nil {
			if t.Path.Next() != r {
				log.Printf("tribe %d is moving to region %d, which is not the next step in the path %d", t.ID, r, t.Path.Peek())
			}
			if t.Path.Done() {
				if t.Culture == nil || t.Culture.Language == nil {
					log.Printf("tribe %d has no culture or language", t.ID)
				}
				m.completeSettle(t)
			}
		}
	}
}
