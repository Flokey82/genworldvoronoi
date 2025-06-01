package genworldvoronoi

import (
	"github.com/Flokey82/genworldvoronoi/bio"
	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/genworldvoronoi/geo"
)

// Config is a struct that holds all configuration options for the map generation.
type Config struct {
	*geo.GeoConfig
	*civ.CivConfig
	*bio.BioConfig
}

// NewConfig returns a new Config with default values.
func NewConfig() *Config {
	return &Config{
		GeoConfig: geo.NewGeoConfig(),
		CivConfig: civ.NewCivConfig(),
		BioConfig: bio.NewBioConfig(),
	}
}
