package genworldvoronoi

import (
	"image/color"
	"math"
	"math/rand"
	"sort"
	"sync"

	"github.com/Flokey82/go_gens/utils"
	"github.com/Flokey82/go_gens/vectors"
)

// getCentroidOfTriangle returns the centroid of a triangle defined by
// the xyz coordinates a, b, c as a vectors.Vec3.
func getCentroidOfTriangle(a, b, c []float64) vectors.Vec3 {
	return vectors.Vec3{
		X: (a[0] + b[0] + c[0]) / 3,
		Y: (a[1] + b[1] + c[1]) / 3,
		Z: (a[2] + b[2] + c[2]) / 3,
	}.Normalize()
}

var zero2 = [2]float64{0, 0}

// dist2 returns the eucledian distance between two points.
func dist2(a, b [2]float64) float64 {
	xDiff := a[0] - b[0]
	yDiff := a[1] - b[1]
	return math.Sqrt(xDiff*xDiff + yDiff*yDiff)
}

// dot2 returns the dot product of two vectors.
func dot2(a, b [2]float64) float64 {
	return a[0]*b[0] + a[1]*b[1]
}

// len2 returns the length of the given vector.
func len2(a [2]float64) float64 {
	return math.Sqrt(a[0]*a[0] + a[1]*a[1])
}

// normalize2 returns the normalized vector of the given vector.
func normalize2(a [2]float64) [2]float64 {
	l := 1.0 / len2(a)
	return [2]float64{
		a[0] * l,
		a[1] * l,
	}
}

// angle2 returns the angle between two vectors.
func angle2(a, b [2]float64) float64 {
	return math.Acos(dot2(a, b) / (len2(a) * len2(b)))
}

// add2 returns the sum of two vectors.
func add2(a, b [2]float64) [2]float64 {
	return [2]float64{
		a[0] + b[0],
		a[1] + b[1],
	}
}

// scale2 returns the scaled vector of the given vector.
func scale2(v [2]float64, s float64) [2]float64 {
	return [2]float64{
		v[0] * s,
		v[1] * s,
	}
}

func setMagnitude2(v [2]float64, mag float64) [2]float64 {
	oldMag := math.Sqrt(v[0]*v[0] + v[1]*v[1])
	if oldMag == 0 {
		return v
	}
	return [2]float64{v[0] * mag / oldMag, v[1] * mag / oldMag}
}

// normal2 returns the normalized vector of the given vector.
func normal2(v [2]float64) [2]float64 {
	l := 1.0 / len2(v)
	return [2]float64{
		v[0] * l,
		v[1] * l,
	}
}

// rotate2 returns the rotated vector of the given vector.
func rotate2(v [2]float64, angle float64) [2]float64 {
	sin := math.Sin(angle)
	cos := math.Cos(angle)
	return [2]float64{
		v[0]*cos - v[1]*sin,
		v[0]*sin + v[1]*cos,
	}
}

// cross2 returns the cross product of two vectors.
func cross2(a, b [2]float64) float64 {
	return a[0]*b[1] - a[1]*b[0]
}

// sub2 returns the difference of two vectors.
func sub2(a, b [2]float64) [2]float64 {
	return [2]float64{
		a[0] - b[0],
		a[1] - b[1],
	}
}

// distToSegment2 returns the distance between a point p and a line
// segment defined by the points v and w.
func distToSegment2(v, w, p [2]float64) float64 {
	l2 := dist2(v, w)
	if l2 == 0 {
		// If the line segment has a length of 0, we can just return
		// the distance between the point and any of the two line
		// segment points.
		return dist2(p, v)
	}
	t := math.Max(0, math.Min(1, ((p[0]-v[0])*(w[0]-v[0])+(p[1]-v[1])*(w[1]-v[1]))/(l2*l2)))
	return dist2(p, [2]float64{v[0] + t*(w[0]-v[0]), v[1] + t*(w[1]-v[1])})
}

// convToVec3 converts a float slice containing 3 values into a vectors.Vec3.
func convToVec3(xyz []float64) vectors.Vec3 {
	return vectors.Vec3{
		X: xyz[0],
		Y: xyz[1],
		Z: xyz[2],
	}
}

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

func radToDeg(rad float64) float64 {
	return rad * 180 / math.Pi
}

