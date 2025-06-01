package civ

import (
	"log"
)

// chooseCombat will determine if the tribe will choose to fight for the region.
func chooseCombat(isDestination bool, attacking, defending *Tribe) bool {
	// Compare the two tribes and log if they'd like each other.
	if score := attacking.compare(defending); score < 0.5 {
		log.Printf("Tribe %d does not like tribe %d with a score of %f.", attacking.ID, defending.ID, score)
	} else {
		log.Printf("Tribe %d likes tribe %d with a score of %f.", attacking.ID, defending.ID, score)
	}

	// For now we don't allow sacking of capitals.
	if defending.CityState != nil && defending.CityState.Capital == defending.Settlement ||
		defending.Empire != nil && defending.Empire.Capital == defending.Settlement {
		log.Printf("Tribe %d is trying to sack the capital of tribe %d.", attacking.ID, defending.ID)
		return false
	}

	// Determine if the tribe will sack the settlement at the destination
	// if there is a tribe settled there and it is our destination.
	disallowSacking := false

	// Check if the defending tribe has settled there.
	if defending.Type > TribeTypeSettling && defending.doneSettling && (disallowSacking || !isDestination) {
		return false
	}

	// Depending on our aggressiveness, the strength of the other tribe,
	// and if this is our destination, we might choose to fight for the region.
	// TODO:
	// - Some religions might be peaceful, while others might be more aggressive.
	// - Consider culture, expansionism, aggressiveness, etc.
	multiplier := 1.0
	if isDestination {
		multiplier += 0.1 // We are more likely to fight for our destination.
	}
	if attacking.gotVision {
		multiplier += 0.1 // Religion might be a powerful motivator for combat.
	}

	// We multiply our own strength by the multiplier.
	confidence := float64(attacking.Population) * multiplier
	return confidence > float64(defending.Population)
}

// combatSuccessSatisfaction is the value that will be added to the satisfaction of the tribes
// after a combat outcome.
const combatSuccessSatisfaction = 0.5

func (s *simState) handleCombat(t, occupier *Tribe, nextRegion int) bool {
	if s.m.fightForRegion(t, occupier, nextRegion) {
		// We have won the fight for the region.
		t.changeSatisfaction(combatSuccessSatisfaction)
		occupier.changeSatisfaction(-combatSuccessSatisfaction)

		// Switch the tribes' regions.
		s.switchTribes(t, occupier)

		// Add a history entry for both tribes.
		s.m.History.AddEventf("Combat", t.Ref(), "Tribe %s has fought for region %d and won against tribe %s.", t.String(), nextRegion, occupier.String())
		s.m.History.AddEventf("Combat", occupier.Ref(), "Tribe %s has been displaced by tribe %s after a fierce battle.", occupier.String(), t.String())
		return true
	}

	// We have lost the fight for the region.
	t.changeSatisfaction(-combatSuccessSatisfaction)
	occupier.changeSatisfaction(combatSuccessSatisfaction)

	// Add a history entry for both tribes.
	s.m.History.AddEventf("Combat", t.Ref(), "Tribe %s has fought for region %d and lost against tribe %s.", t.String(), nextRegion, occupier.String())
	s.m.History.AddEventf("Combat", occupier.Ref(), "Tribe %s has successfully defended against tribe %s.", occupier.String(), t.String())
	return false
}

func (s *simState) handleSackingAttack(attacker, defender *Tribe, nextRegion int) bool {
	if s.handleCombat(attacker, defender, nextRegion) {
		// We have won, the other tribe has lost its settlement.
		sackedSettlement := defender.Settlement
		sackedSettlement.Population = 0 // Depopulate the settlement.
		defender.Settlement = nil       // Remove the settlement.

		// The other tribe has to find a new place to settle.
		defender.doneSettling = false
		defender.Type = TribeTypeSettling

		// Try to find a new home for the other tribe. If we fail, it will become nomadic.
		if !s.findNewRegionToSettle(defender, []int{attacker.RegionID, defender.RegionID, nextRegion}) {
			defender.makeNomadic(s.m.History)
		}

		// Add a history entry mentioning the settlement we successfully sacked.
		s.m.History.AddEventf("Combat", attacker.Ref(), "Tribe %s has sacked the settlement %q of tribe %s.", attacker.String(), sackedSettlement.Name, defender.String())

		// Add a history entry mentioning that the other tribe has lost its settlement.
		s.m.History.AddEventf("Combat", defender.Ref(), "Tribe %s has lost its settlement %q after an attack by tribe %s.", defender.String(), sackedSettlement.Name, attacker.String())
		return true
	}

	// Add a history entry mentioning the settlement we unsuccessfully tried to sack.
	s.m.History.AddEventf("Combat", attacker.Ref(), "Tribe %s has tried to sack the settlement %q of tribe %s, but failed.", attacker.String(), defender.Settlement.Name, defender.String())

	// Add a history entry mentioning that the defending tribe has successfully defended its settlement.
	s.m.History.AddEventf("Combat", defender.Ref(), "Tribe %s has successfully defended its settlement %q against tribe %s.", defender.String(), defender.Settlement.Name, attacker.String())
	return false
}

func (s *simState) handleRaidingAttack(attacker, defender *Tribe, nextRegion int) bool {
	// Attack with a focus to raid the tribe / settlement for resources.
	// TODO: Implement raiding.
	return s.handleCombat(attacker, defender, nextRegion)
}

func (s *simState) handleDisplacementAttack(attacker, defender *Tribe, nextRegion int) bool {
	if s.handleCombat(attacker, defender, nextRegion) {
		// If the other tribe is on the move, we need to find a new path for it.
		if defender.hasPath() && !s.findNewPath(defender, defender.Path.To) {
			// If the other tribe couldn't find a new path to the destination, it will become nomadic.
			defender.makeNomadic(s.m.History)
		}

		// Add a history entry mentioning that we successfully displaced the other tribe.
		s.m.History.AddEventf("Combat", attacker.Ref(), "Tribe %s has displaced tribe %s.", attacker.String(), defender.String())

		// Add a history entry mentioning that the other tribe has been displaced.
		s.m.History.AddEventf("Combat", defender.Ref(), "Tribe %s has been displaced by tribe %s.", defender.String(), attacker.String())
		return true
	}

	// Add a history entry mentioning that we unsuccessfully tried to displace the other tribe.
	s.m.History.AddEventf("Combat", attacker.Ref(), "Tribe %s has tried to displace tribe %s, but failed.", attacker.String(), defender.String())

	// Add a history entry mentioning that the defending tribe has successfully defended against displacement.
	s.m.History.AddEventf("Combat", defender.Ref(), "Tribe %s has successfully defended against tribe %s who tried to displace them.", defender.String(), attacker.String())
	return false
}
