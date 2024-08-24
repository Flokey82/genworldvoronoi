package genworldvoronoi

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/go_gens/genstory"
)

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

func pickVictim(m *Civ, p *Person) *relation {
	options := pickRelatives(p)
	if len(options) > 0 {
		// Sort options by opinion.
		// This is sort of a "who do we hate the most" function.
		sort.Slice(options, func(i, j int) bool {
			return p.Opinions.GetOpinion(options[i].p) < p.Opinions.GetOpinion(options[j].p)
		})
		// TODO: Maybe pick a random from the top 3.
		return &options[0]
	}

	// Pick a random person.
	nearby := m.getPeopleAt(p.Region)
	if len(nearby) <= 1 {
		log.Println("no nearby people found")
		return nil
	}

	// Filter out the current person
	filteredNearby := make([]*Person, 0, len(nearby)-1)
	for _, person := range nearby {
		if person != p && !person.Dead() {
			filteredNearby = append(filteredNearby, person)
		}
	}

	if len(filteredNearby) == 0 {
		log.Println("no person found")
		return nil
	}

	picked := filteredNearby[rand.Intn(len(filteredNearby))]

	return &relation{
		p:                picked,
		relationToPerson: "random stranger",
		relationOfPerson: "random stranger",
	}
}

// dislikesGoodDeeds checks if the person dislikes good deeds.
func dislikesGoodDeeds(p *Person) bool {
	return p.Traits.HasTrait(geneticshuman.TraitCruel) || p.Traits.HasTrait(geneticshuman.TraitAggressive)
}

// dislikesBadDeeds checks if the person dislikes bad deeds.
func dislikesBadDeeds(p *Person) bool {
	return !p.Traits.HasTrait(geneticshuman.TraitCruel) && !p.Traits.HasTrait(geneticshuman.TraitAggressive)
}

