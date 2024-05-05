package geo

import (
	"testing"
)

func TestCalculateSunPosition(t *testing.T) {
	for _, test := range []struct {
		lat       float64
		lon       float64
		dayOfYear int
		hour      float64
	}{{lat: 0.0, lon: 0.0, dayOfYear: 1, hour: 12.0}} {
		calculateSunPosition(test.lat, test.lon, 0, test.dayOfYear, test.hour)
	}
}
