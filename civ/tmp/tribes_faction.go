package civ

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/go_gens/gengovernment"
)

// getPreferredLeadershipForm returns the preferred leadership form of the tribe.
func (t *Tribe) getPreferredLeadershipForm() gengovernment.LeadershipForm {
	if t.Type <= TribeTypeSettling {
		return gengovernment.LeadershipFormChiefdom
	}
	if t.Type == TribeTypeCity {
		return gengovernment.LeadershipFormMonarchy
	}
	if t.Type == TribeTypeCityState {
		return gengovernment.LeadershipFormRepublic
	}
	if t.Type == TribeTypeEmpire {
		return gengovernment.LeadershipFormDictatorship
	}
	return gengovernment.LeadershipFormChiefdom
}

// getPossibleLeadershipForms returns the possible leadership forms of the tribe.
func (t *Tribe) getPossibleLeadershipForms() []gengovernment.LeadershipForm {
	if t.Type <= TribeTypeSettling {
		return []gengovernment.LeadershipForm{gengovernment.LeadershipFormChiefdom}
	}
	if t.Type == TribeTypeCity {
		return []gengovernment.LeadershipForm{gengovernment.LeadershipFormMonarchy}
	}
	if t.Type == TribeTypeCityState {
		return []gengovernment.LeadershipForm{gengovernment.LeadershipFormMonarchy, gengovernment.LeadershipFormRepublic}
	}
	if t.Type == TribeTypeEmpire {
		return []gengovernment.LeadershipForm{gengovernment.LeadershipFormDictatorship, gengovernment.LeadershipFormMonarchy, gengovernment.LeadershipFormRepublic}
	}
	return []gengovernment.LeadershipForm{gengovernment.LeadershipFormChiefdom}
}

func (t *Tribe) findNaturalProgression() gengovernment.LeadershipForm {
	// TODO: Pick from the preferred leadership forms.
	var currentInfluence gengovernment.LeadershipInfluence
	var fallbackForm gengovernment.LeadershipForm
	switch t.Type {
	case TribeTypeNomadic, TribeTypeSettling:
		currentInfluence = gengovernment.LeadershipInfluenceTribe
		fallbackForm = gengovernment.LeadershipFormChiefdom
	case TribeTypeCity:
		currentInfluence = gengovernment.LeadershipInfluenceSettlement
		fallbackForm = gengovernment.LeadershipFormChiefdom
	case TribeTypeCityState:
		currentInfluence = gengovernment.LeadershipInfluenceCityState
		fallbackForm = gengovernment.LeadershipFormMonarchy
	case TribeTypeEmpire:
		currentInfluence = gengovernment.LeadershipInfluenceEmpire
		fallbackForm = gengovernment.LeadershipFormDictatorship
	default:
		log.Printf("!!!%s has an unknown tribe type: %d", t.String(), t.Type)
		return t.Leadership.Form
	}

	// Check if we need to change the leadership form.
	// There is a chance we just retain the current form.
	curMin, curMax := t.Leadership.Form.RangeInfluence()
	if curMin <= currentInfluence && curMax >= currentInfluence && rand.Intn(100) < 20 {
		return t.Leadership.Form
	}

	natProg := t.Leadership.Form.NaturalProgression()
	if len(natProg) == 0 {
		log.Printf("!!!%s has no natural progression for %s", t.String(), t.Leadership.Form)
		return fallbackForm
	}

	for _, i := range rand.Perm(len(natProg)) {
		frm := natProg[i]
		minInf, maxInf := frm.RangeInfluence()
		if minInf <= currentInfluence && maxInf >= currentInfluence {
			return frm
		}
	}
	return t.Leadership.Form
}

