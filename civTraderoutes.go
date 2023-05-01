package genworldvoronoi

import (
	"log"
	"math"
	"sort"

	goastar "github.com/beefsack/go-astar"
)

func (m *Civ) getTradeRoutes() ([][]int, [][]int) {
	// TODO: Allow persistent trading routes, so we can run multiple times without
	//       destroying existing routes.
	// Major cities will produce major trading routes that ensure that trade will be
	// as efficiently as possible. Minor cities will produce minor trading routes
	// that will connect them to the major trading routes.
	// Currently we only connect cities / settlement by proximity, which is not
	// how it works in reality. Of course along major trade routes, settlements
	// will experience growth through trade passing through, which is something
	// to consider later.
	log.Println("Generating trade routes...")
	nodeCache := make(map[int]*TradeTile)
	steepness := m.GetSteepness()

	cities := m.Cities
	isCity := make(map[int]bool)
	for _, c := range cities {
		isCity[c.ID] = true
	}

	_, maxElevation := minMax(m.Elevation)

	// linking will store which cities are linked through a trade route crossing
	// the given region.
	linking := make([][]int, m.SphereMesh.numRegions)

	// visited will store which city pairs have already been visited.
	visited := make(map[[2]int]bool)

	// visitedPathSeg will store how often path segments (neighboring regions connected by a trade route)
	// have been used
	visitedPathSeg := make(map[[2]int]int)

	// wasVisited returns how often the path segment between the two given regions has been used.
	var wasVisited func(i, j int) int
	wasVisited = func(i, j int) int {
		var seg [2]int
		if i < j {
			seg = [2]int{i, j}
		} else {
			seg = [2]int{j, i}
		}
		return visitedPathSeg[seg]
	}

	// getTile returns the TradeTile for the given index from the node cache.
	var getTile func(i int) *TradeTile
	getTile = func(i int) *TradeTile {
		// Make sure we re-use pre-existing nodes.
		n, ok := nodeCache[i]
		if ok {
			return n
		}

		// If we have no cached node for this index,
		// create a new one.
		n = &TradeTile{
			steepness:    steepness,
			r:            m,
			index:        i,
			getTile:      getTile,
			isCity:       isCity,
			wasVisited:   wasVisited,
			maxElevation: maxElevation,
		}
		nodeCache[i] = n
		return n
	}

	// Paths contains a list of all trade routes represented through
	// a list of connected regions.
	//
	// Note that we still double up if two trade routes happen to
	// share a common section leading up to a city.
	var paths [][]int

	// TODO: Pair up by import/export of goods and taxes to the capital.
	sortCityIdx := make([]int, len(cities))
	for i := range sortCityIdx {
		sortCityIdx[i] = i
	}

	connectNClosest := 5
	for i, startC := range cities {
		start := startC.ID
		// Sort by distance to start as we try to connect the closest towns first.
		// NOTE: Wouldn't it make sense to connect the largest cities first?
		sort.Slice(sortCityIdx, func(j, k int) bool {
			return m.GetDistance(start, cities[sortCityIdx[j]].ID) < m.GetDistance(start, cities[sortCityIdx[k]].ID)
		})
		for nidx, j := range sortCityIdx {
			if nidx >= connectNClosest {
				break
			}
			// We don't want to link a city to itself and we try to avoid double
			// links (a->b and b->a) as well as we try to only connect towns within
			// the same territory.
			if i == j {
				continue
			}

			var curEdge [2]int
			end := cities[j].ID
			if start < end {
				curEdge = [2]int{start, end}
			} else {
				curEdge = [2]int{end, start}
			}
			if visited[curEdge] ||
				m.RegionToEmpire[start] != m.RegionToEmpire[end] ||
				m.Landmasses[start] != m.Landmasses[end] { //  || math.Abs(float64(i-j)) > float64(5)
				continue
			}

			// Make sure we note that we have visited this city pair.
			visited[curEdge] = true

			// Attempt to find a path between the two cities.
			path, _, found := goastar.Path(getTile(start), getTile(end))
			if !found {
				continue
			}
			var newPath []int
			for idx, n := range path {
				// Mark the node as used.
				nti := n.(*TradeTile)
				nti.SetUsed()
				nIdx := nti.index
				if idx > 0 {
					var seg [2]int
					if nIdx < newPath[idx-1] {
						seg = [2]int{nIdx, newPath[idx-1]}
					} else {
						seg = [2]int{newPath[idx-1], nIdx}
					}
					visitedPathSeg[seg]++
				}

				// Check if the cities are already in our list for
				// the given region (aka "node index").
				if !isInIntList(linking[nIdx], start) {
					linking[nIdx] = append(linking[nIdx], start)
				}
				if !isInIntList(linking[nIdx], end) {
					linking[nIdx] = append(linking[nIdx], end)
				}

				// Append the region to the path.
				newPath = append(newPath, nIdx)
			}
			paths = append(paths, newPath)
		}
		log.Println("Done connecting city", i, "of", len(cities))
	}

	log.Println("Done generating trade routes.")
	return paths, linking
}

