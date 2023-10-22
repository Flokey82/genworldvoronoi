package various

import "math"

// RoundToDecimals rounds the given float to the given number of decimals.
func RoundToDecimals(v, d float64) float64 {
	m := math.Pow(10, d)
	return math.Round(v*m) / m
}
