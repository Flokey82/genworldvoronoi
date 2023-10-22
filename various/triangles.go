package various

import (
	"math"

	"github.com/Flokey82/go_gens/vectors"
)

// GetCentroidOfTriangle returns the centroid of a triangle defined by
// the xyz coordinates a, b, c as a vectors.Vec3.
func GetCentroidOfTriangle(a, b, c []float64) vectors.Vec3 {
	return vectors.Vec3{
		X: (a[0] + b[0] + c[0]) / 3,
		Y: (a[1] + b[1] + c[1]) / 3,
		Z: (a[2] + b[2] + c[2]) / 3,
	}.Normalize()
}

// HeronsTriArea returns the area of a triangle given the three sides.
// See: https://www.mathopenref.com/heronsformula.html
func HeronsTriArea(a, b, c float64) float64 {
	p := (a + b + c) / 2
	return math.Sqrt(p * (p - a) * (p - b) * (p - c))
}

// CalcHeightInTriangle calculates the height of a point in a triangle.
func CalcHeightInTriangle(p1, p2, p3, p [2]float64, z1, z2, z3 float64) float64 {
	// Calculate the barycentric coordinates of the point (xp, yp) with respect to the triangle
	denom := (p2[1]-p3[1])*(p1[0]-p3[0]) + (p3[0]-p2[0])*(p1[1]-p3[1])
	s := ((p2[1]-p3[1])*(p[0]-p3[0]) + (p3[0]-p2[0])*(p[1]-p3[1])) / denom
	t := ((p3[1]-p1[1])*(p[0]-p3[0]) + (p1[0]-p3[0])*(p[1]-p3[1])) / denom
	u := 1 - s - t
	// Calculate the height of our point in the triangle.
	z := z1*s + z2*t + z3*u
	return z
}

// IsPointInTriangle returns true if the point (xp, yp) is inside the triangle or
// on the edge of the triangle.
func IsPointInTriangle(p1, p2, p3, p [2]float64) bool {
	// Calculate the barycentric coordinates of the point (xp, yp) with respect to the triangle
	denom := (p2[1]-p3[1])*(p1[0]-p3[0]) + (p3[0]-p2[0])*(p1[1]-p3[1])
	s := ((p2[1]-p3[1])*(p[0]-p3[0]) + (p3[0]-p2[0])*(p[1]-p3[1])) / denom
	t := ((p3[1]-p1[1])*(p[0]-p3[0]) + (p1[0]-p3[0])*(p[1]-p3[1])) / denom
	u := 1 - s - t

	// Check if the point is inside the triangle
	return s >= 0 && t >= 0 && u >= 0
}
