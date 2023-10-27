package various

import (
	"math"
	"sort"

	"github.com/Flokey82/go_gens/utils"
)

// IsInIntList returns true if the given int is in the given slice.
func IsInIntList(l []int, i int) bool {
	for _, v := range l {
		if v == i {
			return true
		}
	}
	return false
}

// RoundToDecimals rounds the given float to the given number of decimals.
func RoundToDecimals(v, d float64) float64 {
	m := math.Pow(10, d)
	return math.Round(v*m) / m
}

// MergeIndexSegments matches up the ends of the segments (region pairs) and returns
// a slice containing all continuous, connected segments as sequence of connected regions.
func MergeIndexSegments(segs [][2]int) [][]int {
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

// ConvToMap converts a slice of ints into a map of ints to bools.
func ConvToMap(in []int) map[int]bool {
	res := make(map[int]bool)
	for _, v := range in {
		res[v] = true
	}
	return res
}

// ConvToArray converts a map of ints to bools into a slice of ints.
func ConvToArray(in map[int]bool) []int {
	var res []int
	for v := range in {
		res = append(res, v)
	}
	sort.Ints(res)
	return res
}

// InitFloatSlice returns a slice of floats of the given size, initialized to -1.
func InitFloatSlice(size int) []float64 {
	return InitSlice[float64](size)
}

// InitRegionSlice returns a slice of ints of the given size, initialized to -1.
func InitRegionSlice(size int) []int {
	return InitSlice[int](size)
}

// InitTimeSlice returns a slice of int64s of the given size, initialized to -1.
func InitTimeSlice(size int) []int64 {
	return InitSlice[int64](size)
}

// InitSlice returns a slice of the given type of the given size, initialized to -1.
func InitSlice[V utils.Number](size int) []V {
	res := make([]V, size)
	for i := range res {
		res[i] = -1
	}
	return res
}
