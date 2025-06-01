package civ

import (
	"log"
	"math"
)

func (m *Civ) InitSim() {
	// TODO:
	// - Initialize the population per region to 0.
	// - Calculate the suitability of each region for population growth.
	//   - We will use this to calculate the population growth rate for each region.
	// - Continue to expand the civilization from the cradle of civilization.
	// - Have a bunch of clans, and then they can start to form tribes.
	//   - Tribes are like "moving cities/settlements" or "nomadic cities/settlements".
	//   - They can move around and settle in different regions that are suitable for them.
	//   - They can also fight with other tribes, and form alliances.
	//   - They can only move to unoccupied regions, or regions that are not too densely populated.
	//   - They can also trade with other tribes.
	//   - They might be able to form a confederation of tribes.
	//   - At some point they might decide to settle down and form a city.
	// - Once we reach a certain population, we can start to form cities.

	// We have two options:
	// - Per-region population growth rate and create tribes later.
	// - Tribes that move around and settle in different regions.

	// Here's a draft of the first option:
	// Calculate the suitability of each region for population growth.
	// We get a fitness function for each region.
	m.InitSimSettling()
	m.InitSimTribes()
	rNbs := make([]int, 0, 6)
	numYears := 3000
	for i := 0; i < numYears; i++ {
		m.PeopleToRegion.NeedsUpdate = true
		log.Printf("Year: %d\n", m.Geo.Calendar.GetYear())
		m.tickSimSettlement(rNbs)
		m.tickSimTribes()
		m.Geo.Calendar.TickYear()
	}
	log.Println(len(m.Tribes.Objects), "tribes formed.")

	for i, s := range m.Suitability {
		if math.IsNaN(s) {
			panic("NaN suitability")
		}
		if s > 1000 {
			m.Suitability[i] = 0
		}
	}
}

func (m *Civ) TickSim() {
	log.Printf("Year: %d\n", m.Geo.Calendar.GetYear())
	m.Geo.Calendar.TickYear()
	m.tickSimTribes()
	m.tickSimSettlement(rNbs)
}

func (m *Civ) calculateSuitability() []float64 {
	cf := m.Geo.GetFitnessSurviability()
	elevs := m.Elevation.GetValues()
	rNbs := make([]int, 0, 6)
	for i := 0; i < m.NumRegions; i++ {
		if elevs[i] <= 0 || i == m.NumRegions { // The pole has a excessive area.
			// Ocean regions are not suitable for population growth.
			m.Suitability[i] = 0
			continue
		}
		// Get the area of the region (which is the area on a unit sphere).
		regArea := m.GetRegArea(i, rNbs)

		// Now scale it up to earth's surface area.
		// We will use this to calculate the maximum population for each region.
		regArea *= 6371 * 6371 // This will give us the area in km^2.

		// NOTE: The (preferred) population density changes over time.

		// Calculate the maximum population for each region.
		m.Suitability[i] = regArea * cf(i)
	}

	// Normalize the suitability.
	return m.Suitability
}

const popDensity = 10 // 10 people per km^2

func (m *Civ) maxPopReg(i int) int {
	return int(m.Suitability[i] * popDensity)
}

func dedupInts(ints []int) []int {
	seen := make(map[int]bool)
	j := 0
	for _, i := range ints {
		if seen[i] {
			continue
		}
		seen[i] = true
		ints[j] = i
		j++
	}
	return ints[:j]
}

const maxNAvg = 1000

type RunningBool struct {
	RunningAverage
}

func NewRunningBool() *RunningBool {
	return &RunningBool{}
}

func (ra *RunningBool) Add(v bool) {
	if v {
		ra.RunningAverage.Add(1)
	} else {
		ra.RunningAverage.Add(0)
	}
}

func (ra *RunningBool) CopyTo(other *RunningBool) {
	ra.RunningAverage.CopyTo(&other.RunningAverage)
}

func (ra *RunningBool) Current() bool {
	return ra.avg > 0.5
}

func (ra *RunningBool) Reset() {
	ra.RunningAverage.Reset()
}

// RunningaverageLimit is a running average that limits the number of historical
// values that are taken into account.
type RunningAverageLimit struct {
	RunningAverage
	limit int
}

// NewRunningAverageLimit creates a new running average with a limit.
func NewRunningAverageLimit(limit int) *RunningAverageLimit {
	return &RunningAverageLimit{
		limit: limit,
	}
}

// Add adds a new value to the running average.
func (ra *RunningAverageLimit) Add(v float64) {
	ra.RunningAverage.Add(v)
	if ra.n > ra.limit {
		ra.n = ra.limit
	}
}

// CopyTo copies the running average to the other running average.
func (ra *RunningAverageLimit) CopyTo(other *RunningAverageLimit) {
	ra.RunningAverage.CopyTo(&other.RunningAverage)
	other.limit = ra.limit
}

// RunningAverage is a running average that takes the historical values into account.
type RunningAverage struct {
	avg float64
	n   int
}

