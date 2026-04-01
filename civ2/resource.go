package civ2

import (
	"github.com/Flokey82/genworldvoronoi/geo"
)

// Various additional resource types.
var (
	ResourceTypeManufactured geo.ResourceType = geo.ResourceTypeMax
)

// Various manufactured resources.
var (
	ResLeather = &geo.Resource{
		Type: ResourceTypeManufactured,
		Name: "Leather",
	}
	ResFood = &geo.Resource{
		Type: ResourceTypeManufactured,
		Name: "Food",
	}
	ResTools = &geo.Resource{
		Type: ResourceTypeManufactured,
		Name: "Tools",
	}
	ResHousing = &geo.Resource{
		Type: ResourceTypeManufactured,
		Name: "Housing",
	}
)

type LocalResources struct {
	Resources []*geo.Resource
}

func (m *Civ) getResources(r int, incNeighbors bool) LocalResources {
	var rsc LocalResources
	for _, res := range m.Geo.Resources {
		if m.Geo.Location[res][r] {
			rsc.Resources = append(rsc.Resources, res)
		}
	}

	if incNeighbors {
		seenResources := make(map[*geo.Resource]bool)
		for _, res := range rsc.Resources {
			seenResources[res] = true
		}
		for _, nb := range m.Geo.R_circulate_r(nil, r) {
			for _, res := range m.Geo.Resources {
				if m.Geo.Location[res][nb] && !seenResources[res] {
					rsc.Resources = append(rsc.Resources, res)
					seenResources[res] = true
				}
			}
		}
	}
	return rsc
}
