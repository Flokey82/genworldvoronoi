package geo

// GeoConfig is a struct that holds all configuration options for the geography / geology / climate generation.
type GeoConfig struct {
	NumPlates          int     // Number of generated plates
	NumVolcanoes       int     // Number of generated volcanoes
	NumPoints          int     // Number of generated points / regions
	TectonicFalloff    bool    // Use square falloff to make mountains more peaky and flatlands more flat.
	NormalizeElevation bool    // Normalize elevation to 0-1 range
	MultiplyNoise      bool    // Multiply noise instead of adding
	Jitter             float64 // Jitter factor (randomness in point distribution)
}

// NewGeoConfig returns a new config for geography / geology / climate generation.
func NewGeoConfig() *GeoConfig {
	return &GeoConfig{
		NumPoints:          400000,
		NumPlates:          25,
		NumVolcanoes:       10,
		TectonicFalloff:    true,
		NormalizeElevation: true,
		MultiplyNoise:      true,
		Jitter:             0.0,
	}
}
