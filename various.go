package genworldvoronoi

import (
	"image/color"
	"math"
	"math/rand"
	"sort"
	"sync"

	"github.com/Flokey82/go_gens/utils"
)

// convToMap converts a slice of ints into a map of ints to bools.
func convToMap(in []int) map[int]bool {
	res := make(map[int]bool)
	for _, v := range in {
		res[v] = true
	}
	return res
}

// convToArray converts a map of ints to bools into a slice of ints.
func convToArray(in map[int]bool) []int {
	var res []int
	for v := range in {
		res = append(res, v)
	}
	sort.Ints(res)
	return res
}

// isInIntList returns true if the given int is in the given slice.
func isInIntList(l []int, i int) bool {
	for _, v := range l {
		if v == i {
			return true
		}
	}
	return false
}

var minMax = utils.MinMax[float64]
var minMax64 = utils.MinMax[int64]

// initFloatSlice returns a slice of floats of the given size, initialized to -1.
func initFloatSlice(size int) []float64 {
	return initSlice[float64](size)
}

// initRegionSlice returns a slice of ints of the given size, initialized to -1.
func initRegionSlice(size int) []int {
	return initSlice[int](size)
}

// initTimeSlice returns a slice of int64s of the given size, initialized to -1.
func initTimeSlice(size int) []int64 {
	return initSlice[int64](size)
}

// initSlice returns a slice of the given type of the given size, initialized to -1.
func initSlice[V utils.Number](size int) []V {
	res := make([]V, size)
	for i := range res {
		res[i] = -1
	}
	return res
}

// mergeIndexSegments matches up the ends of the segments (region pairs) and returns
// a slice containing all continuous, connected segments as sequence of connected regions.
func mergeIndexSegments(segs [][2]int) [][]int {
	adj := make(map[int][]int)
	for i := 0; i < len(segs); i++ {
		seg := segs[i]
		adj[seg[0]] = append(adj[seg[0]], seg[1])
		adj[seg[1]] = append(adj[seg[1]], seg[0])
	}
	var paths [][]int
	var path []int
	for len(segs) > 0 {
		if path == nil {
			seg := segs[0]
			segs = segs[1:]
			path = []int{seg[0], seg[1]}
		}
		var changed bool
		for i := 0; i < len(segs); i++ {
			seg := segs[i]
			if len(adj[path[0]]) == 2 && (seg[0] == path[0] || seg[1] == path[0]) {
				if seg[0] == path[0] {
					path = unshiftIndexPath(path, seg[1])
				} else {
					path = unshiftIndexPath(path, seg[0])
				}
				segs = append(segs[:i], segs[i+1:]...)
				changed = true
				break
			}
			if len(adj[path[len(path)-1]]) == 2 && (seg[0] == path[len(path)-1] || seg[1] == path[len(path)-1]) {
				if seg[0] == path[len(path)-1] {
					path = append(path, seg[1])
				} else {
					path = append(path, seg[0])
				}
				segs = append(segs[:i], segs[i+1:]...)
				changed = true
				break
			}
		}
		if !changed {
			paths = append(paths, path)
			path = nil
		}
	}
	return paths
}

func unshiftIndexPath(path []int, p int) []int {
	res := make([]int, len(path)+1)
	res[0] = p
	copy(res[1:], path)
	return res
}

// weightedToArray converts a map of weighted values to an array.
func weightedToArray(weighted map[string]int) []string {
	var res []string
	for key, weight := range weighted {
		for j := 0; j < weight; j++ {
			res = append(res, key)
		}
	}
	return res
}

// probability shorthand
func P(probability float64) bool {
	if probability >= 1.0 {
		return true
	}
	if probability <= 0 {
		return false
	}
	return rand.Float64() < probability
}

// queueEntry is a single entry in the priority queue.
type queueEntry struct {
	index       int     // index of the item in the heap.
	score       float64 // priority of the item in the queue.
	origin      int     // origin region / ID
	destination int     // destination region / ID
}

// ascPriorityQueue implements heap.Interface and holds Items.
// Priority is ascending (lowest score first).
type ascPriorityQueue []*queueEntry

