package geo

// getLakeBorders returns the borders of each lake (regions with same drainage region) as list of triangle center points.
func (m *Geo) GetLakeBorders() [][]int {
	return m.GetCustomBorders(m.Drainage)
}

// GetCustomBorders returns the borders/contours of all region in the supplied slice that have the same value.
func (m *Geo) GetCustomBorders(regionToID []int) [][]int {
	elevs := m.Elevation.GetValues()
	return m.GetCustomContour(func(idxA, idxB int) bool {
		if elevs[idxA] < 0 || elevs[idxB] < 0 ||
			(regionToID[idxA] < 0 && regionToID[idxB] < 0) {
			return false
		}
		return regionToID[idxA] != regionToID[idxB]
	})
}

// GetLandmassBorders returns the borders of each landmass (neighboring regions above sea level)
// as list of triangle center points.
func (m *Geo) GetLandmassBorders() [][]int {
	elevs := m.Elevation.GetValues()
	return m.GetCustomContour(func(idxA, idxB int) bool {
		return elevs[idxA] >= 0 && elevs[idxB] < 0 || elevs[idxA] < 0 && elevs[idxB] >= 0
	})
}
