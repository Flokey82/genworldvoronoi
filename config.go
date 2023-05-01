package genworldvoronoi

// Config is a struct that holds all configuration options for the map generation.
type Config struct {
	*GeoConfig
	*CivConfig
	*BioConfig
}

// NewConfig returns a new Config with default values.
func NewConfig() *Config {
	return &Config{
		GeoConfig: NewGeoConfig(),
		CivConfig: NewCivConfig(),
		BioConfig: NewBioConfig(),
	}
}

// GeoConfig is a struct that holds all configuration options for the geography / geology / climate generation.
type GeoConfig struct {
	NumPlates    int     // Number of generated plates
	NumVolcanoes int     // Number of generated volcanoes
	NumPoints    int     // Number of generated points / regions
	Jitter       float64 // Jitter factor (randomness in point distribution)
}

// NewGeoConfig returns a new config for geography / geology / climate generation.
func NewGeoConfig() *GeoConfig {
	return &GeoConfig{
		NumPoints:    400000,
		NumPlates:    25,
		NumVolcanoes: 10,
		Jitter:       0.0,
	}
}

// CivConfig is a struct that holds all configuration options for civilization generation.
type CivConfig struct {
	NumCultures              int  // (Min) Number of generated cultures
	NumOrganizedReligions    int  // (Min) Number of generated religions
	NumEmpires               int  // Number of generated territories
	NumCities                int  // Number of generated cities (regions)
	NumCityStates            int  // Number of generated city states
	NumMiningTowns           int  // Number of generated mining towns
	NumMiningGemsTowns       int  // Number of generated (gem) mining towns
	NumQuarryTowns           int  // Number of generated quarry towns
	NumFarmingTowns          int  // Number of generated farming towns
	NumTradingTowns          int  // Number of generated trading towns
	NumDesertOasis           int  // Number of generated desert oases
	EnableCityAging          bool // Enable city aging
	EnableOrganizedReligions bool // Enable organized religion generation
}

// NewCivConfig returns a new config for civilization generation.
func NewCivConfig() *CivConfig {
	return &CivConfig{
		NumCultures:              30,
		NumOrganizedReligions:    20,
		NumEmpires:               10,
		NumCities:                150,
		NumCityStates:            150,
		NumMiningTowns:           60,
		NumMiningGemsTowns:       60,
		NumQuarryTowns:           60,
		NumFarmingTowns:          60,
		NumTradingTowns:          0,
		NumDesertOasis:           10,
		EnableCityAging:          true,
		EnableOrganizedReligions: true,
	}
}

// BioConfig is a struct that holds all configuration options for biology generation.
type BioConfig struct {
	EnableRandomSpecies bool // Enable random species generation
	NumSpecies          int  // Number of randomly generated species
}

// NewBioConfig returns a new config for biology generation.
func NewBioConfig() *BioConfig {
	return &BioConfig{
		EnableRandomSpecies: false,
		NumSpecies:          100,
	}
}
