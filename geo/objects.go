package geo

import (
	"sort"

	"github.com/Flokey82/go_gens/utils"
)

type PropertySliceNoUpdate struct {
	Values  []float64
	Impacts []SetNeedsUpdate
	Min     float64
	Max     float64
}

func NewPropertySliceNoUpdate(numRegions int) *PropertySliceNoUpdate {
	return &PropertySliceNoUpdate{
		Values: make([]float64, numRegions),
	}
}

func (ps *PropertySliceNoUpdate) SetValues(values []float64) {
	ps.Values = values
	ps.Min, ps.Max = utils.MinMax(values)
	for _, impact := range ps.Impacts {
		impact.SetNeedsUpdate()
	}
}

func (ps *PropertySliceNoUpdate) GetValues() []float64 {
	return ps.Values
}

type SetNeedsUpdate interface {
	SetNeedsUpdate()
}

type PSliceMinMax[T float64 | int] struct {
	Values      []T
	Impacts     []SetNeedsUpdate
	NeedsUpdate bool
	UpdateFunc  func() []T
	Min         T
	Max         T
}

func NewPSliceMinMax[T float64 | int](numRegions int, updateFunc func() []T) *PSliceMinMax[T] {
	return &PSliceMinMax[T]{
		Values:      make([]T, numRegions),
		NeedsUpdate: true,
		UpdateFunc:  updateFunc,
	}
}

