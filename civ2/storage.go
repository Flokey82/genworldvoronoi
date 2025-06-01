package civ2

import "github.com/Flokey82/genworldvoronoi/geo"

// Storage should be extensible and limited.
// Nomadic tribes can develop and construct wagons and other means of transport.
// Settlements can develop and construct buildings like barns or granaries.
// Should each type of storage have its own struct and limit the type of resources it can store?
type Storage struct {
	Res      []*geo.Resource
	ResCount map[*geo.Resource]int
}

func NewStorage() *Storage {
	return &Storage{
		ResCount: make(map[*geo.Resource]int),
	}
}

func (s *Storage) AddResource(r *geo.Resource, n int) {
	if _, ok := s.ResCount[r]; !ok {
		s.Res = append(s.Res, r)
	}
	s.ResCount[r] += n
}

func (s *Storage) RemoveResource(r *geo.Resource, n int) bool {
	if s.ResCount[r] < n {
		return false
	}
	s.ResCount[r] -= n
	if s.ResCount[r] <= 0 {
		delete(s.ResCount, r)
		for i, res := range s.Res {
			if res == r {
				s.Res = append(s.Res[:i], s.Res[i+1:]...)
				break
			}
		}
	}
	return true
}
