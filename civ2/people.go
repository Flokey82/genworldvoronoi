package civ2

import (
	"math"
	"math/rand"
)

// calcPopulationGrowth calculates the population growth after nDays.
// Use the formula: P(t) = P0 * e^(rt)
// where P(t) is the population at time t, P0 is the initial population,
// r is the growth rate, and t is the time in years.
func calcPopulationGrowth(population int, growthRate float64, nDays int) float64 {
	return float64(population) * (math.Exp(growthRate*float64(nDays)/365) - 1)
}

// calcCombat returns the losses of each side in a combat.
// TODO: Deduplicate this function with the one in tribe.go.
func calcCombat(attacker, defender int) (int, int) {
	// Calculate the losses for each tribe.
	lossesA := rand.Intn(attacker)
	lossesD := rand.Intn(defender)
	return lossesA, lossesD
}
