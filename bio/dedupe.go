package bio

import (
	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/go_gens/utils"
)

// isInIntList returns true if the given int is in the given slice.
var isInIntList = various.IsInIntList

var minMax = utils.MinMax[float64]
var minMax64 = utils.MinMax[int64]

var convToMap = various.ConvToMap
var convToArray = various.ConvToArray

var initFloatSlice = various.InitFloatSlice
var initRegionSlice = various.InitRegionSlice
var initTimeSlice = various.InitTimeSlice
