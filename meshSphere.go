package genworldvoronoi

import (
	"math"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/geoquad"
	"github.com/fogleman/delaunay"
)

// generateFibonacciSphere generates a number of points along a spiral on a sphere
// as a flat array of lat, lon coordinates.
func generateFibonacciSphere(seed int64, numPoints int, jitter float64) []float64 {
	rnd := rand.New(rand.NewSource(seed))

	// Second algorithm from http://web.archive.org/web/20120421191837/http://www.cgafaq.info/wiki/Evenly_distributed_points_on_sphere
	s := 3.6 / math.Sqrt(float64(numPoints))
	dlong := math.Pi * (3 - math.Sqrt(5)) // ~2.39996323
	dz := 2.0 / float64(numPoints)

	var latLon []float64
	for k := 0; k < numPoints; k++ {
		// Calculate latitude as z value from -1 to 1.
		z := 1 - (dz / 2) - float64(k)*dz

		// Calculate longitude in rad.
		long := float64(k) * dlong

		// Calculate the radius at the given z.
		r := math.Sqrt(1 - z*z)

		// Calculate latitude and longitude in degrees.
		latDeg := math.Asin(z) * 180 / math.Pi
		lonDeg := long * 180 / math.Pi

		// Apply jitter if any is set.
		if jitter > 0 {
			latDeg += jitter * (rnd.Float64() - rnd.Float64()) * (latDeg - math.Asin(math.Max(-1, z-dz*2*math.Pi*r/s))*180/math.Pi)
			lonDeg += jitter * (rnd.Float64() - rnd.Float64()) * (s / r * 180 / math.Pi)
		}

		latLon = append(latLon, latDeg, math.Mod(lonDeg, 360.0))
	}
	return latLon
}

/** Add south pole back into the mesh.
 *
 * We run the Delaunay Triangulation on all points *except* the south
 * pole, which gets mapped to infinity with the stereographic
 * projection. This function adds the south pole into the
 * triangulation. The Delaunator guide explains how the halfedges have
 * to be connected to make the mesh work.
 * <https://mapbox.github.io/delaunator/>
 *
 * Returns the new {triangles, halfedges} for the triangulation with
 * one additional point added around the convex hull.
 */
func addSouthPoleToMesh(southPoleId int, d *delaunay.Triangulation) *delaunay.Triangulation {
	// This logic is from <https://github.com/redblobgames/dual-mesh>,
	// where I use it to insert a "ghost" region on the "back" side of
	// the planar map. The same logic works here. In that code I use
	// "s" for edges ("sides"), "r" for regions ("points"), t for triangles
	triangles := d.Triangles
	numSides := len(triangles)
	s_next_s := func(s int) int {
		if s%3 == 2 {
			return s - 2
		}
		return s + 1
	}

	halfedges := d.Halfedges
	numUnpairedSides := 0
	firstUnpairedSide := -1
	pointIdToSideId := make(map[int]int) // seed to side
	for s := 0; s < numSides; s++ {
		if halfedges[s] == -1 {
			numUnpairedSides++
			pointIdToSideId[triangles[s]] = s
			firstUnpairedSide = s
		}
	}

	newTriangles := make([]int, numSides+3*numUnpairedSides)
	copy(newTriangles, triangles)

	newHalfedges := make([]int, numSides+3*numUnpairedSides)
	copy(newHalfedges, halfedges)

	for i, s := 0, firstUnpairedSide; i < numUnpairedSides; i++ {
		// Construct a pair for the unpaired side s
		newSide := numSides + 3*i
		newHalfedges[s] = newSide
		newHalfedges[newSide] = s
		newTriangles[newSide] = newTriangles[s_next_s(s)]

		// Construct a triangle connecting the new side to the south pole
		newTriangles[newSide+1] = newTriangles[s]
		newTriangles[newSide+2] = southPoleId
		k := numSides + (3*i+4)%(3*numUnpairedSides)
		newHalfedges[newSide+2] = k
		newHalfedges[k] = newSide + 2
		s = pointIdToSideId[newTriangles[s_next_s(s)]]
	}

	return &delaunay.Triangulation{
		Triangles: newTriangles,
		Halfedges: newHalfedges,
	}
}

// stereographicProjection converts 3d coordinates into two dimensions.
// See: https://en.wikipedia.org/wiki/Stereographic_projection
func stereographicProjection(xyz []float64) []float64 {
	numPoints := len(xyz) / 3
	xy := make([]float64, 0, 2*numPoints)
	for r := 0; r < numPoints; r++ {
		x := xyz[3*r]
		y := xyz[3*r+1]
		z := xyz[3*r+2]
		xy = append(xy, x/(1-z), y/(1-z)) // Append projected 2d coordinates.
	}
	return xy
}

