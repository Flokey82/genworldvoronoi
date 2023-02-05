package genworldvoronoi

import (
	"strings"

	"github.com/Flokey82/go_gens/genlanguage"
	"github.com/Flokey82/go_gens/genreligion"
)

// getOrganizedReligionName generates a name for the given form and deity at the given center.
// This code is based on:
// https://github.com/Azgaar/Fantasy-Map-Generator/blob/master/modules/religions-generator.js
func (m *Civ) getOrganizedReligionName(rlgGen *genreligion.Generator, c *Culture, lang *genlanguage.Language, deity *genreligion.Deity, rel *genreligion.Classification, r int) (string, string) {
	if c == nil {
		return "MISSING_CULTURE", "MISSING_CULTURE"
	}

	// Returns the name of the city, -state, or culture at the given region.
	place := func() string {
		// Get the name of the city at the region.
		for _, city := range m.Cities {
			if city.ID == r {
				// Trim the vowels from the name and return it.
				return TrimVowels(strings.Split(city.Name, " ")[0], 3)
			}
		}

		// If unsuccessful, Check if we have a state at the region.
		if stateId := m.RegionToCityState[r]; stateId >= 0 {
			for _, city := range m.Cities {
				if city.ID == stateId {
					// Trim the vowels from the name and return it.
					return TrimVowels(strings.Split(city.Name, " ")[0], 3)
				}
			}
		}

		// If unsuccessful, use the culture name.
		// Trim the vowels from the name and return it.
		return TrimVowels(strings.Split(c.Name, " ")[0], 3)
	}

	// Attempt to generate a name for the religion.
	switch rlgGen.RandGenMethod() {
	case genreligion.MethodFaithOfSupreme:
		if deity != nil {
			// Example: "Grognark, The Supreme Being" -> "Faith of Grognark, The Supreme Being"
			return rlgGen.GenNameFaitOfSupreme(deity.FullName()), ReligionExpGlobal
		}
	case genreligion.MethodRandomType:
		// GenNamedTypeOfForm generates a name for a named religion type based on a given
		// religion form ("Polytheism", "Dualism", etc).
		// E.g. "Pradanium deities".
		return lang.MakeName() + " " + rel.Type, ReligionExpGlobal
	case genreligion.MethodPlaceIanType:
		// GenNamedTypeOfForm generates a name for a named religion type based on a given
		// religion form ("Polytheism", "Dualism", etc).
		// E.g. "Pradanium deities".
		return genlanguage.GetAdjective(place()) + "ian " + rel.Type, ReligionExpState
	case genreligion.MethodCultureType:
		// GenNamedTypeOfForm generates a name for a named religion type based on a given
		// religion form ("Polytheism", "Dualism", etc).
		// E.g. "Pradanium deities".
		return c.Name + " " + rel.Type, ReligionExpCulture
	case genreligion.MethodSurpremeIsm:
		if deity != nil {
			// Example: "Grognark, The Supreme Being" -> "Grognarkism"
			return rlgGen.GenNamedIsm(deity.Name), ReligionExpGlobal
		}
	case genreligion.MethodRandomIsm:
		return rlgGen.GenNamedIsm(lang.MakeName()), ReligionExpGlobal
	case genreligion.MethodCultureIsm:
		return rlgGen.GenNamedIsm(c.Name), ReligionExpCulture
	case genreligion.MethodPlaceIsm:
		return place() + "ism", ReligionExpState
	}
	return rlgGen.GenNamedIsm(lang.MakeName()), ReligionExpGlobal
}

const (
	// Expansion modes.
	ReligionExpGlobal  = "global"
	ReligionExpState   = "state"
	ReligionExpCulture = "culture"
)
