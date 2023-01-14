package genworldvoronoi

import (
	"fmt"
	"math/rand"
	"sort"
)

// Religion represents a religion in the world.
//
// TODO: Ensure we can infer symbolisms from events and other things.
//
// For example, if they worship the 99 beer bottles on the wall, we should
// be able to infer that they highly value beer and the number 99, as well
// as walls. They might be fearful of the number 100, and might have a
// taboo against the number 1.
// They might look kindly on people who can drink 99 beers in a row.
//
// Another example: If they worship the sun, we should be able to infer
// that they highly value the sun, and that they might be fearful of the
// moon. They might have a celebration during the summer solstice and consider
// a total eclipse of the sun to be a bad omen and a moon eclipse to be a good
// omen.
//
// # DEITIES AND SYMBOLS
//
// Folk religions that are purely based on the culture might warship
// nature itself, such as the sun, summer, the rain, a particular animal,
// or a particular plant. They might worship one or multiple deities that
// represent nature, like the sun god, the rain god, the god of the forest.
//
// Organized religions might either worship one or multiple gods, or a single
// person that is considered to be a god (or chosen).
//
// # GRAPH
//
// All these themes and connections could be represented as a graph, which
// would allow us to infer the relationships between deities and symbols and
// if mundane events hold any significance for a particular religion.
type Religion struct {
	ID           int
	Origin       int
	Name         string
	Culture      *Culture
	Type         string
	Form         string
	Deity        string
	Expansion    string
	Expansionism float64
	Parent       *Religion
}

func (r *Religion) String() string {
	return fmt.Sprintf("%s (%s, %s, %s), worshipping %s", r.Name, r.Type, r.Expansion, r.Form, r.Deity)
}

// genFolkReligion generates a folk religion for the given culture.
// This code is based on:
// https://github.com/Azgaar/Fantasy-Map-Generator/blob/master/modules/religions-generator.js
func (m *Civ) genFolkReligion(c *Culture) *Religion {
	r := m.newFolkReligion(c)
	m.Religions = append(m.Religions, r)
	return r
}

func (m *Civ) newFolkReligion(c *Culture) *Religion {
	form := rw(forms[ReligionGroupFolk])
	r := &Religion{
		Origin:       c.ID,
		Name:         m.getFolkReligionName(c, form),
		Culture:      c,
		Type:         ReligionGroupFolk,
		Form:         form,
		Expansion:    ReligionExpCulture,
		Expansionism: c.Expansionism * rand.Float64() * 1.5,
	}
	if form != ReligionFormAnimism {
		r.Deity = getDeityName(c)
	}
	return r
}

// genOrganizedReligion generates organized religions.
// This code is based on:
// https://github.com/Azgaar/Fantasy-Map-Generator/blob/master/modules/religions-generator.js
func (m *Civ) genOrganizedReligion() []*Religion {
	var religions []*Religion
	cities := make([]*City, len(m.Cities))
	copy(cities, m.Cities)
	sort.Slice(cities, func(i, j int) bool {
		return cities[i].Score > cities[j].Score
	})
	if len(cities) > m.NumOrganizedReligions {
		cities = cities[:m.NumOrganizedReligions]
	}
	for _, c := range cities {
		religions = append(religions, m.newOrganizedReligion(c))
	}
	return religions
}

func (m *Civ) newOrganizedReligion(c *City) *Religion {
	form := rw(forms[ReligionGroupOrganized])
	// const state = cells.state[center];
	culture := m.GetCulture(c.ID)

	var deity string
	if form != ReligionFormNontheism {
		deity = getDeityName(culture)
	}

	// Check if we have a state at this location
	name, expansion := m.getReligionName(form, deity, c.ID)
	if expansion == ReligionExpState && m.RegionToCityState[c.ID] == -1 {
		expansion = ReligionExpGlobal
	}
	if expansion == ReligionExpCulture && culture == nil {
		expansion = ReligionExpGlobal
	}

	// For now, set the origin to the city.
	origin := c.ID
	// if expansion == "state" {
	// 	origin = m.RegionToCityState[c.ID]
	// }
	// if expansion == "culture" {
	// 	origin = culture.ID
	// }

	// if (!cells.burg[center] && cells.c[center].some(c => cells.burg[c]))
	//  center = cells.c[center].find(c => cells.burg[c]);
	// const [x, y] = cells.p[center];

	// const s = spacing * gauss(1, 0.3, 0.2, 2, 2); // randomize to make the placement not uniform
	// if (religionsTree.find(x, y, s) !== undefined) continue; // to close to existing religion

	// add "Old" to name of the folk religion on this culture
	// isFolkBased := expansion == "culture" || P(0.5)
	// folk := isFolkBased && religions.find(r => r.culture === culture && r.type === "Folk");
	// if (folk && expansion === "culture" && folk.name.slice(0, 3) !== "Old") folk.name = "Old " + folk.name;

	// const origins = folk ? [folk.i] : getReligionsInRadius({x, y, r: 150 / count, max: 2});
	// const expansionism = rand(3, 8);
	// const baseColor = religions[culture]?.color || states[state]?.color || getRandomColor();
	// const color = getMixedColor(baseColor, 0.3, 0);
	return &Religion{
		Origin:       origin,
		Name:         name,
		Culture:      culture,
		Type:         ReligionGroupOrganized,
		Form:         form,
		Deity:        deity,
		Expansion:    expansion,
		Expansionism: culture.Expansionism*rand.Float64()*1.5 + 0.5,
	}
}
