package genworldvoronoi

import (
	"fmt"
	"log"
	"math/rand"
	"sort"

	"github.com/Flokey82/genetics"
	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/go_gens/gameconstants"
	"github.com/Flokey82/go_gens/genlanguage"
)

func (m *Civ) getNextPersonID() int {
	m.nextPersonID++
	return m.nextPersonID
}

// PersonalAction represents an action that a person can take.
type PersonalAction struct {
	probability  func(*Person) float64
	requires     func(*Person) bool
	consequences func(m *Civ, p *Person)
}

func (a *PersonalAction) Execute(m *Civ, p *Person) {
	if a.consequences != nil {
		a.consequences(m, p)
	}
}

// pickAction picks an action based on the probabilities of the actions.
func pickAction(p *Person, actions []*PersonalAction) *PersonalAction {
	totalProbability := 0.0
	for _, action := range actions {
		probability := action.probability(p)
		totalProbability += probability
	}

	randomValue := rand.Float64() * totalProbability
	for _, action := range actions {
		probability := action.probability(p)
		randomValue -= probability
		if randomValue < 0 {
			return action
		}
	}
	// Should never reach here (all probabilities should be covered)
	return nil
}

// tickPerson advances the person by nDays and returns any new born child.å
// TODO: Twins, triplets, etc.
func (m *Civ) tickPerson(p *Person, nDays int, cf func(int) *Culture) *Person {
	if !m.doesPersonExist(p) {
		return nil
	}
	// Calculate age.
	m.tickPersonAge(p, nDays)

	// Advance pregnancy.
	var child *Person
	if p.Prengancy != nil {
		child = m.tickPersonPregnancy(p, nDays, cf)
	}

	// Check if person dies of natural causes.
	m.tickPersonDeath(p, nDays)

	// If the person is dead, we don't need to do anything else.
	if p.Death.IsSet() {
		return nil
	}

	type relation struct {
		p                *Person
		relationToPerson string
		relationOfPerson string
	}

	getSiblingRelationString := func(p *Person) string {
		switch p.Gender() {
		case geneticshuman.GenderFemale:
			return "sister"
		case geneticshuman.GenderMale:
			return "brother"
		default:
			return "sibling"
		}
	}
	getParentRelationString := func(p *Person) string {
		switch p.Gender() {
		case geneticshuman.GenderFemale:
			return "mother"
		case geneticshuman.GenderMale:
			return "father"
		default:
			return "parent"
		}
	}
	getChildRelationString := func(p *Person) string {
		switch p.Gender() {
		case geneticshuman.GenderFemale:
			return "daughter"
		case geneticshuman.GenderMale:
			return "son"
		default:
			return "child"
		}
	}
	getSpouseRelationString := func(p *Person) string {
		switch p.Gender() {
		case geneticshuman.GenderFemale:
			return "wife"
		case geneticshuman.GenderMale:
			return "husband"
		default:
			return "spouse"
		}
	}
	pickRelatives := func() []relation {
		var options []relation
		relChild := getChildRelationString(p)
		if len(p.Children) > 0 {
			relParent := getParentRelationString(p)
			for _, c := range p.Children {
				if c.Dead() {
					continue
				}
				options = append(options, relation{
					p:                c,
					relationToPerson: relParent,
					relationOfPerson: getChildRelationString(c),
				})
			}
		}
		if p.Spouse != nil && !p.Spouse.Dead() {
			options = append(options, relation{
				p:                p.Spouse,
				relationToPerson: getSpouseRelationString(p),
				relationOfPerson: getSpouseRelationString(p.Spouse),
			})
		}

		// Our relationship to our siblings. (brother, sister, sibling)
		seen := make(map[*Person]bool)
		relSibling := getSiblingRelationString(p)
		getSiblingsAndParent := func(parent *Person) []relation {
			var options []relation
			if !parent.Dead() {
				options = append(options, relation{
					p:                parent,
					relationToPerson: relChild,
					relationOfPerson: getParentRelationString(parent),
				})
			}
			// Add parent's children.
			for _, c := range parent.Children {
				if c == p || seen[c] || c.Dead() {
					continue
				}
				seen[c] = true
				relSiblingHere := relSibling
				siblingRel := getSiblingRelationString(c)
				if c.Father != p.Father && c.Mother != p.Mother {
					relSiblingHere = "step-" + relSibling
					siblingRel = "step-" + siblingRel
				} else if c.Father != p.Father || c.Mother != p.Mother {
					relSiblingHere = "half-" + relSibling
					siblingRel = "half-" + siblingRel
				}
				options = append(options, relation{
					p:                c,
					relationToPerson: relSiblingHere,
					relationOfPerson: siblingRel,
				})
			}
			return options
		}

		// Add mother and children.
		if p.Mother != nil {
			options = append(options, getSiblingsAndParent(p.Mother)...)
		}

		// Add father and children.
		if p.Father != nil {
			options = append(options, getSiblingsAndParent(p.Father)...)
		}
		return options
	}
	pickVictim := func() relation {
		options := pickRelatives()
		// TODO: Add siblings, cousins, etc.
		if len(options) == 0 {
			// Pick a random person.
			// TODO: Make sure we don't pick ourselves.
			options = append(options, relation{m.People[rand.Intn(len(m.People))], "random person", ""})
		}
		// Sort options by opinion.
		// This is sort of a "who do we hate the most" function.
		sort.Slice(options, func(i, j int) bool {
			return p.Opinions.GetOpinion(options[i].p) < p.Opinions.GetOpinion(options[j].p)
		})
		// TODO: Maybe pick a random from the top 3.
		return options[0]
	}

	// Check who we hate in our family.
	var hated, loved []relation
	for _, c := range pickRelatives() {
		// Check if we hate the person.
		if p.Opinions.GetOpinion(c.p) < -0.5 {
			hated = append(hated, c)
		} else if p.Opinions.GetOpinion(c.p) > 0.5 {
			loved = append(loved, c)
		}
	}

	nemesis := p.Opinions.GetNemesis()
	if nemesis != nil {
		// We have a nemesis.
		m.AddEvent("Nemesis", fmt.Sprintf("%s has a nemesis: %s (%.2f)", p.String(), nemesis.String(), p.Opinions.GetOpinion(nemesis)), p.Ref())
	}

	// Add an entry to the history.
	if len(hated) > 0 {
		var str string
		for _, h := range hated {
			str += fmt.Sprintf("%s (%s %.2f), ", h.p.String(), h.relationOfPerson, p.Opinions.GetOpinion(h.p))
		}
		str = str[:len(str)-2]
		m.AddEvent("Hate", fmt.Sprintf("%s hates %s", p.String(), str), p.Ref())
	}
	if len(loved) > 0 {
		var str string
		for _, h := range loved {
			str += fmt.Sprintf("%s (%s %.2f), ", h.p.String(), h.relationOfPerson, p.Opinions.GetOpinion(h.p))
		}
		str = str[:len(str)-2]
		m.AddEvent("Love", fmt.Sprintf("%s loves %s", p.String(), str), p.Ref())
	}

	// There should be different actions for each person.
	// - A work action
	// - A personal action
	// - A family action
	// - A faction action

	// TODO: The events that we have defined down there should be separated into mundane
	// events and life altering events.
	// - Mundane events should be things that happen every day and have a small impact.
	// - Life altering events should be things that happen once in a while and have a big impact.
	// - The events should be based on the person's traits, responsibilities, etc.

	// Adults:
	// - Bad people might do bad things.
	//  - Mistreat family members
	//  - Bully others
	//  - Attack others
	//  - Cheat
	// - Good people might do good things.
	//  - Help family members, spend time with them
	//  - Help others
	//  - Volunteer
	// - All people might do various things
	//  - Go on an adventure
	//  - Explore
	//  - Socialize
	//  - Hobby, art, music, etc.

	// During these activities, there is a chance that something life altering might happen.

	// An action has an observable outcome, and might have consequences.

	// Personal decisions.
	// Depending on the number of days ticked, we select from a list of possible actions.
	// There are some actions that happen once a year and have more impact, and some that
	// can happen every day with less impact.
	// If we tick a person for a year, we pick a random event that happens once a year and
	// disregard the daily events.
	// The actions should also depend on the responsibilites, personality, traits, etc.
	// For example, a person that is the leader of a faction might have different actions
	// than a person that is a farmer.
	getLivingRelatives := func(p *Person) []*Person {
		var rels []*Person
		for _, c := range p.Children {
			if c.Dead() {
				continue
			}
			rels = append(rels, c)
		}
		if p.Spouse != nil && !p.Spouse.Dead() {
			rels = append(rels, p.Spouse)
		}
		seen := make(map[*Person]bool)
		if p.Mother != nil {
			if !p.Mother.Dead() {
				rels = append(rels, p.Mother)
			}
			// Add mother's children.
			for _, c := range p.Mother.Children {
				if c == p || seen[c] || c.Dead() {
					continue
				}
				seen[c] = true
				rels = append(rels, c)
			}
		}
		if p.Father != nil && !p.Father.Dead() {
			rels = append(rels, p.Father)
			// Add father's children.
			for _, c := range p.Father.Children {
				if c == p || seen[c] || c.Dead() {
					continue
				}
				seen[c] = true
				rels = append(rels, c)
			}
		}
		return rels
	}

	dislikesGoodDeeds := func(p *Person) bool {
		// Check if the person dislikes good deeds.
		return p.Traits.HasTrait(geneticshuman.TraitCruel) || p.Traits.HasTrait(geneticshuman.TraitAggressive)
	}

	dislikesBadDeeds := func(p *Person) bool {
		// Check if the person dislikes bad deeds.
		return !p.Traits.HasTrait(geneticshuman.TraitCruel) && !p.Traits.HasTrait(geneticshuman.TraitAggressive)
	}

	changeOptsBadDeeds := func(p *Person, changeDislike, changeLike float64, ev *Event) {
		// TODO: This should be more specific and depend on the people's values, traits, etc.
		// Get all living relatives.
		rels := getLivingRelatives(p)
		// Change opinion for all relatives.
		for _, c := range rels {
			if dislikesBadDeeds(c) {
				c.Opinions.AddOpinion(p, changeDislike, ev)
			} else {
				c.Opinions.AddOpinion(p, changeLike, ev)
			}
		}
	}

	changeOptsGoodDeeds := func(p *Person, changeLike, changeDislike float64, ev *Event) {
		// TODO: This should be more specific and depend on the people's values, traits, etc.
		// Get all living relatives.
		rels := getLivingRelatives(p)
		// Change opinion for all relatives.
		for _, c := range rels {
			if dislikesGoodDeeds(c) {
				c.Opinions.AddOpinion(p, changeDislike, ev)
			} else {
				c.Opinions.AddOpinion(p, changeLike, ev)
			}
		}
	}

	actionMurder := &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.0
			// Primary trait.
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				prob += 0.5
			}
			// Secondary traits.
			if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
				prob += 0.2
			}
			// Negative traits.
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob -= 0.3
			}
			if p.Traits.HasTrait(geneticshuman.TraitCareful) {
				prob -= 0.2
			}
			if prob < 0 {
				prob = 0
			}
			return prob
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			// Pick a possible victim.
			victim := pickVictim()
			// We might attack someone and potentially kill them.
			if rand.Intn(100) < 50 {
				methods := []string{"stabbed", "shot", "poisoned", "strangled", "drowned", "burned", "beaten"}
				method := methods[rand.Intn(len(methods))]
				// Kill the person.
				ev := m.killPerson(victim.p, fmt.Sprintf("(%s) being %s by %s (%s)", victim.relationOfPerson, method, p.Name(), victim.relationToPerson))
				// Add opinion for all relatives.
				if !p.Traits.HasTrait(geneticshuman.TraitCareful) {
					changeOptsBadDeeds(p, -0.9, -0.1, ev)
				}
			} else {
				// Add history event.
				ev := m.AddEvent("Cruelty", fmt.Sprintf("%s (%s) was attacked by %s (%s)", victim.p.Name(), victim.relationOfPerson, p.Name(), victim.relationToPerson), p.Ref())
				// Add opinion.
				victim.p.Opinions.AddOpinion(p, -1.0, ev)
				if !p.Traits.HasTrait(geneticshuman.TraitCareful) {
					// Add opinion for all relatives.
					changeOptsBadDeeds(p, -0.7, 0.1, ev)
					// Change reputation.
					p.Popularity.Add(-0.4)
				}
			}
		},
	}
	actionCruelty := &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.01
			// Primary trait.
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				prob += 0.5
			}

			// Secondary traits.
			if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
				prob += 0.2
			}

			// Negative traits.
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob -= 0.3
			}

			if prob < 0 {
				prob = 0
			}

			return prob
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			// Pick a possible victim.
			victim := pickVictim()
			// We might be cruel in other ways.
			ev := m.AddEvent("Cruelty", fmt.Sprintf("%s (%s) was cruel to %s (%s)", p.Name(), victim.relationToPerson, victim.p.Name(), victim.relationOfPerson), p.Ref())
			// Add opinion.
			victim.p.Opinions.AddOpinion(p, -0.7, ev)

			if !p.Traits.HasTrait(geneticshuman.TraitCareful) {
				// Add opinion for all relatives.
				changeOptsBadDeeds(p, -0.5, 0.15, ev)

				// Change reputation.
				p.Popularity.Add(-0.1)
			}
		},
	}
	actionKindness := &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.1
			// Primary trait.
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob += 0.5
			}
			// Secondary traits.
			if p.Traits.HasTrait(geneticshuman.TraitTrusting) {
				prob += 0.2
			}
			// Negative traits.
			if p.Traits.HasTrait(geneticshuman.TraitCareless) {
				prob -= 0.1
			}
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				prob -= 0.2
			}
			if prob < 0 {
				prob = 0
			}
			return prob
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			// Kind people might help others.
			// Pick a random person.
			// TODO: Make sure we don't pick ourselves.
			personInNeed := m.People[rand.Intn(len(m.People))]
			options := []string{
				"helped",
				"gave food to",
				"offered shelter to",
				"offered money to",
				"gave advice to",
				"gave a gift to",
			}

			// TODO: Check the person's traits and check if they are up to no good.
			action := options[rand.Intn(len(options))]
			ev := m.AddEvent("Kindness", fmt.Sprintf("%s %s %s", p.Name(), action, personInNeed.Name()), p.Ref())
			// Add opinion.
			personInNeed.Opinions.AddOpinion(p, 1.0, ev)
			// Add opinion for all relatives.
			changeOptsGoodDeeds(p, 0.7, -0.15, ev)
			// Change reputation.
			p.Popularity.Add(0.1)
		},
	}
	actionIdle := &PersonalAction{
		probability: func(p *Person) float64 {
			return 1.0
		},
		requires: func(p *Person) bool {
			return true
		},
		consequences: func(m *Civ, p *Person) {
			// Nothing happens.
			// Recover the opinion of all relatives.
			rels := getLivingRelatives(p)
			for _, c := range rels {
				p.Opinions.AddOpinion(c, 0.0, nil)
			}
		},
	}

	actionChildPlay := &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.3
			// Primary trait.
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob += 0.5
			}
			// Secondary traits.
			if p.Traits.HasTrait(geneticshuman.TraitTrusting) {
				prob += 0.2
			}
			return prob
		},
		requires: func(p *Person) bool {
			return p.Age < 12 && p.Age > 2
		},
		consequences: func(m *Civ, p *Person) {
			// Pick a random child.
			var children []*Person
			for _, c := range m.People {
				if c == p {
					continue
				}
				if c.Age < 12 && c.Age > 2 {
					children = append(children, c)
				}
			}

			if len(children) == 0 {
				log.Println("no children to play with")
				return
			}

			// Pick a random child.
			child := children[rand.Intn(len(children))]

			// TODO: Prioritize children we like and that have a good reputation.
			// But one of the two children is a troublemaker, someting bad might happen
			// or the bad child might hurt the good child.
			// Pick a random game.
			options := []string{
				"hide and seek",
				"tag",
				"catch",
				"climbing trees",
				"swimming",
				"building a fort",
				"playing with dolls",
			}
			game := options[rand.Intn(len(options))]
			ev := m.AddEvent("Child Play", fmt.Sprintf("%s played %s with %s", p.Name(), game, child.Name()), p.Ref())
			// Add opinion.
			child.Opinions.AddOpinion(p, 0.7, ev)
			// Add opinion for all relatives.
			changeOptsGoodDeeds(p, 0.4, -0.1, ev)
		},
	}
	actionChildExploration := &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.1
			// TODO: High openness, bravery, etc.
			// Primary trait.
			if p.Traits.HasTrait(geneticshuman.TraitBrave) {
				prob += 0.2
			}
			if p.Traits.HasTrait(geneticshuman.TraitCareless) {
				prob += 0.1
			}
			return prob
		},
		requires: func(p *Person) bool {
			return p.Age < 16 && p.Age > 8
		},
		consequences: func(m *Civ, p *Person) {
			// Find a nearby location to explore.
			// There might be treasure to find, or danger.
			places := []string{
				"forest",
				"cave",
				"river",
				"mountain",
				"abandoned house",
				"graveyard",
				"ruin",
				"swamp",
				"lake",
				"beach",
				"hut",
			}
			place := places[rand.Intn(len(places))]
			// We explore a location and there is a chance of finding something interesting,
			// nothing happening, getting hurt, getting lost, or finding something dangerous.
			// TODO: Add actual benefits and consequences.
			// - If we encounter a wild animal and escape, defeat or make friends with it,
			// we might get a trait.
			// - If we find a treasure, we might get change our fortune.
			// - If we get hurt, we might get a scar or a trait.
			switch rand.Intn(50) {
			case 0:
				// We find a treasure.
				// TODO:
				// - This should trigger some kind of personal quest.
				// - Use genlanguage.TextConfig
				treasure := []string{
					"stash of gold coins",
					"sword inscribed with runes",
					"magical amulet",
					"mirror made of silver",
					"crown",
					"ring",
					"necklace",
					"gemstone",
					"book of spells",
					"scroll",
					"potion",
					"strange coin",
					"small statue",
					"clockwork device",
					"rusty helmet",
					"strange key",
					"dusty diary",
					"tattered letter",
					"drawing of a beautiful woman",
					"crystal shard",
					"doll",
					"glass eye",
					"locket",
					"small box",
				}
				t := treasure[rand.Intn(len(treasure))]

				// TODO: Add item to inventory.

				// Check if it is cursed.
				if rand.Intn(100) < 10 {
					// TODO: Add various curses.
					t += " (cursed)"
					p.Traits |= geneticshuman.TraitDeceptive
					p.Traits |= geneticshuman.TraitAmbitious
					p.Traits |= geneticshuman.TraitCruel

					if rand.Intn(100) < 50 {
						p.Traits |= geneticshuman.TraitCareful
						p.Traits &^= geneticshuman.TraitCareless
					}

					// Remove positive traits.
					p.Traits &^= geneticshuman.TraitKind
					p.Traits &^= geneticshuman.TraitHonest
					p.Traits &^= geneticshuman.TraitContent
					// p.Traits &^= geneticshuman.TraitBrave // Let's keep this one for now.
					p.NickName = "the Cursed"
					p.Popularity.Add(5.0)
				} else if rand.Intn(100) < 10 {
					t += " (blessed)"
					p.Traits |= geneticshuman.TraitKind
					p.Traits |= geneticshuman.TraitBrave
					p.Traits |= geneticshuman.TraitHonest
					p.Traits |= geneticshuman.TraitCareful

					if rand.Intn(100) < 50 {
						p.Traits |= geneticshuman.TraitAmbitious
						p.Traits &^= geneticshuman.TraitContent
					}

					// Remove negative traits.
					p.Traits &^= geneticshuman.TraitCruel
					p.Traits &^= geneticshuman.TraitDeceptive
					p.Traits &^= geneticshuman.TraitCareless
					// p.Traits &^= geneticshuman.TraitAggressive // Let's keep this one for now.
					p.NickName = "the Blessed"
					p.Popularity.Add(1.0)
				} else {
					p.Popularity.Add(1.0)
				}
				ev := m.AddEvent("Exploration", fmt.Sprintf("%s found a %s in a %s", p.Name(), t, place), p.Ref())
				// Add opinion for all relatives.
				changeOptsGoodDeeds(p, 0.7, -0.4, ev)
			case 1:
				// We find a dangerous animal.
				// TODO: The animals should be different depending on the region.
				animal := []string{
					"wild dog",
					"bear",
					"snake",
					"wolf",
					"wildcat",
					"boar",
					"dangerous squirrel",
					"wild rabbit",
				}

				// TODO: The outcome should depend on the person's traits.
				t := animal[rand.Intn(len(animal))]
				if rand.Intn(100) < 50 {
					// We gain bravery, lose cowardice.
					p.Traits |= geneticshuman.TraitBrave
					p.Traits &^= geneticshuman.TraitCowardly

					p.Popularity.Add(0.7)

					// We escape.
					ev := m.AddEvent("Exploration", fmt.Sprintf("%s encountered a %s while exploring a %s and was able to chase it away", p.Name(), t, place), p.Ref())
					// Add opinion for all relatives.
					changeOptsGoodDeeds(p, 0.3, -0.1, ev)
				} else {
					// We gain cowardice or carefullness.
					if rand.Intn(100) < 50 {
						p.Traits |= geneticshuman.TraitCowardly
						p.Traits &^= geneticshuman.TraitBrave
					} else {
						p.Traits |= geneticshuman.TraitCareful
						p.Traits &^= geneticshuman.TraitCareless
					}
					p.Popularity.Add(-0.2)

					// We get hurt.
					ev := m.AddEvent("Exploration", fmt.Sprintf("%s encountered a %s while exploring a %s and got hurt", p.Name(), t, place), p.Ref())
					// Add opinion for all relatives.
					changeOptsBadDeeds(p, -0.2, 0.0, ev)
				}
			case 2:
				// We get hurt.
				ev := m.AddEvent("Exploration", fmt.Sprintf("%s got hurt while exploring a %s", p.Name(), place), p.Ref())
				// Add opinion for all relatives.
				changeOptsBadDeeds(p, -0.2, 0.0, ev)
			case 3:
				// We get lost.
				ev := m.AddEvent("Exploration", fmt.Sprintf("%s got lost while exploring a %s", p.Name(), place), p.Ref())
				// Add opinion for all relatives.
				changeOptsBadDeeds(p, -0.2, 0.0, ev)
			default:
				// Nothing happens.
				ev := m.AddEvent("Exploration", fmt.Sprintf("%s explored a %s", p.Name(), place), p.Ref())
				// Add opinion for all relatives.
				changeOptsGoodDeeds(p, 0.2, -0.1, ev)
			}
		},
	}

	// A cruel child might hurt another child or torture an animal.
	actionChildBully := &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.01
			// Primary trait.
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				prob = 0.2
			} else if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob = 0.0
			}
			// Secondary traits.
			if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
				prob += 0.2
			}
			if prob < 0 {
				prob = 0
			}
			return prob
		},
		requires: func(p *Person) bool {
			return p.Age < 12 && p.Age > 2
		},
		consequences: func(m *Civ, p *Person) {
			// Pick a random child.
			var children []*Person
			for _, c := range p.Children {
				if c == p {
					continue
				}
				if c.Age < 12 && c.Age > 2 {
					children = append(children, c)
				}
			}

			if len(children) == 0 {
				return
			}

			// Pick a random child.
			child := children[rand.Intn(len(children))]
			// We bully another child.
			ev := m.AddEvent("Child Cruelty", fmt.Sprintf("%s bullied %s", p.Name(), child.Name()), p.Ref())
			// Add opinion.
			child.Opinions.AddOpinion(p, -1.0, ev)
			// Add opinion for all relatives.
			changeOptsBadDeeds(p, -0.7, 0.15, ev)
		},
	}
	actionChildTortureAnimal := &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.01
			// Primary trait.
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				prob = 0.2
			} else if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob = 0.0
			}
			// Secondary traits.
			if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
				prob += 0.2
			}
			if prob < 0 {
				prob = 0
			}
			return prob
		},
		requires: func(p *Person) bool {
			return p.Age < 12 && p.Age > 2
		},
		consequences: func(m *Civ, p *Person) {
			// Pick a random animal.
			animals := []string{"cat", "dog", "bird", "rabbit", "squirrel", "mouse", "fish"}
			animal := animals[rand.Intn(len(animals))]
			// We torture an animal.
			ev := m.AddEvent("Child Cruelty", fmt.Sprintf("%s tortured a %s", p.Name(), animal), p.Ref())
			// Add opinion for all relatives.
			changeOptsBadDeeds(p, -0.5, 0.25, ev)
		},
	}

	var actions []*PersonalAction
	if actionMurder.requires(p) {
		actions = append(actions, actionMurder)
	}
	if actionCruelty.requires(p) {
		actions = append(actions, actionCruelty)
	}
	if actionKindness.requires(p) {
		actions = append(actions, actionKindness)
	}
	if actionIdle.requires(p) {
		actions = append(actions, actionIdle)
	}
	if actionChildPlay.requires(p) {
		actions = append(actions, actionChildPlay)
	}
	if actionChildExploration.requires(p) {
		actions = append(actions, actionChildExploration)
	}
	if actionChildBully.requires(p) {
		actions = append(actions, actionChildBully)
	}
	if actionChildTortureAnimal.requires(p) {
		actions = append(actions, actionChildTortureAnimal)
	}

	// Pick an action.
	if action := pickAction(p, actions); action != nil {
		action.Execute(m, p)
	}
	// TODO: Add actions that depends on multiple characteristics...
	// - An aggressive, paranoid person might attack someone they think is plotting against them
	//  or assassinate a leader of a rival faction.
	// - A kind, trusting person might help someone in need and either be deceived or make a new friend
	//  or ally. Maybe even help a leader or wealthy person and gain influence.
	// - A lazy, careless person might cause an accident or get into trouble.

	// Other options:
	// - Kind people doing kind things.
	// - Deceptive people conning others.
	// - Trusting people being deceived or robbed.
	// - Careless people getting into accidents.
	// - Aggressive people getting into fights.
	// - People with criminal intent committing crimes.

	// TODO: Add consequences for actions.
	// - A person that is cruel to their children might have their children run away or turn against them.
	// - If someone witnesses a murder, they might report it to the authorities.
	// - Criminal acts might lead to prison.

	// Maybe we should define these "roles" formally, which bundle a set of actions and the
	// related entities that the person can interact with.

	// Traditions
	// Add the concept of traditions, which are actions that are performed on a regular basis.
	// For example, a person might have the tradition of taking a child hunting on their
	// n-th birthday, or the tradition of visiting a particular place on a particular day.
	// These traditions could be passed down from generation to generation, and could be
	// a source of conflict or bonding between people.
	// A cruel person might have the tradition of killing a pet on someone's birthday, while
	// a kind person might have the tradition of giving a gift to someone on their birthday.
	//
	// Traditions should have conditions and/or a cadence or trigger.

	return child
}

