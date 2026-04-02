package civ2

import (
	"log"

	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/go_gens/genlanguage"
	"github.com/Flokey82/go_gens/genreligion"
)

func (m *Civ) placeNCultures(n int) {
	for i := 0; i < n; i++ {
		// Pick a random region for the culture.
		r := m.Rand.Intn(m.Geo.NumRegions)
		if m.Cultures.GetIDAt(r) != -1 {
			continue // Already has a culture.
		}
		lang := genlanguage.GenLanguage(int64(m.getNextFactionID()))
		m.newCulture(r, lang, m.cultureFunc(r))
	}
}

func (m *Civ) placeSpecializedCities(pop int) {
	m.placeNCitiesOfType(m.NumFarmingTowns, civ.CityTypeFarming, pop)
	m.placeNCitiesOfType(m.NumDesertOasis, civ.CityTypeDesertOasis, pop)
	m.placeNCitiesOfType(m.NumMiningTowns, civ.CityTypeMining, pop)
	m.placeNCitiesOfType(m.NumMiningGemsTowns, civ.CityTypeMiningGems, pop)
	m.placeNCitiesOfType(m.NumQuarryTowns, civ.CityTypeQuarry, pop)
	m.placeNCitiesOfType(m.NumTradingTowns, civ.CityTypeTrading, pop)
}

func (m *Civ) placeNCitiesOfType(n int, cType civ.CityType, pop int) {
	if n <= 0 {
		return
	}
	// Pick best regions based on civ.CityType.GetFitnessFunction.
	// NOTE: We're using the legacy fitness function from the civ package.
	// Since civ2.Civ doesn't satisfy civ.Civ, we have to bridge it or adapt it.
	// For now, let's use the cradles logic.
	regions := m.pickNCradlesOfCivilization(n, false)
	for _, r := range regions {
		culture := m.GetCulture(r)
		if culture == nil {
			lang := genlanguage.GenLanguage(int64(m.getNextFactionID()))
			culture = m.newCulture(r, lang, m.cultureFunc(r))
		}
		m.NewSpecializedCity(r, pop, culture, cType)
	}
	log.Printf("Placed %d %s cities", n, cType)
}

func (m *Civ) placeNOrganizedReligions(n int) {
	cities := m.Cities.Objects
	for i := 0; i < n && i < len(cities); i++ {
		m.genOrganizedReligion(cities[i])
	}
}

func (m *Civ) genOrganizedReligion(c *City) *Religion {
	return m.placeReligionAt(c.ID, -1, genreligion.GroupOrganized, c.Culture, nil)
}
