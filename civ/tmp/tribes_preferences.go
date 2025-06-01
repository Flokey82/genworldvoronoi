package civ

import (
	"math"
	"math/rand"

	"github.com/Flokey82/genworldvoronoi/geo"
)

type TribePreference struct {
	LastBiomes        *Last100[geo.HackyBiome] // The last 1000 biomes the tribe has been in.
	LastCultures      *Last100[CultureType]    // The last 1000 cultures the tribe has been in.
	RiverProximity    *RunningBool
	LakeProximity     *RunningBool
	OceanProximity    *RunningBool
	MountainProximity *RunningBool
}

func newTribePreference() TribePreference {
	return TribePreference{
		LastBiomes:        newLast100[geo.HackyBiome]("biome"),
		LastCultures:      newLast100[CultureType]("culture"),
		RiverProximity:    NewRunningBool(),
		LakeProximity:     NewRunningBool(),
		OceanProximity:    NewRunningBool(),
		MountainProximity: NewRunningBool(),
	}
}

func (t *TribePreference) String() string {
	var proxStr string
	if t.RiverProximity.Current() {
		proxStr += "R"
	} else {
		proxStr += "-"
	}
	if t.LakeProximity.Current() {
		proxStr += "L"
	} else {
		proxStr += "-"
	}
	if t.OceanProximity.Current() {
		proxStr += "O"
	} else {
		proxStr += "-"
	}
	if t.MountainProximity.Current() {
		proxStr += "M"
	} else {
		proxStr += "-"
	}
	return proxStr
}

// RandomReset resets some of the tribe preferences randomly.
func (t *TribePreference) RandomReset() {
	if rand.Intn(100) < 50 {
		t.LastBiomes.Reset()
	}
	if rand.Intn(100) < 50 {
		t.RiverProximity.Reset()
	}
	if rand.Intn(100) < 50 {
		t.LakeProximity.Reset()
	}
	if rand.Intn(100) < 50 {
		t.OceanProximity.Reset()
	}
	if rand.Intn(100) < 50 {
		t.MountainProximity.Reset()
	}
}

func (t *TribePreference) CopyTo(other *TribePreference) {
	t.LastBiomes.CopyTo(other.LastBiomes)
	t.LastCultures.CopyTo(other.LastCultures)
	t.RiverProximity.CopyTo(other.RiverProximity)
	t.LakeProximity.CopyTo(other.LakeProximity)
	t.OceanProximity.CopyTo(other.OceanProximity)
	t.MountainProximity.CopyTo(other.MountainProximity)
}

func (t *TribePreference) addRegionProp(curRegProp *geo.RegionProp) {
	// Set the current biome of the tribe and get the preferred biome of the tribe.
	t.LastBiomes.Add(curRegProp.Biome)

	// Set water proximity and get the preferred water proximity.
	t.RiverProximity.Add(curRegProp.River)

	// Set lake proximity and get the preferred lake proximity.
	t.LakeProximity.Add(curRegProp.Lake)

	// Set ocean proximity and get the preferred ocean proximity.
	t.OceanProximity.Add(curRegProp.Ocean)

	// Set mountain proximity and get the preferred mountain proximity.
	t.MountainProximity.Add(curRegProp.Mountain)
}

func (t *TribePreference) getPreferredRegionProp() *geo.RegionProp {
	preferredBiome, _ := t.LastBiomes.Preferred()
	return &geo.RegionProp{
		Biome: preferredBiome,
		RegionProximity: geo.RegionProximity{
			River:    t.RiverProximity.Current(),
			Lake:     t.LakeProximity.Current(),
			Ocean:    t.OceanProximity.Current(),
			Mountain: t.MountainProximity.Current(),
		},
	}
}

func (t *TribePreference) compare(other *TribePreference) float64 {
	// Compare preferences.
	var prefValue float64

	// TODO: Instead let's compare the aversion to the other tribe's preferred biome.
	tBiome, tScore := t.LastBiomes.Preferred()
	oBiome, oScore := other.LastBiomes.Preferred()
	if tBiome != oBiome {
		prefValue -= float64(tScore) + float64(oScore)
	}
	prefValue -= math.Abs(t.MountainProximity.avg - other.MountainProximity.avg)
	prefValue -= math.Abs(t.RiverProximity.avg - other.RiverProximity.avg)
	prefValue -= math.Abs(t.LakeProximity.avg - other.LakeProximity.avg)
	prefValue -= math.Abs(t.OceanProximity.avg - other.OceanProximity.avg)
	return prefValue
}

// Calculate the "attractiveness" multiplier for the region and the tribe.
// This will return a multiplier signifying how well the tribe can extract resources
// or value from the region.
func (t *TribePreference) compareSuitability(gotProp geo.RegionProp) float64 {
	wantProp := t.getPreferredRegionProp()
	multiplier := 1.0

	// Penalize if the biome doesn't match the preferred biome.
	if wantProp.Biome != gotProp.Biome {
		multiplier -= 0.2 * (1 - t.LastBiomes.GetScoreOf(gotProp.Biome))
	}

	// If there is a specific preference for the region, we penalize
	// if the region doesn't match the preference.
	if wantProp.River && !gotProp.River {
		multiplier -= 0.2 * t.RiverProximity.avg
	}
	if wantProp.Lake && !gotProp.Lake {
		multiplier -= 0.2 * t.LakeProximity.avg
	}
	if wantProp.Ocean && !gotProp.Ocean {
		multiplier -= 0.2 * t.OceanProximity.avg
	}
	if wantProp.Mountain && !gotProp.Mountain {
		multiplier -= 0.2
	}
	return multiplier
}