const (
	ageOfAdulthood     = 18
	ageEndChildbearing = 45
)

func (m *Civ) tickPersonAge(p *Person, nDays int) {
	// Calculate current age.
	if m.History.GetDayOfYear() < p.Birth.Day {
		p.Age = int(m.History.GetYear()) - p.Birth.Year - 1
	} else {
		p.Age = int(m.History.GetYear()) - p.Birth.Year
	}
}

func (m *Civ) tickPersonDeath(p *Person, nDays int) {
	// Check if person dies of natural causes.
	if gameconstants.DiesAtAgeWithinNDays(p.Age, nDays) {
		// If the person just gave birth, we note that the person
		// died during childbirth.
		options := []string{
			"illness",
			"an accident",
			"a mysterious cause",
			"a heart attack",
			"a stroke",
			"a fall",
			"a lightning strike",
			"a snake bite",
			"a wild animal attack",
			"a drowning",
			"a fire",
			"a poisoning",
		}

		// Depending on the personality, the person might die of different causes.
		// For example, a person with low agreeableness might die in a duel, while
		// a person with low conscientiousness might die in an accident.

		if p.Age > 14 {
			options = append(options, "old age")

			// High aggression might lead to death in a duel, fight, or challenging a wild animal.
			if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
				options = append(options, "a duel", "a fight", "challenging a wild animal")
				options = append(options, "a heart attack", "a stroke")
			}

			// Carelessness might lead to death in an accident, eating spoiled food, or a fall.
			if p.Traits.HasTrait(geneticshuman.TraitCareless) {
				options = append(options, "an accident", "eating spoiled food", "a fall")
			}

			// Carelessness might lead to death during new experiences.
			// - Poisioning from trying new food.
			// - Dying duing extreme sports.
			// - Drowning while swimming.
			if p.Traits.HasTrait(geneticshuman.TraitCareless) {
				options = append(options, "a mysterious cause", "a poisoning")
			}

			// Trusting might lead to death by being deceived, robbed, or poisoned.
			if p.Traits.HasTrait(geneticshuman.TraitTrusting) {
				options = append(options, "being deceived", "being robbed", "being poisoned")
			}
		}
		m.killPerson(p, options[rand.Intn(len(options))]) // Random cause?
	}
}

