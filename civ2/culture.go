package civ2

import (
	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/go_gens/genlanguage"
)

// Culture represents a culture.
type Culture struct {
	ID       int                   // Region where the culture originates
	Name     string                // Name of the culture
	Type     civ.CultureType       // Type of the culture
	Language *genlanguage.Language // Language of the culture
	Religion *Religion             // Religion of the culture
	Skills   []*Skill              // Skills developed by the culture
}

// GetID returns the ID of the culture.
func (c Culture) GetID() int {
	return c.ID
}

func (c *Culture) Ref() civ.ObjectReference {
	return civ.ObjectReference{ID: c.ID, Type: civ.ObjectTypeCulture}
}

func (m *Civ) newCulture(id int, lang *genlanguage.Language, t civ.CultureType) *Culture {
	c := &Culture{
		ID:       id,
		Name:     lang.MakeName(),
		Type:     t,
		Language: lang,
		Skills:   cultureTypeToStartingSkills(t),
	}
	m.Cultures.PlaceObjectAt(c, id)
	return c
}

func (m *Civ) GetCulture(id int) *Culture {
	for _, c := range m.Cultures.Objects {
		if c.ID == id {
			return c
		}
	}
	return nil
}