type TradeTile struct {
	r            *Civ                   // Reference to the Civ object
	getTile      func(i int) *TradeTile // Fetch tiles from cache
	index        int                    // region index
	used         int                    // number of times this node was used for a trade route
	steepness    []float64              // cached steepness of all regiones
	wasVisited   func(i, j int) int     // quick lookup if a segment was already visited
	isCity       map[int]bool           // quick lookup if an index is a city
	maxElevation float64
}

func (n *TradeTile) SetUsed() {
	n.used++
}

// PathNeighbors returns the direct neighboring nodes of this node which
// can be pathed to.
func (n *TradeTile) PathNeighbors() []goastar.Pather {
	nbs := make([]goastar.Pather, 0, 6)
	for _, i := range n.r.GetRegNeighbors(n.index) {
		nbs = append(nbs, n.getTile(i))
	}
	return nbs
}

// PathNeighborCost calculates the exact movement cost to neighbor nodes.
func (n *TradeTile) PathNeighborCost(to goastar.Pather) float64 {
	tot := to.(*TradeTile)

	// Discourage underwater paths.
	if n.r.Elevation[n.index] <= 0 || n.r.Elevation[tot.index] <= 0 {
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

	// Bonus if the neighbor is a city.
	if n.isCity[nIdx] {
		cost /= 4.0
	}

	// Bonus if along coast.
	for _, nbnb := range n.r.GetRegNeighbors(nIdx) {
		if n.r.Elevation[nbnb] <= 0 {
			cost /= 2.0
			break
		}
	}

	if n.r.isRegRiver(n.index) && n.r.isRegRiver(nIdx) {
		cost *= 0.8 // Bonus if along rivers.
	} else if n.r.isRegRiver(n.index) != n.r.isRegRiver(nIdx) {
		cost *= 1.4 // Cost of crossing rivers.
	}

	// Penalty for crossing into a new territory
	if n.r.RegionToEmpire[n.index] != n.r.RegionToEmpire[nIdx] {
		cost *= 2.0
	}

	return cost
}

// PathEstimatedCost is a heuristic method for estimating movement costs
// between non-adjacent nodes.
func (n *TradeTile) PathEstimatedCost(to goastar.Pather) float64 {
	return n.r.GetDistance(n.index, to.(*TradeTile).index)
}

func (m *Civ) getTradeRoutesInLatLonBB(minLat, minLon, maxLat, maxLon float64) [][]int {
	// Convert the trade route paths to segments.
	tr := m.tradeRoutes
	var links [][2]int
	seen := make(map[[2]int]bool)
	for _, path := range tr {
		for i := 0; i < len(path)-1; i++ {
			a := path[i]
			b := path[i+1]
			var seg [2]int
			if a > b {
				seg[0] = b
				seg[1] = a
			} else {
				seg[0] = a
				seg[1] = b
			}
			if seen[seg] {
				continue
			}
			seen[seg] = true
			links = append(links, seg)
		}
	}

	filter := false

	// Find the segments that are within the bounding box.
	var filtered [][2]int
	for _, link := range links {
		if filter {
			lat1, lon1 := m.LatLon[link[0]][0], m.LatLon[link[0]][1]
			lat2, lon2 := m.LatLon[link[1]][0], m.LatLon[link[1]][1]

			// If both points are outside the bounding box, skip the segment.
			if (lat1 < minLat || lat1 > maxLat || lon1 < minLon || lon1 > maxLon) &&
				(lat2 < minLat || lat2 > maxLat || lon2 < minLon || lon2 > maxLon) {
				continue
			}
		}
		filtered = append(filtered, link)
	}
	return mergeIndexSegments(filtered)
}
