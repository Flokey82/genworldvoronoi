package civ2

import (
	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/go_gens/gengovernment"
	"github.com/Flokey82/go_gens/genlanguage"
)

// peopleThing represents an entity that has population, storage, and leadership.
type peopleThing interface {
	GetID() int
	GetType() byte
	Ref() civ.ObjectReference
	GetPopulation() int
	SetPopulation(int)
	GetCulture() *Culture
	SetCulture(*Culture)
	GetStorage() *Storage
	GetPeople() []*Person
	GetGoverningPeople() *GoverningPeople
	GetInfrastructure() *Infrastructure
	GetConstructionQueue() *ConstructionQueue
	GetMilitary() *Military
	GetRegions() []int
	NewRandomPerson(m *Civ, gender geneticshuman.Gender) *Person
	NewRandomChild(m *Civ, gender geneticshuman.Gender, parent *Person) *Person
	getPreferredLeadershipForm() gengovernment.LeadershipForm
	getPossibleLeadershipForms() []gengovernment.LeadershipForm
	findNaturalProgression() gengovernment.LeadershipForm
	findCoupProgression() gengovernment.LeadershipForm
	getLanguage() *genlanguage.Language
	getFactionActions(m *Civ, f *Faction) []*FactionAction
	String() string
}

// Constructible represents an entity that can build infrastructure.
type Constructible interface {
	GetID() int
	Ref() civ.ObjectReference
	GetPopulation() int
	SetPopulation(int)
	GetStorage() *Storage
	GetInfrastructure() *Infrastructure
	GetConstructionQueue() *ConstructionQueue
}

// tickEconomyBase handles the common economic steps: production and consumption.
func (m *Civ) tickEconomyBase(e peopleThing, nDays int) {
	id := e.GetID()
	storage := e.GetStorage()
	population := e.GetPopulation()

	// Update the resources.
	m.handleProduction(id, storage, population, e)
	m.handleConsumption(storage, population)

	// Update infrastructure production/consumption/upkeep.
	m.tickInfrastructure(e, nDays)

	// Advance construction projects.
	m.tickConstruction(e, nDays)

	// Handle disasters.
	m.tickDisasters(e, nDays)

	// Update military.
	m.tickMilitaryBase(e, nDays)

	// Handle trade if there are partners.
	// This is still specific to the entity types because finding partners
	// might differ. For now, we leave it to the specific tick functions
	// or provide a helper to find partners.
}

// tickMilitaryBase handles the common military steps.
func (m *Civ) tickMilitaryBase(e peopleThing, nDays int) {
	m.tickMilitary(e, nDays)
}

// tickDiplomacyBase handles common diplomatic evaluation.
func (m *Civ) tickDiplomacyBase(e peopleThing, nDays int) {
	switch e.GetType() {
	case byte(civ.ObjectTypeCity):
		if c, ok := e.(*City); ok {
			m.tickDiplomacyCity(c)
		}
	case byte(civ.ObjectTypeCityState):
		if cs, ok := e.(*CityState); ok {
			m.tickDiplomacyCityState(cs)
		}
	case byte(civ.ObjectTypeEmpire):
		if emp, ok := e.(*Empire); ok {
			m.tickDiplomacyEmpire(emp)
		}
	}
}
