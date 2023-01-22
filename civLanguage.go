package genworldvoronoi

import (
	"github.com/Flokey82/go_gens/genlanguage"
)

var GenLanguage = genlanguage.GenLanguage

var newFantasyName = genlanguage.NewFantasyName

// GetAdjective get adjective form from noun
var GetAdjective = genlanguage.GetAdjective

// IsVowel returns true if the given rune is a vowel.
var IsVowel = genlanguage.IsVowel

// TrimVowels remove vowels from the end of the string.
var TrimVowels = genlanguage.TrimVowels

// GetNounPlural returns the plural form of a noun.
// This takes in account "witch" and "fish" which are
// irregular.
var GetNounPlural = genlanguage.GetNounPlural
