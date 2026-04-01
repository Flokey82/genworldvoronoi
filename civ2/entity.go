package civ2

import (
	"math/rand"

	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/genworldvoronoi/civ"
)

// BaseEntity represents common fields and methods for simulation entities.
type BaseEntity struct {
	ID         int
	Name       string
	Population int
	Culture    *Culture
	Type       byte // civ.ObjectType
	*Storage
	*GoverningPeople
	*Infrastructure
	*ConstructionQueue
	*Military
	Regions    []int
}
	
func (b BaseEntity) GetType() byte {
	return b.Type
}

func (b BaseEntity) GetID() int {
	return b.ID
}

func (b BaseEntity) Ref() civ.ObjectReference {
	return civ.ObjectReference{
		ID:   b.ID,
		Type: b.Type,
	}
}

func (b *BaseEntity) GetPopulation() int {
	return b.Population
}

func (b *BaseEntity) SetPopulation(pop int) {
	b.Population = pop
}

func (b *BaseEntity) GetStorage() *Storage {
	return b.Storage
}

func (b *BaseEntity) GetGoverningPeople() *GoverningPeople {
	return b.GoverningPeople
}

func (b *BaseEntity) GetInfrastructure() *Infrastructure {
	return b.Infrastructure
}

func (b *BaseEntity) GetConstructionQueue() *ConstructionQueue {
	return b.ConstructionQueue
}

func (b *BaseEntity) GetMilitary() *Military {
	return b.Military
}

func (b *BaseEntity) GetRegions() []int {
	return b.Regions
}

func (b *BaseEntity) AddRegion(r int) {
	for _, reg := range b.Regions {
		if reg == r {
			return
		}
	}
	b.Regions = append(b.Regions, r)
}

func (b *BaseEntity) RemoveRegion(r int) {
	for i, reg := range b.Regions {
		if reg == r {
			b.Regions = append(b.Regions[:i], b.Regions[i+1:]...)
			return
		}
	}
}

func (b *BaseEntity) GetPeople() []*Person {
	// TODO: Filter people by entity
	return nil
}

func (b *BaseEntity) NewRandomPerson(m *Civ, gender geneticshuman.Gender) *Person {
	return m.newRandomPersonAt(b.ID, b.Culture, gender, nil)
}

func (b *BaseEntity) NewRandomChild(m *Civ, gender geneticshuman.Gender, parent *Person) *Person {
	return m.newRandomPersonAt(b.ID, b.Culture, gender, parent)
}

func (b *BaseEntity) String() string {
	return b.Name
}

// Grow handles population growth for the entity.
func (b *BaseEntity) Grow(nDays int, growthRate float64) {
	// Apply infrastructure modifiers.
	growthRate += b.GetGrowthModifier()

	if growth := calcPopulationGrowth(b.Population, growthRate, nDays); growth >= 1 {
		b.Population += int(growth)
	} else if rand.Float64() < growth {
		b.Population++
	}

	// Cap population if it's a city.
	if b.Type == civ.ObjectTypeCity {
		if maxPop := b.GetMaxPopulation(); b.Population > maxPop {
			b.Population = maxPop
		}
	}
}

func (b *BaseEntity) GetGrowthModifier() float64 {
	var modifier float64
	if b.Infrastructure != nil {
		for _, building := range b.Infrastructure.Buildings {
			if building == BlueprintAquaeduct {
				modifier += 0.002 // +0.2% growth
			}
		}
	}
	return modifier
}

func (b *BaseEntity) GetMaxPopulation() int {
	baseMax := 5000 // Base limit for a settlement
	if b.Infrastructure != nil {
		for _, building := range b.Infrastructure.Buildings {
			if building == BlueprintGranary {
				baseMax += 5000
			} else if building == BlueprintHut {
				baseMax += 100
			}
		}
	}
	return baseMax
}
