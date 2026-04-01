package civ2

import (
	"fmt"
	"math"

	"github.com/Flokey82/genetics/geneticshuman"
	"math/rand"
)

// Military tracks the military strength and state of an entity.
type Military struct {
	Power          float64 // Current military strength
	Aggressiveness float64 // Likelihood of starting a conflict [0.0 - 1.0]
}

func NewMilitary() *Military {
	return &Military{
		Aggressiveness: 0.1,
	}
}

// tickMilitary handles military power buildup and maintenance.
func (m *Civ) tickMilitary(e peopleThing, nDays int) {
	mil := e.GetMilitary()
	if mil == nil {
		return
	}

	pop := e.GetPopulation()
	storage := e.GetStorage()
	gp := e.GetGoverningPeople()

	// 1. Recruitment & Training
	// Based on population and martial influence.
	martialBonus := 1.0
	if gp != nil && gp.Leadership != nil && gp.Leadership.Type == FactionTypeMartial {
		martialBonus = 1.5
	}

	// We recruit from the population (abstractly).
	// Let's say 1% of the population can be in the military.
	potentialPower := float64(pop) * 0.01 * martialBonus
	
	// Power buildup (takes time to train).
	// Scale growth by nDays, assuming original 0.05 was per year (365 days).
	growthFactor := 1.0 - math.Pow(1.0-0.05, float64(nDays)/365.0)
	mil.Power += (potentialPower - mil.Power) * growthFactor

	// 2. Maintenance
	// Military consumes food and potentially gold.
	// We'll keep it simple: each unit of power consumes 1 food per day.
	foodNeeded := int(mil.Power) * nDays
	if foodNeeded > 0 {
		avail := storage.GetResource(ResFood)
		if avail < foodNeeded {
			// Famine among soldiers! Loss of power proportional to deficit.
			ratio := float64(avail) / float64(foodNeeded)
			storage.RemoveResource(ResFood, avail)
			mil.Power *= (1.0 - 0.1*(1.0-ratio))
		} else {
			storage.RemoveResource(ResFood, foodNeeded)
		}
	}

	// Aggressiveness drift towards leadership's preference.
	if gp != nil && gp.Leadership != nil && gp.Leadership.Leadership != nil && gp.Leadership.Leadership.Leader != nil {
		leader := gp.Leadership.Leadership.Leader
		if leader.Traits.HasTrait(geneticshuman.TraitAggressive) {
			mil.Aggressiveness = math.Min(1.0, mil.Aggressiveness+0.01)
		} else if leader.Traits.HasTrait(geneticshuman.TraitKind) {
			mil.Aggressiveness = math.Max(0.0, mil.Aggressiveness-0.01)
		}
	}
}

// resolveCombat calculates the outcome of a battle between two entities.
// Returns true if the attacker won.
func (m *Civ) resolveCombat(attacker, defender peopleThing) bool {
	aMil := attacker.GetMilitary()
	dMil := defender.GetMilitary()
	if aMil == nil || dMil == nil {
		// Fallback to population-based combat if military is missing.
		attStrength := float64(attacker.GetPopulation()) * (0.8 + 0.4*rand.Float64())
		defStrength := float64(defender.GetPopulation()) * (0.8 + 0.4*rand.Float64())
		return attStrength > defStrength
	}

	// Factors:
	// - Military Power
	// - Infrastructure (Walls)
	// - Randomness

	defenseBonus := 1.0
	if infra := defender.GetInfrastructure(); infra != nil {
		for _, b := range infra.Buildings {
			if b == BlueprintWalls {
				defenseBonus += 0.5
			}
		}
	}

	attStrength := aMil.Power * (0.8 + 0.4*rand.Float64())
	defStrength := dMil.Power * defenseBonus * (0.8 + 0.4*rand.Float64())

	// History logging
	if attStrength > defStrength {
		// Attacker wins
		lossesA := aMil.Power * 0.1
		lossesD := dMil.Power * 0.2
		aMil.Power -= lossesA
		dMil.Power -= lossesD

		// Looting: Transfer some resources
		m.loot(attacker, defender, 0.2)
		
		m.History.AddEvent("combat", fmt.Sprintf("%s won a battle against %s.", attacker.String(), defender.String()), attacker.Ref())
		m.History.AddEvent("combat", fmt.Sprintf("%s lost a battle against %s.", defender.String(), attacker.String()), defender.Ref())
		return true
	} else {
		// Defender wins
		lossesA := aMil.Power * 0.2
		lossesD := dMil.Power * 0.1
		aMil.Power -= lossesA
		dMil.Power -= lossesD

		m.History.AddEvent("combat", fmt.Sprintf("%s failed an attack on %s.", attacker.String(), defender.String()), attacker.Ref())
		m.History.AddEvent("combat", fmt.Sprintf("%s successfully defended against %s.", defender.String(), attacker.String()), defender.Ref())
		return false
	}
}

func (m *Civ) loot(attacker, defender peopleThing, ratio float64) {
	aS := attacker.GetStorage()
	dS := defender.GetStorage()
	if aS == nil || dS == nil {
		return
	}

	for res, amount := range dS.Resources {
		lootAmount := int(float64(amount) * ratio)
		if lootAmount > 0 {
			dS.RemoveResource(res, lootAmount)
			aS.AddResource(res, lootAmount)
		}
	}
}

