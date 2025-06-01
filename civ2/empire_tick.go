package civ2

func (m *Civ) tickEmpires(nDays int) {
	for _, e := range m.Empires.Objects {
		// Empires might collapse if they lose on influence.
		if e.Capital == nil {
			continue
		}

		// We might expand or contract our influence.
		// Look at surrounding settlements and cities that would be willing to join us.
	}
}