// Adds a vector to a latitude and longitude in degrees.
// The vector's x coordinate is modified by the cosine of the latitude to
// account for the fact that the distance between degrees of longitude
// decreases as the latitude increases.
func addVecToLatLong(lat, lon float64, vec [2]float64) (float64, float64) {
	return lat + vec[1], lon + vec[0]/math.Cos(degToRad(lat+vec[1]))
}

// I'm not sure if this is correct, but it seems to work.
func vectorToLatLong(vec [2]float64) (float64, float64) {
	return radToDeg(math.Asin(vec[0])), // Lat
		radToDeg(math.Atan2(vec[1], math.Sqrt(1-vec[0]*vec[0]))) // Lon
}

// calcVecFromLatLong calculates the vector between two lat/long pairs.
func calcVecFromLatLong(lat1, lon1, lat2, lon2 float64) [2]float64 {
	// Old implementation, which is wrong.
	// convert to radians
	// lat1 = degToRad(lat1)
	// lon1 = degToRad(lon1)
	// lat2 = degToRad(lat2)
	// lon2 = degToRad(lon2)
	//
	// 	return [2]float64{
	// 		math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(lon2-lon1), // X
	// 		math.Sin(lon2-lon1) * math.Cos(lat2),                                              // Y
	// 	}

	// Calculate the vector between two lat/long pairs using the bearing we calculate
	// from the lat/long pairs.
	// Note that the bearing is 0 when facing north, and increases clockwise, so we need to
	// convert it so that int is 0 when facing east, and increases counter-clockwise (and to radians).
	bearing := math.Pi/2 - calcBearingRad(lat1, lon1, lat2, lon2)

	// Convert the bearing to a vector and scale it to the distance between the two points.
	dist := haversine(lat1, lon1, lat2, lon2)
	return [2]float64{math.Cos(bearing) * dist, math.Sin(bearing) * dist}
}

// calcBearing calculates the bearing between two lat/long pairs.
func calcBearingRad(lat1, lon1, lat2, lon2 float64) float64 {
	// convert to radians
	lat1 = degToRad(lat1)
	lon1 = degToRad(lon1)
	lat2 = degToRad(lat2)
	lon2 = degToRad(lon2)

	return math.Atan2(
		math.Sin(lon2-lon1)*math.Cos(lat2),                                              // y
		math.Cos(lat1)*math.Sin(lat2)-math.Sin(lat1)*math.Cos(lat2)*math.Cos(lon2-lon1), // x
	)
}

// latLonToCartesian converts latitude and longitude to x, y, z coordinates.
// See: https://rbrundritt.wordpress.com/2008/10/14/conversion-between-spherical-and-cartesian-coordinates-systems/
func latLonToCartesian(latDeg, lonDeg float64) []float64 {
	latRad := (latDeg / 180.0) * math.Pi
	lonRad := (lonDeg / 180.0) * math.Pi
	return []float64{
		math.Cos(latRad) * math.Cos(lonRad),
		math.Cos(latRad) * math.Sin(lonRad),
		math.Sin(latRad),
	}
}

// latLonFromVec3 converts a vectors.Vec3 to latitude and longitude.
// See: https://rbrundritt.wordpress.com/2008/10/14/conversion-between-spherical-and-cartesian-coordinates-systems/
func latLonFromVec3(position vectors.Vec3, sphereRadius float64) (float64, float64) {
	// See https://stackoverflow.com/questions/46247499/vector3-to-latitude-longitude
	return radToDeg(math.Asin(position.Z / sphereRadius)), // theta
		radToDeg(math.Atan2(position.Y, position.X)) // phi
}

// haversine returns the great arc distance between two lat/long pairs.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	dLatSin := math.Sin(degToRad(lat2-lat1) / 2)
	dLonSin := math.Sin(degToRad(lon2-lon1) / 2)
	a := dLatSin*dLatSin + dLonSin*dLonSin*math.Cos(degToRad(lat1))*math.Cos(degToRad(lat2))
	return 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// crossArc calculates the shortest distance between an arc (defined by p1 and p2)
