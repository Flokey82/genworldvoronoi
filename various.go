package genworldvoronoi

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/go_gens/utils"
)

var convToMap = various.ConvToMap
var convToArray = various.ConvToArray

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

var initFloatSlice = various.InitFloatSlice
var initRegionSlice = various.InitRegionSlice
var initTimeSlice = various.InitTimeSlice

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

// QueueEntry is a single entry in the priority queue.
type QueueEntry struct {
	Index       int     // index of the item in the heap.
	Score       float64 // priority of the item in the queue.
	Origin      int     // origin region / ID
	Destination int     // destination region / ID
}

// AscPriorityQueue implements heap.Interface and holds Items.
// Priority is ascending (lowest score first).
type AscPriorityQueue []*QueueEntry

func (pq AscPriorityQueue) Len() int { return len(pq) }

func (pq AscPriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	// return pq[i].score > pq[j].score // 3, 2, 1

	// We want Pop to give us the lowest, not highest, priority so we use less than here.
	return pq[i].Score < pq[j].Score // 1, 2, 3
}

func (pq *AscPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *AscPriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*QueueEntry)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq AscPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index, pq[j].Index = i, j
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
