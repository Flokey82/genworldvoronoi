package civ

import (
	"fmt"
	"log"
)

type familyUnit struct {
	Adults   []*Person
	Children []*Person
	Home     *Construction
}

func (fu *familyUnit) String() string {
	return fmt.Sprintf("Family unit: %d adults, %d children, home: %s", len(fu.Adults), len(fu.Children), fu.Home.Name)
}

func (fu *familyUnit) assignHome(co *Construction) {
	fu.Home = co
	for _, a := range fu.Adults {
		a.Home = co
	}
	for _, c := range fu.Children {
		c.Home = co
	}
}

func (s *simState) buildThings(t *Tribe) {
	// TODO: If we arent nomadic, the housing should be constructed in the region where we are.
	var ds *ComboStorage
	if t.Type <= TribeTypeSettling {
		ds = t.ComboStorage
	} else {
		ds = t.Settlement.ComboStorage
	}

	// TODO: For nomadic tribes we will construct tents, for which we need less wood
	// but also either hide or cloth.

	// Sum up housing that we already have.
	// TODO: Sum up family units so we can calculate how many houses or tents we need.
	// Directly simulated people can build their own houses, but for the rest of the population
	// we need to construct them.
	seenPeople := make(map[*Person]bool)
	// Start with all children and their parents or adult childless people and their spouses.
	// Then walk up the generations and put the parents in houses together.
	familyUnits := make(map[*Person]*familyUnit)
	for _, p := range t.People {
		if seenPeople[p] || p.Age < 18 {
			continue
		}
		seenPeople[p] = true
		fu := &familyUnit{
			Adults: []*Person{p},
			Home:   p.Home,
		}

		// Add the spouse if there is one.
		if p.Spouse != nil {
			seenPeople[p.Spouse] = true
			fu.Adults = append(fu.Adults, p.Spouse)
		}

		// Add all children that aren't adults yet.
		for _, c := range p.Children {
			if c.Age < 18 {
				seenPeople[c] = true
				fu.Children = append(fu.Children, c)
			}
		}

		livesInParenthome := func(p, parent *Person) bool {
			return parent == nil || p.Home == parent.Home
		}

		// Check if the adults have a home.
		// If they live in the same house as their parents, unset the home.
		// If they live in different houses, abandon one of the houses.
		// Assign a common house to the parents and their children.
		for _, a := range fu.Adults {
			if a.Home != nil {
				// Check if they live with their parents.
				if livesInParenthome(a, a.Mother) || livesInParenthome(a, a.Father) {
					a.Home = nil
					continue
				}
				if fu.Home == nil {
					fu.Home = a.Home
				} else if fu.Home != a.Home {
					a.Home = nil // Abandon the house.
				}
			}
		}

		// Update the home of the family unit.
		fu.assignHome(fu.Home)
		familyUnits[p] = fu
	}

	// Log all family units.
	var needsHome []*familyUnit
	seenHome := make(map[*Construction]bool)
	housedPeople := 0
	for _, members := range familyUnits {
		if members.Home == nil {
			needsHome = append(needsHome, members)
			log.Printf("Family unit: (%d members) needs a home", len(members.Adults)+len(members.Children))
			housedPeople += len(members.Adults) + len(members.Children)
		} else {
			seenHome[members.Home] = true
			log.Printf("Family unit: (%d members)", len(members.Adults)+len(members.Children))
		}
		for _, m := range members.Adults {
			log.Printf(" - %s (adult)", m.String())
		}
		for _, m := range members.Children {
			log.Printf(" - %s", m.String())
		}
	}

	// TODO: Track the amount of resources we spend on construction.
	housingNeeded := t.Population
	housingAvailable := ds.Resources[ResHousing]
	if housingAvailable < housingNeeded {
		log.Printf("Tribe %s needs more housing. %d/%d", t.String(), housingAvailable, housingNeeded)
		// Select the construction cost.
		for housingAvailable < housingNeeded {
			// Check what we can construct.
			if ds.CanConstruct(constHut) {
				ds.Construct(constHut)
			} else if ds.CanConstruct(constTent) {
				ds.Construct(constTent)
			} else {
				log.Printf("Tribe %s can't construct more housing. %d/%d", t.String(), housingAvailable, housingNeeded)
				break
			}
			housingAvailable = ds.Resources[ResHousing]
		}
	}

	// See if we have any unclaimed homes that we can assign to the family units.
	needHomeIdx := 0
	for _, members := range needsHome {
		for ; needHomeIdx < len(ds.consts); needHomeIdx++ {
			home := ds.consts[needHomeIdx]
			if !seenHome[home] {
				seenHome[home] = true
				members.assignHome(home)
				break
			}
		}

		if members.Home == nil {
			log.Printf("Tribe %s has no home for family unit: (%d members)", t.String(), len(members.Adults)+len(members.Children))
			// Check if we can construct a new home.
			if ds.CanConstruct(constHut) {
				ds.Construct(constHut)
				home := ds.consts[len(ds.consts)-1]
				members.assignHome(home)
				seenHome[home] = true
			} else if ds.CanConstruct(constTent) {
				ds.Construct(constTent)
				home := ds.consts[len(ds.consts)-1]
				members.assignHome(home)
				seenHome[home] = true
			} else {
				log.Printf("Tribe %s can't construct more housing. %d/%d", t.String(), housedPeople, t.Population)
				break
			}

			// TODO: Check if we can buy a home.
		}
	}

	// Check how much temporary housing we have in form of tents.
	// NOTE: This is really stupid. Isn't there a better way to do this?
	if t.Type > TribeTypeSettling {
		housingAvailable += t.Resources[ResHousing]
	}

	if housingAvailable < housingNeeded {
		// We need to reduce satisfaction if we don't have enough housing.
		// TODO:
		// - Also reduce satisfaction of we only have temporary housing.
		// - If we don't have enough housing, some people might die.
		t.Satisfaction.Add(float64(housingAvailable-housingNeeded) / float64(t.Population))
		log.Printf("Tribe %s needs more housing. %d (+%d tents)/%d", t.String(), housingAvailable, t.Resources[ResHousing], housingNeeded)
	}

	// TODO: If we have not enough housing, we need to reduce satisfaction.
	log.Printf("Tribe %s has %s", t.String(), ds.String())
}
