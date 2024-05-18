package genworldvoronoi

import (
	"log"
	"math"
	"math/rand"

	"github.com/Flokey82/go_gens/genreligion"
)

func (s *simState) handleSettlingTribe(t *Tribe) {
	// Check if the tribe has a path set. If not, we need to find a suitable region to settle in.
	if !t.hasPath() {
		// There is a random chance that the tribe will receive a vision from the gods,
		// which will help them find a suitable region to settle in.
		t.gotVision = rand.Intn(100) < 99

		// Check if we could find a suitable region to settle in.
		if !s.findNewRegionToSettle(t, nil) {
			log.Println("Tribe", t.ID, "couldn't find a new region to settle or a path to the new region.")
		}

		// Couldn't find a path to a new region, so we need to try again next turn.
		if !t.hasPath() {
			return
		}
	}

	// We have a path set, so we need to move the tribe to the next region in the path.
	// In general, if we want to move to the next region, check if the region is already occupied.
	//
	// So we just "peek" at the next region to see if it's occupied and either:
	// - choose combat
	// - find a way around
	//
	// If the next region would be our destination and it is occupied we can:
	// - choose combat
	// - merge with the other tribe
	// - find a new region
	nextRegion := t.Path.Peek()

	// Check if the next region is already occupied and if the occupying tribe is not us.
	if occupier := s.tribeAtRegion[nextRegion]; occupier != nil && occupier != t {
		var success bool

		// Check if the next region is the destination.
		isDestination := t.Path.PeekDone()

		// Check if occupier has settled in the region.
		isSettled := occupier.Settlement != nil && occupier.Settlement.ID == nextRegion

		// We might want to fight for the region.
		if chooseCombat(isDestination, t, occupier) {
			// TODO:
			// - Check if either tribe has died out after the fight.
			// - If we have won, we need to dislodge the other tribe, if it still exists.
			// - Store new opinion of the tribes about each other.
			if isDestination && isSettled {
				// We need to handle the possibility that the settlement is the capital of a city state or empire.
				if occupier.CityState != nil || occupier.Empire != nil {
					panic("captured city is the capital of a city state or empire")
				}
				success = s.handleSackingAttack(t, occupier, nextRegion)
			} else {
				success = s.handleDisplacementAttack(t, occupier, nextRegion)
			}
		}

		// Check if we have not succeeded or attempted to fight for the region.
		if !success {
			if isDestination {
				// If the next region is the destination, we need to find a new region to settle in.
				success = s.findNewRegionToSettle(t, []int{nextRegion, t.RegionID, occupier.RegionID})
				if success {
					log.Println("Failed to settle in destination; Tribe", t.ID, "has found a new region to settle in as tribe", occupier.ID, "has occupied the region", nextRegion, ".")
				} else {
					log.Println("Failed to settle in destination; Tribe", t.ID, "failed to find a new region to settle in as tribe", occupier.ID, "has occupied the region", nextRegion, ".")
				}
			} else {
				// Find a way around the region, or find a new region.
				// TODO: The pathfinding should avoid regions that are occupied by other tribes.
				// ... but if no path avoiding other tribes can be found, we might need to fight for the region.
				success = s.findNewPath(t, t.Path.To)
				if success {
					log.Println("Failed to pass through region; Tribe", t.ID, "has found a new path to region", t.Path.To, "as tribe", occupier.ID, "has occupied the region", nextRegion, ".")
				} else {
					log.Println("Failed to pass through region; Tribe", t.ID, "failed to find a new path to region", t.Path.To, "as tribe", occupier.ID, "has occupied the region", nextRegion, ".")
				}
			}
		}

		// If we didn't succeed in dislodging the other tribe or in finding a new region to settle in.
		// What to do? Stay in the current region?
		if !success {
			log.Println("$$$$$$$$$$$$$$Tribe", t.ID, "failed to move to region", nextRegion, " and will stay in region", t.RegionID, " as tribe", occupier.ID, "has occupied the region", nextRegion, ".")
			return
		}
	}

	// Move the tribe to the next region in the path.
	nextRegion = t.Path.Next()

	// Check if the next region in the path is still occupied by another tribe.
	// If this happens, we need to debug what happened, so panic.
	if s.tribeAtRegion[nextRegion] != nil && s.tribeAtRegion[nextRegion] != t {
		log.Println("NOTE: Tribe", t.ID, "wants to move to region", nextRegion, "but it's already occupied by tribe", s.tribeAtRegion[nextRegion].ID, ".")
		panic("tribe wants to move to an occupied region")
	}

	// Move the tribe to the next region in the path.
	s.moveTribe(t, nextRegion)
	log.Println("Tribe", t.ID, "is moving to region", nextRegion, "in order to settle.")

	// Check if the tribe has arrived at the new region.
	if t.Path.Done() {
		s.handleEstablishingSettlement(t)
	}
}

