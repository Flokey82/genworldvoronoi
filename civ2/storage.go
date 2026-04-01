package civ2

import "github.com/Flokey82/genworldvoronoi/geo"

// Storage should be extensible and limited.
// Nomadic tribes can develop and construct wagons and other means of transport.
// Settlements can develop and construct buildings like barns or granaries.
// Should each type of storage have its own struct and limit the type of resources it can store?
type Storage struct {
	MaxStorage int
	Resources  map[*geo.Resource]int
}

func NewStorage() *Storage {
	return &Storage{
		Resources: make(map[*geo.Resource]int),
	}
}

func (s *Storage) GetResource(res *geo.Resource) int {
	return s.Resources[res]
}

func (s *Storage) AddResources(r LocalResources) {
	for _, res := range r.Resources {
		s.Resources[res]++
	}
}

func (s *Storage) AddResource(res *geo.Resource, count int) {
	s.Resources[res] += count
}

func (s *Storage) RemoveResource(res *geo.Resource, count int) bool {
	if s.Resources[res] < count {
		return false
	}
	s.Resources[res] -= count
	return true
}

func (s *Storage) HasNOfType(rType geo.ResourceType, n int) bool {
	count := 0
	for res, c := range s.Resources {
		if res.Type == rType {
			count += c
		}
	}
	return count >= n
}

func (s *Storage) TakeNOfType(rType geo.ResourceType, n int) bool {
	if !s.HasNOfType(rType, n) {
		return false
	}
	for res, c := range s.Resources {
		if res.Type == rType {
			if c >= n {
				s.Resources[res] -= n
				return true
			}
			n -= c
			s.Resources[res] = 0
		}
	}
	return true
}

func (s *Storage) HasAnyOfType(rType geo.ResourceType) bool {
	for res := range s.Resources {
		if res.Type == rType {
			return true
		}
	}
	return false
}
