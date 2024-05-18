package genworldvoronoi

import (
	"log"
	"math/rand"

	"github.com/davecgh/go-spew/spew"
)

// fightForRegion will execute a fight between the attacking and defending tribe.
// If the attacking tribe wins, the function will return true.
func fightForRegion(attacking, defending *Tribe) bool {
	// TODO:
	// - The defending tribe should have a higher chance of inflicting losses on the attacking tribe.
	// - The losses should depend on the population of the opponent.
	// - The losses should depend on the culture of the tribes.
	// - The losses should depend on the skills of the tribes.
	// - The casulties should depend on the terrain, defense structures, etc.

	// Calculate the losses for each tribe.
	lossesDefendingTribe := rand.Intn(defending.Population)
	defending.Population -= lossesDefendingTribe

	lossesAttackingTribe := rand.Intn(attacking.Population)
	attacking.Population -= lossesAttackingTribe

	// The tribe with the largest remaining population wins.
	return attacking.Population > defending.Population
}

// chooseCombat will determine if the tribe will choose to fight for the region.
func chooseCombat(isDestination bool, attacking, defending *Tribe) bool {
	// Determine if the tribe will sack the settlement at the destination
	// if there is a tribe settled there and it is our destination.
	disallowSacking := false

	// For now we don't allow sacking of capitals.
	if defending.CityState != nil && defending.CityState.Capital == defending.Settlement ||
		defending.Empire != nil && defending.Empire.Capital == defending.Settlement {
		log.Printf("Tribe %d is trying to sack the capital of tribe %d.", attacking.ID, defending.ID)
		return false
	}

	// Check if the defending tribe has settled there:
	if defending.Type > TribeTypeSettling && defending.doneSettling && (disallowSacking || !isDestination) {
		return false
	}

	// Depending on our aggressiveness, the strength of the other tribe,
	// and if this is our destination, we might choose to fight for the region.
	multiplier := 1.0

	// This is where we want to settle.
	if isDestination {
		multiplier += 0.1
	}

	// Religion might be a powerful motivator for combat.
	// TODO: Some religions might be peaceful, while others might be more aggressive.
	if attacking.gotVision {
		multiplier += 0.1
	}
	// Consider culture.
	// TODO: Consider expansionism, aggressiveness, etc.

	// We multiply our own strength by the multiplier.
	confidence := float64(attacking.Population) * multiplier
	return confidence > float64(defending.Population)
}

func (s *simState) handleCombat(t, occupier *Tribe, nextRegion int) bool {
	if !fightForRegion(t, occupier) {
		// The attacking tribe has lost the fight.
		log.Println("Tribe", t.ID, "has fought for region", nextRegion, "and lost against tribe", occupier.ID)
		// We have lost the fight for the region.
		t.changeSatisfaction(-combatSuccessSatisfaction)
		occupier.changeSatisfaction(combatSuccessSatisfaction)
		return false
	}

	// Attack with a focus to sack the settlement of the defending tribe.
	// When we defeat the tribe, we will take control of the settlement.
	// If we have succeeded, the tribe at the destination will be moved to our current one
	// and we will move to the destination.
	log.Println("Tribe", t.ID, "has fought for region", nextRegion, "and won against tribe", occupier.ID)

	// Change satisfaction of the tribes.
	t.changeSatisfaction(combatSuccessSatisfaction)
	occupier.changeSatisfaction(-combatSuccessSatisfaction)

	// Move the other tribe to the current region.
	s.switchTribes(t, occupier)

	return true
}

func (s *simState) handleSackingAttack(attacker, defender *Tribe, nextRegion int) bool {
	if !s.handleCombat(attacker, defender, nextRegion) {
		return false
	}

	// The other tribe has lost its settlement.
	defender.Settlement.Population = 0 // Depopulate the settlement.
	defender.Settlement = nil          // Remove the settlement.

	// The other tribe has to find a new place to settle.
	defender.doneSettling = false
	defender.Type = TribeTypeSettling
	log.Printf("Tribe %d has sacked the settlement of tribe %d.", attacker.ID, defender.ID)

	log.Println("TODO: avoid moving back to the region where we were just kicked out of")
	log.Println("1111111jfdkjlskjfsdjlsjfklsdjfkdfjksdjflksjfksdjfksjflksdjfksjlsjfksdflksjfsjl")
	spew.Dump([]int{attacker.RegionID, defender.RegionID, nextRegion})
	if attacker.RegionID != nextRegion && defender.RegionID != nextRegion {
		log.Printf("t: %s", attacker.String())
		log.Printf("occupier: %s", defender.String())
		log.Printf("WARNING!!!! t.RegionID %d, nextRegion %d, occupier.RegionID %d", attacker.RegionID, nextRegion, defender.RegionID)
	}

	// Try to find a new home for the other tribe.
	if !s.findNewRegionToSettle(defender, []int{attacker.RegionID, defender.RegionID, nextRegion}) {
		// If the other tribe couldn't find a new region to settle in, it will become nomadic.
		defender.makeNomadic()
	}
	return true
}

func (s *simState) handleRaidingAttack(attacker, defender *Tribe, nextRegion int) bool {
	if !s.handleCombat(attacker, defender, nextRegion) {
		return false
	}
	// Attack with a focus to raid the tribe / settlement.
	// When we defeat the tribe, we will take some resources and leave.
	return true
}

func (s *simState) handleDisplacementAttack(attacker, defender *Tribe, nextRegion int) bool {
	if !s.handleCombat(attacker, defender, nextRegion) {
		return false
	}

	// Move the other tribe to the current region.
	s.switchTribes(attacker, defender)

	// If the other tribe is on the move, we need to find a new path for it.
	if defender.hasPath() && !s.findNewPath(defender, defender.Path.To) {
		// If the other tribe couldn't find a new path to the destination, it will become nomadic.
		defender.makeNomadic()
	}
	return true
}