func (s *simState) handleEstablishingSettlement(t *Tribe) {
	// TODO: This value should depend on the tribe's preferences.
	// (if it is a spiritual calling, the satisfaction should be higher, etc.)
	t.changeSatisfaction(settlingSuccessSatisfaction)

	// Get the culture type that we should have in this region.
	ctf := s.m.getRegionCultureTypeFunc() // <- VERY COSTLY!!!
	regionType := ctf(t.RegionID)

	// Get the existing culture of the region.
	existingCulture := s.m.GetCulture(t.RegionID)

	// If the tribe did not have yet its own culture, we either adopt the existing culture
	// of the region, or we create a new culture based on the culture type of the region.
	if t.Culture == nil {
		if existingCulture == nil || existingCulture.Type != regionType {
			// No existing culture in the region or the existing culture is not of the right type
			// for the current region.
			t.Culture = s.m.PlaceCultureAt(t.RegionID, false, t.Language)
		} else {
			// We adopt the existing culture of the region.
			log.Println("Tribe", t.ID, "is adopting the culture of the region", t.RegionID, ".!!!!!!!!!!!!!!!!!!!!1")
			t.Culture = existingCulture
		}
	} else if existingCulture != t.Culture {
		// We are settling in a region that has a different culture than our own,
		// so we retain our own, and fork the culture that we brought with us,
		// which is the culture of the parent tribe.

		// Fork culture.
		// TODO:
		// - Make this by random chance and distance to the original culture.
		// - Make sure that there is no culture that has this ID.
		t.Culture = s.m.AddCultureAt(t.RegionID, t.Culture.Fork(t.RegionID), false)
		t.Culture.SetNewType(regionType) // Set the new type of the culture.
	}

	// Update the type of the new culture based on the preferences of the tribe.
	// TODO: Update all preferences to match the new location.
	log.Println("Tribe", t.ID, "with a culture of", t.GetCultureType(), "should have a culture of", regionType)

	// Check if there is already a settlement in the region and either update it or create a new one.
	t.Settlement = s.m.GetCity(t.RegionID)
	if t.Settlement != nil {
		// We have an existing settlement in the region, so we update it.
		// TOOD:
		// - Rename the settlement using the language of the tribe.
		// - Modify the settlement to match the tribe's preferences.
		t.Settlement.Name += " (captured)"
		t.Settlement.Culture = t.Culture
		t.Settlement.Religion = t.Religion
		t.Settlement.Population = t.Population
	} else {
		// We need to create a new settlement in the region.
		t.Settlement = s.m.placeCityAt(t.RegionID, -1, s.m.getRegCityType(t.RegionID), t.Population, 1.0)

		// Get secondary type of the settlement, which is the types of neighboring regions.
		seenSecondaryTypes := make(map[TownType]bool)
		for _, nb := range s.m.R_circulate_r(rNbs, t.RegionID) {
			if tt := s.m.getRegCityType(nb); tt != t.Settlement.Type && !seenSecondaryTypes[tt] {
				t.Settlement.SecondaryTypes = append(t.Settlement.SecondaryTypes, tt)
				seenSecondaryTypes[tt] = true
			}
		}
	}

	// Set the tribe as a settled tribe.
	if t.CityState != nil || t.Empire != nil {
		panic("Tribe is a city state or empire")
	}
	t.doneSettling = true
	t.Type = TribeTypeCity

	// If we found this region through the gods, we establish a new religion, either
	// as a variant of the original religion or as a new, independent religion.
	if t.gotVision {
		// This will generate a religion based on the culture and the original religion of the "parent tribe"
		// and update the religion of the tribe's culture.
		// TODO: We should create a new religion as soon as we receive a vision from the gods
		// not only when we settle.
		group := genreligion.GroupFolk
		if t.Religion != nil {
			group = t.Religion.Group
		}
		t.Religion = s.m.placeReligionAt(t.RegionID, -1, group, t.Culture, t.Culture.Language, t.Religion)
		t.Culture.Religion = t.Religion
		log.Println("Tribe", t.ID, "has received a vision from the gods and settled in region", t.RegionID, "and follows the religion of", t.Religion.String())
	} else {
		log.Println("Tribe", t.ID, "has settled in region", t.RegionID, "with a population of", t.Population, "and a culture of", t.Culture.Type)
	}
}

// Outline:
// - No path set:
//   - Find a suitable region to settle in and set a path to the region.
// - Path set:
//   - Peek at the next region in the path and see if it is unoccupied.
//   - If it is unoccupied:
//     - Move to the region.
//   - If it is occupied:
//     - We might fight the tribe for the region.
//     - If it is the destination and we can't defeat the tribe or decide not to fight:
//       - Find a new region to settle
//       - Set a path to the new region.
//     - If it is not the destination:
//       - Find a new path to the destination.
// - If we can move to the next region in the path and it is the destination:
//   - Move to the region.
//   - Settle in the region.

