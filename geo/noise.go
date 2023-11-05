package geo

// fbmNoiseCustom returns a function that returns the 'fractal bownian motion'-ish noise value for a given region.
func (m *Geo) fbmNoiseCustom(octaves int, persistence, mx, my, mz, dx, dy, dz float64) func(int) float64 {
	// https://thebookofshaders.com/13/
	return func(r int) float64 {
		nx, ny, nz := m.XYZ[3*r]*mx+dx, m.XYZ[3*r+1]*my+dy, m.XYZ[3*r+2]*mz+dz
		var sum, sumOfAmplitudes float64
		amplitude := 1.0
		for octave := 0; octave < octaves; octave++ {
			frequency := 1 << octave
			fFreq := float64(frequency)
			sum += amplitude * m.noise.OS.Eval3(nx*fFreq, ny*fFreq, nz*fFreq) * float64(octave)
			sumOfAmplitudes += amplitude * float64(octave)
			amplitude *= persistence
		}
		return (sum / sumOfAmplitudes)
	}
}

// genFbmNoise returns the 'fractal bownian motion'-ish noise value for each region.
func (m *Geo) genFbmNoise() []float64 {
	fn := m.fbmNoiseCustom(2, 1, 2, 2, 2, 0, 0, 0) // This should be a parameter.
	n := make([]float64, m.SphereMesh.NumRegions)
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		n[r] = fn(r)
	}
	return n
}

// getNoiseBandIntersection returns true if a noise value is within a band.
func getNoiseBandIntersection(noisevalue, bandvalue, bandwidth float64) bool {
	return bandvalue-bandwidth/2 <= noisevalue && noisevalue <= bandvalue+bandwidth/2
}
