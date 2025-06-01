package civ

import (
	"log"
	"sort"

	"github.com/Flokey82/genworldvoronoi/various"
	goastar "github.com/beefsack/go-astar"
)

func (m *Civ) GetTradeRoutes() ([][]int, [][]int) {
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

	cities := m.Cities.Objects

	// linking will store which cities are linked through a trade route crossing
	// the given region.
	linking := make([][]int, m.SphereMesh.NumRegions)

	// visited will store which city pairs have already been visited.
	visited := make(map[[2]int]bool)

	// tribeMigrationCost returns the modified cost of migrating from one region to another and if the migration is possible.
	tradeMigrationCustomCost := func(from, to *NavTile, cost float64) (float64, bool) {
		// Bonus if the neighbor is a city.
		if m.Cities.GetIDAt(to.ID) != -1 {
			cost /= 4.0
		}

		// Penalty for crossing into a new territory
		if m.Empires.Regions[from.ID] != m.Empires.Regions[to.ID] {
			cost *= 2.0
		}
		return cost, true
	}

	nodeCache := NewNavCache(m.Geo, tradeMigrationCustomCost)

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

		var connections int
		for _, j := range sortCityIdx {
			if connections >= connectNClosest {
				break
			}
			// We don't want to link a city to itself and we try to avoid double
			// links (a->b and b->a) as well as we try to only connect towns within
			// the same territory.
			if i == j {
				continue
			}

			end := cities[j].ID
			curEdge := getSegment(start, end)
			if visited[curEdge] ||
				m.Empires.Regions[start] != m.Empires.Regions[end] ||
				m.Landmasses[start] != m.Landmasses[end] { //  || math.Abs(float64(i-j)) > float64(5)
				continue
			}
			connections++

			// Make sure we note that we have visited this city pair.
			visited[curEdge] = true

			// Attempt to find a path between the two cities.
			path, _, found := goastar.Path(nodeCache.GetTile(start), nodeCache.GetTile(end))
			if !found {
				continue
			}
			var newPath []int
			for idx, n := range path {
				// Mark the node as used.
				nti := n.(*NavTile)
				nti.SetUsed()
				nIdx := nti.ID
				if idx > 0 {
					nodeCache.visitedPathSeg[getSegment(newPath[idx-1], nIdx)]++
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

func (m *Civ) GetTradeRoutesInLatLonBB(minLat, minLon, maxLat, maxLon float64) [][]int {
	// Convert the trade route paths to segments.
	tr := m.TradeRoutes
	var links [][2]int
	seen := make(map[[2]int]bool)
	for _, path := range tr {
		for i := 0; i < len(path)-1; i++ {
			seg := getSegment(path[i], path[i+1])
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
	return various.MergeIndexSegments(filtered)
}
