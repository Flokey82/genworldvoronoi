package various

import (
	"math"

	"github.com/Flokey82/go_gens/vectors"
)

func DegToRad(deg float64) float64 {
	return deg * math.Pi / 180
}

func RadToDeg(rad float64) float64 {
	return rad * 180 / math.Pi
}

// I'm not sure if this is correct, but it seems to work.
func VectorToLatLong(vec [2]float64) (float64, float64) {
	return RadToDeg(math.Asin(vec[0])), // Lat
		RadToDeg(math.Atan2(vec[1], math.Sqrt(1-vec[0]*vec[0]))) // Lon
}

// LatLonToCartesian converts latitude and longitude to x, y, z coordinates.
// See: https://rbrundritt.wordpress.com/2008/10/14/conversion-between-spherical-and-cartesian-coordinates-systems/
func LatLonToCartesian(latDeg, lonDeg float64) []float64 {
	latRad := (latDeg / 180.0) * math.Pi
	lonRad := (lonDeg / 180.0) * math.Pi
	return []float64{
		math.Cos(latRad) * math.Cos(lonRad),
		math.Cos(latRad) * math.Sin(lonRad),
		math.Sin(latRad),
	}
}

// LatLonFromVec3 converts a vectors.Vec3 to latitude and longitude.
// See: https://rbrundritt.wordpress.com/2008/10/14/conversion-between-spherical-and-cartesian-coordinates-systems/
func LatLonFromVec3(position vectors.Vec3, sphereRadius float64) (float64, float64) {
	// See https://stackoverflow.com/questions/46247499/vector3-to-latitude-longitude
	return RadToDeg(math.Asin(position.Z / sphereRadius)), // theta
		RadToDeg(math.Atan2(position.Y, position.X)) // phi
}

// Adds a vector to a latitude and longitude in degrees.
// The vector's x coordinate is modified by the cosine of the latitude to
// account for the fact that the distance between degrees of longitude
// decreases as the latitude increases.
func AddVecToLatLong(lat, lon float64, vec [2]float64) (float64, float64) {
	return lat + vec[1], lon + vec[0]/math.Cos(DegToRad(lat+vec[1]))
}

// CalcVecFromLatLong calculates the vector between two lat/long pairs.
func CalcVecFromLatLong(lat1, lon1, lat2, lon2 float64) [2]float64 {
	// Old implementation, which is wrong.
	// convert to radians
	// lat1 = DegToRad(lat1)
	// lon1 = DegToRad(lon1)
	// lat2 = DegToRad(lat2)
	// lon2 = DegToRad(lon2)
	//
	// 	return [2]float64{
	// 		math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(lon2-lon1), // X
	// 		math.Sin(lon2-lon1) * math.Cos(lat2),                                              // Y
	// 	}

	// Calculate the vector between two lat/long pairs using the bearing we calculate
	// from the lat/long pairs.
	// Note that the bearing is 0 when facing north, and increases clockwise, so we need to
	// convert it so that int is 0 when facing east, and increases counter-clockwise (and to radians).
	bearing := math.Pi/2 - CalcBearingRad(lat1, lon1, lat2, lon2)

	// Convert the bearing to a vector and scale it to the distance between the two points.
	dist := Haversine(lat1, lon1, lat2, lon2)
	return [2]float64{math.Cos(bearing) * dist, math.Sin(bearing) * dist}
}

// calcBearing calculates the bearing between two lat/long pairs.
func CalcBearingRad(lat1, lon1, lat2, lon2 float64) float64 {
	// convert to radians
	lat1 = DegToRad(lat1)
	lon1 = DegToRad(lon1)
	lat2 = DegToRad(lat2)
	lon2 = DegToRad(lon2)

	return math.Atan2(
		math.Sin(lon2-lon1)*math.Cos(lat2),                                              // y
		math.Cos(lat1)*math.Sin(lat2)-math.Sin(lat1)*math.Cos(lat2)*math.Cos(lon2-lon1), // x
	)
}

// Haversine returns the great arc distance between two lat/long pairs.
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	dLatSin := math.Sin(DegToRad(lat2-lat1) / 2)
	dLonSin := math.Sin(DegToRad(lon2-lon1) / 2)
	a := dLatSin*dLatSin + dLonSin*dLonSin*math.Cos(DegToRad(lat1))*math.Cos(DegToRad(lat2))
	return 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

// CrossArc calculates the shortest distance between an arc (defined by p1 and p2)
// and a third point, p3. The input is expected in degrees.
// See: https://stackoverflow.com/questions/32771458/distance-from-lat-lng-point-to-minor-arc-segment
func CrossArc(lat1, lon1, lat2, lon2, lat3, lon3 float64) float64 {
	// dis Finds the distance between two lat/lon points.
	dis := func(latA, lonA, latB, lonB float64) float64 {
		return math.Acos(math.Sin(latA)*math.Sin(latB) + math.Cos(latA)*math.Cos(latB)*math.Cos(lonB-lonA))
	}

	// bearing Finds the bearing from one lat/lon point to another.
	bearing := func(latA, lonA, latB, lonB float64) float64 {
		return math.Atan2(math.Sin(lonB-lonA)*math.Cos(latB), math.Cos(latA)*math.Sin(latB)-math.Sin(latA)*math.Cos(latB)*math.Cos(lonB-lonA))
	}

	lat1 = DegToRad(lat1)
	lat2 = DegToRad(lat2)
	lat3 = DegToRad(lat3)
	lon1 = DegToRad(lon1)
	lon2 = DegToRad(lon2)
	lon3 = DegToRad(lon3)

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