func (m *Civ) killPerson(p *Person, reason string) *Event {
	p.Death.Day = int(m.History.GetDayOfYear())
	p.Death.Year = int(m.History.GetYear())
	p.Death.Region = p.Region

	// If they have a spouse, unset their spouse.
	if p.Spouse != nil {
		p.Spouse.Spouse = nil
	}

	// :(
	if p.Prengancy != nil {
		m.killPerson(p.Prengancy, fmt.Sprintf("mother %s dying due to %s", p.Name(), reason))
	}

	var deathStr string
	name := p.Name()
	if name == "" {
		if p.Birth.IsSet() {
			name = "unknown person"
		} else if p.Gender() == geneticshuman.GenderFemale {
			name = "unborn girl"
		} else if p.Gender() == geneticshuman.GenderMale {
			name = "unborn boy"
		} else {
			name = "unborn child"
		}
	}
	if reason == "" {
		deathStr = fmt.Sprintf("%s died at age %d", name, p.Age)
	} else {
		deathStr = fmt.Sprintf("%s died at age %d due to %s", name, p.Age, reason)
	}
	return m.AddEvent("Death", deathStr, p.Ref())

	/*
		var str string
		if p.Spouse != nil {
			str += "[spouse]"
		}
		if len(p.Children) > 0 {
			str += fmt.Sprintf("[%d children]", len(p.Children))
		}
		log.Println("killed", p.Name(), "at", p.Death.Region, "aged", p.Age, str)
	*/
}

