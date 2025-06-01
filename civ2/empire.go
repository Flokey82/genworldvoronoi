package civ2

import (
	"github.com/Flokey82/genworldvoronoi/civ"
)

// Empire contains information about a territory with the given ID.
// TODO: Maybe drop the regions since we can get that info
// relatively cheaply.
type Empire struct {
	ID      int      // Region where the empire originates (capital)
	Name    string   // Name of the empire
	Capital *City    // Capital city
	Culture *Culture // Primary culture of the empire
	Founded int64    // Year when the empire was founded
	*Storage
}

func (e Empire) GetID() int {
	return e.ID
}

// Ref returns the object reference of the empire.
func (e *Empire) Ref() civ.ObjectReference {
	return civ.ObjectReference{
		ID:   e.ID,
		Type: civ.ObjectTypeEmpire,
	}
}

func (m *Civ) GetEmpire(id int) *Empire {
	return m.Empires.Get(id)
}
