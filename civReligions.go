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
	ID           int       // The region where the religion was founded
	Name         string    // The name of the religion
	Culture      *Culture  // The culture that the religion is based on
	Parent       *Religion // The parent religion (if any)
	Type         string    // The type of the religion
	Form         string    // The form of the religion
	Deity        string    // The main deity of the religion (if any)
	DeityMeaning string    // The meaning of the main deity of the religion (if any)
	Expansion    string    // How the religion wants to expand
	Expansionism float64   // How much the religion wants to expand
	Founded      int64     // Year when the religion was founded
}

func (r *Religion) GetDeityName() string {
	if r.Deity == "" {
		return ""
	}
	return r.Deity + ", The " + r.DeityMeaning
}

func (r *Religion) String() string {
	if r.Deity == "" {
		return fmt.Sprintf("%s (%s, %s, %s)", r.Name, r.Type, r.Expansion, r.Form)
	}
	return fmt.Sprintf("%s (%s, %s, %s)\n=%s", r.Name, r.Type, r.Expansion, r.Form, r.GetDeityName())
}

// genFolkReligion generates a folk religion for the given culture.
func (m *Civ) genFolkReligion(c *Culture) *Religion {
	return m.placeReligionAt(c.ID, -1, ReligionGroupFolk, c)
}

// genOrganizedReligions generates organized religions.
func (m *Civ) genOrganizedReligions() []*Religion {
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
		religions = append(religions, m.placeReligionAt(c.ID, -1, ReligionGroupOrganized, c.Culture))
	}
	return religions
}

// placeReligionAt places a religion of the given group at the given region.
// This code is based on:
// https://github.com/Azgaar/Fantasy-Map-Generator/blob/master/modules/religions-generator.js
func (m *Civ) placeReligionAt(r int, founded int64, group string, culture *Culture) *Religion {
	// If founded is -1, we take the current year.
	if founded == -1 {
		founded = m.History.GetYear()
	}
	form := rw(forms[group])

	relg := &Religion{
		ID:      r,
		Culture: culture,
		Type:    group,
		Form:    form,
		Founded: founded,
	}

	// If appropriate, add a deity to the religion.
	if form != ReligionFormNontheism && form != ReligionFormAnimism {
		relg.Deity, relg.DeityMeaning = getDeityName(culture)
	}

	// Select name, expansion, and expansionism.
	if group == ReligionGroupOrganized {
		// Check if we have a state at this location
		name, expansion := m.getReligionName(form, relg.GetDeityName(), r)
		if expansion == ReligionExpState && m.RegionToCityState[r] == -1 {
			expansion = ReligionExpGlobal
		}
		if expansion == ReligionExpCulture && culture == nil {
			expansion = ReligionExpGlobal
		}
		relg.Name = name
		relg.Expansion = expansion
		relg.Expansionism = culture.Expansionism*rand.Float64()*1.5 + 0.5
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
	} else if group == ReligionGroupFolk {
		relg.Name = m.getFolkReligionName(culture, form)
		relg.Expansion = ReligionExpCulture
		relg.Expansionism = culture.Expansionism * rand.Float64() * 1.5
	}
	m.Religions = append(m.Religions, relg)
	return relg
}
