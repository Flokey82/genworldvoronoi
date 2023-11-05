package genworldvoronoi

import (
	"image/color"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/go_gens/utils"
)

var convToMap = various.ConvToMap
var convToArray = various.ConvToArray

// isInIntList returns true if the given int is in the given slice.
var isInIntList = various.IsInIntList

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
