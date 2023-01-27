package genworldvoronoi

import (
	"strings"

	"github.com/Flokey82/go_gens/genlanguage"
	"github.com/Flokey82/go_gens/genreligion"
)

func (m *Civ) getFolkReligionName(rlgGen *genreligion.Generator, c *Culture, form string) string {
	if c == nil {
		return "MISSING_CULTURE"
	}

	return c.Name + " " + rlgGen.RandTypeFromForm(form)
}

// getReligionName generates a name for the given form and deity at the given center.
// This code is based on:
// https://github.com/Azgaar/Fantasy-Map-Generator/blob/master/modules/religions-generator.js
func (m *Civ) getReligionName(rlgGen *genreligion.Generator, c *Culture, form, deity string, r int) (string, string) {
	if c == nil {
		return "MISSING_CULTURE", "MISSING_CULTURE"
	}

	// Returns a random name from the culture at the given region.
	random := func() string {
		return c.Language.MakeName()
	}

	// Splits the deity name into parts and returns the first part.
	// Example: "Grognark, The Supreme Being" -> "Grognark"
	supreme := func() string {
		return strings.Split(deity, (", "))[0]
	}

	// Returns the name of the culture at the given region.
	culture := func() string {
		return c.Name
	}

	// Returns the name of the city at the given region.
	city := func() string {
		for _, city := range m.Cities {
			if city.ID == r {
				return city.Name
			}
		}
		return ""
	}

	// Returns the name of the city state at the given region.
	state := func() string {
		if stateId := m.RegionToCityState[r]; stateId >= 0 {
			for _, city := range m.Cities {
				if city.ID == stateId {
					return city.Name
				}
			}
		}
		return ""
	}

	// Returns the name of the city, -state, or culture at the given region.
	place := func() string {
		// Get the name of the city at the region.
		base := city()

		// If unsuccessful, Check if we have a state at the region.
		if base == "" {
			base = state()
		}

		// If unsuccessful, use the culture name.
		if base == "" {
			base = culture()
		}

		// If unsuccessful, return a placeholder.
		if base == "" {
			return "TODO_PLACE"
		}

		// Trim the vowels from the name and return it.
		return TrimVowels(strings.Split(base, " ")[0], 3)
	}

	// Attempt to generate a name for the religion.
	switch rlgGen.RandGenMethod() {
	case genreligion.MethodFaithOfSupreme:
		if deity != "" {
			return rlgGen.GenNameFaitOfSupreme(deity), ReligionExpGlobal
		}
	case genreligion.MethodRandomType:
		return rlgGen.GenNamedTypeOfForm(random(), form), ReligionExpGlobal
	case genreligion.MethodPlaceIanType:
		placeAdj := genlanguage.GetAdjective(place()) // Generate adjective for the place.
		return rlgGen.GenNamedTypeOfForm(placeAdj+"ian", form), ReligionExpState
	case genreligion.MethodCultureType:
		return rlgGen.GenNamedTypeOfForm(culture(), form), ReligionExpCulture
	case genreligion.MethodSurpremeIsm:
		if deity != "" {
			return rlgGen.GenNamedIsm(supreme()), ReligionExpGlobal
		}
	case genreligion.MethodRandomIsm:
		return rlgGen.GenNamedIsm(random()), ReligionExpGlobal
	case genreligion.MethodCultureIsm:
		return rlgGen.GenNamedIsm(culture()), ReligionExpCulture
	case genreligion.MethodPlaceIsm:
		return place() + "ism", ReligionExpState
	}
	return rlgGen.GenNamedIsm(random()), ReligionExpGlobal
}

const (
	// Expansion modes.
	ReligionExpGlobal  = "global"
	ReligionExpState   = "state"
	ReligionExpCulture = "culture"
)