// and a third point, p3. The input is expected in degrees.
// See: https://stackoverflow.com/questions/32771458/distance-from-lat-lng-point-to-minor-arc-segment
func crossArc(lat1, lon1, lat2, lon2, lat3, lon3 float64) float64 {
	// dis Finds the distance between two lat/lon points.
	dis := func(latA, lonA, latB, lonB float64) float64 {
		return math.Acos(math.Sin(latA)*math.Sin(latB) + math.Cos(latA)*math.Cos(latB)*math.Cos(lonB-lonA))
	}

	// bearing Finds the bearing from one lat/lon point to another.
	bearing := func(latA, lonA, latB, lonB float64) float64 {
		return math.Atan2(math.Sin(lonB-lonA)*math.Cos(latB), math.Cos(latA)*math.Sin(latB)-math.Sin(latA)*math.Cos(latB)*math.Cos(lonB-lonA))
	}

	lat1 = degToRad(lat1)
	lat2 = degToRad(lat2)
	lat3 = degToRad(lat3)
	lon1 = degToRad(lon1)
	lon2 = degToRad(lon2)
	lon3 = degToRad(lon3)

	// Prerequisites for the formulas
	bear12 := bearing(lat1, lon1, lat2, lon2)
	bear13 := bearing(lat1, lon1, lat3, lon3)
	dis13 := dis(lat1, lon1, lat3, lon3)

	diff := math.Abs(bear13 - bear12)
	if diff > math.Pi {
		diff = 2*math.Pi - diff
	}
	// Is relative bearing obtuse?
	if diff > math.Pi/2 {
		return dis13
	}
	// Find the cross-track distance.
	dxt := math.Asin(math.Sin(dis13) * math.Sin(bear13-bear12))

	// Is p4 beyond the arc?
	dis12 := dis(lat1, lon1, lat2, lon2)
	dis14 := math.Acos(math.Cos(dis13) / math.Cos(dxt))
	if dis14 > dis12 {
		return dis(lat2, lon2, lat3, lon3)
	}
	return math.Abs(dxt)
}

// calcHeightInTriangle calculates the height of a point in a triangle.
func calcHeightInTriangle(p1, p2, p3, p [2]float64, z1, z2, z3 float64) float64 {
	// Calculate the barycentric coordinates of the point (xp, yp) with respect to the triangle
	denom := (p2[1]-p3[1])*(p1[0]-p3[0]) + (p3[0]-p2[0])*(p1[1]-p3[1])
	s := ((p2[1]-p3[1])*(p[0]-p3[0]) + (p3[0]-p2[0])*(p[1]-p3[1])) / denom
	t := ((p3[1]-p1[1])*(p[0]-p3[0]) + (p1[0]-p3[0])*(p[1]-p3[1])) / denom
	u := 1 - s - t
	// Calculate the height of our point in the triangle.
	z := z1*s + z2*t + z3*u
	return z
}

// isPointInTriangle returns true if the point (xp, yp) is inside the triangle or
// on the edge of the triangle.
func isPointInTriangle(p1, p2, p3, p [2]float64) bool {
	// Calculate the barycentric coordinates of the point (xp, yp) with respect to the triangle
	denom := (p2[1]-p3[1])*(p1[0]-p3[0]) + (p3[0]-p2[0])*(p1[1]-p3[1])
	s := ((p2[1]-p3[1])*(p[0]-p3[0]) + (p3[0]-p2[0])*(p[1]-p3[1])) / denom
	t := ((p3[1]-p1[1])*(p[0]-p3[0]) + (p1[0]-p3[0])*(p[1]-p3[1])) / denom
	u := 1 - s - t

	// Check if the point is inside the triangle
	return s >= 0 && t >= 0 && u >= 0
}

// heronsTriArea returns the area of a triangle given the three sides.
// See: https://www.mathopenref.com/heronsformula.html
func heronsTriArea(a, b, c float64) float64 {
	p := (a + b + c) / 2
	return math.Sqrt(p * (p - a) * (p - b) * (p - c))
}

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

// roundToDecimals rounds the given float to the given number of decimals.
func roundToDecimals(v, d float64) float64 {
	m := math.Pow(10, d)
	return math.Round(v*m) / m
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

func genBlue(intensity float64) color.NRGBA {
	return color.NRGBA{
		R: uint8(intensity * 255),
		G: uint8(intensity * 255),
		B: 255,
		A: 255,
	}
}

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