func (m *Civ) doesPersonExist(p *Person) bool {
	return !p.Death.IsSet() && p.Birth.IsSet() && p.Birth.Year < int(m.History.GetYear())
}

func (m *Civ) updatePersonLocation(p *Person, r int) {
	// Update location.
	// NOTE: We should differentiate between people who live in the city and
	// people who work in or visit the city.
	p.Region = r
	// p.City = m.GetCity(r)
	// TODO: Add person to city population?
}

// LifeEvent represents a date and place in the world.
type LifeEvent struct {
	Year   int
	Day    int
	Region int
}

// IsSet returns true if the life event is set.
func (l LifeEvent) IsSet() bool {
	return l.Year != 0 || l.Day != 0 || l.Region != 0
}

// Person represents a person in the world.
// TODO: Improve efficiency of this struct.
//   - We could drop age, and use day-ticks for birth and death instead.
//   - Also, we can get the gender directly from the genes.
//   - We might be able to drop the pregnancy counter and use the birth life event
//     of the child as a counter.
//   - We can use use the Region for the location and derive the city from that.
//   - A lot of this stuff is identical to simvillage_simple, so we could probably
//     merge the person logic somehow, or move it to a separate package.
type Person struct {
	ID          int                      // ID of the person
	Region      int                      // Location of the person
	Genes       genetics.Genes           // Genes.
	Personality geneticshuman.FiveFactor // Personality of the person
	Traits      geneticshuman.Trait      // Traits of the person
	City        *City                    // City of the person
	Culture     *Culture                 // Culture of the person
	Opinions    *Opinions                // Opinions of the person
	Popularity  ClampedVal               // Popularity of the person (reputation)

	// Todo: Allow different naming conventions.
	FirstName string
	LastName  string
	NickName  string
	Title     string // Title of the person. TODO: Improve this.

	// Birth, death...
	// TODO: Add death cause.
	Age   int // Age of the person.
	Birth LifeEvent
	Death LifeEvent

	// Pregnancy
	PregnancyCounter int     // Days of pregnancy
	Prengancy        *Person // baby (TODO: twins, triplets, etc.)

	// Family (TODO: Distinguish between known and unknown family members.)
	// Maybe use a map of relations to people?
	Mother   *Person
	Father   *Person
	Spouse   *Person   // TODO: keep track of spouses that might have perished?
	Children []*Person // TODO: Split into known and unknown children.
}

