package civ2

import (
	"log"
)

const maxNAvg = 1000

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

// Last1000 is a counter that keeps track of the last 1000 values.
type Last1000[T ~int] struct {
	vals        [1000]T
	idx         int
	currentPref T
	label       string
}

func newLast1000[T ~int](label string) *Last1000[T] {
	return &Last1000[T]{
		label: label,
	}
}

func (l *Last1000[T]) Current() T {
	return l.vals[l.idx]
}

func (l *Last1000[T]) Add(v T) {
	l.idx = (l.idx + 1) % 1000
	l.vals[l.idx] = v
}

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
		if b == 0 {
			continue
		}
		c := entries[b]
		if c > maxCount {
			maxCount = c
			maxEntry = b
		}
	}

	if l.currentPref != maxEntry {
		log.Printf("Entity prefers now %v %s with a strength of %f", maxEntry, l.label, float64(maxCount)/1000)
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