// findCoupProgression returns the possible progression of the leadership form through a coup.
func (t *Tribe) findCoupProgression() gengovernment.LeadershipForm {
	// TODO: Pick from the preferred leadership forms.
	var currentInfluence gengovernment.LeadershipInfluence
	var fallbackForm gengovernment.LeadershipForm
	switch t.Type {
	case TribeTypeNomadic, TribeTypeSettling:
		currentInfluence = gengovernment.LeadershipInfluenceTribe
		fallbackForm = gengovernment.LeadershipFormChiefdom
	case TribeTypeCity:
		currentInfluence = gengovernment.LeadershipInfluenceSettlement
		fallbackForm = gengovernment.LeadershipFormChiefdom
	case TribeTypeCityState:
		currentInfluence = gengovernment.LeadershipInfluenceCityState
		fallbackForm = gengovernment.LeadershipFormMonarchy
	case TribeTypeEmpire:
		currentInfluence = gengovernment.LeadershipInfluenceEmpire
		fallbackForm = gengovernment.LeadershipFormDictatorship
	default:
		log.Printf("!!!%s has an unknown tribe type: %d", t.String(), t.Type)
		return t.Leadership.Form
	}

	// Check if we need to change the leadership form.
	// There is a chance we just retain the current form.
	curMin, curMax := t.Leadership.Form.RangeInfluence()
	if curMin <= currentInfluence && curMax >= currentInfluence && rand.Intn(100) < 20 {
		return t.Leadership.Form
	}

	natProg := t.Leadership.Form.CoupProgression()
	if len(natProg) == 0 {
		log.Printf("!!!%s has no natural progression for %s", t.String(), t.Leadership.Form)
		return fallbackForm
	}

	for _, i := range rand.Perm(len(natProg)) {
		frm := natProg[i]
		minInf, maxInf := frm.RangeInfluence()
		if minInf <= currentInfluence && maxInf >= currentInfluence {
			return frm
		}
	}
	return t.Leadership.Form
}

func (s *simState) handleLeadershipTribe(t *Tribe) {
	m := s.m

	// Update the factions.
	// This will trigger election of new leaders, etc.
	t.Leadership.TickTribe(t, m)
	for _, f := range t.Factions {
		f.TickTribe(t, m)
	}

	// Update existing factions.
	// - If a faction has a popularity of 0, it should be removed.
	// - Either exile, kill, or merge with another faction.
	// Factions should have independent actions, etc.
	// - Factions with high popularity unlock new actions, etc.
	// - Available action depends on sponsorship, popularity, etc.
	// - We will need to keep track of notable sponsors and members.
	// Example actions:
	// - Charitable actions (help the poor, etc.)
	// - Userper actions (try to take over the tribe)
	// - Religious actions (try to convert the tribe to a new religion)
	// - Intrigue actions (try to kill the leadership, etc.)

	t.GoverningPeople.handleLeadership(t, s.m)
}

// TickTribe the faction for one year.
func (f *Faction) TickTribe(t *Tribe, m *Civ) {
	f.Tick(t, m)

	// TODO: Add action for faction.
	// - This might be an action to reduce the popularity of a rival faction or leader.
	// - This might be an action to increase the popularity of the faction.
	// Execute a random action.
	if rand.Float64() < 0.1 {
		action := f.pickAction(t, t.GoverningPeople, m)
		if action != nil {
			action.Execute(t, m, f)
		}
	}
}