func (m *Civ) newRandomPersonAt(r int, culture *Culture, gender geneticshuman.Gender, parent *Person) *Person {
	if parent != nil && parent.Age < ageOfAdulthood {
		panic("Parent is too young to have children.")
	}
	// Random genes / gender.
	var genes genetics.Genes
	if parent != nil {
		genes = genetics.Mix(parent.Genes, genetics.NewRandom(), 1)
	} else {
		genes = genetics.NewRandom()
	}
	geneticshuman.SetGender(&genes, gender)

	lang := culture.Language

	// Create the person.

	// If the first/last name pool is large enough, we should
	// start reusing names because generating new names is expensive.
	//
	// TODO: With increasing pool size, we should increase the chance
	// of reusing names.
	var firstName string
	if poolSize := lang.GetFirstNamePoolSize(); poolSize > 100 && rand.Intn(poolSize) > 10 {
		firstName = lang.GetFirstName()
	}
	if firstName == "" {
		firstName = lang.MakeFirstName()
	}

	// Same for last names.
	var lastName string
	if parent != nil {
		lastName = parent.LastName
	} else if poolSize := lang.GetLastNamePoolSize(); poolSize > 300 && rand.Intn(poolSize) > 10 {
		lastName = lang.GetLastName()
	}
	if lastName == "" {
		lastName = lang.MakeLastName()
	}

	// Infer personality and traits from genes.
	fiveFactor := geneticshuman.GetFiveFactor(&genes)
	traits := geneticshuman.GetTraits(fiveFactor)

	// Random age.
	var age int
	if parent != nil {
		age = max(rand.Intn(parent.Age-16), parent.Age/2)
	} else {
		age = ageOfAdulthood + rand.Intn(2*ageOfAdulthood)
	}

	// TODO: Calculate age.
	p := &Person{
		ID:          m.getNextPersonID(),
		Culture:     culture,
		Opinions:    NewOpinions(),
		Genes:       genes,
		FirstName:   firstName,
		LastName:    lastName,
		Personality: fiveFactor,
		Traits:      traits,
		Age:         age,
		Birth: LifeEvent{
			Year:   int(m.History.GetYear()) - age,
			Day:    rand.Intn(365),
			Region: r, // TODO: Pick a birth region that makes sense.
		},
	}

	if parent != nil {
		if parent.Gender() == GenderFemale {
			p.Mother = parent
		} else {
			p.Father = parent
		}

		// Assign as child to parent.
		parent.Children = append(parent.Children, p)
	}

	// Update location.
	m.updatePersonLocation(p, r)

	// TODO: Random spouse, children, etc.?
	m.People = append(m.People, p)
	return p
}

