package genworldvoronoi

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/Flokey82/go_gens/genlanguage"
)

// Leadership represents the leadership of a tribe or faction.
type Leadership struct {
	// TODO: Add leadership properties.
	// Type (e.g. democratic, autocratic, etc.
	// Titles (e.g. king, council, etc.)
	// Leader(s) (e.g. council, king, etc.)
	Leaders []string
	// Influence  ClampedVal // Influence or popularity will determine the power of the leadership.
	Popularity ClampedVal // Popularity will determine the happiness of the tribe.
}

// TODO: Also add culture maybe?
func genLeadership(lang *genlanguage.Language) *Leadership {
	leadership := &Leadership{
		Leaders: []string{lang.MakeFirstName() + " " + lang.MakeLastName()},
		// Influence:  1.0,
		Popularity: 1.0,
	}
	return leadership
}

// String returns a string representation of the leadership.
func (l *Leadership) String() string {
	return fmt.Sprintf("%s (%f)", strings.Join(l.Leaders, ", "), l.Popularity)
}

// Faction represents a faction within a tribe or a city state.
type Faction struct {
	Name string // Name of the faction.
	*Leadership
	// TODO: Add faction properties.
	// Type (e.g. religious, military, etc.)
}

// String returns a string representation of the faction.
func (f *Faction) String() string {
	return fmt.Sprintf("%s (%s)", f.Name, f.Leadership.String())
}

// genFaction generates a new faction with the given name.
func genFaction(lang *genlanguage.Language) *Faction {
	// TODO: Migrate the name generation to a dedicated function.
	var name string
	if rand.Float64() < 0.5 {
		name = "The "
	}
	prefixes := []string{"Brothers", "Sisters", "Warriors", "Mystics", "Hunters", "Fishers", "Farmers", "Craftsmen", "Artisans"}
	name += prefixes[rand.Intn(len(prefixes))]

	if rand.Float64() < 0.5 {
		name += " of "
		if rand.Float64() < 0.5 {
			suffixes := []string{"the Sun", "the Moon", "the Stars", "the Earth", "the Sea", "the Mountains", "the Forest", "the River", "the Lake", "the Ocean"}
			name += suffixes[rand.Intn(len(suffixes))]
		} else {
			// TODO: Like religion, allow place names, etc.
			name += lang.MakeName()
		}
	}

	f := &Faction{
		Name:       name,
		Leadership: genLeadership(lang),
	}
	return f
}
