package civ2

import (
	"log"
)

// tickConstruction advances the current construction project.
func (m *Civ) tickConstruction(e Constructible, nDays int) {
	q := e.GetConstructionQueue()
	if q == nil {
		return
	}

	// 1. If queue is empty, try to find something to build.
	if q.Current == nil {
		m.tickConstructionAI(e)
	}

	// 2. Advance current project.
	if q.Current == nil {
		return
	}

	p := q.Current
	storage := e.GetStorage()

	// Check if we can fulfill the cost for this tick.
	// For now, we'll assume a project takes a fixed amount of resources
	// and we try to fulfill it daily. 
	// TODO: Make construction speed dependent on population/labor.
	
	if storage.CanFulfill(p.Blueprint.Cost) {
		storage.Fulfill(p.Blueprint.Cost)
		p.Progress += 0.1 // 10% progress per tick for now.
		if p.Progress >= 1.0 {
			m.completeProject(e, p)
		}
	} else {
		log.Printf("Entity %d: Not enough resources to continue building %s", e.GetID(), p.Blueprint.Name)
	}
}

// tickInfrastructure handles upkeep, production, and effects of completed buildings.
func (m *Civ) tickInfrastructure(e Constructible, nDays int) {
	infra := e.GetInfrastructure()
	if infra == nil {
		return
	}

	storage := e.GetStorage()
	for _, b := range infra.Buildings {
		// Handle upkeep.
		if b.Upkeep != nil {
			if !storage.Fulfill(b.Upkeep) {
				log.Printf("Entity %d: Not enough resources for upkeep of %s", e.GetID(), b.Name)
				continue
			}
		}

		// Handle production.
		for _, pr := range b.Produces {
			storage.AddResource(pr.Res, pr.Amount)
		}

		// Handle active effects.
		if b.Effects != nil {
			b.Effects(e, m)
		}
	}
}

func (m *Civ) completeProject(e Constructible, p *Project) {
	infra := e.GetInfrastructure()
	infra.Buildings = append(infra.Buildings, p.Blueprint)
	
	q := e.GetConstructionQueue()
	log.Printf("Entity %d: Completed construction of %s", e.GetID(), p.Blueprint.Name)
	
	// Start next project if available.
	if len(q.Pending) > 0 {
		q.Current = q.Pending[0]
		q.Pending = q.Pending[1:]
	} else {
		q.Current = nil
	}

	m.History.AddEvent(HistoryEventConstruction, "Completed construction of "+p.Blueprint.Name, e.Ref())
}

func (m *Civ) tickConstructionAI(e Constructible) {
	infra := e.GetInfrastructure()
	if infra == nil {
		return
	}

	// Helper to check if entity already has a building.
	hasBuilding := func(name string) bool {
		for _, b := range infra.Buildings {
			if b.Name == name {
				return true
			}
		}
		return false
	}

	// Try to find a suitable blueprint.
	for _, b := range AllBlueprints {
		// Skip if we already have it.
		if hasBuilding(b.Name) {
			continue
		}

		// Skip if requirements are not met.
		if b.Requires != nil && !b.Requires(e, m) {
			continue
		}

		// Found a building! Add it to the queue.
		m.AddProject(e, b)
		return
	}
}

// AddProject adds a new construction project to the entity's queue.
func (m *Civ) AddProject(e Constructible, b *ConstructionBlueprint) {
	q := e.GetConstructionQueue()
	if q == nil {
		return
	}

	p := &Project{
		Blueprint: b,
		Progress:  0,
	}

	if q.Current == nil {
		q.Current = p
	} else {
		q.Pending = append(q.Pending, p)
	}
}

// Helper methods for Storage to handle ConstructionCost
func (s *Storage) CanFulfill(c *ConstructionCost) bool {
	if c == nil {
		return true
	}
	for _, rt := range c.ResourceTypes {
		if !s.HasNOfType(rt.Type, rt.Amount) {
			return false
		}
	}
	for _, res := range c.Resources {
		if s.Resources[res.Res] < res.Amount {
			return false
		}
	}
	return true
}

func (s *Storage) Fulfill(c *ConstructionCost) bool {
	if !s.CanFulfill(c) {
		return false
	}
	for _, rt := range c.ResourceTypes {
		s.TakeNOfType(rt.Type, rt.Amount)
	}
	for _, res := range c.Resources {
		s.Resources[res.Res] -= res.Amount
	}
	return true
}
