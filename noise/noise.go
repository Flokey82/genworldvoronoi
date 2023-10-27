package noise

import (
	"math"

	"github.com/ojrac/opensimplex-go"
)

// Noise is a wrapper for opensimplex.Noise, initialized with
// a given seed, persistence, and number of octaves.
type Noise struct {
	Octaves     int
	Persistence float64
	Amplitudes  []float64
	Seed        int64
	OS          opensimplex.Noise
}

// NewNoise returns a new Noise.
func NewNoise(octaves int, persistence float64, seed int64) *Noise {
	n := &Noise{
		Octaves:     octaves,
		Persistence: persistence,
		Amplitudes:  make([]float64, octaves),
		Seed:        seed,
		OS:          opensimplex.NewNormalized(seed),
	}

	// Initialize the amplitudes.
	for i := range n.Amplitudes {
		n.Amplitudes[i] = math.Pow(persistence, float64(i))
	}

	return n
}

// Eval3 returns the noise value at the given point.
func (n *Noise) Eval3(x, y, z float64) float64 {
	var sum, sumOfAmplitudes float64
	for octave := 0; octave < n.Octaves; octave++ {
		frequency := 1 << octave
		fFreq := float64(frequency)
		sum += n.Amplitudes[octave] * n.OS.Eval3(x*fFreq, y*fFreq, z*fFreq)
		sumOfAmplitudes += n.Amplitudes[octave]
	}
	return sum / sumOfAmplitudes
}

// Eval2 returns the noise value at the given point.
func (n *Noise) Eval2(x, y float64) float64 {
	var sum, sumOfAmplitudes float64
	for octave := 0; octave < n.Octaves; octave++ {
		frequency := 1 << octave
		fFreq := float64(frequency)
		sum += n.Amplitudes[octave] * n.OS.Eval2(x*fFreq, y*fFreq)
		sumOfAmplitudes += n.Amplitudes[octave]
	}
	return sum / sumOfAmplitudes
}

// PlusOneOctave returns a new Noise with one more octave.
func (n *Noise) PlusOneOctave() *Noise {
	return NewNoise(n.Octaves+1, n.Persistence, n.Seed)
}