func MakeSphere(seed int64, numPoints int, jitter float64) (*SphereMesh, error) {
	// Generate a Fibonacci sphere.
	latlong := generateFibonacciSphere(seed, numPoints, jitter)

	// Convert the lat/lon coordinates to x,y,z.
	var xyz []float64
	var latLon [][2]float64
	for r := 0; r < len(latlong); r += 2 {
		// HACKY! Fix this properly!
		nla, nlo := various.LatLonFromVec3(various.ConvToVec3(various.LatLonToCartesian(latlong[r], latlong[r+1])).Normalize(), 1.0)
		latLon = append(latLon, [2]float64{nla, nlo})

		// This calculates x,y,z from the spherical coordinates lat,lon.
		xyz = append(xyz, various.LatLonToCartesian(latlong[r], latlong[r+1])...)
	}
	return newSphereMesh(latLon, xyz, true)
}

type SphereMesh struct {
	*TriangleMesh
	XYZ         []float64         // Region coordinates
	LatLon      [][2]float64      // Region latitude and longitude
	TriXYZ      []float64         // Triangle xyz coordinates
	TriLatLon   [][2]float64      // Triangle latitude and longitude
	regQuadTree *geoquad.QuadTree // Quadtree for region lookup
	triQuadTree *geoquad.QuadTree // Quadtree for triangle lookup
}

func newSphereMesh(latLon [][2]float64, xyz []float64, addSouthPole bool) (*SphereMesh, error) {
	// Map the sphere on a plane using the stereographic projection.
	xy := stereographicProjection(xyz)

	// Create a Delaunay triangulation of the points.
	pts := make([]delaunay.Point, 0, len(xy)/2)
	for i := 0; i < len(xy); i += 2 {
		pts = append(pts, delaunay.Point{X: xy[i], Y: xy[i+1]})
	}
	tri, err := delaunay.Triangulate(pts)
	if err != nil {
		return nil, err
	}

	// Close the hole at the south pole if requested.
	// TODO: rotate an existing point into this spot instead of creating one.
	if addSouthPole {
		xyz = append(xyz, 0, 0, 1)
		latLon = append(latLon, [2]float64{-90.0, 45.0})
		tri = addSouthPoleToMesh((len(xyz)/3)-1, tri)
	}

	// Create a mesh from the triangulation.
	m := &SphereMesh{
		TriangleMesh: NewTriangleMesh(len(latLon), tri.Triangles, tri.Halfedges),
		XYZ:          xyz,
		LatLon:       latLon,
	}

	// Iterate over all triangles and generates the centroids for each.
	tXYZ := make([]float64, 0, m.numTriangles*3)
	tLatLon := make([][2]float64, 0, m.numTriangles)
	for t := 0; t < m.numTriangles; t++ {
		a := m.s_begin_r(3 * t)
		b := m.s_begin_r(3*t + 1)
		c := m.s_begin_r(3*t + 2)
		v3 := various.GetCentroidOfTriangle(
			m.XYZ[3*a:3*a+3],
			m.XYZ[3*b:3*b+3],
			m.XYZ[3*c:3*c+3])
		tXYZ = append(tXYZ, v3.X, v3.Y, v3.Z)
		nla, nlo := various.LatLonFromVec3(v3, 1.0)
		tLatLon = append(tLatLon, [2]float64{nla, nlo})
	}
	m.TriLatLon = tLatLon
	m.TriXYZ = tXYZ

	// Create a quadtree for region lookup.
	m.regQuadTree = newQuadTreeFromLatLon(m.LatLon)

	// Create a quadtree for triangle lookup.
	m.triQuadTree = newQuadTreeFromLatLon(m.TriLatLon)

	return m, nil
}

func newQuadTreeFromLatLon(latLon [][2]float64) *geoquad.QuadTree {
	var points []geoquad.Point
	for i := range latLon {
		ll := latLon[i]
		points = append(points, geoquad.Point{
			Lat:  ll[0],
			Lon:  ll[1],
			Data: i,
		})
	}
	return geoquad.NewQuadTree(points)
}

// MakeCoarseSphereMesh returns a sphere mesh with 1/step density.
func (m *SphereMesh) MakeCoarseSphereMesh(step int) (*SphereMesh, error) {
	// Convert the lat/lon coordinates to x,y,z. (skip the existing south pole)
	var xyz []float64
	var latLon [][2]float64
	for r := 0; r < len(m.LatLon)-1; r += step {
		xyz = append(xyz, m.XYZ[3*r:3*r+3]...)
		latLon = append(latLon, m.LatLon[r])
	}

	// Now adjust the indices of the triangles
	return newSphereMesh(latLon, xyz, true)
}