// compare returns the similarity between two people.
func (p *Person) compare(other *Person) float64 {
	if p == other {
		return 1.0
	}
	if p == nil || other == nil {
		return -1.0
	}
	cultureValue := p.Culture.compare(other.Culture)
	languageValue := compareLanguage(p.Culture.Language, other.Culture.Language)
	// religionValue := p.Religion.compare(other.Religion)
	traitValue := p.Traits.Compare(other.Traits)

	return (cultureValue + languageValue + traitValue) / 3
}

// Name returns the name of the person.
func (p *Person) Name() string {
	if p.FirstName == "" && p.LastName == "" {
		if p.NickName != "" {
			return p.NickName
		}
		return ""
	}
	if p.NickName != "" {
		return fmt.Sprintf("%s %q %s", p.FirstName, p.NickName, p.LastName)
	}
	return p.FirstName + " " + p.LastName
}

// Ref returns the object reference of the person.
func (p *Person) Ref() ObjectReference {
	return ObjectReference{
		ID:   p.ID,
		Type: ObjectTypePerson,
	}
}

// StringGenes returns the string representation of the person.
func (p *Person) StringGenes() string {
	return geneticshuman.String(p.Genes)
}

// String returns a string representation of the leader.
func (p *Person) String() string {
	name := p.Name()
	if p.Title != "" {
		name = p.Title + " " + name
	}
	gender := "°"
	if p.Gender() == geneticshuman.GenderFemale {
		gender = "♀"
	} else if p.Gender() == geneticshuman.GenderMale {
		gender = "♂"
	}
	isDead := " "
	if p.Dead() {
		isDead = "†"
	}
	str := fmt.Sprintf("%s (%s%s%d)", name, gender, isDead, p.Age)
	if p.Traits != 0 {
		str += " [" + p.Traits.String() + "]"
	}
	return str
}