// getFactionActions returns a list of actions that a faction of a tribe can take.
func (t *Tribe) getFactionActions(m *Civ, f *Faction) []*FactionAction {
	g := t.GoverningPeople
	h := m.History

	// Depending on the form of leadership, the actions will be different and decided by different people.
	// Right now, we just have a single leader in each type of leadership.

	// TODO: The actions should depend on the personality of the faction or faction leader.
	// TODO: Can consequences be chains of actions which are then narrated?
	// There is a tree or sequence of actions, which will be played out and the
	// resulting chain of events will be used to generate the narrative.
	// We can store the chain of events to store the whole sequence of events.
	actionsLeaderNegative := []*FactionAction{{
		probability: func(fac *Faction) float64 { return 0.1 },
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitDeceptive) // && !leader.Traits.HasTrait(geneticshuman.TraitCareful)
		},
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"spy", "traitor", "criminal", "fraud"}
			flavor := flavors[rand.Intn(len(flavors))]

			// Update the popularity of the faction.
			fac.Popularity.Add(-0.1)

			// Update the popularity of the leader.
			fac.Leader.Popularity.Add(-0.2)

			// Add a history entry.
			historyMsg := facActionString("[LEADER] of faction [FACTION] is discovered to be a [FLAVOR].", fac, flavor)
			h.AddEvent("Scandal (Faction)", historyMsg, fac.Ref())

			oldLeader := fac.ChooseNewLeader(t, fac, m)
			fac.ExecutePerson(oldLeader, "treason", m)
		},
	}, {
		probability: func(fac *Faction) float64 { return 0.1 },
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitDeceptive) // && !leader.Traits.HasTrait(geneticshuman.TraitCareful)
		},
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"stealing", "lying", "abusing power"}
			flavor := flavors[rand.Intn(len(flavors))]

			leader := fac.Leader
			detectionChance := 0.5
			if leader.Traits.HasTrait(geneticshuman.TraitCareful) {
				detectionChance = 0.2
			} else if leader.Traits.HasTrait(geneticshuman.TraitCareless) {
				detectionChance = 0.8
			}

			// Check if the scandal is discovered.
			if rand.Float64() < detectionChance {
				// The leader is discovered and will be replaced.
				// Update the popularity of the faction.
				fac.Popularity.Add(-0.2)

				// Update the popularity of the leader.
				fac.Leader.Popularity.Add(-0.3)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has been caught %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Scandal (Faction)", historyMsg, fac.Ref())

				oldLeader := fac.ChooseNewLeader(t, fac, m)
				fac.ExecutePerson(oldLeader, "corruption", m)
			} else {
				// The leader will be able to cover up the scandal.
				// Update the popularity of the faction.
				fac.Popularity.Add(-0.1)

				// Update the popularity of the leader.
				fac.Leader.Popularity.Add(-0.1)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has almost been caught %s but managed to cover it up.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Scandal (Faction)", historyMsg, fac.Ref())
			}
		},
	}, {
		probability: func(fac *Faction) float64 { return 0.1 },
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			if leader.Traits.HasTrait(geneticshuman.TraitKind) || leader.Traits.HasTrait(geneticshuman.TraitContent) {
				return false
			}
			// TODO: Check if the leader has enough foresight to check if they have a chance of success.
			return (leader.Traits.HasTrait(geneticshuman.TraitAmbitious) || leader.Traits.HasTrait(geneticshuman.TraitDeceptive)) && fac != g.Leadership
		},
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"coup", "rebellion", "uprising"}
			flavor := flavors[rand.Intn(len(flavors))]

			leader := fac.Leader
			detectionChance := 0.5
			if leader.Traits.HasTrait(geneticshuman.TraitCareful) {
				detectionChance = 0.2
			} else if leader.Traits.HasTrait(geneticshuman.TraitCareless) {
				detectionChance = 0.8
			}

			// Check if the plan is discovered.
			if rand.Float64() < detectionChance {
				// The plan is discovered and will be stopped.
				// Update the popularity of the faction and the leader.
				fac.Popularity.Add(-0.2)
				fac.Leader.Popularity.Add(-0.3)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has been caught planning a violent %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Intrigue (Faction)", historyMsg, fac.Ref())
			} else {
				// The plan will be executed.
				// Update the popularity of the faction.
				fac.Popularity.Add(0.2)

				// Update the popularity of the leader.
				fac.Leader.Popularity.Add(0.3)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully executed a violent %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Uprising (Faction)", historyMsg, fac.Ref())

				// The faction will take over.
				// Depending on chance and popularity, it might kill the leadership, or simply replace it.
				// If the faction is more popular than the leadership, it will simply replace it.
				executeCoup(t, g, fac, m)
			}
		},
	}}
	actionsLeaderPositive := []*FactionAction{{
		probability: func(fac *Faction) float64 { return 0.1 },
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitAmbitious) && !leader.Traits.HasTrait(geneticshuman.TraitDeceptive)
		},
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"hero", "saint", "genius", "visionary"}
			flavor := flavors[rand.Intn(len(flavors))]

			// Update the popularity of the faction.
			fac.Popularity.Add(0.1)

			// Update the popularity of the leader.
			fac.Leader.Popularity.Add(0.3)

			// Add a history entry.
			historyMsg := facActionString("[LEADER] of faction [FACTION] is discovered to be a [FLAVOR].", fac, flavor)
			h.AddEvent("Scandal (Faction)", historyMsg, fac.Ref())
		},
	}, {
		probability: func(fac *Faction) float64 { return 0.1 },
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitAmbitious) && !leader.Traits.HasTrait(geneticshuman.TraitDeceptive)
		},
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"charity", "innovation", "reform"}
			flavor := flavors[rand.Intn(len(flavors))]

			// Update the popularity of the faction.
			fac.Popularity.Add(0.1)

			// Update the popularity of the leader.
			fac.Leader.Popularity.Add(0.2)

			if flavor == "reform" {
				// Change the type of the faction.
				newType := FactionType(rand.Intn(int(FactionTypeMax)))
				for newType == fac.Type {
					newType = FactionType(rand.Intn(int(FactionTypeMax)))
				}
				oldType := fac.ChangeType(newType, f.LeaderPreferredForm(), h)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has reformed the faction from %s to %s.", fac.ID, fac.Leader.Name(), fac.Name, oldType.String(), newType.String())
				h.AddEvent("Reform (Faction)", historyMsg, fac.Ref())
			} else {
				// Add a history entry.
				historyMsg := facActionString("[LEADER] of faction [FACTION] is pursuing [FLAVOR].", fac, flavor)
				h.AddEvent("Innovation (Faction)", historyMsg, fac.Ref())
			}
		},
	}}
	actionsLeaderManipulative := []*FactionAction{{
		probability: func(fac *Faction) float64 { return 0.1 },
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitAmbitious) && (leader.Traits.HasTrait(geneticshuman.TraitDeceptive) || leader.Traits.HasTrait(geneticshuman.TraitCruel))
		},
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"hatred", "fear", "distrust"}
			flavor := flavors[rand.Intn(len(flavors))]

			// TODO: Select rival faction.
			rivalIsLeader := true
			rival := g.Leadership
			if rival == fac { // Select a different faction.
				rivalIsLeader = false
				for _, i := range rand.Perm(len(g.Factions)) {
					if g.Factions[i] != fac {
						rival = g.Factions[i]
						break
					}
				}
			}

			// There is a chance that this plan will backfire and our popularity will decrease
			// while the rival faction will gain popularity.
			// The lower the popularity of the rival faction, the higher the chance that this plan will backfire.
			if rand.Float64() > float64(fac.Popularity)*0.9 {
				// The rival faction will become more popular and we will lose popularity.
				rival.Popularity.Add(0.3)
				if rivalIsLeader {
					g.Satisfaction.Add(0.1)
				}
				fac.Popularity.Add(-0.3)

				// Reduce the popularity of the leader.
				fac.Leader.Popularity.Add(-0.2)

				// Increase the popularity of the rival faction leader.
				rival.Leader.Popularity.Add(0.3)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s tried to stoke %s against faction %d's leader %s of %s but failed.", fac.ID, fac.Leader.Name(), fac.Name, flavor, rival.ID, rival.Leader.Name(), rival.Name)
				h.AddEvent("Intrigue (Faction)", historyMsg, fac.Ref())
			} else {
				// The rival faction will lose popularity and we will gain popularity.
				rival.Popularity.Add(-0.3)
				if rivalIsLeader {
					g.Satisfaction.Add(-0.1)
				}
				fac.Popularity.Add(0.3)

				// Increase the popularity of the leader.
				fac.Leader.Popularity.Add(0.2)

				// Reduce the popularity of the rival faction leader.
				rival.Leader.Popularity.Add(-0.3)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully stoked %s against faction %d's leader %s of %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor, rival.ID, rival.Leader.Name(), rival.Name)
				h.AddEvent("Intrigue (Faction)", historyMsg, fac.Ref())
			}
		},
	}, {
		probability: func(fac *Faction) float64 { return 0.1 },
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"assassination", "extortion"}
			flavor := flavors[rand.Intn(len(flavors))]

			// TODO: Depending on the traits of the leader or the target, the chance of success might change.
			// Carelessness might lead to failure, while carefulness might lead to success.
			// Paranoid leaders might be harder to assassinate, while trusting leaders might be easier.
			// Depending how clever "we" are, we might choose a target that is easier to extort or assassinate,
			// alternatively we choose who we like least. If we are cruel, we might choose the most popular target.
			switch flavor {
			case "extortion":
				// There is a chance that the extortion will be successful.
				// Find a target for the extortion.
				rivalFaction := g.Leadership
				if rivalFaction == fac {
					for _, i := range rand.Perm(len(g.Factions)) {
						if g.Factions[i] != fac {
							rivalFaction = g.Factions[i]
							break
						}
					}
				}
				// There is a chance that the extortion will be successful.
				origin := fac.Leader
				target := rivalFaction.Leader
				if rand.Float64() > float64(rivalFaction.Popularity)*0.9 {
					// The extortion was successful.
					// For now, we just reduce the popularity of the rival faction.
					rivalFaction.Popularity.Add(-0.3)
					fac.Popularity.Add(0.3)

					// Reduce the popularity of the rival faction leader.
					target.Popularity.Add(-0.2)

					// Increase the popularity of the faction leader.
					origin.Popularity.Add(0.1)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully extorted faction %d's leader %s of %s.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					h.AddEvent("Extortion (Faction)", historyMsg, fac.Ref())
				} else {
					// The extortion failed.
					// Increase the popularity of the rival faction leader and decrease the popularity of the faction leader.
					rivalFaction.Popularity.Add(0.3)
					fac.Popularity.Add(-0.3)

					// Decrease the popularity of the leader.
					origin.Popularity.Add(-0.2)

					// Increase the popularity of the target.
					target.Popularity.Add(0.2)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has tried to extort faction %d's leader %s of %s but failed.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					h.AddEvent("Extortion (Faction)", historyMsg, fac.Ref())
				}
			case "assassination":
				// There is a chance that the assassination will be successful.
				// Find a target for the assassination.
				rivalFaction := g.Leadership
				if rivalFaction == fac {
					for _, i := range rand.Perm(len(g.Factions)) {
						if g.Factions[i] != fac {
							rivalFaction = g.Factions[i]
							break
						}
					}
				}
				// There is a chance that the assassination will be successful.
				origin := fac.Leader
				target := rivalFaction.Leader
				if rand.Float64() > float64(rivalFaction.Popularity)*0.9 {
					// The assassination was successful.
					// Increase the popularity of the faction.
					fac.Popularity.Add(0.1)

					// Increase the popularity of the leader.
					fac.Leader.Popularity.Add(0.2)

					m.killPerson(target, "assassination by faction "+f.Name)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully assassinated faction %d's leader %s of %s.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					h.AddEvent("Assassination (Faction)", historyMsg, fac.Ref())
				} else {
					// The assassination failed.
					// Decrease the popularity of the faction.
					fac.Popularity.Add(-0.2)

					// Decrease the popularity of the leader.
					fac.Leader.Popularity.Add(-0.3)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has tried to assassinate faction %d's leader %s of %s but failed.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					event := h.AddEvent("Assassination (Faction)", historyMsg, fac.Ref())
					// TODO: There is a chance that the target will become aware of who tried to assassinate them.
					if rand.Float64() > 0.5 {
						target.Opinions.AddOpinion(origin, -0.5, event)
					}
				}
			default:
				if rand.Float64() > 0.5 {
					// The action was successful.
					// Increase the popularity of the faction.
					fac.Popularity.Add(0.1)

					// Increase the popularity of the leader.
					fac.Leader.Popularity.Add(0.1)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully organized a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
					h.AddEvent("Criminal (Faction)", historyMsg, fac.Ref())
				} else {
					// The action failed.
					// Decrease the popularity of the faction.
					fac.Popularity.Add(-0.2)

					// Decrease the popularity of the leader.
					fac.Leader.Popularity.Add(-0.3)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has tried to organize a %s but failed.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
					h.AddEvent("Criminal (Faction)", historyMsg, fac.Ref())
				}
			}
		},
	}}
	actionsFactionReligious := []*FactionAction{{
		probability: func(fac *Faction) float64 { return 0.1 },
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"turning wine into water", "making a piglet talk", "walking on beer"}
			flavor := flavors[rand.Intn(len(flavors))]

			leaderIsCharlatan := false
			if rand.Float64() > 0.9 {
				leaderIsCharlatan = true
			}

			// If the leader is a charlatan, there is a higher chance that the miracle will fail.
			threshold := float64(fac.Popularity)
			if leaderIsCharlatan {
				threshold *= 0.7
			}
			if rand.Float64() > threshold {
				// The miracle failed.
				fac.Popularity.Add(-0.1)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s tried to perform a miracle by %s but failed.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Religious (Faction)", historyMsg, fac.Ref())

				// TODO: If the leader is a charlatan, there is a chance that the leader will be exposed, depending on the popularity of the faction.
				if leaderIsCharlatan && rand.Float64() > float64(fac.Popularity)*0.9 {
					// Decrease the popularity of the leader.
					fac.Leader.Popularity.Add(-0.5)

					// The leader was exposed.
					oldLeader := fac.ChooseNewLeader(t, fac, m)
					fac.ExecutePerson(oldLeader, "charlatanism", m)
				} else {
					// Decrease the popularity of the leader.
					fac.Leader.Popularity.Add(-0.1)
				}
			} else {
				// The miracle was successful.
				fac.Popularity.Add(0.3)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s performed a miracle by %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Religious (Faction)", historyMsg, fac.Ref())
			}
		},
	}, {
		probability: func(fac *Faction) float64 { return 0.1 },
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"end of the world", "coming of the savior", "return of the gods"}
			flavor := flavors[rand.Intn(len(flavors))]

			// Increase the popularity of the faction.
			fac.Popularity.Add(0.1)

			// Increase the popularity of the leader.
			fac.Leader.Popularity.Add(0.1)

			// Add a history entry.
			historyMsg := facActionString("Leader [LEADER] of faction [FACTION] is preaching about the [FLAVOR].", fac, flavor)
			h.AddEvent("Religious (Faction)", historyMsg, fac.Ref())
		},
	}, {
		probability: func(fac *Faction) float64 { return 0.1 },
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"festival", "pilgrimage", "sacrifice", "ceremony", "ritual"}
			flavor := flavors[rand.Intn(len(flavors))]

			// Just for fun, if it is a ceremony, there is a chance a freak accident will happen.
			// Like: The leader being hit by lightning, a statue falling on someone, etc.
			// Maybe a portal to another dimension opens up and something comes through and eats someone.
			misfortuneChance := 0.1

			// If the leader is a charlatan, there is a higher chance that the miracle will fail
			// or that something bad will happen.
			if fac.Leader.Traits.HasTrait(geneticshuman.TraitDeceptive) {
				misfortuneChance = 0.4
			}

			misfortune := rand.Float64() < misfortuneChance
			if !misfortune {
				// The ceremony was successful.
				// Increase the popularity of the faction.
				fac.Popularity.Add(0.1)

				// Increase the popularity of the leader.
				fac.Leader.Popularity.Add(0.1)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s organized a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Religious (Faction)", historyMsg, fac.Ref())
				return
			}
			switch flavor {
			case "ceremony", "ritual":
				// A freak accident happened.
				const (
					IncidentTypeLightning = iota
					IncidentTypeStatue
					IncidentTypePortal
				)
				incidentType := rand.Intn(3)
				switch incidentType {
				case IncidentTypeLightning:
					// The popularity of the faction will increase, because it was funny.
					fac.Popularity.Add(0.3)

					// The leader was hit by lightning.
					m.killPerson(fac.Leader, "hit by lightning during "+flavor)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s was hit by lightning during a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
					h.AddEvent("Misfortune (Faction)", historyMsg, fac.Ref())
				case IncidentTypeStatue:
					// A statue fell on someone.
					// The popularity of the faction will decrease, because it was tragic.
					fac.Popularity.Add(-0.3)

					// Decrease the popularity of the leader.
					fac.Leader.Popularity.Add(-0.2)

					// Find a random person to kill.
					people := t.GetPeople()
					victim := people[rand.Intn(len(people))]
					m.killPerson(victim, "hit by falling statue during "+flavor)

					// Add a history entry.
					historyMsg := fmt.Sprintf("A statue fell on %s during a %s organized by faction %d's leader %s of %s.", victim.Name(), flavor, fac.ID, fac.Leader.Name(), fac.Name)
					h.AddEvent("Misfortune (Faction)", historyMsg, fac.Ref())
				case IncidentTypePortal:
					// A portal to another dimension opened up.
					// The popularity of the faction will decrease, because it was tragic.
					fac.Popularity.Add(-0.3)

					// Decrease the popularity of the leader.
					fac.Leader.Popularity.Add(-0.2)

					// Pick a random monster to come through the portal.
					const (
						MonsterTypeDemon = iota
						MonsterTypeAlien
						MonsterTypeEldritch
					)
					monsterType := rand.Intn(3)
					monsterName := "unknown monster"
					switch monsterType {
					case MonsterTypeDemon:
						monsterName = "demon"
					case MonsterTypeAlien:
						monsterName = "alien"
					case MonsterTypeEldritch:
						monsterName = "eldritch horror"
					}
					// Find a random person to kill.
					people := t.GetPeople()
					victim := people[rand.Intn(len(people))]
					m.killPerson(victim, "eaten by "+monsterName+" from another dimension")
					// Add a history entry.
					historyMsg := fmt.Sprintf("A portal to another dimension opened up during a %s organized by faction %d's leader %s of %s and %s came through and ate %s.", flavor, fac.ID, fac.Leader.Name(), fac.Name, monsterName, victim.Name())
					h.AddEvent("Misfortune (Faction)", historyMsg, fac.Ref())
				}
			case "sacrifice":
				// A freak accident happened.
				// Increase the popularity of the faction, because it was funny.
				fac.Popularity.Add(0.1)

				// Decrease the popularity of the leader.
				fac.Leader.Popularity.Add(-0.1)

				// Pick an animal to sacrifice.
				animals := []string{
					"sheep", "goat", "pig", "cow", "chicken", "duck", "goose", "turkey", "rabbit", "dog", "cat", "horse", "donkey", "elephant", "rhinoceros", "hippopotamus", "giraffe", "zebra", "lion", "tiger", "bear", "wolf", "fox", "deer", "moose", "elk", "buffalo", "bison", "antelope", "gazelle", "impala", "kudu", "oryx", "springbok", "ibex", "chamois", "goral", "tahr", "muskox", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison",
				}
				animal := animals[rand.Intn(len(animals))]

				// Add a body part to be bitten by the animal.
				bodyParts := []string{
					"hand", "foot", "leg", "arm", "head", "ear", "nose", "eye", "mouth", "tongue", "cheek", "chin", "forehead", "neck", "shoulder", "back", "chest", "stomach", "belly", "waist", "hip", "buttocks", "thigh", "knee", "calf", "ankle", "heel", "toe", "finger", "thumb", "nail", "palm", "wrist", "elbow", "forearm", "armpit", "breast", "nipple", "heart", "lung", "liver", "kidney", "bladder", "intestine", "stomach", "spleen", "pancreas", "appendix", "brain", "skull", "rib", "spine", "pelvis", "hip", "thorax", "abdomen", "groin", "genitals",
				}
				bodyPart := bodyParts[rand.Intn(len(bodyParts))]

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s was bitten by a %s during a %s on the %s.", fac.ID, fac.Leader.Name(), fac.Name, animal, flavor, bodyPart)
				h.AddEvent("Misfortune (Faction)", historyMsg, fac.Ref())
			default:
				// The ceremony was successful.
				// Increase the popularity of the faction.
				fac.Popularity.Add(0.1)

				// Increase the popularity of the leader.
				fac.Leader.Popularity.Add(0.1)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s organized a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Religious (Faction)", historyMsg, fac.Ref())
			}
		},
	}}
	actionsFactionMartial := []*FactionAction{{
		probability: func(fac *Faction) float64 { return 0.1 },
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitAggressive)
		},
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"purge of the weak", "mandatory conscription"}
			flavor := flavors[rand.Intn(len(flavors))]

			// For this to have a positive effect, the faction must have extremely high popularity.
			if rand.Float64() > float64(fac.Popularity)*0.9 {
				// This action will increase the popularity of the faction.
				fac.Popularity.Add(0.4 * rand.Float64())

				// Increase the popularity of the leader.
				fac.Leader.Popularity.Add(0.2)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has initiated a %s which was welcomed by the members.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Martial (Faction)", historyMsg, fac.Ref())
			} else {
				// This action will decrease the popularity of the faction.
				fac.Popularity.Add(-0.4 * rand.Float64())

				// Decrease the popularity of the leader.
				fac.Leader.Popularity.Add(-0.2)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has initiated a %s which was met with resistance.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Martial (Faction)", historyMsg, fac.Ref())
			}
		},
	}, {
		probability: func(fac *Faction) float64 { return 0.1 },
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitAggressive) && leader.Traits.HasTrait(geneticshuman.TraitAmbitious)
		},
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"war", "raid", "battle", "training", "tournament"}
			flavor := flavors[rand.Intn(len(flavors))]

			//Increase the popularity of the faction.
			fac.Popularity.Add(0.1)

			// Increase the popularity of the leader.
			fac.Leader.Popularity.Add(0.1)

			// Add a history entry.
			historyMsg := facActionString("Leader [LEADER] of faction [FACTION] is preparing for a [FLAVOR].", fac, flavor)
			h.AddEvent("Martial (Faction)", historyMsg, fac.Ref())
		},
	}}
	actionsFactionCivil := []*FactionAction{{
		probability: func(fac *Faction) float64 { return 0.1 },
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"festival", "market", "election", "celebration", "meeting"}
			flavor := flavors[rand.Intn(len(flavors))]

			// Increase the popularity of the faction.
			fac.Popularity.Add(0.1)

			// Increase the popularity of the leader.
			fac.Leader.Popularity.Add(0.1)

			// Add a history entry.
			historyMsg := facActionString("Leader [LEADER] of faction [FACTION] is organizing a [FLAVOR].", fac, flavor)
			h.AddEvent("Martial (Faction)", historyMsg, fac.Ref())
		},
	}}
	actionsFactionCriminal := []*FactionAction{{
		probability: func(fac *Faction) float64 { return 0.1 },
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"raid", "heist", "smuggling", "assassination", "extortion"}
			flavor := flavors[rand.Intn(len(flavors))]
			switch flavor {
			case "assassination":
				// Find a target for the assassination.
				rivalFaction := g.Leadership
				if rivalFaction == fac {
					for _, i := range rand.Perm(len(g.Factions)) {
						if g.Factions[i] != fac {
							rivalFaction = g.Factions[i]
							break
						}
					}
				}
				origin := fac.Leader
				target := rivalFaction.Leader

				// There is a chance that the assassination will be successful.
				if rand.Float64() > float64(rivalFaction.Popularity)*0.9 {
					// The assassination was successful.
					// Increase the popularity of the faction.
					fac.Popularity.Add(0.1)

					// Increase the popularity of the leader.
					fac.Leader.Popularity.Add(0.2)

					// Decrease the popularity of the target faction.
					rivalFaction.Popularity.Add(-0.3)

					// Kill the target.
					m.killPerson(target, "assassination by faction "+f.Name)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully assassinated faction %d's leader %s of %s.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					h.AddEvent("Assassination (Faction)", historyMsg, fac.Ref())
				} else {
					// The assassination failed.
					// Decrease the popularity of the faction.
					fac.Popularity.Add(-0.2)

					// Decrease the popularity of the leader.
					fac.Leader.Popularity.Add(-0.3)

					// Increase the popularity of the target faction.
					rivalFaction.Popularity.Add(0.1)

					// Increase the popularity of the target.
					target.Popularity.Add(0.2)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has tried to assassinate faction %d's leader %s of %s but failed.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					event := h.AddEvent("Assassination (Faction)", historyMsg, fac.Ref())
					// TODO: There is a chance that the target will become aware of who tried to assassinate them.
					if rand.Float64() > 0.5 {
						target.Opinions.AddOpinion(origin, -0.5, event)
					}
				}
			default:
				if rand.Float64() > 0.5 {
					// The action was successful.
					// Increase the popularity of the faction.
					fac.Popularity.Add(0.1)

					// Increase the popularity of the leader.
					fac.Leader.Popularity.Add(0.1)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully organized a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
					h.AddEvent("Criminal (Faction)", historyMsg, fac.Ref())
				} else {
					// The action failed.
					// Decrease the popularity of the faction.
					fac.Popularity.Add(-0.2)

					// Decrease the popularity of the leader.
					fac.Leader.Popularity.Add(-0.3)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has tried to organize a %s but failed.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
					h.AddEvent("Criminal (Faction)", historyMsg, fac.Ref())
				}
			}
		},
	}}
	actionsFactionMerchant := []*FactionAction{{
		probability: func(fac *Faction) float64 { return 0.1 },
		consequences: func(t peopleThing, m *Civ, fac *Faction) {
			// Pick a flavor.
			flavors := []string{"trade", "market", "caravan", "deal", "negotiation"}
			flavor := flavors[rand.Intn(len(flavors))]

			// Increase the popularity of the faction.
			fac.Popularity.Add(0.1)

			// Increase the popularity of the leader.
			fac.Leader.Popularity.Add(0.1)

			// Add a history entry.
			historyMsg := facActionString("Leader [LEADER] of faction [FACTION] is organizing a [FLAVOR].", fac, flavor)
			h.AddEvent("Merchant (Faction)", historyMsg, fac.Ref())
		},
	}}

	// Pick an action based on the type of the faction.
	var actions []*FactionAction
	switch f.Type {
	case FactionTypeReligious:
		actions = actionsFactionReligious
	case FactionTypeMartial:
		actions = actionsFactionMartial
	case FactionTypeCivil:
		actions = actionsFactionCivil
	case FactionTypeCriminal:
		actions = actionsFactionCriminal
	case FactionTypeMerchant:
		actions = actionsFactionMerchant
	}

	actions = append(actions, actionsLeaderNegative...)
	actions = append(actions, actionsLeaderPositive...)
	actions = append(actions, actionsLeaderManipulative...)
	return actions
}