// Add adds a new value to the running average.
func (ra *RunningAverage) Add(v float64) {
	ra.avg = (ra.avg*float64(ra.n) + v) / float64(ra.n+1)
	ra.n++
	if ra.n > maxNAvg {
		ra.n = maxNAvg
	}
}

// CopyTo copies the running average to the other running average.
func (ra *RunningAverage) CopyTo(other *RunningAverage) {
	other.avg = ra.avg
	other.n = ra.n
}

// Current returns the current average.
func (ra *RunningAverage) Current() float64 {
	return ra.avg
}

// Reset resets the running average.
func (ra *RunningAverage) Reset() {
	ra.avg = 0
	ra.n = 0
}

type Last1000[T ~int] struct {
	vals        [1000]T
	idx         int
	currentPref T
	label       string
}

func newLast1000[T ~int](label string) *Last1000[T] {
	var vals [1000]T
	for i := range vals {
		vals[i] = 0 // Initialize with a generic zero value for type T
	}
	return &Last1000[T]{
		vals:        vals,
		idx:         0,
		currentPref: 0, // Use zero value for T
		label:       label,
	}
}

func (l *Last1000[T]) Current() T {
	return l.vals[l.idx]
}

func (l *Last1000[T]) Add(v T) {
	l.idx = (l.idx + 1) % 1000
	l.vals[l.idx] = v
}

// getScoreOf returns the score of the given value in the last 1000 values.
func (l *Last1000[T]) GetScoreOf(v T) float64 {
	entries := make(map[T]int)
	for _, b := range l.vals {
		entries[b]++
	}
	return float64(entries[v]) / 1000
}

func (l *Last1000[T]) Preferred() (T, float64) {
	entries := make(map[T]int)
	for _, b := range l.vals {
		entries[b]++
	}
	var maxEntry T
	var maxCount int
	for i := 0; i < 1000; i++ {
		b := l.vals[(l.idx+1000-i)%1000]
		if b == 0 { // Check for generic zero value
			continue
		}
		c := entries[b]
		if c > maxCount {
			maxCount = c
			maxEntry = b
		}
	}

	if l.currentPref != maxEntry {
		log.Printf("Tribe prefers now %v %s with a strength of %f", maxEntry, l.label, float64(maxCount)/1000)
		l.currentPref = maxEntry
	}

	return maxEntry, float64(maxCount) / 1000
}

func (l *Last1000[T]) CopyTo(other *Last1000[T]) {
	other.vals = l.vals
	other.idx = l.idx
	other.currentPref = l.currentPref
}

// Reset resets the counter.
func (l *Last1000[T]) Reset() {
	l.idx = 0
	l.currentPref = 0
}

// Last100 is a counter that keeps track of the last 100 values.
type Last100[T ~int] struct {
	vals        [100]T
	idx         int
	currentPref T
	label       string
}

// NewLast100 creates a new counter that keeps track of the last 100 values.
func newLast100[T ~int](label string) *Last100[T] {
	var vals [100]T
	for i := range vals {
		vals[i] = 0 // Initialize with a generic zero value for type T
	}
	return &Last100[T]{
		vals:        vals,
		idx:         0,
		currentPref: 0, // Use zero value for T
		label:       label,
	}
}

// Current returns the current value.
func (l *Last100[T]) Current() T {
	return l.vals[l.idx]
}

// Add adds a new value to the counter.
func (l *Last100[T]) Add(v T) {
	l.idx = (l.idx + 1) % 100
	l.vals[l.idx] = v
}

// GetScoreOf returns the score of the given value in the last 100 values.
func (l *Last100[T]) GetScoreOf(v T) float64 {
	entries := make(map[T]int)
	for _, b := range l.vals {
		entries[b]++
	}
	return float64(entries[v]) / 100
}

// Preferred returns the preferred value and the score of the preferred value.
func (l *Last100[T]) Preferred() (T, float64) {
	entries := make(map[T]int)
	for _, b := range l.vals {
		entries[b]++
	}
	var maxEntry T
	var maxCount int
	for i := 0; i < 100; i++ {
		b := l.vals[(l.idx+100-i)%100]
		if b == 0 { // Check for generic zero value
			continue
		}
		c := entries[b]
		if c > maxCount {
			maxCount = c
			maxEntry = b
		}
	}

	if l.currentPref != maxEntry {
		log.Printf("Tribe prefers now %v %s with a strength of %f", maxEntry, l.label, float64(maxCount)/100)
		l.currentPref = maxEntry
	}

	return maxEntry, float64(maxCount) / 100
}

// CopyTo copies the counter to the other counter.
func (l *Last100[T]) CopyTo(other *Last100[T]) {
	other.vals = l.vals
	other.idx = l.idx
	other.currentPref = l.currentPref
}

// Reset resets the counter.
func (l *Last100[T]) Reset() {
	l.idx = 0
	l.currentPref = 0
}
