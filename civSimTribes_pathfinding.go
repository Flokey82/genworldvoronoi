package genworldvoronoi

import (
	"math"

	goastar "github.com/beefsack/go-astar"
)

// wasVisited returns how often the path segment between the two given regions has been used.
func (s *simState) wasVisited(i, j int) int {
	return s.visitedPathSeg[getSegment(i, j)]
}

func (s *simState) getTile(i int) *MigrationTile {
	// Make sure we re-use pre-existing nodes.
	n, ok := s.nodeCache[i]
	if ok {
		return n
	}

	// If we have no cached node for this index,
	// create a new one.
	n = &MigrationTile{
		steepness:     s.steepness,
		r:             s.m,
		index:         i,
		getTile:       s.getTile,
		wasVisited:    s.wasVisited,
		maxElevation:  s.maxElevation,
		isCity:        s.isCity,
		tribeAtRegion: s.tribeAtRegion,
	}
	s.nodeCache[i] = n
	return n
}

// findNewPath will find a new path to the destination region and assign it to the tribe.
func (s *simState) findNewPath(t *Tribe, destination int) bool {
	newPath, found := planPath(s.getTile(t.RegionID), s.getTile(destination))
	if found {
		if len(newPath.Steps) == 1 {
			panic("path length is 1")
		}
		t.SetPath(newPath)
	}
	return found
}

type MigrationTile struct {
	r             *Civ                       // Reference to the Civ object
	getTile       func(i int) *MigrationTile // Fetch tiles from cache
	index         int                        // region index
	used          int                        // number of times this node was used for a trade route
	steepness     []float64                  // cached steepness of all regiones
	wasVisited    func(i, j int) int         // quick lookup if a segment was already visited
	isCity        map[int]bool               // quick lookup if an index is a city
	maxElevation  float64
	tribeAtRegion []*Tribe // mapping of tribes to region
}

func (n *MigrationTile) SetUsed() {
	n.used++
}

// PathNeighbors returns the direct neighboring nodes of this node which
// can be pathed to.
func (n *MigrationTile) PathNeighbors() []goastar.Pather {
	nbs := make([]goastar.Pather, 0, 6)
	for _, i := range n.r.GetRegNeighbors(n.index) {
		nbs = append(nbs, n.getTile(i))
	}
	return nbs
}

// PathNeighborCost calculates the exact movement cost to neighbor nodes.
func (n *MigrationTile) PathNeighborCost(to goastar.Pather) float64 {
	tot := to.(*MigrationTile)

	// Discourage underwater paths and occupied regions.
	// TODO: Make sure that tribeAtRegion is not infinite but just very high cost.
	if n.r.Elevation[n.index] <= 0 || n.r.Elevation[tot.index] <= 0 || n.tribeAtRegion[tot.index] != nil {
		return math.Inf(1)
	}

	// TODO: Fix this... this is highly inefficient.
	nIdx := tot.index

	// Altitude changes come with a cost (downhill is cheaper than uphill)
	cost := 1.0 + (n.r.Elevation[nIdx]-n.r.Elevation[n.index])/n.maxElevation

	// The steeper the terrain, the more expensive.
	cost *= 1.0 + n.steepness[nIdx]*n.steepness[nIdx]

	// Highly incentivize re-using used segments
	if nvis := n.wasVisited(n.index, nIdx); nvis > 0 {
		cost /= 8.0 * float64(nvis) * float64(nvis)
	} else {
		cost *= 8.0
	}

	// Heavily incentivize re-using existing roads.
	if nUsed := tot.used; nUsed > 0 {
		cost /= 8.0 * float64(nUsed) * float64(nUsed)
	} else {
		cost *= 8.0
	}

	// Penalty if the neighbor is a city.
	if n.isCity[nIdx] {
		cost *= 4.0
	}

	// Bonus if along coast.
	for _, nbnb := range n.r.GetRegNeighbors(nIdx) {
		if n.r.Elevation[nbnb] <= 0 {
			cost /= 2.0
			break
		}
	}

	// Bonus for moving along rivers.
	// TODO: This should depend on the culture of the tribe.
	// River cultures should not be penalized for crossing rivers.
	if n.r.isDownstream(n.index, nIdx) || n.r.isUpstream(n.index, nIdx) {
		cost /= 2.0
	} else if n.r.IsRegRiver(n.index) != n.r.IsRegRiver(nIdx) {
		cost *= 1.4 // Cost of crossing rivers.
	}

	// Penalty for crossing into a new territory
	if n.r.RegionToEmpire[n.index] != n.r.RegionToEmpire[nIdx] {
		cost *= 2.0
	}

	// Penalty for crossing into a new culture.
	if n.r.RegionToCulture[n.index] != n.r.RegionToCulture[nIdx] {
		cost *= 2.0
	}

	return cost
}

// PathEstimatedCost is a heuristic method for estimating movement costs
// between non-adjacent nodes.
func (n *MigrationTile) PathEstimatedCost(to goastar.Pather) float64 {
	return n.r.GetDistance(n.index, to.(*MigrationTile).index)
}

type Path struct {
	From, To int
	Steps    []int
	Idx      int
}

func planPath(from, to *MigrationTile) (*Path, bool) {
	path, _, found := goastar.Path(from, to)
	if !found {
		return nil, false
	}
	p := &Path{
		From: from.index,
		To:   to.index,
	}
	for _, n := range path {
		nti := n.(*MigrationTile)
		p.Steps = append(p.Steps, nti.index)
	}
	return p, true
}

// Peek returns the next region in the path without advancing the path.
func (p *Path) Peek() int {
	if p.Idx >= len(p.Steps) {
		return -1
	}
	return p.Steps[len(p.Steps)-p.Idx-1]
}

// Next returns the next region in the path and advances the path.
func (p *Path) Next() int {
	if p.Idx >= len(p.Steps) {
		return -1
	}
	// The path is reversed, so we need to go backwards.
	next := p.Steps[len(p.Steps)-p.Idx-1]
	p.Idx++
	return next
}

// PeekDone returns true if the path is done after the next step.
func (p *Path) PeekDone() bool {
	return p.Idx+1 >= len(p.Steps)
}

// Done returns true if the path is done.
func (p *Path) Done() bool {
	return p.Idx >= len(p.Steps)
}

// NumRemaining returns the number of steps remaining in the path.
func (p *Path) NumRemaining() int {
	return len(p.Steps) - p.Idx
}
