package genworldvoronoi

import (
	"fmt"
	"log"
	"strings"
)

func (s *simState) buildThings(t *Tribe) {
	// TODO: If we arent nomadic, the housing should be constructed in the region where we are.
	var ds *dumbStorage
	var c *Construction
	if t.Type <= TribeTypeSettling {
		c = constTent
		ds = t.dumbStorage
	} else {
		c = constHut
		ds = t.Settlement.dumbStorage
	}

	// TODO: For nomadic tribes we will construct tents, for which we need less wood
	// but also either hide or cloth.

	// TODO: Track the amount of resources we spend on construction.
	housingNeeded := t.Population
	housingAvailable := ds.resources[StorageHousing]
	if housingAvailable < housingNeeded {
		// Select the construction cost.
		for housingAvailable < housingNeeded {
			if !ds.CanConstruct(c) {
				break
			}
			ds.Construct(c)
			housingAvailable = ds.resources[StorageHousing]
		}
	}

	// Check how much temporary housing we have in form of tents.
	// NOTE: This is really stupid. Isn't there a better way to do this?
	if t.Type > TribeTypeSettling {
		housingAvailable += t.resources[StorageHousing]
	}

	if housingAvailable < housingNeeded {
		// We need to reduce satisfaction if we don't have enough housing.
		// TODO:
		// - Also reduce satisfaction of we only have temporary housing.
		// - If we don't have enough housing, some people might die.
		t.Satisfaction.Add(float64(housingAvailable-housingNeeded) / float64(t.Population))
		log.Printf("Tribe %s needs more housing. %d (+%d tents)/%d", t.String(), housingAvailable, t.resources[StorageHousing], housingNeeded)
	}

	// TODO: If we have not enough housing, we need to reduce satisfaction.
	log.Printf("Tribe %s has %s", t.String(), ds.String())
}

type StorageType int

const (
	StorageWood StorageType = iota
	StorageFood
	StorageLeather
	StorageHousing
	StorageTools
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
	counts := make(map[string]int)
	var uniqueConsts []string
	for _, c := range ds.consts {
		counts[c.Name]++
		if counts[c.Name] == 1 {
			uniqueConsts = append(uniqueConsts, c.Name)
		}
	}
	for _, c := range uniqueConsts {
		str = append(str, fmt.Sprintf("%s x%d (construction)", c, counts[c]))
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

var (
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
	constCarpenter = &Construction{
		Name: "Carpenter",
		Cost: map[StorageType]int{
			StorageWood: 10,
		},
		Produces: map[StorageType]int{
			StorageTools: 1,
		},
	}
)

var constructions = []*Construction{
	constTent,
	constHut,
	constCarpenter,
}

type Construction struct {
	Name     string
	Cost     map[StorageType]int
	Produces map[StorageType]int // Produces each turn.
}