// Dead returns true if the person is dead.
func (p *Person) Dead() bool {
	return p.Death.IsSet()
}

// Gender returns the gender of the person.
func (p *Person) Gender() geneticshuman.Gender {
	return geneticshuman.GetGender(&p.Genes)
}

func (p *Person) isOfChildbearingAge() bool {
	return p.Age >= ageOfAdulthood && p.Age < ageEndChildbearing
}

// isElegibleSingle returns true if the person is old enough to look for a partner and single.
func (p *Person) isEligibleSingle() bool {
	return p.Age > ageOfAdulthood && p.Spouse == nil // Old enough and single.
}

// canBePregnant returns true if the person is old enough and not pregnant.
func (p *Person) canBePregnant() bool {
	// Female, has a spouse (implies old enough), and is currently not pregnant.
	// TODO: Set randomized upper age limit.
	return p.Gender() == GenderFemale && p.Spouse != nil && p.Prengancy == nil
}

const pregnancyDays = 280 // for humans

func (p *Person) newPersonPregnancy(id int, father *Person) *Person {
	// Mix genes.
	var genes genetics.Genes
	if father != nil {
		genes = genetics.Mix(p.Genes, father.Genes, 1)
	} else {
		genes = genetics.Mix(p.Genes, genetics.NewRandom(), 1)
	}

	// Fix genes wrt. gender (the genetic mix doesn't limit gender varaition)
	geneticshuman.SetGender(&genes, randGender())

	fiveFactor := geneticshuman.GetFiveFactor(&genes)
	traits := geneticshuman.GetTraits(fiveFactor)

	// We need to set the name after birth, because the parents might not know the gender of the baby
	// until birth. (If there's magic, only wealthy people would be able to determine the gender before)
	child := &Person{
		ID:          id,
		Genes:       genes,
		Mother:      p,
		Father:      father,
		Personality: fiveFactor,
		Traits:      traits,
		Opinions:    NewOpinions(),
	}

	p.PregnancyCounter = pregnancyDays
	p.Prengancy = child
	return child
}