// changeOptsBadDeeds changes the opinions of the person's relatives based on the person's bad deeds.
func changeOptsBadDeeds(p *Person, changeDislike, changeLike float64, ev *Event) {
	// Change Karma for the person.
	p.Karma.Add(changeDislike)

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

// changeOptsGoodDeeds changes the opinions of the person's relatives based on the person's good deeds.
func changeOptsGoodDeeds(p *Person, changeLike, changeDislike float64, ev *Event) {
	// Change Karma for the person.
	p.Karma.Add(changeLike)

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

var (
	actionMurder = &PersonalAction{
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
			return max(prob, 0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			// Pick a possible victim.
			victim := pickVictim(m, p)
			if victim == nil {
				log.Println("no victim found")
				return
			}
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
	actionCruelty = &PersonalAction{
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
			return max(prob, 0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			// Pick a possible victim.
			victim := pickVictim(m, p)
			if victim == nil {
				log.Println("no victim found")
				return
			}

			// We might be cruel in other ways.
			ev := m.AddEvent("Cruelty", fmt.Sprintf("%s (%s) was cruel to %s (%s)", p.Name(), victim.relationToPerson, victim.p.Name(), victim.relationOfPerson), p.Ref())
			// Add opinion.
			victim.p.Opinions.AddOpinion(p, -0.7, ev)

			// There is a chance that we steal one of the artifacts from the victim.
			if len(victim.p.Artifacts) > 0 && rand.Intn(100) < 50 {
				// Pick a random artifact.
				art := victim.p.Artifacts[rand.Intn(len(victim.p.Artifacts))]
				// Steal the artifact.
				victim.p.transferArtifact(m, art, p)
				// Add event.
				m.AddEvent("Artifact", fmt.Sprintf("%s was stolen from %s by %s", art.NameWithArticle(), victim.p.Name(), p.Name()), p.Ref())
			}

			if !p.Traits.HasTrait(geneticshuman.TraitCareful) {
				// Add opinion for all relatives.
				changeOptsBadDeeds(p, -0.5, 0.15, ev)

				// Change reputation.
				p.Popularity.Add(-0.1)
			}
		},
	}
	actionHangout = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.5
			// Primary trait.
			if p.Traits.HasTrait(geneticshuman.TraitGregarious) {
				prob += 0.3
			}
			// Secondary traits.
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob += 0.1
			}
			// Negative traits.
			if p.Traits.HasTrait(geneticshuman.TraitShy) {
				prob -= 0.2
			}
			return max(prob, 0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			// Pick the person we like most but that is not a relative.
			nearby := m.getPeopleAt(p.Region)
			filteredNearby := make([]*Person, 0, len(nearby)-1)
			for _, person := range nearby {
				if person == p || person.Dead() || isRelated(p, person) {
					continue
				}
				filteredNearby = append(filteredNearby, person)
			}
			if len(filteredNearby) == 0 {
				log.Println("no person found")
				return
			}
			sort.Slice(filteredNearby, func(i, j int) bool {
				return p.Opinions.GetOpinion(filteredNearby[i]) > p.Opinions.GetOpinion(filteredNearby[j])
			})
			picked := filteredNearby[0]

			// We might hang out with someone.
			activities := []string{"drinking", "eating", "talking", "walking", "fishing", "hunting", "playing games", "watching the stars", "watching the sunset", "watching the sunrise", "watching the moon", "watching the clouds", "watching the rain", "watching the snow", "watching the birds", "watching the animals", "watching the people", "watching the children", "watching the old", "watching the young", "watching the middle-aged", "watching the plants", "watching the trees", "watching the flowers", "watching the grass"}
			activity := activities[rand.Intn(len(activities))]

			// Relationshipdistance
			// relDist := calcRelationshipDistance(p, picked)
			relDist := 0

			// Compare the two people and if they are similar they have a positive experience.
			cmp := p.compare(picked)
			if cmp > 0.5 {
				// They are very similar.
				ev := m.AddEvent("Hangout", fmt.Sprintf("%s went %s with %s (dist %d)", p.Name(), activity, picked.Name(), relDist), p.Ref())
				// Add opinion.
				picked.Opinions.AddOpinion(p, 0.7, ev)
				p.Opinions.AddOpinion(picked, 0.7, ev)
				// TODO: Add opinion for all relatives.
				// If it is someone not liked by the relatives, they might get a bad opinion.
			} else {
				// They are not very similar.
				ev := m.AddEvent("Hangout", fmt.Sprintf("%s went %s with %s (dist %d)", p.Name(), activity, picked.Name(), relDist), p.Ref())
				// Add opinion.
				picked.Opinions.AddOpinion(p, 0.3, ev)
				p.Opinions.AddOpinion(picked, 0.3, ev)
			}
		},
	}

	actionKindness = &PersonalAction{
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
			return max(prob, 0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			// Kind people might help others.

			// Pick a random person.
			var personInNeed *Person
			for i := 0; i < 100; i++ {
				// TODO: Only pick people we are physically close to.
				personInNeed = m.People[rand.Intn(len(m.People))]

				// Make sure we don't pick ourselves or a dead person.
				// TODO: Do not pick people we outright hate.
				if personInNeed == p || personInNeed.Dead() {
					continue
				}
				break
			}

			if personInNeed == nil {
				log.Println("no person in need found")
				return
			}

			options := []string{
				"helped",
				"gave food to",
				"offered shelter to",
				"offered money to",
				"gave advice to",
				"gave a gift to",
			}
			action := options[rand.Intn(len(options))]

			// TODO: Check the person's traits and check if they are up to no good.
			// If they are, we might get tricked or something bad might happen.
			// There is also the option that if we are blessed, we might change the person's life for the better.
			// This would result in bad traits being removed and/or good traits being added.
			if personInNeed.Traits.HasTrait(geneticshuman.TraitCruel | geneticshuman.TraitAggressive | geneticshuman.TraitDeceptive) {
				// TODO: Check for blessing, add event.
				// If we are blessed, we might even be able to remove a curse?
				if rand.Intn(100) < 10 {
					personInNeed.Traits &^= geneticshuman.TraitCruel | geneticshuman.TraitAggressive | geneticshuman.TraitDeceptive
					personInNeed.Popularity.Add(0.5)
				}
			}

			ev := m.AddEvent("Kindness", fmt.Sprintf("%s %s %s", p.Name(), action, personInNeed.Name()), p.Ref())
			// Add opinion.
			personInNeed.Opinions.AddOpinion(p, 1.0, ev)
			// Add opinion for all relatives.
			changeOptsGoodDeeds(p, 0.7, -0.15, ev)
			// Change reputation.
			p.Popularity.Add(0.1)
		},
	}

	actionIdle = &PersonalAction{
		probability: func(p *Person) float64 {
			return 1.0
		},
		requires: func(p *Person) bool {
			return true
		},
		consequences: func(m *Civ, p *Person) {
			// Nothing happens. Recover the opinion of all relatives.
			rels := getLivingRelatives(p)
			for _, c := range rels {
				p.Opinions.AddOpinion(c, 0.0, nil)
			}
		},
	}

	actionLoseArtifact = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.01
			// Primary trait.
			if p.Traits.HasTrait(geneticshuman.TraitDeceptive) {
				prob += 0.2
			}
			// Secondary traits.
			if p.Traits.HasTrait(geneticshuman.TraitCareless) {
				prob += 0.1
			}
			return max(prob, 0)
		},
		requires: func(p *Person) bool {
			return len(p.Artifacts) > 0
		},
		consequences: func(m *Civ, p *Person) {
			// Pick a random artifact.
			art := p.Artifacts[rand.Intn(len(p.Artifacts))]
			// Lose the artifact.
			p.removeArtifact(m, art)
			// Add event.
			ev := m.AddEvent("Artifact", fmt.Sprintf("%s lost %s", p.Name(), art.NameWithArticle()), p.Ref())
			// Add opinion for all relatives.
			changeOptsBadDeeds(p, -0.3, 0.1, ev)

			// Thre is a chance that the artifact is found by someone.
			if rand.Intn(100) < 50 {
				// Pick a random person.
				var person *Person
				peopleAtRegion := m.getPeopleAt(p.Region)
				for i := 0; i < 100; i++ {
					person = peopleAtRegion[rand.Intn(len(peopleAtRegion))]
					if person == p || person.Dead() {
						continue
					}
					break
				}
				if person != nil {
					// Add event.
					m.AddEvent("Artifact", fmt.Sprintf("%s found %s", person.Name(), art.NameWithArticle()), person.Ref())
					person.addArtifact(m, art)

					// There is a chance we've witnessed the event and if we are kind or honest, or if we like the person we might return the artifact.
					if p.Traits.HasTrait(geneticshuman.TraitKind) || p.Traits.HasTrait(geneticshuman.TraitHonest) || p.Opinions.GetOpinion(person) > 0.5 {
						// Add event.
						m.AddEvent("Artifact", fmt.Sprintf("%s returned %s to %s", p.Name(), art.NameWithArticle(), person.Name()), p.Ref())
						person.transferArtifact(m, art, p)
						// Add opinion.
						p.Opinions.AddOpinion(person, 0.7, ev)
						// Change karma.
						p.Karma.Add(0.7)
						// TODO: Add maybe a small reward.
						// Maybe add a little buf or positive memory.
					}
				}
			}
		},
	}

	actionChildPlay = &PersonalAction{
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
			return max(prob, 0)
		},
		requires: func(p *Person) bool {
			return p.Age > 2 && p.Age < 12
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
	actionChildExploration = &PersonalAction{
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
			return max(prob, 0)
		},
		requires: func(p *Person) bool {
			return p.Age > 8 && p.Age < 16
		},
		consequences: func(m *Civ, p *Person) {
			// Find a nearby location to explore.
			// There might be treasure to find, or danger.
			// Generate a random location.
			rls := &genstory.Rules{
				Expansions: map[string][]string{
					"location": {
						"forest",
						"cave",
						"river",
						"mountain",
						"house",
						"graveyard",
						"ruin",
						"swamp",
						"lake",
						"beach",
						"hut",
					},
					"adjective": {
						"dark",
						"mysterious",
						"abandoned",
						"haunted",
						"enchanted",
						"hidden",
						"secret",
						"forbidden",
						"dangerous",
						"treacherous",
						"forlorn",
						"desolate",
						"lonely",
						"lost",
						"ancient",
						"forgotten",
						"overgrown",
					},
					"location_phrase": {
						"[adjective:a] [location]",
						"[location:a]",
					},
				},
				Start: "[location_phrase]",
			}
			placeStory := rls.NewStory(time.Now().UnixNano())

			place, err := placeStory.Expand()
			if err != nil {
				log.Println("error expanding place story:", err)
			}
			// We explore a location and there is a chance of finding something interesting,
			// nothing happening, getting hurt, getting lost, or finding something dangerous.
			// TODO: Add actual benefits and consequences.
			// - If we encounter a wild animal and escape, defeat or make friends with it,
			// we might get a trait.
			// - If we find a treasure, we might get change our fortune.
			// - If we get hurt, we might get a scar or a trait.
			const (
				outcomeFoundTreasure = 0
				outcomeFoundDanger   = 1
				outcomeGotHurt       = 2
				outcomeGotLost       = 3
			)
			switch rand.Intn(50) {
			case outcomeFoundTreasure:
				// We find a treasure.
				// TODO:
				// - This should trigger some kind of personal quest.
				// - Use genstory.TextConfig

				// Generate a random treasure.
				treasure := NewTreasure()

				// We are neither cursed nor blessed.
				if treasure.Condition == nil {
					// Gain some popularity.
					p.Popularity.Add(1.0)
				}

				// Add item to inventory.
				p.addArtifact(m, treasure)

				// Add event for the artifact.
				m.AddEvent("Artifact", fmt.Sprintf("%s was found by %s in %s", treasure.NameWithArticle(), p.Name(), place), p.Ref())

				// Add event.
				ev := m.AddEvent("Exploration", fmt.Sprintf("%s found a %s in %s", p.Name(), treasure.NameWithArticle(), place), p.Ref())
				// Add opinion for all relatives.
				changeOptsGoodDeeds(p, 0.7, -0.4, ev)
			case outcomeFoundDanger:
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

				// If we have an artifact, we might be able to use it to scare the animal away.
				if len(p.Artifacts) > 0 && rand.Intn(100) < 50 {
					// Pick a random artifact.
					art := p.Artifacts[rand.Intn(len(p.Artifacts))]

					// Add event.
					m.AddEvent("Artifact", fmt.Sprintf("%s was used by %s to scare away a %s", art.NameWithArticle(), p.Name(), t), p.Ref())

					// We scare the animal away.
					ev := m.AddEvent("Exploration", fmt.Sprintf("%s encountered a %s while exploring %s and was able to chase it away", p.Name(), t, place), p.Ref())
					// Add opinion for all relatives. They will be proud.
					changeOptsGoodDeeds(p, 0.3, -0.1, ev)
				} else if rand.Intn(100) < 50 {
					// Things go well! We gain bravery, lose cowardice.
					lessonForTheBrave.apply(m, p)

					// We gain some popularity.
					p.Popularity.Add(0.7)

					// We escape.
					ev := m.AddEvent("Exploration", fmt.Sprintf("%s encountered a %s while exploring %s and was able to chase it away", p.Name(), t, place), p.Ref())
					// Add opinion for all relatives. They will be proud.
					changeOptsGoodDeeds(p, 0.3, -0.1, ev)
				} else {
					// Things go badly. We gain cowardice or carefullness.
					// TODO: Move this to a condition.
					if rand.Intn(100) < 50 {
						traumaOfTheCowardly.apply(m, p)
					} else {
						lessonForTheCareful.apply(m, p)
					}

					// We lose some popularity.
					p.Popularity.Add(-0.2)

					// We get hurt.
					ev := m.AddEvent("Exploration", fmt.Sprintf("%s encountered a %s while exploring %s and got hurt", p.Name(), t, place), p.Ref())
					// Add opinion for all relatives. They will be worried.
					changeOptsBadDeeds(p, -0.2, 0.0, ev)
				}
			case outcomeGotHurt:
				// We get hurt.
				ev := m.AddEvent("Exploration", fmt.Sprintf("%s got hurt while exploring %s", p.Name(), place), p.Ref())
				// Add opinion for all relatives.
				changeOptsBadDeeds(p, -0.2, 0.0, ev)
			case outcomeGotLost:
				// We get lost.
				ev := m.AddEvent("Exploration", fmt.Sprintf("%s got lost while exploring %s", p.Name(), place), p.Ref())
				// Add opinion for all relatives.
				changeOptsBadDeeds(p, -0.2, 0.0, ev)
			default:
				// Nothing happens and we just had a good time.
				ev := m.AddEvent("Exploration", fmt.Sprintf("%s explored %s", p.Name(), place), p.Ref())
				// Add opinion for all relatives.
				changeOptsGoodDeeds(p, 0.2, -0.1, ev)
			}
		},
	}

	// A cruel child might hurt another child or torture an animal.
	actionChildBully = &PersonalAction{
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
			return max(prob, 0)
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
	actionChildTortureAnimal = &PersonalAction{
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
			return max(prob, 0)
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
	actionWriteBook = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.01
			// Primary trait.
			if p.Traits.HasTrait(geneticshuman.TraitCareful) {
				prob = 0.2
			}
			return max(prob, 0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			var art *Artifact

			// If we have a career, we write a book about the career.
			// TODO: Make this dependent on the skills and knowledge of the person,
			// as well as motivation, and ambition.
			if p.Career != nil {
				// Write a book about the career.
				art = NewBookInstruction(p.Career.Profession.Name)
			} else {
				// Write a book.
				art = NewBook()
			}

			// Add item to inventory.
			p.addArtifact(m, art)
		},
	}
)

// pickAction picks an action based on the probabilities of the actions.
func pickAction(p *Person, actions []*PersonalAction) *PersonalAction {
	totalProbability := 0.0
	var remainingActions []*PersonalAction
	for _, action := range actions {
		// Check if the basic requirements are met.
		if !action.requires(p) {
			continue
		}

		// Get the probability of the action.
		probability := action.probability(p)
		if probability <= 0 {
			continue // Skip actions with zero probability.
		}

		// Add the probability to the total.
		totalProbability += probability
		remainingActions = append(remainingActions, action)
	}

	// Pick a random action from the remaining actions.
	randomValue := rand.Float64() * totalProbability
	for _, action := range remainingActions {
		probability := action.probability(p)
		randomValue -= probability
		if randomValue < 0 {
			return action
		}
	}

	// Should never reach here (all probabilities should be covered)
	// log.Println("no action picked")
	return nil
}

func (m *Civ) tickAction(p *Person, nDays int) {
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
	if p.Career != nil {
		if action := pickAction(p, p.Career.Profession.Actions); action != nil {
			action.Execute(m, p)
		}
	}

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

	actions := []*PersonalAction{
		actionMurder,
		actionCruelty,
		actionHangout,
		actionKindness,
		actionIdle,
		actionLoseArtifact,
		actionWriteBook,
		actionChildPlay,
		actionChildExploration,
		actionChildBully,
		actionChildTortureAnimal,
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

}
