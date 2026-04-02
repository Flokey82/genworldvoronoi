package civ2

import (
	"container/heap"
	"log"
	"math"

	"github.com/Flokey82/genworldvoronoi/geo"
)

func (m *Civ) tickEmpires(nDays int) {
	for _, e := range m.Empires.Objects {
		// Empires might collapse if they lose on influence.
		if e.Capital == nil || e.Capital.Population == 0 {
			continue
		}

		log.Printf("Empire %d: %s", e.ID, e.Name)

		// Centralized economic tick.
		m.tickEconomyBase(e, nDays)

		// Tick diplomatic relations.
		m.tickDiplomacyBase(e, nDays)

		// Tick leadership.
		m.tickLeadership(e, nDays)
	}

	// Perform territorial expansion organically.
	m.recalcEmpiresTerritoriesBounded()
}

func (m *Civ) getCityScoreForMartial(c *City) float64 {
	// Simple martial score based on population and culture martialism.
	// We need to count the regions of the culture.
	numRegions := 0
	for _, rID := range m.Cultures.Regions {
		if rID == c.Culture.ID {
			numRegions++
		}
	}
	return float64(c.Population) * c.Culture.Type.Martialism() * float64(numRegions)
}

func (m *Civ) legacyExpandEmpires() {
	// Empires do not expand region by region, but rather occupy entire city states.
	var queue geo.AscPriorityQueue
	heap.Init(&queue)

	cityIDToCity := make(map[int]*City)
	for _, c := range m.Cities.Objects {
		cityIDToCity[c.ID] = c
	}

	// cityIDToEmpireID maps city IDs to the ID of the empire that controls them.
	cityIDToEmpireID := make(map[int]int)
	for _, c := range m.Cities.Objects {
		cityIDToEmpireID[c.ID] = -1
	}

	// Start with the city states that are the core of the empires.
	for _, empire := range m.Empires.Objects {
		cityIDToEmpireID[empire.ID] = empire.ID

		// Get the martial score of the empire / city state we are expanding from.
		empireScore := m.getCityScoreForMartial(empire.Capital)
		for _, r := range m.getTerritoryNeighbors(empire.ID, empire.Regions, m.CityStates.Regions) {
			// We only expand to city states that have a lower martial score than the empire.
			if destCity := cityIDToCity[r]; destCity != nil {
				if destScore := m.getCityScoreForMartial(destCity); destScore <= empireScore {
					heap.Push(&queue, &geo.QueueEntry{
						Score:       destScore,
						Origin:      empire.ID,
						Destination: r,
					})
				}
			}
		}
	}

	// Extend territories until the queue is empty.
	for queue.Len() > 0 {
		u := heap.Pop(&queue).(*geo.QueueEntry)
		if cityIDToEmpireID[u.Destination] >= 0 {
			continue
		}
		cityIDToEmpireID[u.Destination] = u.Origin

		// Get the martial score of the city state we are expanding from.
		originScore := m.getCityScoreForMartial(cityIDToCity[u.Origin])

		// Check if there are any neighbors that we can expand to.
		// We need to get the regions of the city state we just absorbed.
		cs := m.CityStates.Get(u.Destination)
		if cs == nil {
			continue
		}

		for _, v := range m.getTerritoryNeighbors(u.Destination, cs.Regions, m.CityStates.Regions) {
			if cityIDToEmpireID[v] >= 0 {
				continue
			}

			// Get the martial score of the city state we want to expand to.
			destCity := cityIDToCity[v]
			if destCity == nil {
				continue
			}
			destScore := m.getCityScoreForMartial(destCity)

			// If the destination score is higher than the origin score, we can't expand
			// to this city state since they would resist our expansion successfully.
			if destScore >= 0 && destScore < originScore {
				heap.Push(&queue, &geo.QueueEntry{
					Score:       destScore + u.Score, // The further away, the higher the score, the lower the rank in the queue.
					Origin:      u.Origin,
					Destination: v,
				})
			}
		}
	}

	// Now update the empire regions based on the city states they occupy.
	m.Empires.ResetRegions()
	for _, e := range m.Empires.Objects {
		e.Regions = e.Regions[:0]
	}

	for r, csID := range m.CityStates.Regions {
		if csID >= 0 {
			if empID, ok := cityIDToEmpireID[csID]; ok && empID >= 0 {
				m.Empires.SetIDAt(r, empID)
				if emp := m.Empires.Get(empID); emp != nil {
					emp.AddRegion(r)
				}
			}
		}
	}
}

func (m *Civ) recalcEmpiresTerritoriesBounded() {
	m.Empires.ResetRegions()
	for _, e := range m.Empires.Objects {
		e.Regions = e.Regions[:0]
	}

	var queue geo.AscPriorityQueue
	heap.Init(&queue)

	// maxInfluence maps e.ID to their max range
	maxInfluence := make(map[int]float64)

	for _, e := range m.Empires.Objects {
		if e.Capital == nil || e.Capital.Population == 0 {
			continue
		}
		// Calculate max range based on martial score
		influence := m.getCityScoreForMartial(e.Capital) / 100.0 // Tune this factor
		if influence < 1.0 {
			influence = 1.0
		}
		maxInfluence[e.ID] = influence

		heap.Push(&queue, &geo.QueueEntry{
			Score:       0,
			Origin:      e.ID,
			Destination: e.Capital.ID,
		})
	}

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
		eID := current.Origin

		if cost > maxInfluence[eID] {
			continue
		}

		if cost >= costCache[region] {
			continue
		}
		costCache[region] = cost

		// Empires claim it
		if m.Empires.Regions[region] == -1 {
			m.Empires.SetIDAt(region, eID)
			emp := m.Empires.Get(eID)
			if emp != nil {
				emp.AddRegion(region)
				m.Empires.PlaceObjectAt(emp, region)
			}
		}

		for _, nb := range m.SphereMesh.R_circulate_r(outReg, region) {
			if elevs[nb] <= 0 {
				continue // For organic growth, don't auto-cross water
			}

			stepCost := 1.0
			elevDiff := math.Abs(elevs[nb] - elevs[region])
			elevCost := (elevDiff / maxElev) * 50.0
			stepCost += elevCost
			bCost := 0.0
			if biomeWeight != nil {
				bCost = biomeWeight(eID, region, nb)
			}
			stepCost += bCost

			nextCost := cost + stepCost

			if nextCost <= maxInfluence[eID] && nextCost < costCache[nb] {
				heap.Push(&queue, &geo.QueueEntry{
					Score:       nextCost,
					Origin:      eID,
					Destination: nb,
				})
			}
		}
	}
}
