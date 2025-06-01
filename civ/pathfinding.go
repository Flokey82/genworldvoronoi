package civ

import (
	"math"

	"github.com/Flokey82/genworldvoronoi/geo"
	goastar "github.com/beefsack/go-astar"
)

// CustomTraverseCostFunc is a function that can be used to customize the cost of moving between two regions.
// The function should return the new cost and a boolean indicating if the traversal is possible.
type CustomTraverseCostFunc func(from, to *NavTile, score float64) (float64, bool)

type NavCache struct {
	m              *geo.Geo               // Reference to the civilization
	tiles          map[int]*NavTile       // reusable path nodes
	visitedPathSeg map[[2]int]int         // path segment visited count
	customCost     CustomTraverseCostFunc // custom cost function
}

func NewNavCache(m *geo.Geo, customCost CustomTraverseCostFunc) *NavCache {
	return &NavCache{
		m:              m,
		tiles:          make(map[int]*NavTile),
		visitedPathSeg: make(map[[2]int]int),
		customCost:     customCost,
	}
}

// wasVisited returns how often the path segment between the two given regions has been used.
func (s *NavCache) wasVisited(i, j int) int {
	return s.visitedPathSeg[getSegment(i, j)]
}

func (s *NavCache) GetTile(r int) *NavTile {
	// Make sure we re-use pre-existing nodes.
	n, ok := s.tiles[r]
	if ok {
		return n
	}

	// If we have no cached node for this index,
	// create a new one.
	n = &NavTile{
		ID:       r,
		NavCache: s,
	}
	s.tiles[r] = n
	return n
}

type NavTile struct {
	ID        int // region index
	used      int // number of times this node was used for a trade route
	*NavCache     // Reference to the cache
}

func (n *NavTile) SetUsed() {
	n.used++
}

// PathNeighbors returns the direct neighboring nodes of this node which
// can be pathed to.
func (n *NavTile) PathNeighbors() []goastar.Pather {
	nbs := make([]goastar.Pather, 0, 6)
	for _, i := range n.m.GetRegNeighbors(n.ID) {
		nbs = append(nbs, n.GetTile(i))
	}
	return nbs
}

// baseNeighborCost calculates the base cost of moving to a neighbor and if it is possible.
func (n *NavTile) baseNeighborCost(to *NavTile) (float64, bool) {
	// Discourage underwater paths.
	if n.m.Elevation.Values[n.ID] <= 0 || n.m.Elevation.Values[to.ID] <= 0 {
		return math.Inf(1), false
	}

	// Altitude changes come with a cost (downhill is cheaper than uphill)
	cost := 1.0 + (n.m.Elevation.Values[to.ID]-n.m.Elevation.Values[n.ID])/n.m.Elevation.Max

	// The steeper the terrain, the more expensive.
	if st := n.m.Steepness.Values[to.ID]; st > 0.0 {
		cost *= 1.0 + st*st
	}

	if nvis := n.wasVisited(n.ID, to.ID); nvis > 0 {
		// Highly incentivize re-using used segments
		cost /= 8.0 * float64(nvis*nvis)
	} else if n.m.IsDownstream(n.ID, to.ID) || n.m.IsUpstream(n.ID, to.ID) {
		// Bonus for moving along rivers.
		// TODO: This should depend on the culture of the tribe.
		// River cultures should not be penalized for crossing rivers.
		cost /= 2.0
	}

	/*
		if nUsed := to.used; nUsed > 0 {
			// Incentivize re-using existing roads.
			cost /= 4.0 * float64(nUsed*nUsed)
		} else {
			cost *= 4.0
		}
	*/

	// Bonus if along coast.
	for _, nbnb := range n.m.GetRegNeighbors(to.ID) {
		if n.m.IsRegLakeOrWaterBody(nbnb) {
			cost /= 2.0
			break
		}
	}

	return cost, true
}

// PathNeighborCost calculates the exact movement cost to neighbor nodes.
func (n *NavTile) PathNeighborCost(to goastar.Pather) float64 {
	tot := to.(*NavTile)

	cost, ok := n.baseNeighborCost(tot)
	if !ok {
		return math.Inf(1)
	}

	cost, ok = n.customCost(n, tot, cost)
	if !ok {
		return math.Inf(1)
	}

	return cost
}

// PathEstimatedCost is a heuristic method for estimating movement costs
// between non-adjacent nodes.
func (n *NavTile) PathEstimatedCost(to goastar.Pather) float64 {
	return n.m.GetDistance(n.ID, to.(*NavTile).ID)
}

type Path struct {
	From, To int
	Steps    []int
	Idx      int
}

func PlanPath(from, to *NavTile) (*Path, bool) {
	// Find the path.
	path, _, found := goastar.Path(from, to)
	if !found {
		return nil, false
	}

	// Create a new path.
	p := &Path{
		From: from.ID,
		To:   to.ID,
	}

	// Convert the path to a list of region indices.
	// TODO: Maybe just reverse the path right away?
	for _, n := range path {
		nti := n.(*NavTile)
		p.Steps = append(p.Steps, nti.ID)
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

func getSegment(a, b int) [2]int {
	if a < b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}
