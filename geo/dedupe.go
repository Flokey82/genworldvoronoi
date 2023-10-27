package geo

import (
	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/go_gens/utils"
)

var minMax = utils.MinMax[float64]
var minMax64 = utils.MinMax[int64]

var convToMap = various.ConvToMap
var convToArray = various.ConvToArray

var initFloatSlice = various.InitFloatSlice
var initRegionSlice = various.InitRegionSlice
var initTimeSlice = various.InitTimeSlice
