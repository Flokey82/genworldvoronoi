package main

import (
	"flag"
	"log"
	"os"

	"github.com/Flokey82/genworldvoronoi"
	"github.com/Flokey82/genworldvoronoi/cmd2/maptiles"
	"github.com/hajimehoshi/ebiten"
)

var (
	seed                    int64   = 12345
	numPlates               int     = 25
	oceanPlatesFraction     float64 = 0.65
	oceanPlatesAltSelection bool    = true
	numPoints               int     = 400000
	numVolcanoes            int     = 10
	jitter                  float64 = 0.0
)

func init() {
	flag.Int64Var(&seed, "seed", seed, "the world seed")
	flag.IntVar(&numPlates, "num_plates", numPlates, "number of plates")
	flag.IntVar(&numPoints, "num_points", numPoints, "number of points")
	flag.IntVar(&numVolcanoes, "num_volcanoes", numVolcanoes, "number of volcanoes")
	flag.Float64Var(&jitter, "jitter", jitter, "jitter")
}

func main() {
	// Generate a new world.
	flag.Parse()

	// Initialize the config.
	cfg := genworldvoronoi.NewConfig()
	cfg.GeoConfig.NumPlates = numPlates
	cfg.GeoConfig.OceanPlatesFraction = oceanPlatesFraction
	cfg.GeoConfig.OceanPlatesAltSelection = oceanPlatesAltSelection
	cfg.GeoConfig.NumPoints = numPoints
	cfg.GeoConfig.NumVolcanoes = numVolcanoes
	cfg.GeoConfig.Jitter = jitter

	sp, err := genworldvoronoi.NewMapFromConfig(seed, cfg)
	if err != nil {
		log.Fatal(err)
	}

	g, err := maptiles.NewGame(sp)
	if err != nil {
		os.Exit(1)
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
