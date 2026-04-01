package civ2

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/geo"
)

func (m *Civ) tickDisasters(e peopleThing, nDays int) {
	// Only trigger disasters occasionally.
	// Roughly once every 50 years on average if ticking yearly.
	if rand.Intn(50) != 0 {
		return
	}

	// 1. Collect potential disasters.
	// Geo disasters from the region.
	disasters := m.disasterFunc(e.GetID()).GetDisasters()

	// Internal disasters.
	pop := e.GetPopulation()
	if pop > 1000 {
		disasters = append(disasters, geo.DisDisease)
	}
	if pop > 10000 {
		disasters = append(disasters, geo.DisPlague)
	}

	// Check storage for famine.
	storage := e.GetStorage()
	if storage != nil {
		// Calculate total food (Plants + Animals).
		foodCount := 0
		for res, count := range storage.Resources {
			if res.Type == geo.ResourceTypePlant || res.Type == geo.ResourceTypeAnimal {
				foodCount += count
			}
		}
		if foodCount < pop/10 {
			disasters = append(disasters, geo.DisFamine)
		}
	}

	// Industry-specific disasters.
	if m.hasResourceAny(e.GetID(), geo.ResourceTypeMetal) || m.hasResourceAny(e.GetID(), geo.ResourceTypeGem) || m.hasResourceAny(e.GetID(), geo.ResourceTypeStone) {
		disasters = append(disasters, geo.DisCaveIn)
	}

	// Biome-specific disasters.
	if m.GetRegTemperature(e.GetID()) > 30 && m.Moisture.Values[e.GetID()] < 0.1 {
		disasters = append(disasters, geo.DisSandstorm)
	}

	if len(disasters) == 0 {
		return
	}

	// 2. Pick a disaster.
	dis := geo.RandDisaster(disasters)
	if dis == geo.DisNone {
		return
	}

	// 3. Apply mitigation based on infrastructure.
	popLoss := dis.PopulationLoss * (0.8 + rand.Float64()*0.4) // Random variance 0.8-1.2
	infra := e.GetInfrastructure()
	if infra != nil {
		for _, b := range infra.Buildings {
			switch b.Name {
			case "Aquaeduct":
				if dis.Name == geo.DisDrought.Name {
					popLoss *= 0.5 // Aquaeducts mitigate droughts.
				}
			case "Granary":
				if dis.Name == geo.DisFamine.Name {
					popLoss *= 0.7 // Granaries mitigate famine.
				}
			case "Walls":
				if dis.Name == geo.DisStorm.Name || dis.Name == geo.DisWildfire.Name {
					popLoss *= 0.8 // Walls provide some protection.
				}
			}
		}
	}

	// 4. Resolve disaster effects.
	dead := int(math.Ceil(float64(pop) * popLoss))
	if dead > pop {
		dead = pop
	}

	// Apply population loss.
	m.Population[e.GetID()] -= dead
	if m.Population[e.GetID()] < 0 {
		m.Population[e.GetID()] = 0
	}
	e.SetPopulation(m.Population[e.GetID()])

	// Logging and History.
	m.History.AddEvent(HistoryEventDisaster, fmt.Sprintf("A %s has struck %s, causing %d deaths.", dis.Name, e.String(), dead), e.Ref())
}