func minMax2[T float64 | int](values []T) (T, T) {
	min := values[0]
	max := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

func (ps *PSliceMinMax[T]) SetValues(values []T) {
	ps.Values = values
	ps.Min, ps.Max = minMax2(values)
	ps.NeedsUpdate = false
	for _, impact := range ps.Impacts {
		impact.SetNeedsUpdate()
	}
}

func (ps *PSliceMinMax[T]) GetValues() []T {
	if ps.NeedsUpdate {
		if ps.UpdateFunc == nil {
			panic("PropertySlice has no update function")
		}
		ps.SetValues(ps.UpdateFunc())
	}
	return ps.Values
}

func (ps *PSliceMinMax[T]) SetNeedsUpdate() {
	ps.NeedsUpdate = true
}

type PSlice[T any] struct {
	Values      []T
	Impacts     []SetNeedsUpdate
	NeedsUpdate bool
	UpdateFunc  func([]T) []T
}

func NewPSlice[T any](numRegions int, updateFunc func(oldVals []T) []T) *PSlice[T] {
	return &PSlice[T]{
		Values:      make([]T, numRegions),
		NeedsUpdate: true,
		UpdateFunc:  updateFunc,
	}
}

func (ps *PSlice[T]) SetValues(values []T) {
	ps.Values = values
	ps.NeedsUpdate = false
	for _, impact := range ps.Impacts {
		impact.SetNeedsUpdate()
	}
}

func (ps *PSlice[T]) GetValues() []T {
	if ps.NeedsUpdate {
		if ps.UpdateFunc == nil {
			panic("PropertySlice has no update function")
		}
		ps.SetValues(ps.UpdateFunc(ps.Values))
	}
	return ps.Values
}

func (ps *PSlice[T]) SetNeedsUpdate() {
	ps.NeedsUpdate = true
}

// Generic type constraint for T
type IDable interface {
	GetID() int
}

// Generic References struct
type References[T IDable] struct {
	Objects   []*T
	ObjectMap map[int]*T
	Regions   []int
}

func NewReferences[T IDable](numRegions int) *References[T] {
	return &References[T]{
		ObjectMap: make(map[int]*T),
		Regions:   initRegionSlice(numRegions),
	}
}

// ResetRegions resets the region IDs to -1.
func (ref *References[T]) ResetRegions() {
	for i := range ref.Regions {
		ref.Regions[i] = -1
	}
}

// Clean removes all objects that are not associated with a region.
func (ref *References[T]) Clean() {
	seenObjects := make(map[int]bool)
	for _, id := range ref.Regions {
		if id >= 0 {
			seenObjects[id] = true
		}
	}

	// Remove objects that are not associated with a region.
	newObjects := make([]*T, 0, len(ref.Objects))
	for _, obj := range ref.Objects {
		id := (*obj).GetID()
		if seenObjects[id] {
			newObjects = append(newObjects, obj)
		} else {
			delete(ref.ObjectMap, id)
		}
	}
	ref.Objects = newObjects
}

// Get returns the object with the specified ID.
func (ref *References[T]) Get(id int) *T {
	return ref.ObjectMap[id]
}

// GetAt retrieves the object at the specified region ID.
func (ref *References[T]) GetAt(r int) *T {
	return ref.ObjectMap[ref.Regions[r]]
}

// GetIDAt retrieves the object ID at the specified region ID.
func (ref *References[T]) GetIDAt(r int) int {
	return ref.Regions[r]
}

// SetIDAt sets the object ID at the specified region ID.
func (ref *References[T]) SetIDAt(r, id int) {
	ref.Regions[r] = id
	// TODO: What to do if there are no regions associated with an object anymore?
	// Shouldn't we remove the object from the map?
}

// PlaceObjectAt places an object at the specified region ID.
func (ref *References[T]) PlaceObjectAt(obj *T, r int) {
	id := (*obj).GetID()
	if _, ok := ref.ObjectMap[id]; !ok {
		ref.Objects = append(ref.Objects, obj)
		ref.ObjectMap[id] = obj
	}
	ref.Regions[r] = id
}

// GetNeighbours returns the neighbours of the specified object.
func (ref *References[T]) GetNeighbours(r *Geo, obj *T) []*T {
	seenRegions := make(map[int]bool)    // IDs of the regions.
	seenNeighbours := make(map[int]bool) // IDs of the neighbours.
	var nbs []*T
	rNbs := make([]int, 0, 6) // Neighbour regions.
	// Iterate over the regions and find the neighbours.
	objID := (*obj).GetID()
	for i, id := range ref.Regions {
		if id == objID {
			for _, nb := range r.R_circulate_r(rNbs, i) {
				if seenRegions[nb] {
					continue
				}
				seenRegions[nb] = true
				if nbID := ref.Regions[nb]; nbID >= 0 && !seenNeighbours[nbID] {
					seenNeighbours[nbID] = true
					nbs = append(nbs, ref.ObjectMap[nbID])
				}
			}
		}
	}
	return nbs
}

// Sort sorts the objects based on the given comparison function.
func (ref *References[T]) Sort(less func(a, b *T) bool) {
	sort.Slice(ref.Objects, func(i, j int) bool {
		return less(ref.Objects[i], ref.Objects[j])
	})
}

// TODO: Create a new version where an object can only be placed at one region.

// SoloReferences is a struct that only allows an object to be placed at one region.
type SoloReferences[T IDable] struct {
	Objects        []*T
	ObjectMap      map[int]*T
	ObjectLocation map[int]int
	Regions        []int
}

// NewSoloReferences creates a new SoloReferences struct.
func NewSoloReferences[T IDable](numRegions int) *SoloReferences[T] {
	return &SoloReferences[T]{
		ObjectMap:      make(map[int]*T),
		ObjectLocation: make(map[int]int),
		Regions:        initRegionSlice(numRegions),
	}
}

// ResetRegions resets the region IDs to -1.
func (ref *SoloReferences[T]) ResetRegions() {
	for i := range ref.Regions {
		ref.Regions[i] = -1
	}
}

// Get returns the object with the specified ID.
func (ref *SoloReferences[T]) Get(id int) *T {
	return ref.ObjectMap[id]
}

// GetAt retrieves the object at the specified region ID.
func (ref *SoloReferences[T]) GetAt(r int) *T {
	return ref.ObjectMap[ref.Regions[r]]
}

// GetIDAt retrieves the object ID at the specified region ID.
func (ref *SoloReferences[T]) GetIDAt(r int) int {
	return ref.Regions[r]
}

// PlaceObjectAt places an object at the specified region ID.
func (ref *SoloReferences[T]) PlaceObjectAt(obj *T, r int) {
	id := (*obj).GetID()
	if _, ok := ref.ObjectMap[id]; !ok {
		ref.Objects = append(ref.Objects, obj)
		ref.ObjectMap[id] = obj
	}

	// Remove the object from the previous region.
	if prevRegion, ok := ref.ObjectLocation[id]; ok && prevRegion >= 0 {
		ref.Regions[prevRegion] = -1
	}
	ref.Regions[r] = id
	ref.ObjectLocation[id] = r
}

// RemoveObjectAt removes the object at the specified region ID.
func (ref *SoloReferences[T]) RemoveObjectAt(r int) {
	id := ref.Regions[r]
	ref.Regions[r] = -1
	delete(ref.ObjectMap, id)
	delete(ref.ObjectLocation, id)

	// Remove the object from the slice.
	for i, o := range ref.Objects {
		if (*o).GetID() == id {
			ref.Objects = append(ref.Objects[:i], ref.Objects[i+1:]...)
			break
		}
	}
}

// RemoveObject removes the object from the references.
func (ref *SoloReferences[T]) RemoveObject(obj *T) {
	id := (*obj).GetID()
	if r, ok := ref.ObjectLocation[id]; ok {
		ref.Regions[r] = -1
		delete(ref.ObjectMap, id)
		delete(ref.ObjectLocation, id)
	}
	for i, o := range ref.Objects {
		if (*o).GetID() == id {
			ref.Objects = append(ref.Objects[:i], ref.Objects[i+1:]...)
			break
		}
	}
}

// SwitchRegions switches the regions of two objects.
func (ref *SoloReferences[T]) SwitchRegions(r1, r2 int) {
	id1 := ref.Regions[r1]
	id2 := ref.Regions[r2]
	ref.Regions[r1] = id2
	ref.Regions[r2] = id1
	ref.ObjectLocation[id1] = r2
	ref.ObjectLocation[id2] = r1
}

// Sort sorts the objects based on the given comparison function.
func (ref *SoloReferences[T]) Sort(less func(a, b *T) bool) {
	sort.Slice(ref.Objects, func(i, j int) bool {
		return less(ref.Objects[i], ref.Objects[j])
	})
}

// MultiReferences is a struct that allows multiple objects to be placed at one or more regions.
type MultiReferences[T IDable] struct {
	Objects        []*T
	ObjectMap      map[int]*T
	ObjectToRegion map[int][]bool
	NumRegions     int
}

// NewMultiReferences creates a new MultiReferences struct.
func NewMultiReferences[T IDable](numRegions int) *MultiReferences[T] {
	return &MultiReferences[T]{
		ObjectMap:      make(map[int]*T),
		ObjectToRegion: make(map[int][]bool),
		NumRegions:     numRegions,
	}
}

// ResetRegions resets the region IDs to -1.
func (ref *MultiReferences[T]) ResetRegions() {
	for _, obj := range ref.Objects {
		id := (*obj).GetID()
		for i := range ref.ObjectToRegion[id] {
			ref.ObjectToRegion[id][i] = false
		}
	}
}

// Get returns the object with the specified ID.
func (ref *MultiReferences[T]) Get(id int) *T {
	return ref.ObjectMap[id]
}

// GetAt retrieves the object at the specified region ID.
func (ref *MultiReferences[T]) GetAt(r int) []*T {
	var objs []*T
	for _, obj := range ref.Objects {
		id := (*obj).GetID()
		if ref.ObjectToRegion[id][r] {
			objs = append(objs, obj)
		}
	}
	return objs
}

// GetIDAt retrieves the object ID at the specified region ID.
func (ref *MultiReferences[T]) GetIDAt(r int) []int {
	var ids []int
	for _, obj := range ref.Objects {
		id := (*obj).GetID()
		if ref.ObjectToRegion[id][r] {
			ids = append(ids, id)
		}
	}
	return ids
}

// SetIDAt sets the object ID at the specified region ID.
func (ref *MultiReferences[T]) SetIDAt(r, id int) {
	if _, ok := ref.ObjectMap[id]; !ok {
		return
	}
	ref.ObjectToRegion[id][r] = true
}

// PlaceObjectAt places an object at the specified region ID.
func (ref *MultiReferences[T]) PlaceObjectAt(obj *T, r int) {
	id := (*obj).GetID()
	if _, ok := ref.ObjectMap[id]; !ok {
		ref.Objects = append(ref.Objects, obj)
		ref.ObjectMap[id] = obj
		ref.ObjectToRegion[id] = make([]bool, ref.NumRegions)
	}
	ref.ObjectToRegion[id][r] = true
}

// RemoveObjectAt removes the object at the specified region ID.
func (ref *MultiReferences[T]) RemoveObjectAt(obj *T, r int) {
	id := (*obj).GetID()
	if _, ok := ref.ObjectMap[id]; !ok {
		return
	}
	ref.ObjectToRegion[id][r] = false
}
