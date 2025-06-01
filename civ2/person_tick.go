package civ2

import (
	civ "github.com/Flokey82/genworldvoronoi/civ/tmp"
	"github.com/Flokey82/go_gens/gameconstants"
)

func (m *Civ) tickPeople(nDays int) {
}

func (m *Civ) tickPerson(p *Person, nDays int) {
	// People might die.
	if gameconstants.DiesAtAgeWithinNDays(int(m.Geo.Calendar.GetYear())-p.Birth.Year, nDays) {
		// Die.
		p.Death = &civ.LifeEvent{
			Day:    m.Geo.Calendar.GetDayOfYear(),
			Year:   int(m.Geo.Calendar.GetYear()),
			Region: p.Birth.Region, // TOOD: Set to current region.
		}
	}
}
