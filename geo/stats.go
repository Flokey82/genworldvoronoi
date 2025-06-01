package geo

import (
	"log"

	"github.com/Flokey82/genbiome"
	"github.com/Flokey82/go_gens/gameconstants"
)

// Stats holds aggregated statistics about a number of regions.
type Stats struct {
	NumRegions int
	Resouces   map[*Resource]int
	TotalArea  float64
	Biomes     map[int]int
	Desert     int
	Forest     int
	RainForest int
	Snow       int
	Swamp      int
	Wetlands   int
	Rivers     int
	Coastal    int
}

// NewStats returns a new Stats object.
func NewStats() *Stats {
	return &Stats{
		Biomes:   make(map[int]int),
		Resouces: make(map[*Resource]int),
	}
}

func (m *Geo) GetStats(rr []int) *Stats {
	st := NewStats()
	st.NumRegions = len(rr)

	outRegs := make([]int, 0, 6)
	biomeFunc := m.GetRegWhittakerModBiomeFunc()
	for _, res := range m.Resources {
		for _, r := range rr {
			if m.Location[res][r] {
				st.Resouces[res]++
			}
		}
	}
	for _, r := range rr {
		st.TotalArea += m.GetRegArea(r, outRegs)
		b := biomeFunc(r)
		st.Biomes[b]++

		switch b {
		case genbiome.WhittakerModBiomeColdDesert, genbiome.WhittakerBiomeSubtropicalDesert:
			st.Desert++
		case genbiome.WhittakerModBiomeTropicalRainforest,
			genbiome.WhittakerModBiomeTemperateRainforest:
			st.RainForest++
		case genbiome.WhittakerModBiomeTropicalSeasonalForest,
			genbiome.WhittakerModBiomeTemperateSeasonalForest:
			st.Forest++
		case genbiome.WhittakerModBiomeSnow:
			st.Snow++
		case genbiome.WhittakerModBiomeHotSwamp:
			st.Swamp++
		case genbiome.WhittakerModBiomeWetlands:
			st.Wetlands++
		}
		// Check if we border to a waterbody.
		for _, nb := range m.R_circulate_r(outRegs, r) {
			if m.IsRegLakeOrWaterBody(nb) {
				st.Coastal++
				break
			}
		}

		// Check if we have a river.
		if m.IsRegRiver(r) {
			st.Rivers++
		}
	}
	return st
}

func (s *Stats) Log() {
	log.Printf("Total Area: %.2f km2", s.TotalArea*gameconstants.EarthSurface/gameconstants.SphereSurface)
	for res, n := range s.Resouces {
		log.Printf("Resource: %s: %d (%.3f%%)", res.Name, n, 100*float64(n)/float64(s.NumRegions))
	}
	log.Printf("Desert: %.2f%%", 100*float64(s.Desert)/float64(s.NumRegions))
	log.Printf("RainForest: %.2f%%", 100*float64(s.RainForest)/float64(s.NumRegions))
	log.Printf("Forest: %.2f%%", 100*float64(s.Forest)/float64(s.NumRegions))
	log.Printf("Snow: %.2f%%", 100*float64(s.Snow)/float64(s.NumRegions))
	log.Printf("Swamp: %.2f%%", 100*float64(s.Swamp)/float64(s.NumRegions))
	log.Printf("Wetlands: %.2f%%", 100*float64(s.Wetlands)/float64(s.NumRegions))
	log.Printf("Rivers: %.2f%%", 100*float64(s.Rivers)/float64(s.NumRegions))
	log.Printf("Coastal: %.2f%%", 100*float64(s.Coastal)/float64(s.NumRegions))
}