func (pq ascPriorityQueue) Len() int { return len(pq) }

func (pq ascPriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	// return pq[i].score > pq[j].score // 3, 2, 1

	// We want Pop to give us the lowest, not highest, priority so we use less than here.
	return pq[i].score < pq[j].score // 1, 2, 3
}

func (pq *ascPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *ascPriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*queueEntry)
	item.index = n
	*pq = append(*pq, item)
}

func (pq ascPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index, pq[j].index = i, j
}

// getDiversionFromRange returns the amount a value diverges from a range.
func getDiversionFromRange(x float64, rng [2]float64) float64 {
	if x < rng[0] {
		return rng[0] - x
	}
	if x > rng[1] {
		return x - rng[1]
	}
	return 0
}

// getRangeFit returns 1 if a value fits within the range, and a value between 0 and 1 if it doesn't.
// 0 means the value is outside the range, 1 means it's at the edge of the range.
// If the value is more than 20% outside the range, -1 is returned.
// If the value is exactly 20% outside the range, 0 is returned.
func getRangeFit(x float64, rng [2]float64) float64 {
	if x < rng[0] {
		if x < rng[0]*0.8 {
			return -1
		}
		return math.Abs(rng[0]-x) / (rng[0] * 0.2)
	}
	if x > rng[1] {
		if x > rng[1]*1.2 {
			return -1
		}
		return math.Abs(x-rng[1]) / (rng[1] * 0.2)
	}
	return 1
}

// genBlue returns a blue color with the given intensity (0.0-1.0).
func genBlue(intensity float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(intensity * 255),
		G: uint8(intensity * 255),
		B: 255,
		A: 255,
	}
}

// genGreen returns a green color with the given intensity (0.0-1.0).
func genGreen(intensity float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(intensity * 255),
		B: uint8((1 - intensity) * 255),
		G: 255,
		A: 255,
	}
}

// genBlackShadow returns a black color that is more transparent the higher the intensity.
func genBlackShadow(intensity float64) color.NRGBA {
	return color.NRGBA{
		A: uint8((1 - intensity) * 255),
	}
}

func genColor(col color.Color, intensity float64) color.Color {
	var col2 color.NRGBA
	cr, cg, cb, _ := col.RGBA()
	col2.R = uint8(float64(255) * float64(cr) / float64(0xffff))
	col2.G = uint8(float64(255) * float64(cg) / float64(0xffff))
	col2.B = uint8(float64(255) * float64(cb) / float64(0xffff))
	col2.A = 255
	return col2
}

func kickOffChunkWorkers(totalItems int, fn func(start, end int)) {
	numWorkers := 8

	var wg sync.WaitGroup
	var chunkStart int
	chunkSize := (totalItems / numWorkers) + 1
	for i := 0; i < numWorkers; i++ {
		curChunk := chunkSize
		if rem := totalItems - chunkStart; rem < curChunk {
			curChunk = rem
		}
		if curChunk <= 0 {
			break
		}
		wg.Add(1)
		go func(start, end int) {
			fn(start, end)
			wg.Done()
		}(chunkStart, chunkStart+curChunk)
		chunkStart += curChunk
	}
	wg.Wait()
}

// randPerm returns a random permutation of the given slice indices.
// This works like rand.Perm, but it reuses the same slice.
func (m *BaseObject) randPerm(perm []int, n int) []int {
	perm = perm[:utils.Min(cap(perm), n)]
	if diff := len(perm) - n; diff < 0 {
		perm = append(perm, make([]int, -diff)...)
	}
	// In the following loop, the iteration when i=0 always swaps m[0] with m[0].
	// A change to remove this useless iteration is to assign 1 to i in the init
	// statement. But Perm also effects r. Making this change will affect
	// the final state of r. So this change can't be made for compatibility
	// reasons for Go 1.
	for i := 0; i < n; i++ {
		j := m.rand.Intn(i + 1)
		perm[i] = perm[j]
		perm[j] = i
	}
	return perm
}
