package genworldvoronoi

type TriangleMesh struct {
	RegInSide            []int
	Triangles            []int
	Halfedges            []int
	RegionNeighborsCache [][]int
	numSides             int
	numRegions           int
	numTriangles         int
	numHalfedges         int
}

// NewTriangleMesh takes partial mesh information and fills in the rest; the
// partial information is generated in create.js or in fromDelaunator.
func NewTriangleMesh(numRegions int, tris, halfEdges []int) *TriangleMesh {
	// Update internal data structures to match the input mesh.
	tm := &TriangleMesh{
		Triangles:            tris,
		Halfedges:            halfEdges,
		RegionNeighborsCache: make([][]int, numRegions),
		numRegions:           numRegions,
		numSides:             len(tris),
		numTriangles:         len(tris) / 3,
		numHalfedges:         len(halfEdges),
	}

	// Construct an index for finding sides connected to a region
	//
	// Use if you have updated the triangles/halfedges with Delaunator
	// and want the dual mesh to match the updated data. Note that
	// this DOES not update boundary regions or ghost elements.
	tm.RegInSide = make([]int, tm.numRegions)
	for s := 0; s < len(tm.Triangles); s++ {
		endpoint := tm.Triangles[s_next_s(s)]
		if tm.RegInSide[endpoint] == 0 || tm.Halfedges[s] == -1 {
			tm.RegInSide[endpoint] = s
		}
	}

	// Cache the neighbors for each region and return.
	for r := 0; r < tm.numRegions; r++ {
		tm.RegionNeighborsCache[r] = tm.r_circulate_r_no_cache(nil, r)
	}
	return tm
}

func s_to_t(s int) int {
	return (s / 3)
}

func s_prev_s(s int) int {
	if s%3 == 0 {
		return s + 2
	}
	return s - 1
}

func s_next_s(s int) int {
	if s%3 == 2 {
		return s - 2
	}
	return s + 1
}

// r_circulate_r returns the regions adjacent to r using the cached
// neighbors. This is faster than r_circulate_r_no_cache, but it
// requires that the cached neighbors are up to date.
func (tm *TriangleMesh) r_circulate_r(out_r []int, r int) []int {
	out_r = out_r[:0]
	for _, rn := range tm.RegionNeighborsCache[r] {
		out_r = append(out_r, rn)
	}
	return out_r
}

// r_circulate_r_no_cache returns the regions adjacent to r without
// using the cached neighbors. This is slower than r_circulate_r, but
// it does not require that the cached neighbors are up to date.
func (tm *TriangleMesh) r_circulate_r_no_cache(out_r []int, r int) []int {
	s0 := tm.RegInSide[r]
	incoming := s0
	out_r = out_r[:0]
	for {
		out_r = append(out_r, tm.s_begin_r(incoming))
		outgoing := s_next_s(incoming)
		incoming = tm.Halfedges[outgoing]
		if incoming == -1 || incoming == s0 {
			break
		}
	}
	return out_r
}

func (tm *TriangleMesh) r_circulate_t(out_t []int, r int) []int {
	s0 := tm.RegInSide[r]
	incoming := s0
	out_t = out_t[:0]
	for {
		out_t = append(out_t, s_to_t(incoming))
		outgoing := s_next_s(incoming)
		incoming = tm.Halfedges[outgoing]
		if incoming == -1 || incoming == s0 {
			break
		}
	}
	return out_t
}

func (tm *TriangleMesh) t_circulate_s(out_s []int, t int) []int {
	out_s = out_s[:0]
	for i := 0; i < 3; i++ {
		out_s = append(out_s, 3*t+i)
	}
	return out_s
}

func (tm *TriangleMesh) t_circulate_r(out_r []int, t int) []int {
	out_r = out_r[:0]
	for i := 0; i < 3; i++ {
		out_r = append(out_r, tm.Triangles[3*t+i])
	}
	return out_r
}

func (tm *TriangleMesh) s_end_r(s int) int {
	return tm.Triangles[s_next_s(s)]
}

func (tm *TriangleMesh) s_begin_r(s int) int {
	return tm.Triangles[s]
}

func (tm *TriangleMesh) s_opposite_s(s int) int {
	return tm.Halfedges[s]
}

func (tm *TriangleMesh) s_inner_t(s int) int {
	return s_to_t(s)
}

func (tm *TriangleMesh) s_outer_t(s int) int {
	return s_to_t(tm.Halfedges[s])
}
