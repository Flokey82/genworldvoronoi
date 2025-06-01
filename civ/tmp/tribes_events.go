package civ

import (
	"fmt"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/geo"
)

func (s *simState) handleEvents(t *Tribe, curRegProp geo.RegionProp) {
	m := s.m
	// Random bad things can happen.
	// TODO:
	// - If there is a religion, the tribe might associate the random events with the gods.
	// - But there is a chance that a event is being caused by the god(s) or by the religion itself.
	// - If the tribe is in good standing with the god(s), they might get a bonus, if not, they might get a penalty.
	// - Maybe introduce some modifiers that are gained by these events.
	//
	// Good events:
	// - Good harvest could increase the population growth rate or improve soil fertility / reduce soil exhaustion.
	// - Good weather could give a bonus to positve satisfaction changes and reduce negative satisfaction changes.
	// - Good fortune could improve defense or offense of the tribe, or some magical or religious event.
	//   Maybe they gain a relic or a new skill.
	// Bad events:
	// - Bad events could reduce economic output or destroy resources.
	// - Floods could reduce population growth or destroy resources but also increase soil fertility.
	// TODO: The magnitude of change should depend on the severity of the bad or good thing.
	const (
		goodThingSatChange = 0.1
		badThingSatChange  = -0.1
	)

	type event struct {
		name      string
		satChange float64
	}

	getBadEvent := func() event {
		badEvents := []event{
			{name: "bad weather", satChange: -0.1},
			{name: "bad fortune", satChange: -0.2},
			{name: "bad harvest", satChange: -0.5},
			{name: "illness", satChange: -0.7},
		}
		if curRegProp.Mountain {
			badEvents = append(badEvents,
				event{name: "earthquake", satChange: -0.8},
				event{name: "rockslide", satChange: -0.6},
			)
		}
		if curRegProp.Lake {
			badEvents = append(badEvents, event{name: "malaria", satChange: -0.7})
		}
		if curRegProp.River || curRegProp.Lake {
			badEvents = append(badEvents, event{name: "flood", satChange: -0.6})
		}
		if curRegProp.Ocean {
			badEvents = append(badEvents, event{name: "tsunami", satChange: -0.8})
		}
		return badEvents[rand.Intn(len(badEvents))]
	}

	getGoodEvent := func() event {
		goodEvents := []event{
			{name: "good weather", satChange: 0.1},
			{name: "good fortune", satChange: 0.2},
			{name: "good harvest", satChange: 0.5},
		}
		return goodEvents[rand.Intn(len(goodEvents))]
	}

	if rand.Intn(1000) < 2 {
		// Random good things can happen, increase the satisfaction of the tribe.
		goodThing := getGoodEvent()
		t.changeSatisfaction(goodThing.satChange)

		// Add a history entry.
		historyMsg := fmt.Sprintf("Tribe %d (%d) has benefited from %s.", t.ID, t.Population, goodThing.name)
		m.History.AddEvent("Benefit", historyMsg, t.Ref())
	} else if rand.Intn(1000) < 2 {
		// Random bad things can happen, reduce the satisfaction of the tribe.
		badThing := getBadEvent()
		t.changeSatisfaction(badThing.satChange)

		// Add a history entry.
		historyMsg := fmt.Sprintf("Tribe %d (%d) has suffered from %s.", t.ID, t.Population, badThing.name)
		m.History.AddEvent("Disaster", historyMsg, t.Ref())
	}
}
