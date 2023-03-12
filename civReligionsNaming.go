package genworldvoronoi

import (
	"strings"

	"github.com/Flokey82/go_gens/genlanguage"
	"github.com/Flokey82/go_gens/genreligion"
	"github.com/Flokey82/go_gens/genstory"
)

func (m *Civ) getFolkReligionName(rlgGen *genreligion.Generator, c *Culture, lang *genlanguage.Language, deity *genreligion.Deity, rel *genreligion.Classification, r int) (*genstory.Generated, string) {
	// Calculate all available tokens.
	//
	// TODO: Find a way to only generate the tokens when needed.
	// This might work by using the modifier functions of the generator,
	// which are only called when a token is used.
	var tokens []genstory.TokenReplacement
	tokens = append(tokens, genstory.TokenReplacement{
		Token:       genreligion.TokenCulture,
		Replacement: c.Name,
	}, genstory.TokenReplacement{
		Token:       genreligion.TokenType,
		Replacement: rel.Type,
	})
	if deity != nil {
		tokens = append(tokens, genstory.TokenReplacement{
			Token:       genreligion.TokenSurpreme,
			Replacement: deity.Name,
		})
	}

	// Generate the name and method.
	gen, err := rlgGen.GenFaithName(tokens)
	if err != nil {
		return nil, err.Error()
	}
	return gen, ReligionExpCulture
}

// getOrganizedReligionName generates a name for the given form and deity at the given center.
// This code is based on:
// https://github.com/Azgaar/Fantasy-Map-Generator/blob/master/modules/religions-generator.js
func (m *Civ) getOrganizedReligionName(rlgGen *genreligion.Generator, c *Culture, lang *genlanguage.Language, deity *genreligion.Deity, rel *genreligion.Classification, r int) (*genstory.Generated, string) {
	// Returns the name of the city, -state, or culture at the given region.
	place := func() string {
		// Get the name of the city at the region.
		for _, city := range m.Cities {
			if city.ID == r {
				return city.Name
			}
		}

		// If unsuccessful, Check if we have a state at the region
		// and use the capital name.
		if stateId := m.RegionToCityState[r]; stateId >= 0 {
			for _, city := range m.Cities {
				if city.ID == stateId {
					return city.Name
				}
			}
		}

		// TODO: Try the empire name.
		return "" // If unsuccessful, return an empty string.
	}

	// Calculate all available tokens.
	//
	// TODO: Find a way to only generate the tokens when needed.
	// This might work by using the modifier functions of the generator,
	// which are only called when a token is used.
	var tokens []genstory.TokenReplacement
	tokens = append(tokens, genstory.TokenReplacement{
		Token:       genreligion.TokenCulture,
		Replacement: c.Name,
	}, genstory.TokenReplacement{
		Token:       genreligion.TokenType,
		Replacement: rel.Type,
	}, genstory.TokenReplacement{
		Token:       genreligion.TokenRandom,
		Replacement: lang.MakeName(), // TODO: Use the religion generator to generat religous terms.
	})

	// Try to find a location token.
	if place := place(); place != "" {
		tokens = append(tokens, genstory.TokenReplacement{
			Token:       genreligion.TokenPlace,
			Replacement: strings.Split(place, " ")[0],
		})
	}

	// If we have a deity, add it to the tokens.
	if deity != nil {
		tokens = append(tokens, genstory.TokenReplacement{
			Token:       genreligion.TokenSurpreme,
			Replacement: deity.Name,
		})
	}

	// Generate the name and method.
	gen, err := rlgGen.GenFaithName(tokens)
	if err != nil {
		return nil, err.Error()
	}

	// Select expansion based on the method which indicates the
	// focus of the faith (culture, location, or... global).
	var expansion string
	switch gen.Template {
	case genreligion.MethodCultureIsm,
		genreligion.MethodCultureType:
		expansion = ReligionExpCulture
	case genreligion.MethodPlaceIanType,
		genreligion.MethodPlaceIsm:
		expansion = ReligionExpState
	default:
		expansion = ReligionExpGlobal
	}
	return gen, expansion
}

const (
	// Expansion modes.
	ReligionExpGlobal  = genreligion.ReligionExpGlobal
	ReligionExpState   = genreligion.ReligionExpState
	ReligionExpCulture = genreligion.ReligionExpCulture
)
