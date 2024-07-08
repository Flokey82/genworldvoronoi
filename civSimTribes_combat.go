package genworldvoronoi

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/davecgh/go-spew/spew"
)

// fightForRegion will execute a fight between the attacking and defending tribe.
// If the attacking tribe wins, the function will return true.
func (s *simState) fightForRegion(attacking, defending *Tribe) bool {
	// TODO:
	// - The defending tribe should have a higher chance of inflicting losses on the attacking tribe.
	// - The losses should depend on the population of the opponent.
	// - The losses should depend on the culture of the tribes.
	// - The losses should depend on the skills of the tribes.
	// - The casulties should depend on the terrain, defense structures, etc.

	// Calculate the losses for each tribe.
	lossesDefendingTribe := rand.Intn(defending.Population)
	remainingDefending := defending.Population - lossesDefendingTribe
	lossesAttackingTribe := rand.Intn(attacking.Population)
	remainingAttacking := attacking.Population - lossesAttackingTribe

	// Add a history entry with the losses.
	historyMsg := fmt.Sprintf("Tribe %d (%d-%d=%d) has fought against tribe %d (%d-%d=%d)", attacking.ID, attacking.Population, lossesAttackingTribe, remainingAttacking, defending.ID, defending.Population, lossesDefendingTribe, remainingDefending)
	s.m.History.AddEvent("Combat", historyMsg, attacking.Ref())

	// The tribe with the largest remaining population wins.
	defending.Population = remainingDefending
	attacking.Population = remainingAttacking
	return attacking.Population > defending.Population
}

// chooseCombat will determine if the tribe will choose to fight for the region.
func chooseCombat(isDestination bool, attacking, defending *Tribe) bool {
	// Compare the two tribes and log if they'd like each other.
	if score := attacking.compare(defending); score < 0.5 {
		log.Printf("Tribe %d does not like tribe %d with a score of %f.", attacking.ID, defending.ID, score)
	} else {
		log.Printf("Tribe %d likes tribe %d with a score of %f.", attacking.ID, defending.ID, score)
	}

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
	if !s.fightForRegion(t, occupier) {
		// We have lost the fight for the region.
		t.changeSatisfaction(-combatSuccessSatisfaction)
		occupier.changeSatisfaction(combatSuccessSatisfaction)

		// Add a history entry.
		historyMsg := fmt.Sprintf("Tribe %s has fought for region %d and lost against tribe %s.", t.String(), nextRegion, occupier.String())
		s.m.History.AddEvent("Combat", historyMsg, t.Ref())

		// Add a history entry mentioning that the defending tribe has won the fight.
		historyMsg = fmt.Sprintf("Tribe %s has successfully defended against tribe %s.", occupier.String(), t.String())
		s.m.History.AddEvent("Combat", historyMsg, occupier.Ref())
		return false
	}

	// Attack with a focus to sack the settlement of the defending tribe.
	// When we defeat the tribe, we will take control of the settlement.
	// If we have succeeded, the tribe at the destination will be moved to our current one
	// and we will move to the destination.

	// Change satisfaction of the tribes.
	t.changeSatisfaction(combatSuccessSatisfaction)
	occupier.changeSatisfaction(-combatSuccessSatisfaction)

	// Move the other tribe to the current region.
	s.switchTribes(t, occupier)

	// Add a history entry.
	historyMsg := fmt.Sprintf("Tribe %s has fought for region %d and won against tribe %s.", t.String(), nextRegion, occupier.String())
	s.m.History.AddEvent("Combat", historyMsg, t.Ref())

	// Add a history entry mentioning that the other tribe has lost the fight.
	historyMsg = fmt.Sprintf("Tribe %s has been displaced by tribe %s after a fierce battle.", occupier.String(), t.String())
	s.m.History.AddEvent("Combat", historyMsg, occupier.Ref())
	return true
}

