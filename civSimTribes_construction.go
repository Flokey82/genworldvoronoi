package genworldvoronoi

import (
	"fmt"
	"log"
	"strings"
)

func (s *simState) buildThings(t *Tribe) {
	ds := t.dumbStorage
	// Log the storage
	log.Printf("Tribe %s has %s", t.String(), ds.String())

	const foodPerPerson = 1
	foodNeeded := t.Population * foodPerPerson
	if ds.resources[StorageFood] < foodNeeded {
		log.Printf("Tribe %s needs more food. %d/%d", t.String(), ds.resources[StorageFood], foodNeeded)
		// TODO: Kill off some people
		ds.resources[StorageFood] = 0
	} else {
		ds.resources[StorageFood] -= foodNeeded
	}
	const firewoodPerPerson = 1
	firewoodNeeded := t.Population * firewoodPerPerson
	if ds.resources[StorageWood] < firewoodNeeded {
		log.Printf("Tribe %s needs more firewood. %d/%d", t.String(), ds.resources[StorageWood], firewoodNeeded)
		// TODO: Kill off some people
		ds.resources[StorageWood] = 0
	} else {
		ds.resources[StorageWood] -= firewoodNeeded
	}

	// TODO: For nomadic tribes we will construct tents, for which we need less wood
	// but also either hide or cloth.

	housingNeeded := t.Population
	housingAvailable := ds.resources[StorageHousing]
	if housingAvailable < housingNeeded {
		// Select the construction cost.
		var c *Construction
		if t.Type <= TribeTypeSettling {
			c = constTent
		} else {
			c = constHut
		}
		for housingAvailable < housingNeeded {
			if !ds.CanConstruct(c) {
				break
			}
			ds.Construct(c)
			housingAvailable = ds.resources[StorageHousing]
		}

		if housingAvailable < housingNeeded {
			log.Printf("Tribe %s needs more housing. %d/%d", t.String(), housingAvailable, housingNeeded)
		}

	}
	log.Printf("Tribe %s has %s", t.String(), ds.String())
}

type StorageType int

const (
	StorageWood StorageType = iota
	StorageFood
	StorageLeather
	StorageHousing
	StorageMax
)

func (s StorageType) String() string {
	switch s {
	case StorageWood:
		return "Wood"
	case StorageFood:
		return "Food"
	case StorageLeather:
		return "Leather"
	case StorageHousing:
		return "Housing"
	default:
		return "Unknown"
	}
}

type dumbStorage struct {
	resources map[StorageType]int
	consts    []*Construction
}

func newDumbStorage() *dumbStorage {
	return &dumbStorage{
		resources: make(map[StorageType]int),
	}
}

func (ds *dumbStorage) String() string {
	var str []string
	for i := StorageType(0); i < StorageMax; i++ {
		str = append(str, fmt.Sprintf("%s: %d", i.String(), ds.resources[i]))
	}
	return strings.Join(str, ", ")
}

func (ds *dumbStorage) CanConstruct(c *Construction) bool {
	for k, v := range c.Cost {
		if ds.resources[k] < v {
			return false
		}
	}
	return true
}

func (ds *dumbStorage) Construct(c *Construction) bool {
	if !ds.CanConstruct(c) {
		return false
	}
	ds.RemoveResources(c.Cost)
	ds.AddResources(c.Produces)

	// TODO: Return an instance of the construction
	ds.consts = append(ds.consts, c)
	return true
}

func (ds *dumbStorage) Add(r StorageType, n int) {
	ds.resources[r] += n
}

func (ds *dumbStorage) AddResources(r map[StorageType]int) {
	for i := StorageType(0); i < StorageMax; i++ {
		ds.resources[i] += r[i]
	}
}

func (ds *dumbStorage) RemoveResources(r map[StorageType]int) {
	for i := StorageType(0); i < StorageMax; i++ {
		ds.resources[i] -= r[i]
	}
}

type constructionCost map[StorageType]int

var (
	constCostTent = constructionCost{
		StorageWood:    1,
		StorageLeather: 9,
	}
	constCostHut = constructionCost{
		StorageWood: 10,
	}
	constTent = &Construction{
		Name: "Tent",
		Cost: map[StorageType]int{
			StorageWood:    1,
			StorageLeather: 9,
		},
		Produces: map[StorageType]int{
			StorageHousing: 5,
		},
	}
	constHut = &Construction{
		Name: "Hut",
		Cost: map[StorageType]int{
			StorageWood: 10,
		},
		Produces: map[StorageType]int{
			StorageHousing: 10,
		},
	}
)

type Construction struct {
	Name     string
	Cost     map[StorageType]int
	Produces map[StorageType]int // Produces each turn.
}
