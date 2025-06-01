package civ

import (
	"fmt"
	"math/rand"
)

func (m *Civ) fightForRegion(attacking, defending *Tribe, r int) bool {
	// TODO:
	// - The defending tribe should have a higher chance of inflicting losses on the attacking tribe.
	// - The losses should depend on the population of the opponent.
	// - The losses should depend on the culture of the tribes.
	// - The losses should depend on the skills of the tribes.
	// - The casulties should depend on the terrain, defense structures, etc.

	// Compare the preferences of each each tribe and the destination region's suitability for each tribe.
	// suitA := attacking.compareSuitability(m.GetRegionProp(r))
	// suitD := defending.compareSuitability(m.GetRegionProp(r))

	// Calculate the losses for each tribe.
	lossesA := rand.Intn(attacking.Population)
	remainA := attacking.Population - lossesA

	lossesD := rand.Intn(defending.Population)
	remainD := defending.Population - lossesD

	// Add a history entry with the losses.
	historyMsg := fmt.Sprintf("Tribe %d (%d-%d=%d) has fought against tribe %d (%d-%d=%d)", attacking.ID, attacking.Population, lossesA, remainA, defending.ID, defending.Population, lossesD, remainD)
	m.History.AddEvent("Combat", historyMsg, attacking.Ref())

	// The tribe with the largest remaining population wins.
	attacking.Population = remainA
	defending.Population = remainD
	return attacking.Population > defending.Population
}
