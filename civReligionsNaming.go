package genworldvoronoi

import (
	"strings"

	"github.com/Flokey82/go_gens/genlanguage"
)

var DeityMeaningApproaches = genlanguage.DeityMeaningApproaches

func (m *Civ) getFolkReligionName(c *Culture, form string) string {
	return c.Name + " " + rw(types[form])
}

// getReligionName generates a name for the given form and deity at the given center.
// This code is based on:
// https://github.com/Azgaar/Fantasy-Map-Generator/blob/master/modules/religions-generator.js
func (m *Civ) getReligionName(form, deity string, r int) (string, string) {
	// Returns a random name from the culture at the given region.
	random := func() string {
		c := m.GetCulture(r)
		return c.Language.MakeName()
	}

	// Returns a random, weighted type for the religion form.
	rType := func() string {
		return rw(types[form])
	}

	// Splits the deity name into parts and returns the first part.
	// Example: "Grognark, The Supreme Being" -> "Grognark"
	supreme := func() string {
		return strings.Split(deity, (", "))[0]
	}

	// Returns the name of the culture at the given region.
	culture := func() string {
		c := m.GetCulture(r)
		return c.Name
	}

	// Returns the name of the cit or state at the given region.
	place := func(adj string) string {
		// Check if we have a city at the region.
		var base string
		for _, city := range m.Cities {
			if city.ID == r {
				base = city.Name
				break
			}
		}
		if base == "" {
			// Check if we have a state at the region.
			if stateId := m.RegionToCityState[r]; stateId != 0 {
				for _, city := range m.Cities {
					if city.ID == stateId {
						base = city.Name
						break
					}
				}
			}
		}
		if base == "" {
			// Check if we have a culture at the region.
			if cultureId := m.RegionToCulture[r]; cultureId != 0 {
				base = m.Cultures[cultureId].Name
			}
		}
		if base == "" {
			return "TODO_PLACE"
		}
		name := TrimVowels(strings.Split(base, " ")[0], 3)
		if adj != "" {
			return genlanguage.GetAdjective(name)
		}
		return name
	}

	switch rw(GenReligionMethods) {
	case MethodRandomType:
		return random() + " " + rType(), ReligionExpGlobal
	case MethodRandomIsm:
		return TrimVowels(random(), 3) + "ism", ReligionExpGlobal
	case MethodSurpremeIsm:
		if deity != "" {
			return TrimVowels(supreme(), 3) + "ism", ReligionExpGlobal
		}
	case MethodFaithOfSupreme:
		if deity != "" {
			// Select a random name from the list.
			// but ensure that the name is not a subset of the deity name
			// and vice versa. This is to avoid names like "The Way of The Way".
			var prefix string
			for i := 0; i < 100; i++ {
				prefix = ra([]string{
					"Faith",
					"Way",
					"Path",
					"Word",
					"Truth",
					"Law",
					"Order",
					"Light",
					"Darkness",
					"Gift",
					"Grace",
					"Witnesses",
					"Servants",
					"Messengers",
					"Believers",
					"Disciples",
					"Followers",
					"Children",
					"Brothers",
					"Sisters",
					"Brothers and Sisters",
					"Sons",
					"Daughters",
					"Sons and Daughters",
					"Brides",
					"Grooms",
					"Brides and Grooms",
				})
				if !strings.Contains(strings.ToLower(deity), strings.ToLower(prefix)) &&
					!strings.Contains(strings.ToLower(prefix), strings.ToLower(deity)) {
					break
				}
			}
			return prefix + " of " + supreme(), ReligionExpGlobal
		}
	case MethodPlaceIsm:
		return place("") + "ism", ReligionExpState
	case MethodCultureIsm:
		return TrimVowels(culture(), 3) + "ism", ReligionExpCulture
	case MethodPlaceIanType:
		return place("adj") + " " + rType(), ReligionExpState
	case MethodCultureType:
		return culture() + " " + rType(), ReligionExpCulture
	}
	return TrimVowels(random(), 3) + "ism", ReligionExpGlobal
}

const (
	MethodRandomType     = "Random + type"
	MethodRandomIsm      = "Random + ism"
	MethodSurpremeIsm    = "Supreme + ism"
	MethodFaithOfSupreme = "Faith of + Supreme"
	MethodPlaceIsm       = "Place + ism"
	MethodCultureIsm     = "Culture + ism"
	MethodPlaceIanType   = "Place + ian + type"
	MethodCultureType    = "Culture + type"
)

// genReligionMethods contains a map of religion name generation
// methods and their relative chance to be selected.
var GenReligionMethods = map[string]int{
	MethodRandomType:     3,
	MethodRandomIsm:      1,
	MethodSurpremeIsm:    5,
	MethodFaithOfSupreme: 5,
	MethodPlaceIsm:       1,
	MethodCultureIsm:     2,
	MethodPlaceIanType:   6,
	MethodCultureType:    4,
}