// If the tribe has no path set, we need to scout for the best region
// to settle in within a radius and set a path to the new region.
func (s *simState) getSettleScoreFunc(t *Tribe, spiritualCalling bool) func(r int) (float64, bool) {
	const (
		radiusScout       = 300.0  // km
		radiusGods        = 1000.0 // km
		radiusMaxCityProx = 300.0  // km
	)

	// If the region is determined by the gods, we increase the radius to find
	// the most suitable region (TODO: which should stay unoccupied for a while).
	radius := radiusScout
	if spiritualCalling {
		radius = radiusGods
	}
	log.Println("!!!Tribe", t.ID, "is scouting for a new region to settle in within a radius of", radius, "km.")

	settleRegionScore := func(r int) (float64, bool) {
		// Make sure we don't rate the current region too highly.
		if r == t.RegionID {
			return 0, true
		}

		// Check if there is already a settlement in the region.
		/*
			for _, c := range cities {
				if c.ID == r {
					// DEBUG: Encourage settling in regions with cities so we can test the combat system.
					if c.ID != t.RegionID {
						return 100000, true
					}
					return 0, true
				}
			}
		*/

		// Get the distance to the current region in km.
		dist := s.m.Geo.GetDistance(t.RegionID, r) * unitDistToKm // TODO: Make this a method of the Geo struct.
		if dist > radius {
			log.Printf("Tribe %d has scouted a region %d, but it's too far away (%.2f km).", t.ID, r, dist)
			return 0, false
		}
		log.Printf("Tribe %d has scouted a region %d, and the distance is %.2f km.", t.ID, r, dist)

		// Determine the distance to the closest city.
		// NOTE: In theory we could use a distance field to determine the distance to the closest tribe / city
		// and encourage settling in new regions.
		minDist := math.Inf(1)
		for _, c := range s.cities {
			if dist := s.m.Geo.GetDistance(c.ID, r); dist < minDist {
				minDist = dist
			}
		}
		minDist = min(radiusMaxCityProx, minDist*unitDistToKm) // Cap the distance to 300 km (for now)
		log.Printf("Tribe %d has scouted a region %d, and the distance to the closest city is %.2f km.", t.ID, r, minDist)

		// Calculate the max population for the region and see if it's the best region to settle in
		// and add bonus if the region has similar attributes to the tribe's preferences.
		// Also cap the maxPopPerRegion to twice the current population of the tribe.
		maxPopPerReg := float64(min(s.calcMaxPopPerRegion(t, r), 2*t.Population))
		regMul := s.getRegMultiplier(t, r)

		// TODO: The economic score should depend on the skills of the tribe.
		// For example, anything immediately useful to the tribe should be rated higher,
		// while luxury goods should be secondary to the economic value, especially if
		// the tribe doesn't have the skills to extract the resources.
		ecoMul := s.getRegEconomicMultiplier(r)
		val := maxPopPerReg * regMul * ecoMul * minDist / radiusMaxCityProx

		// Disincentivize settling in regions that are already settled by a culture.
		if s.m.RegionToCulture[r] != -1 {
			val /= 1.2
		}

		// Check if there is already a settlement in the region.
		if c := s.m.GetCity(r); c != nil {
			if c.Population == 0 {
				// If the city is abandoned, we encourage settling in the region.
				// TODO: This value should reflect the value of the city.
				val *= 1.1
			} else {
				// If the city is not abandoned, we discourage settling in the region.
				val /= 2
			}
		}

		log.Printf("Tribe %d has scouted a region %d, and the score is %.2f. regMul %.2f", t.ID, r, val, regMul)
		return val, true
	}
	return settleRegionScore
}

// findNewRegionToSettle will find a new region to settle in and set a new path to the region.
func (s *simState) findNewRegionToSettle(t *Tribe, avoidRegs []int) bool {
	// Find the best region to settle in.
	settleRegionScore := s.getSettleScoreFunc(t, t.gotVision)
	bestRegion, score := s.findBestRegion(t, t.RegionID, settleRegionScore, avoidRegs)
	if bestRegion == -1 {
		log.Println("!!!Tribe", t.ID, "couldn't find a suitable region to settle in.")
		return false // Couldn't find a suitable region to settle in.
	}
	log.Println("!!!Tribe", t.ID, "has found a region to settle in:", bestRegion, "with a score of", score)

	// We have found a new region to settle in, so plot a path to the region
	// and return the outcome of the pathfinding.
	s.printRegionInfo(bestRegion)
	t.printTribePreferences()
	if !s.findNewPath(t, bestRegion) {
		log.Println("!!!Tribe", t.ID, "couldn't find a path to the new region.")
		return false
	}
	return true
}
