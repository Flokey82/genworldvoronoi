package civ

import (
	"math"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/various"
)

var convToMap = various.ConvToMap
var convToArray = various.ConvToArray

// isInIntList returns true if the given int is in the given slice.
var isInIntList = various.IsInIntList

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

type ClampedVal float64

func (c *ClampedVal) Add(v float64) {
	*c = ClampedVal(math.Max(0, math.Min(1, float64(*c)+v)))
}
