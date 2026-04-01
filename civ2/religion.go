package civ2

import (
	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/go_gens/genreligion"
	"github.com/Flokey82/go_gens/genstory"
)

const (
	ReligionExpCulture = "culture"
	ReligionExpState   = "state"
	ReligionExpGlobal  = "global"
)

type Religion struct {
	ID                          int
	Name                        string
	NameGen                     *genstory.Generated
	Culture                     *Culture
	Parent                      *Religion
	*genreligion.Classification
	Deity                       *genreligion.Deity
	Expansion                   string
	Expansionism                float64
	Founded                     int64
}

func (r Religion) GetID() int {
	return r.ID
}

func (r *Religion) Ref() civ.ObjectReference {
	return civ.ObjectReference{
		Type: civ.ObjectTypeReligion,
		ID:   r.ID,
	}
}

func (m *Civ) GetReligion(id int) *Religion {
	return m.Religions.GetAt(id)
}

func (m *Civ) PlaceNFolkReligions(n int) []*Religion {
	var religions []*Religion
	cultures := make([]*Culture, len(m.Cultures.Objects))
	copy(cultures, m.Cultures.Objects)
	// Sort by some criteria, e.g. spirituality (if added to Culture).
	if len(cultures) > n {
		cultures = cultures[:n]
	}
	for _, c := range cultures {
		religions = append(religions, m.genFolkReligion(c))
	}
	m.ExpandReligions()
	return religions
}

func (m *Civ) genFolkReligion(c *Culture) *Religion {
	return m.placeReligionAt(c.ID, -1, genreligion.GroupFolk, c, nil)
}

func (m *Civ) placeReligionAt(r int, founded int64, group string, culture *Culture, parent *Religion) *Religion {
	if founded == -1 {
		founded = m.History.GetYear()
	}

	relg := &Religion{
		ID:      r,
		Culture: culture,
		Founded: founded,
		Parent:  parent,
	}

	rlgGen := genreligion.NewGenerator(int64(r), culture.Language)
	if parent != nil {
		relg.Classification = rlgGen.NewClassificationWithForm(group, parent.Form)
	} else {
		relg.Classification = rlgGen.NewClassification(group)
	}

	if relg.HasDeity() {
		if parent != nil && parent.HasDeity() {
			relg.Deity, _ = rlgGen.GetDeityWithApproach(parent.Deity.Meaning.Template)
		} else {
			relg.Deity, _ = rlgGen.GetDeity()
		}
	}

	// Simplified naming for now.
	relg.Name = culture.Language.MakeName() + "ism"
	relg.Expansionism = 1.0

	m.Religions.PlaceObjectAt(relg, r)
	return relg
}

func (m *Civ) tickReligions(nDays int) {
	for _, r := range m.Religions.Objects {
		m.tickReligion(r, nDays)
	}
	m.ExpandReligions()
}

func (m *Civ) tickReligion(r *Religion, nDays int) {
	// TODO: Handle religious events, schisms, etc.
}

func (m *Civ) ExpandReligions() {
	// The religious centers will be the seed points for the expansion.
	var seeds []int
	for _, r := range m.Religions.Objects {
		seeds = append(seeds, r.ID)
	}

	weightFunc := m.getTerritoryWeightFunc()
	m.Religions.Regions = m.regPlaceNTerritoriesCustom(m.Religions.Regions, seeds, func(o, u, v int) float64 {
		r := m.GetReligion(o)
		if r == nil {
			return -1
		}
		if r.Expansion == ReligionExpCulture && m.Cultures.Regions[v] != r.Culture.ID ||
			r.Expansion == ReligionExpState && m.CityStates.Regions[v] != m.CityStates.Regions[o] {
			return -1
		}
		return weightFunc(o, u, v) / r.Expansionism
	})
}

func (r *Religion) compare(other *Religion) float64 {
	if r == nil || other == nil {
		return -1.0
	}
	if r == other {
		return 1.0
	}
	if r.Parent == other || other.Parent == r || (r.Parent != nil && r.Parent == other.Parent) {
		return 0.5
	}
	return -0.5
}