func (s *simState) handleSackingAttack(attacker, defender *Tribe, nextRegion int) bool {
	if !s.handleCombat(attacker, defender, nextRegion) {
		// Add a history entry mentioning the settlement we unsuccessfully tried to sack.
		historyMsg := fmt.Sprintf("Tribe %s has tried to sack the settlement %q of tribe %s, but failed.", attacker.String(), defender.Settlement.Name, defender.String())
		s.m.History.AddEvent("Combat", historyMsg, attacker.Ref())

		// Add a history entry mentioning that the defending tribe has successfully defended its settlement.
		historyMsg = fmt.Sprintf("Tribe %s has successfully defended its settlement %q against tribe %s.", defender.String(), defender.Settlement.Name, attacker.String())
		s.m.History.AddEvent("Combat", historyMsg, defender.Ref())
		return false
	}

	// The other tribe has lost its settlement.
	sackedSettlement := defender.Settlement
	sackedSettlement.Population = 0 // Depopulate the settlement.
	defender.Settlement = nil       // Remove the settlement.

	// The other tribe has to find a new place to settle.
	defender.doneSettling = false
	defender.Type = TribeTypeSettling

	if attacker.RegionID != nextRegion && defender.RegionID != nextRegion {
		log.Printf("t: %s", attacker.String())
		log.Printf("occupier: %s", defender.String())
		log.Printf("WARNING!!!! t.RegionID %d, nextRegion %d, occupier.RegionID %d", attacker.RegionID, nextRegion, defender.RegionID)
		spew.Dump([]int{attacker.RegionID, defender.RegionID, nextRegion})
	}

	// Try to find a new home for the other tribe.
	if !s.findNewRegionToSettle(defender, []int{attacker.RegionID, defender.RegionID, nextRegion}) {
		// If the other tribe couldn't find a new region to settle in, it will become nomadic.
		defender.makeNomadic(s.m.History)
	}

	// Add a history entry mentioning the settlement we successfully sacked.
	historyMsg := fmt.Sprintf("Tribe %s has sacked the settlement %q of tribe %s.", attacker.String(), sackedSettlement.Name, defender.String())
	s.m.History.AddEvent("Combat", historyMsg, attacker.Ref())

	// Add a history entry mentioning that the other tribe has lost its settlement.
	historyMsg = fmt.Sprintf("Tribe %s has lost its settlement %q after an attack by tribe %s.", defender.String(), sackedSettlement.Name, attacker.String())
	s.m.History.AddEvent("Combat", historyMsg, defender.Ref())
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
		// Add a history entry mentioning that we unsuccessfully tried to displace the other tribe.
		historyMsg := fmt.Sprintf("Tribe %s has tried to displace tribe %s, but failed.", attacker.String(), defender.String())
		s.m.History.AddEvent("Combat", historyMsg, attacker.Ref())

		// Add a history entry mentioning that the defending tribe has successfully defended against displacement.
		historyMsg = fmt.Sprintf("Tribe %s has successfully defended against tribe %s who tried to displace them.", defender.String(), attacker.String())
		s.m.History.AddEvent("Combat", historyMsg, defender.Ref())
		return false
	}

	// If the other tribe is on the move, we need to find a new path for it.
	if defender.hasPath() && !s.findNewPath(defender, defender.Path.To) {
		// If the other tribe couldn't find a new path to the destination, it will become nomadic.
		defender.makeNomadic(s.m.History)
	}

	// Add a history entry mentioning that we successfully displaced the other tribe.
	historyMsg := fmt.Sprintf("Tribe %s has displaced tribe %s.", attacker.String(), defender.String())
	s.m.History.AddEvent("Combat", historyMsg, attacker.Ref())

	// Add a history entry mentioning that the other tribe has been displaced.
	historyMsg = fmt.Sprintf("Tribe %s has been displaced by tribe %s.", defender.String(), attacker.String())
	s.m.History.AddEvent("Combat", historyMsg, defender.Ref())
	return true
}
