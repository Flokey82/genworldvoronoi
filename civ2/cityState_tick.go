package civ2

import (
	"container/heap"
	"log"
	"math"
	"sort"

	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/genworldvoronoi/geo"
)

func (m *Civ) tickCityStates(nDays int) {
	for _, cs := range m.CityStates.Objects {
		// City states might collapse if they lose on influence.
		if cs.Capital == nil || cs.Capital.Population == 0 {
			continue
		}

		log.Printf("City state %d", cs.ID)
		
		// Centralized economic tick.
		m.tickEconomyBase(cs, nDays)

		// Tick diplomatic relations.
		m.tickDiplomacyBase(cs, nDays)

		// Tick leadership.
		m.tickLeadership(cs, nDays)

		// We might expand or contract our influence.
		// (Expansion happens globally after this loop for all city states at once)

		// We might found an empire.
		if m.Empires.GetAt(cs.ID) == nil && cs.Capital.Population > 10000 {
			m.foundEmpire(cs)
		}
	}
	
	// Recalculate territories globally to simulate border pressure and natural boundaries.
	m.recalcCityStatesTerritoriesBounded()
}

func (m *Civ) recalcCityStatesTerritoriesBounded() {
	m.CityStates.ResetRegions()
	for _, cs := range m.CityStates.Objects {
		cs.Regions = cs.Regions[:0]
	}

	var queue geo.AscPriorityQueue
	heap.Init(&queue)

	// maxInfluence maps cs.ID to their max range
	maxInfluence := make(map[int]float64)

	for _, cs := range m.CityStates.Objects {
		if cs.Capital == nil || cs.Capital.Population == 0 {
			continue
		}
		// Calculate max range based on population
		influence := float64(cs.Capital.Population) / 100.0 // tuning parameter
		if influence < 1.0 {
			influence = 1.0
		}
		maxInfluence[cs.ID] = influence

		// Place capital
		heap.Push(&queue, &geo.QueueEntry{
			Score:       0,
			Origin:      cs.ID,
			Destination: cs.Capital.ID,
		})
	}

	// Cost tracking
	costCache := make([]float64, m.SphereMesh.NumRegions)
	for i := range costCache {
		costCache[i] = math.MaxFloat64
	}

	elevs := m.Elevation.GetValues()
	maxElev := m.Elevation.Max
	biomeWeight := m.getTerritoryBiomeWeightFunc()

	var outReg []int
	for queue.Len() > 0 {
		current := heap.Pop(&queue).(*geo.QueueEntry)
		
		region := current.Destination
		cost := current.Score
		csID := current.Origin

		if cost > maxInfluence[csID] {
			continue
		}

		if cost >= costCache[region] {
			continue
		}
		costCache[region] = cost

		// Actually claim it
		if m.CityStates.Regions[region] == -1 {
			m.CityStates.SetIDAt(region, csID)
			cs := m.CityStates.Get(csID)
			if cs != nil {
				cs.AddRegion(region)
				m.CityStates.PlaceObjectAt(cs, region)
			}
		}

		for _, nb := range m.SphereMesh.R_circulate_r(outReg, region) {
			if elevs[nb] <= 0 {
				continue // Organic expansion currently avoids water
			}

			// Base cost
			stepCost := 1.0

			// Elevation difference penalty
			elevDiff := math.Abs(elevs[nb] - elevs[region])
			elevCost := (elevDiff / maxElev) * 50.0 // steep mountains are hard to cross
			stepCost += elevCost

			// Biome transition cost
			bCost := biomeWeight(csID, region, nb)
			stepCost += bCost

			nextCost := cost + stepCost

			if nextCost <= maxInfluence[csID] && nextCost < costCache[nb] {
				heap.Push(&queue, &geo.QueueEntry{
					Score:       nextCost,
					Origin:      csID,
					Destination: nb,
				})
			}
		}
	}
}

func (m *Civ) legacyExpandCityState(cs *CityState, rNbs []int) {
	// Check the neighbours of the controlled regions.
	var numNewRegions int
	var limitExpansionToSettled bool
	var possibleDestinations []int
	seenRegions := make(map[int]bool)
	for _, r := range cs.Regions {
		seenRegions[r] = true
	}

	for _, r := range cs.Regions {
		for _, nb := range m.R_circulate_r(rNbs, r) {
			if m.Elevation.Values[nb] < 0 {
				// We cannot expand to water.
				continue
			}
			if _, ok := seenRegions[nb]; ok || m.CityStates.GetAt(nb) != nil {
				continue
			}
			seenRegions[nb] = true
			if limitExpansionToSettled && m.Settlements.GetIDAt(nb) == -1 && m.Cities.GetIDAt(nb) == -1 {
				continue
			}
			if c := m.Cities.Get(nb); c != nil && c.Population >= cs.Capital.Population {
				continue
			}
			// We found a settlement or a city, so we can expand.
			// Check if the settlement is willing to join us.
			possibleDestinations = append(possibleDestinations, nb)
			numNewRegions++
		}
	}
	if numNewRegions > 0 {
		log.Printf("City state %d has expanded to %d new regions", cs.ID, numNewRegions)
	}
	const limitExpansion = 5
	// Sort candidates by distance.
	distances := make(map[int]float64)
	for _, r := range possibleDestinations {
		distances[r] = m.GetDistance(cs.Capital.ID, r)
	}
	// Sort the possible destinations by distance.
	sort.Slice(possibleDestinations, func(i, j int) bool {
		return distances[possibleDestinations[i]] < distances[possibleDestinations[j]]
	})
	for i, r := range possibleDestinations {
		if i >= limitExpansion {
			break
		}
		m.CityStates.PlaceObjectAt(cs, r)
		cs.AddRegion(r)
	}
}

func (m *Civ) foundEmpire(cs *CityState) {
	e := &Empire{
		BaseEntity: BaseEntity{
			ID:                cs.ID,
			Name:              "Empire of " + cs.Capital.Name,
			Culture:           cs.Culture,
			Type:              civ.ObjectTypeEmpire,
			Storage:           NewStorage(),
			GoverningPeople:   newGoverningPeople(),
			Infrastructure:    NewInfrastructure(),
			ConstructionQueue: NewConstructionQueue(),
			Military:          cs.Military,
		},
		Capital: cs.Capital,
		Founded: m.Geo.Calendar.GetYear(),
	}

	// Place the empire in the world and sync regions.
	m.Empires.PlaceObjectAt(e, e.ID)
	for _, r := range cs.Regions {
		m.Empires.PlaceObjectAt(e, r)
		e.AddRegion(r)
	}
}
