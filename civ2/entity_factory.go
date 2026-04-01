package civ2

import (
	"fmt"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/go_gens/genlanguage"
)

func (m *Civ) NewTribe(region, population int) *Tribe {
	lang := genlanguage.GenLanguage(int64(m.getNextFactionID())) // Using faction ID as seed
	culture := m.newCulture(region, lang, m.cultureFunc(region))
	
	t := &Tribe{
		BaseEntity: BaseEntity{
			ID:              nextTribeID(),
			Name:            lang.MakeName() + " Tribe",
			Population:      population,
			Culture:         culture,
			Type:            civ.ObjectTypeTribe,
			Storage:         NewStorage(),
			GoverningPeople: newGoverningPeople(),
			Infrastructure:    NewInfrastructure(),
			ConstructionQueue: NewConstructionQueue(),
			Military:          NewMilitary(),
		},
		RegionID:   region,
		Aggressive: rand.Float64() < 0.5,
	}
	m.Tribes.PlaceObjectAt(t, region)
	t.AddRegion(region)
	return t
}

func (m *Civ) NewSettlement(region int, population int, culture *Culture) *Settlement {
	s := &Settlement{
		BaseEntity: BaseEntity{
			ID:              region,
			Name:            culture.Language.MakeCityName(),
			Population:      population,
			Culture:         culture,
			Type:            civ.ObjectTypeSettlement,
			Storage:         NewStorage(),
			GoverningPeople: newGoverningPeople(),
			Infrastructure:    NewInfrastructure(),
			ConstructionQueue: NewConstructionQueue(),
			Military:          NewMilitary(),
		},
	}
	m.Settlements.PlaceObjectAt(s, region)
	s.AddRegion(region)
	m.History.AddEvent(HistoryEventFounding, fmt.Sprintf("The settlement of %s was founded.", s.Name), s.Ref())
	return s
}

func (m *Civ) NewCity(region int, population int, culture *Culture) *City {
	c := &City{
		BaseEntity: BaseEntity{
			ID:              region,
			Name:            culture.Language.MakeCityName(),
			Population:      population,
			Culture:         culture,
			Type:            civ.ObjectTypeCity,
			Storage:         NewStorage(),
			GoverningPeople: newGoverningPeople(),
			Infrastructure:    NewInfrastructure(),
			ConstructionQueue: NewConstructionQueue(),
			Military:          NewMilitary(),
		},
		Founded: m.Geo.Calendar.GetYear(),
	}
	m.Cities.PlaceObjectAt(c, region)
	c.AddRegion(region)
	m.History.AddEvent(HistoryEventFounding, fmt.Sprintf("The city of %s was founded.", c.Name), c.Ref())
	return c
}
