package bio

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
