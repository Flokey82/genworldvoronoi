package civ

import (
	"fmt"
	"log"
	"strings"

	"github.com/Flokey82/genworldvoronoi/geo"
)

// TODO: A construction should unlock new actions for leadership.
// Constructions should also be stateful, e.g. a farm should have a state of
// "planted" or "harvested". This would allow us to for example, train an
// army, or copy books, maybe collect books in a library, etc.
type Construction struct {
	Name     string
	Cost     *ConstructionCost // Cost to build.
	Consumes *ConstructionCost // Consumes each turn.
	Produces []ResAmount       // Produces each turn.
}

// Tick runs the construction for one turn.
func (c *Construction) Tick(ds *ComboStorage) bool {
	// TODO: Do not iterate over maps.
	if c.Consumes != nil {
		if !ds.Fulfill(c.Consumes) {
			log.Printf("Not enough resources to consume for %s", c.Name)
			return false
		}
	}
	if c.Produces != nil {
		for _, pr := range c.Produces {
			ds.AddResource(pr.Res, pr.Amount)
		}
	}
	return true
}

type ResTypeAmount struct {
	Type   geo.ResourceType
	Amount int
}

type ResAmount struct {
	Res    *geo.Resource
	Amount int
}

// ConstructionCost represents the cost of constructing a building.
type ConstructionCost struct {
	ResourceTypes []ResTypeAmount // General categories of resources (e.g. wood, stone, metal).
	Resources     []ResAmount     // Specific resources.
	// TODO: Subtypes of resources, e.g. manufactured -> food
}

var (
	constTent = &Construction{
		Name: "Tent",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 1},
			},
			Resources: []ResAmount{
				{Res: ResLeather, Amount: 9},
			},
		},
		Produces: []ResAmount{
			{Res: ResHousing, Amount: 5},
		},
	}
	constHut = &Construction{
		Name: "Hut",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 10},
			},
		},
		Produces: []ResAmount{
			{Res: ResHousing, Amount: 10},
		},
	}
	constCarpenter = &Construction{
		Name: "Carpenter",
		Cost: &ConstructionCost{
			ResourceTypes: []ResTypeAmount{
				{Type: geo.ResourceTypeWood, Amount: 10},
			},
		},
		Produces: []ResAmount{
			{Res: ResTools, Amount: 1},
		},
	}
)

var constructions = []*Construction{
	constTent,
	constHut,
	constCarpenter,
}

type ComboStorage struct {
	*ResourceStorage
	consts []*Construction // TODO: Move this out of here.
}

func newComboStorage(maxStorage int) *ComboStorage {
	return &ComboStorage{
		ResourceStorage: newResourceStorage(maxStorage),
	}
}

func (rs *ComboStorage) Tick() {
	for _, c := range rs.consts {
		c.Tick(rs)
	}
}

func (rs *ComboStorage) CanConstruct(c *Construction) bool {
	return rs.CanFulfill(c.Cost)
}

func (rs *ComboStorage) Construct(c *Construction) bool {
	if !rs.CanConstruct(c) {
		return false
	}

	rs.Fulfill(c.Cost)
	rs.consts = append(rs.consts, c)
	// Create one production cycle.
	c.Tick(rs)
	return true
}

func (rs *ComboStorage) CanFulfill(c *ConstructionCost) bool {
	if c == nil {
		return true
	}

	for _, rt := range c.ResourceTypes {
		if !rs.HasNOfType(rt.Type, rt.Amount) {
			return false
		}
	}
	for _, rc := range c.Resources {
		if rs.Resources[rc.Res] < rc.Amount {
			return false
		}
	}
	return true
}

func (rs *ComboStorage) Fulfill(c *ConstructionCost) bool {
	if c == nil {
		return true
	}

	if !rs.CanFulfill(c) {
		return false
	}

	for _, rt := range c.ResourceTypes {
		rs.TakeNOfType(rt.Type, rt.Amount)
	}
	for _, rc := range c.Resources {
		rs.RemoveResource(rc.Res, rc.Amount)
	}
	return true
}

func (rs *ComboStorage) Log() {
	rs.ResourceStorage.Log()
	for _, c := range rs.consts {
		log.Printf("Constructed: %s", c.Name)
	}
}

func (rs *ComboStorage) String() string {
	resStr := rs.ResourceStorage.String()
	var consts []*Construction
	constCount := make(map[*Construction]int)
	for _, c := range rs.consts {
		if _, ok := constCount[c]; !ok {
			consts = append(consts, c)
		}
		constCount[c]++
	}

	var cStr string
	for _, c := range consts {
		cStr += fmt.Sprintf("%s: %d, ", c.Name, constCount[c])
	}
	return resStr + cStr
}

type ResourceStorage struct {
	maxStorage int
	Resources  map[*geo.Resource]int
}

func newResourceStorage(maxStorage int) *ResourceStorage {
	return &ResourceStorage{
		maxStorage: maxStorage,
		Resources:  make(map[*geo.Resource]int),
	}
}

func (rs *ResourceStorage) Add(r LocalResouces) {
	for _, res := range r.Resources {
		rs.Resources[res]++
	}
}

func (rs *ResourceStorage) AddResource(res *geo.Resource, count int) {
	rs.Resources[res] += count
}

func (rs *ResourceStorage) Remove(r LocalResouces) {
	for _, res := range r.Resources {
		rs.Resources[res]--
	}
}

func (rs *ResourceStorage) RemoveResource(res *geo.Resource, count int) {
	rs.Resources[res] -= count
}
func (rs *ResourceStorage) HasNOfType(rType geo.ResourceType, n int) bool {
	count := 0
	for res, c := range rs.Resources {
		if res.Type == rType {
			count += c
		}
	}
	return count >= n
}

func (rs *ResourceStorage) TakeNOfType(rType geo.ResourceType, n int) {
	for res, c := range rs.Resources {
		if res.Type == rType {
			if c > n {
				rs.Resources[res] -= n
				return
			}
			n -= c
			rs.Resources[res] = 0
		}
	}
}

func (rs *ResourceStorage) HasAnyOfType(rType geo.ResourceType) bool {
	for res := range rs.Resources {
		if res.Type == rType {
			return true
		}
	}
	return false
}

func (rs *ResourceStorage) String() string {
	var ss []string
	for res, count := range rs.Resources {
		ss = append(ss, fmt.Sprintf("%s: %d", res.Name, count))
	}
	return strings.Join(ss, ", ")
}

func (rs *ResourceStorage) Log() {
	for res, count := range rs.Resources {
		log.Printf("Resource: %s, %d", res.Name, count)
	}
}