// tickPersonPregnancy advances the pregnancy of the person.
// TODO: Add twins, triplets, etc.
func (m *Civ) tickPersonPregnancy(p *Person, nDays int, cf func(int) *Culture) *Person {
	if p.Prengancy == nil {
		return nil
	}

	// Reduce pregnancy counter.
	p.PregnancyCounter -= nDays
	if p.PregnancyCounter > 0 {
		return nil
	}

	// Birth!
	wasBornNDaysAgo := -p.PregnancyCounter
	child := p.Prengancy

	// Reset pregnancy.
	p.Prengancy = nil
	p.PregnancyCounter = 0

	// Add child to family and name it.
	// We use spouse since this is the acting father.
	// TODO: Use naming convention of culture to determine if mother or father name the child.
	var lang *genlanguage.Language
	if p.Spouse != nil && rand.Intn(100) < 50 {
		lang = p.Spouse.Culture.Language
	} else {
		lang = p.Culture.Language
	}

	// There is a random chance we generate a new name, but the larger the pool
	// the less likely we are to generate a new name.
	var firstName string
	if poolSize := lang.GetFirstNamePoolSize(); poolSize > 100 && rand.Intn(poolSize) > 10 {
		firstName = lang.GetFirstName()
	}
	if firstName == "" {
		firstName = lang.MakeFirstName()
	}
	child.FirstName = firstName

	// Add child to the children of the mother.
	p.Children = append(p.Children, child)

	// Add child to the children of the "father" uhm.. spouse.
	if p.Spouse != nil {
		p.Spouse.Children = append(p.Spouse.Children, child)
	} else if p.Father != nil {
		// TODO: What if spouse != father?
		p.Father.Children = append(p.Father.Children, child)
	}

	// Use the mother's last name.
	child.LastName = p.LastName

	// Set birth date.
	child.Birth.Region = p.Region
	child.Birth.Year = int(m.History.GetYear())
	child.Birth.Day = m.History.GetDayOfYear() - wasBornNDaysAgo
	if child.Birth.Day < 0 {
		child.Birth.Year--
		child.Birth.Day += 365
		// Age the baby for the number of days it was born ago.
		m.tickPerson(child, wasBornNDaysAgo, cf)
	}

	// Set city.
	child.City = p.City
	if child.City != nil {
		child.City.People = append(child.City.People, child)
		child.Culture = child.City.Culture
	}

	// Set culture.
	// NOTE: Should this be the culture of the mother or father?
	// If mother and father are from different cultures, which one should it be?
	// If the child is born in a different region, should the culture change?
	//
	// I think it'd be great to randomly determine which culture the child
	// should have. This would ba an interesting source of conflict and story.
	if child.Culture == nil {
		child.Culture = cf(p.Region)
	}

	// Update location.
	m.updatePersonLocation(child, p.Region)

	// Add child to world.
	m.People = append(m.People, child)

	// log.Println("New person born:", child.Name())
	return child
}

// isDead returns true if the person is dead.
func (p *Person) isDead() bool {
	return p.Death.IsSet()
}

var (
	GenderFemale = geneticshuman.GenderFemale
	GenderMale   = geneticshuman.GenderMale
)

// randGender returns a random gender.
func randGender() geneticshuman.Gender {
	if rand.Intn(2) == 0 {
		return GenderFemale
	}
	return GenderMale
}

// isRelated returns true if a and b are related (up to first degree).
func isRelated(a, b *Person) bool {
	// Check if there is a parent/child relationship.
	if a == b.Father || a == b.Mother || b == a.Father || b == a.Mother {
		return true
	}

	// If either (or both) of the parents are nil, we assume that they are not related.
	if (a.Father == nil && a.Mother == nil) || (b.Father == nil && b.Mother == nil) {
		return false
	}

	// Check if there is a (half-) sibling relationship.
	return a.Mother == b.Mother || a.Father == b.Father
}
