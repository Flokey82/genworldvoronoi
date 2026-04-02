package civ2

import (
	"fmt"
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

func (m *Civ) tickAction(p *Person, nDays int) {
	// There should be different actions for each person.
	// TODO: Add Career/Profession actions.

	actions := []*PersonalAction{
		actionIdle,
		actionHangout,
		actionChildPlay,
		actionVillainCruelty,
		actionVillainChildBully,
		actionVillainChildTortureAnimal,
		actionHeroKindness,
		actionChangeName,
		actionLoseArtifact,
		actionWriteBook,
		actionVillainMurder,
		actionHeroAdventure,
		actionHeroChildExploration,
	}

	// Pick an action based on probability.
	if action := pickAction(p, actions); action != nil {
		action.Execute(m, p)
	}
}

// pickAction picks an action based on the probabilities of the actions.
func pickAction(p *Person, actions []*PersonalAction) *PersonalAction {
	totalProbability := 0.0
	var remainingActions []*PersonalAction
	for _, action := range actions {
		// Check if the basic requirements are met.
		if action.requires != nil && !action.requires(p) {
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

	if len(remainingActions) == 0 {
		return nil
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

	return nil
}

func (m *Civ) getPeopleAt(r int) []*Person {
	var people []*Person
	for _, p := range m.People {
		if p.Region == r {
			people = append(people, p)
		}
	}
	return people
}

func pickVictim(m *Civ, p *Person) *relation {
	// Either pick from people we know or relatives.
	if rand.Intn(100) < 50 {
		// Pick from relatives.
		options := pickRelatives(p)
		filteredOptions := make([]*relation, 0, len(options))
		for i := range options {
			option := options[i]
			if option.p != p && !option.p.Dead() && p.Region == option.p.Region {
				filteredOptions = append(filteredOptions, &option)
			}
		}

		// If we have relatives, pick one.
		if len(filteredOptions) > 0 {
			// Sort by opinion and comparison.
			sort.Slice(filteredOptions, func(i, j int) bool {
				return p.Opinions.GetOpinion(filteredOptions[i].p)+p.compare(filteredOptions[i].p) < p.Opinions.GetOpinion(filteredOptions[j].p)+p.compare(filteredOptions[j].p)
			})

			// Pick a random person from the top 3.
			return filteredOptions[rand.Intn(min(len(filteredOptions), 3))]
		}
	} else {
		peopleWeknow := p.Opinions.People
		filteredNearby := make([]*Person, 0, len(peopleWeknow))
		for _, person := range peopleWeknow {
			if person != p && !person.Dead() && person.Region == p.Region {
				filteredNearby = append(filteredNearby, person)
			}
		}

		if len(filteredNearby) > 0 {
			// Sort by opinion and comparison.
			sort.Slice(filteredNearby, func(i, j int) bool {
				return p.Opinions.GetOpinion(filteredNearby[i])+p.compare(filteredNearby[i]) < p.Opinions.GetOpinion(filteredNearby[j])+p.compare(filteredNearby[j])
			})

			// Pick a random person from the top 3.
			picked := filteredNearby[rand.Intn(min(len(filteredNearby), 3))]

			// Get the relation to the person.
			relationToPerson := p.GetRelationToPerson(picked)
			if relationToPerson != nil {
				return relationToPerson
			}
			return &relation{
				p:                picked,
				relationOfPerson: "acquaintance",
				relationToPerson: "acquaintance",
			}
		}
	}

	// Pick from nearby.
	nearby := m.getPeopleAt(p.Region)

	// Filter out the current person
	filteredNearby := make([]*Person, 0, len(nearby))
	for _, person := range nearby {
		if person != p && !person.Dead() {
			filteredNearby = append(filteredNearby, person)
		}
	}

	if len(filteredNearby) == 0 {
		return nil
	}

	// Sort by opinion and comparison.
	sort.Slice(filteredNearby, func(i, j int) bool {
		return p.Opinions.GetOpinion(filteredNearby[i])+p.compare(filteredNearby[i]) < p.Opinions.GetOpinion(filteredNearby[j])+p.compare(filteredNearby[j])
	})

	// Pick a random person from the top 3.
	picked := filteredNearby[rand.Intn(min(len(filteredNearby), 3))]

	// Get the relation to the person.
	return p.GetRelationToPerson(picked)
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
	actionHeroAdventure = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.1
			if p.Traits.HasTrait(geneticshuman.TraitBrave) {
				prob += 0.5
			}
			if p.Traits.HasTrait(geneticshuman.TraitAmbitious) {
				prob += 0.2
			}
			if p.Traits.HasTrait(geneticshuman.TraitCareless) {
				prob -= 0.1
			}
			if p.Spouse != nil {
				prob -= 0.2
			}
			if len(p.Children) > 0 {
				prob -= 0.2
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			historyMsg := fmt.Sprintf("%s went on an adventure", p.Name())
			ev := m.History.AddEvent("Adventure", historyMsg, p.Ref())
			changeOptsGoodDeeds(p, 0.1, -0.1, ev)
		},
	}

	actionVillainMurder = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.0
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				prob += 0.05
			}
			if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
				prob += 0.02
			}
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob -= 0.1
			}
			if p.Traits.HasTrait(geneticshuman.TraitCareful) {
				prob -= 0.05
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			victim := pickVictim(m, p)
			if victim == nil {
				return
			}
			if rand.Intn(100) < 50 {
				methods := []string{"stabbed", "shot", "poisoned", "strangled", "drowned", "burned", "beaten"}
				method := methods[rand.Intn(len(methods))]
				ev := m.killPerson(victim.p, fmt.Sprintf("(%s) being %s by %s (%s)", victim.relationOfPerson, method, p.Name(), victim.relationToPerson))
				if !p.Traits.HasTrait(geneticshuman.TraitCareful) {
					changeOptsBadDeeds(p, -0.9, -0.1, ev)
				}
			} else {
				ev := m.History.AddEvent("Cruelty", fmt.Sprintf("%s (%s) was attacked by %s (%s)", victim.p.Name(), victim.relationOfPerson, p.Name(), victim.relationToPerson), p.Ref())
				victim.p.Opinions.AddOpinion(p, -1.0, ev)
				if !p.Traits.HasTrait(geneticshuman.TraitCareful) {
					changeOptsBadDeeds(p, -0.7, 0.1, ev)
					p.Popularity.Add(-0.4)
				}
			}
		},
	}

	actionVillainCruelty = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.01
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				prob += 0.2
			}
			if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
				prob += 0.1
			}
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob -= 0.1
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			victim := pickVictim(m, p)
			if victim == nil {
				return
			}
			ev := m.History.AddEvent("Cruelty", fmt.Sprintf("%s (%s) was cruel to %s (%s)", p.Name(), victim.relationToPerson, victim.p.Name(), victim.relationOfPerson), p.Ref())
			victim.p.Opinions.AddOpinion(p, -0.7, ev)

			if len(victim.p.Artifacts) > 0 && rand.Intn(100) < 50 {
				art := victim.p.Artifacts[rand.Intn(len(victim.p.Artifacts))]
				victim.p.transferArtifact(m, art, p)
				m.History.AddEvent("Artifact", fmt.Sprintf("%s was stolen from %s by %s", art.NameWithArticle(), victim.p.Name(), p.Name()), p.Ref())
			}

			if !p.Traits.HasTrait(geneticshuman.TraitCareful) {
				changeOptsBadDeeds(p, -0.5, 0.15, ev)
				p.Popularity.Add(-0.1)
			}
		},
	}

	actionHangout = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.5
			if p.Traits.HasTrait(geneticshuman.TraitGregarious) {
				prob += 0.3
			}
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob += 0.1
			}
			if p.Traits.HasTrait(geneticshuman.TraitShy) {
				prob -= 0.2
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			nearby := m.getPeopleAt(p.Region)
			if len(nearby) <= 1 {
				return
			}
			var filteredNearby []*Person
			for _, person := range nearby {
				if person == p || person.Dead() || isRelated(p, person) {
					continue
				}
				filteredNearby = append(filteredNearby, person)
			}
			if len(filteredNearby) == 0 {
				return
			}
			sort.Slice(filteredNearby, func(i, j int) bool {
				return p.Opinions.GetOpinion(filteredNearby[i]) > p.Opinions.GetOpinion(filteredNearby[j])
			})
			picked := filteredNearby[rand.Intn(min(len(filteredNearby), 4))]

			activities := []string{"drinking", "eating", "talking", "walking", "fishing", "hunting", "playing games"}
			activity := activities[rand.Intn(len(activities))]

			cmp := p.compare(picked)
			if cmp > 0.5 {
				ev := m.History.AddEvent("Hangout", fmt.Sprintf("%s went %s with %s", p.Name(), activity, picked.Name()), p.Ref())
				picked.Opinions.AddOpinion(p, 0.7, ev)
				p.Opinions.AddOpinion(picked, 0.7, ev)
			} else {
				ev := m.History.AddEvent("Hangout", fmt.Sprintf("%s went %s with %s", p.Name(), activity, picked.Name()), p.Ref())
				picked.Opinions.AddOpinion(p, 0.3, ev)
				p.Opinions.AddOpinion(picked, 0.3, ev)
			}
		},
	}

	actionHeroKindness = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.1
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob += 0.5
			}
			if p.Traits.HasTrait(geneticshuman.TraitTrusting) {
				prob += 0.2
			}
			if p.Traits.HasTrait(geneticshuman.TraitCareless) {
				prob -= 0.1
			}
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				prob -= 0.2
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			peopleNearby := m.getPeopleAt(p.Region)
			if len(peopleNearby) <= 1 {
				return
			}
			var personInNeed *Person
			for i := 0; i < 10; i++ {
				personInNeed = peopleNearby[rand.Intn(len(peopleNearby))]
				if personInNeed != p && !personInNeed.Dead() {
					break
				}
			}

			if personInNeed == nil {
				return
			}

			options := []string{
				"helped",
				"gave food to",
				"offered shelter to",
				"gave a gift to",
			}
			action := options[rand.Intn(len(options))]

			if personInNeed.Traits.HasTrait(geneticshuman.TraitCruel|geneticshuman.TraitAggressive|geneticshuman.TraitDeceptive) {
				if rand.Intn(100) < 10 {
					personInNeed.Traits &^= geneticshuman.TraitCruel | geneticshuman.TraitAggressive | geneticshuman.TraitDeceptive
					personInNeed.Popularity.Add(0.5)
				}
			}

			ev := m.History.AddEvent("Kindness", fmt.Sprintf("%s %s %s", p.Name(), action, personInNeed.Name()), p.Ref())
			personInNeed.Opinions.AddOpinion(p, 1.0, ev)
			changeOptsGoodDeeds(p, 0.7, -0.15, ev)
			p.Popularity.Add(0.1)
		},
	}

	actionLoseArtifact = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.01
			if p.Traits.HasTrait(geneticshuman.TraitDeceptive) {
				prob += 0.2
			}
			if p.Traits.HasTrait(geneticshuman.TraitCareless) {
				prob += 0.1
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return len(p.Artifacts) > 0
		},
		consequences: func(m *Civ, p *Person) {
			art := p.Artifacts[rand.Intn(len(p.Artifacts))]
			p.removeArtifact(m, art)
			ev := m.History.AddEvent("Artifact", fmt.Sprintf("%s lost %s", p.Name(), art.NameWithArticle()), p.Ref())
			changeOptsBadDeeds(p, -0.3, 0.1, ev)

			if rand.Intn(100) < 50 {
				peopleAtRegion := m.getPeopleAt(p.Region)
				var person *Person
				for i := 0; i < 10; i++ {
					person = peopleAtRegion[rand.Intn(len(peopleAtRegion))]
					if person != p && !person.Dead() {
						break
					}
				}
				if person != nil {
					m.History.AddEvent("Artifact", fmt.Sprintf("%s found %s", person.Name(), art.NameWithArticle()), person.Ref())
					person.addArtifact(m, art)

					if p.Traits.HasTrait(geneticshuman.TraitKind) || p.Traits.HasTrait(geneticshuman.TraitHonest) || p.Opinions.GetOpinion(person) > 0.5 {
						m.History.AddEvent("Artifact", fmt.Sprintf("%s returned %s to %s", person.Name(), art.NameWithArticle(), p.Name()), person.Ref())
						person.transferArtifact(m, art, p)
						p.Opinions.AddOpinion(person, 0.7, ev)
						p.Karma.Add(0.7)
					}
				}
			}
		},
	}

	actionChildPlay = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.3
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob += 0.5
			}
			if p.Traits.HasTrait(geneticshuman.TraitTrusting) {
				prob += 0.2
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return p.Age > 2 && p.Age < 12
		},
		consequences: func(m *Civ, p *Person) {
			peopleNearby := m.getPeopleAt(p.Region)
			var children []*Person
			for _, c := range peopleNearby {
				if c != p && c.Age < 12 && c.Age > 2 {
					children = append(children, c)
				}
			}

			if len(children) == 0 {
				return
			}

			sort.Slice(children, func(i, j int) bool {
				return p.Opinions.GetOpinion(children[i]) > p.Opinions.GetOpinion(children[j])
			})

			child := children[rand.Intn(min(len(children), 4))]
			options := []string{"hide and seek", "tag", "catch", "climbing trees", "swimming"}
			game := options[rand.Intn(len(options))]
			ev := m.History.AddEvent("Child Play", fmt.Sprintf("%s played %s with %s", p.Name(), game, child.Name()), p.Ref())
			child.Opinions.AddOpinion(p, 0.7, ev)
			changeOptsGoodDeeds(p, 0.4, -0.1, ev)
		},
	}

	actionHeroChildExploration = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.1
			if p.Traits.HasTrait(geneticshuman.TraitBrave) {
				prob += 0.2
			}
			if p.Traits.HasTrait(geneticshuman.TraitCareless) {
				prob += 0.1
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return p.Age > 8 && p.Age < 16
		},
		consequences: func(m *Civ, p *Person) {
			rls := &genstory.Rules{
				Expansions: map[string][]string{
					"location":        {"forest", "cave", "river", "mountain", "house", "graveyard", "ruin"},
					"adjective":       {"dark", "mysterious", "abandoned", "haunted", "enchanted", "hidden"},
					"location_phrase": {"[adjective:a] [location]", "[location:a]"},
				},
				Start: "[location_phrase]",
			}
			placeStory := rls.NewStory(time.Now().UnixNano())
			place, _ := placeStory.Expand()

			switch rand.Intn(10) {
			case 0: // Found treasure
				treasure := m.NewTreasure()
				p.addArtifact(m, treasure)
				m.History.AddEvent("Artifact", fmt.Sprintf("%s was found by %s in %s", treasure.NameWithArticle(), p.Name(), place), p.Ref())
				ev := m.History.AddEvent("Exploration", fmt.Sprintf("%s found a %s in %s", p.Name(), treasure.NameWithArticle(), place), p.Ref())
				changeOptsGoodDeeds(p, 0.7, -0.4, ev)
			case 1: // Danger
				animal := []string{"bear", "snake", "wolf", "wildcat", "boar"}
				t := animal[rand.Intn(len(animal))]
				if len(p.Artifacts) > 0 && rand.Intn(100) < 50 {
					art := p.Artifacts[rand.Intn(len(p.Artifacts))]
					m.History.AddEvent("Artifact", fmt.Sprintf("%s was used by %s to scare away a %s", art.NameWithArticle(), p.Name(), t), p.Ref())
					ev := m.History.AddEvent("Exploration", fmt.Sprintf("%s encountered a %s while exploring %s and was able to chase it away", p.Name(), t, place), p.Ref())
					changeOptsGoodDeeds(p, 0.3, -0.1, ev)
				} else if rand.Intn(100) < 50 {
					if p.Conditions == nil {
						p.Conditions = NewConditions()
					}
					p.Conditions.Add(lessonForTheBrave, m, p)
					ev := m.History.AddEvent("Exploration", fmt.Sprintf("%s encountered a %s while exploring %s and was able to chase it away", p.Name(), t, place), p.Ref())
					changeOptsGoodDeeds(p, 0.3, -0.1, ev)
				} else {
					if p.Conditions == nil {
						p.Conditions = NewConditions()
					}
					if rand.Intn(100) < 50 {
						p.Conditions.Add(traumaOfTheCowardly, m, p)
					} else {
						p.Conditions.Add(lessonForTheCareful, m, p)
					}
					m.History.AddEvent("Exploration", fmt.Sprintf("%s encountered a %s while exploring %s and was hurt", p.Name(), t, place), p.Ref())
				}
			}
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
			rels := getLivingRelatives(p)
			for _, c := range rels {
				p.Opinions.AddOpinion(c, 0.0, nil)
			}
		},
	}

	actionVillainChildBully = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.01
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				prob = 0.2
			} else if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob = 0.0
			}
			if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
				prob += 0.2
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return p.Age < 12 && p.Age > 2
		},
		consequences: func(m *Civ, p *Person) {
			peopleNearby := m.getPeopleAt(p.Region)
			var children []*Person
			for _, c := range peopleNearby {
				if c != p && c.Age < 12 && c.Age > 2 {
					children = append(children, c)
				}
			}
			if len(children) == 0 {
				return
			}
			sort.Slice(children, func(i, j int) bool {
				return p.Opinions.GetOpinion(children[i]) < p.Opinions.GetOpinion(children[j])
			})
			child := children[rand.Intn(min(len(children), 4))]
			ev := m.History.AddEvent("Child Cruelty", fmt.Sprintf("%s bullied %s", p.Name(), child.Name()), p.Ref())
			child.Opinions.AddOpinion(p, -1.0, ev)
			changeOptsBadDeeds(p, -0.7, 0.15, ev)
		},
	}

	actionVillainChildTortureAnimal = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.01
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				prob = 0.2
			} else if p.Traits.HasTrait(geneticshuman.TraitKind) {
				prob = 0.0
			}
			if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
				prob += 0.2
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return p.Age < 12 && p.Age > 2
		},
		consequences: func(m *Civ, p *Person) {
			animals := []string{"cat", "dog", "bird", "rabbit", "squirrel"}
			animal := animals[rand.Intn(len(animals))]
			ev := m.History.AddEvent("Child Cruelty", fmt.Sprintf("%s tortured a %s", p.Name(), animal), p.Ref())
			changeOptsBadDeeds(p, -0.5, 0.25, ev)
		},
	}

	actionWriteBook = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.01
			if p.Traits.HasTrait(geneticshuman.TraitCareful) {
				prob = 0.2
			}
			return max(prob, 0.0)
		},
		requires: func(p *Person) bool {
			return p.Age > 12
		},
		consequences: func(m *Civ, p *Person) {
			var art *Artifact
			// Profession book? (Not implemented Professions yet)
			art = m.NewBook()
			p.addArtifact(m, art)
			m.History.AddEvent("Artifact", fmt.Sprintf("%s wrote a book: %s", p.Name(), art.Name), p.Ref())
		},
	}

	actionChangeName = &PersonalAction{
		probability: func(p *Person) float64 {
			prob := 0.001
			if p.Mother != nil && p.Father != nil {
				if p.LastName != p.Mother.LastName && p.LastName != p.Father.LastName {
					return 0
				}
				if p.Opinions.GetOpinion(p.Mother) < 0 && p.Opinions.GetOpinion(p.Father) < 0 {
					prob += 0.9
				}
			}
			return prob
		},
		requires: func(p *Person) bool {
			return p.Age > 12 && p.Spouse == nil
		},
		consequences: func(m *Civ, p *Person) {
			origName := p.Name()
			p.LastName = p.Culture.Language.MakeLastName()
			if rand.Intn(100) < 50 {
				p.FirstName = p.Culture.Language.MakeFirstName()
			}
			ev := m.History.AddEvent("Name Change", fmt.Sprintf("%s changed their name to %s", origName, p.Name()), p.Ref())
			changeOptsBadDeeds(p, -0.1, 0.1, ev)
		},
	}
)


