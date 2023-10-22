package various

import (
	"math"

	"github.com/Flokey82/go_gens/vectors"
)

var Zero2 = [2]float64{0, 0}

// Dist2 returns the eucledian distance between two points.
func Dist2(a, b [2]float64) float64 {
	xDiff := a[0] - b[0]
	yDiff := a[1] - b[1]
	return math.Sqrt(xDiff*xDiff + yDiff*yDiff)
}

// Dot2 returns the dot product of two vectors.
func Dot2(a, b [2]float64) float64 {
	return a[0]*b[0] + a[1]*b[1]
}

// Len2 returns the length of the given vector.
func Len2(a [2]float64) float64 {
	return math.Sqrt(a[0]*a[0] + a[1]*a[1])
}

// Normalize2 returns the normalized vector of the given vector.
func Normalize2(a [2]float64) [2]float64 {
	l := 1.0 / Len2(a)
	return [2]float64{
		a[0] * l,
		a[1] * l,
	}
}

// Angle2 returns the angle between two vectors.
func Angle2(a, b [2]float64) float64 {
	return math.Acos(Dot2(a, b) / (Len2(a) * Len2(b)))
}

// Add2 returns the sum of two vectors.
func Add2(a, b [2]float64) [2]float64 {
	return [2]float64{
		a[0] + b[0],
		a[1] + b[1],
	}
}

// Scale2 returns the scaled vector of the given vector.
func Scale2(v [2]float64, s float64) [2]float64 {
	return [2]float64{
		v[0] * s,
		v[1] * s,
	}
}

func SetMagnitude2(v [2]float64, mag float64) [2]float64 {
	oldMag := math.Sqrt(v[0]*v[0] + v[1]*v[1])
	if oldMag == 0 {
		return v
	}
	return [2]float64{v[0] * mag / oldMag, v[1] * mag / oldMag}
}

// Normal2 returns the normalized vector of the given vector.
func Normal2(v [2]float64) [2]float64 {
	l := 1.0 / Len2(v)
	return [2]float64{
		v[0] * l,
		v[1] * l,
	}
}

// Rotate2 returns the rotated vector of the given vector.
func Rotate2(v [2]float64, angle float64) [2]float64 {
	sin := math.Sin(angle)
	cos := math.Cos(angle)
	return [2]float64{
		v[0]*cos - v[1]*sin,
		v[0]*sin + v[1]*cos,
	}
}

// Cross2 returns the cross product of two vectors.
func Cross2(a, b [2]float64) float64 {
	return a[0]*b[1] - a[1]*b[0]
}

// Sub2 returns the difference of two vectors.
func Sub2(a, b [2]float64) [2]float64 {
	return [2]float64{
		a[0] - b[0],
		a[1] - b[1],
	}
}

// DistToSegment2 returns the distance between a point p and a line
// segment defined by the points v and w.
func DistToSegment2(v, w, p [2]float64) float64 {
	l2 := Dist2(v, w)
	if l2 == 0 {
		// If the line segment has a length of 0, we can just return
		// the distance between the point and any of the two line
		// segment points.
		return Dist2(p, v)
	}
	t := math.Max(0, math.Min(1, ((p[0]-v[0])*(w[0]-v[0])+(p[1]-v[1])*(w[1]-v[1]))/(l2*l2)))
	return Dist2(p, [2]float64{v[0] + t*(w[0]-v[0]), v[1] + t*(w[1]-v[1])})
}

// ConvToVec3 converts a float slice containing 3 values into a vectors.Vec3.
func ConvToVec3(xyz []float64) vectors.Vec3 {
	return vectors.Vec3{
		X: xyz[0],
		Y: xyz[1],
		Z: xyz[2],
	}
}
