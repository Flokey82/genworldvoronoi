package genworldvoronoi

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"

	"github.com/Flokey82/geoquad"
)

func (m *Map) GetHeightMapTile(x, y, zoom int) []byte {
	// The tiles are 65x65 vertices and overlap their neighbors at their edges.
	// In other words, at the root, the eastern-most column of heights in the western
	// tile is identical to the western-most column of heights in the eastern tile.

	// The first and most important part of the file is a simple array of 16-bit,
	// little-endian, integer heights arranged from north to south and from west to east
	// - the first 2 bytes are the height in the northwest corner, and the next 2 bytes
	// are the height at the location just to the east of there.
	// Each height is the number of 1/5 meter units above -1000 meters.
	// The total size of the post data is 65 * 65 * 2 = 8450 bytes.

	// Wrap the tile coordinates.
	height := 256
	width := 256

	// Initialize the height map.
	heightMap := make([]uint16, height*width)

	// Wrap the tile coordinates.
	x, y = wrapTileCoordinates(x, y, zoom)
	// Calculate the bounds of the tile.
	tbb := newTileBoundingBox(x, y, zoom)

	// get the width and height of the tile in lat long.
	la1, lo1, la2, lo2 := tbb.toLatLon()
	tbbWidth := lo2 - lo1
	tbbHeight := la2 - la1

	calcHeightMercator := func(p1, p2, p3, p [2]float64, z1, z2, z3 float64) float64 {
		// Convert the points to mercator.
		p1x, p1y := latLonToPixels(p1[0], p1[1], zoom)
		p2x, p2y := latLonToPixels(p2[0], p2[1], zoom)
		p3x, p3y := latLonToPixels(p3[0], p3[1], zoom)
		px, py := latLonToPixels(p[0], p[1], zoom)

		return calcHeightInTriangle([2]float64{p1x, p1y}, [2]float64{p2x, p2y}, [2]float64{p3x, p3y}, [2]float64{px, py}, z1, z2, z3)
	}

	// inTriangleMercator uses the mercator projection to determine if a point is in a triangle.
	inTriangleMercator := func(p1, p2, p3, p [2]float64) bool {
		// Convert the points to mercator.
		p1x, p1y := latLonToPixels(p1[0], p1[1], zoom)
		p2x, p2y := latLonToPixels(p2[0], p2[1], zoom)
		p3x, p3y := latLonToPixels(p3[0], p3[1], zoom)
		px, py := latLonToPixels(p[0], p[1], zoom)

		// Now we can use the regular inTriangle function.
		return isPointInTriangle([2]float64{p1x, p1y}, [2]float64{p2x, p2y}, [2]float64{p3x, p3y}, [2]float64{px, py})
	}

	outTri := make([]int, 0, 7)
	outRegs := make([]int, 0, 7)
	var regs []int
	getHeight := func(latlon [2]float64) float64 {
		// TODO: We should use a quadtree to find the closest region.
		// Then we circulate the region to find the triangle that
		// contains the point.
		minDistIndex := -1

		closestReg, ok := m.regQuadTree.FindNearestNeighbor(geoquad.Point{Lat: latlon[0], Lon: latlon[1]})
		if !ok {
			log.Println("no closest point found")
			// Find the closest triangle center.
			for tri := 0; tri < m.SphereMesh.numTriangles; tri++ {
				//for _, tri := range tris {
				// for speed, we just use the euclidean distance here.
				var dist float64
				//if len(tris) > 1000 {
				// convert to radians
				dist = dist2(latlon, m.TriLatLon[tri])
				//} else {
				//	dist = haversine(latlon[0], latlon[1], m.triLatLon[tri][0], m.triLatLon[tri][1])
				//}
				//log.Println("dist", dist)
				if dist < tbbWidth*tbbHeight {
					minDistIndex = tri

					regs = m.SphereMesh.t_circulate_r(outTri, tri)

					// Check if the point is inside the triangle.
					if inTriangleMercator(m.LatLon[regs[0]], m.LatLon[regs[1]], m.LatLon[regs[2]], latlon) {
						break
					}
				}
			}
		} else {
			cReg := closestReg.Data.(int)
			for _, tri := range m.SphereMesh.r_circulate_t(outTri, cReg) {
				// Check if the point is inside the triangle.
				regs = m.SphereMesh.t_circulate_r(outRegs, tri)
				if inTriangleMercator(m.LatLon[regs[0]], m.LatLon[regs[1]], m.LatLon[regs[2]], latlon) {
					minDistIndex = tri
					break
				}
			}
		}

		// If the tri elevation is below sea level, we just return 0.
		if minDistIndex < 0 {
			log.Println("no triangle found")
			return 0
		}
		if m.triElevation[minDistIndex] <= 0 {
			return 0
		}
		// Now we measure the distance of each point in the triangle from the point
		// and use their respektive weights to calculate the height.
		height := calcHeightMercator(m.LatLon[regs[0]], m.LatLon[regs[1]], m.LatLon[regs[2]], latlon, m.Elevation[regs[0]], m.Elevation[regs[1]], m.Elevation[regs[2]])
		// Reverse lat and long.
		// height := calcHeight(m.LatLon[regs[0]][0], m.LatLon[regs[0]][1], m.Elevation[regs[0]], m.LatLon[regs[1]][0], m.LatLon[regs[1]][1], m.Elevation[regs[1]], m.LatLon[regs[2]][0], m.LatLon[regs[2]][1], m.Elevation[regs[2]], latlon[0], latlon[1])

		if height <= 0 {
			return 0
		}
		return 5 * maxAltitudeFactor * height
	}

	// Get the lat long of each point in the tile, but we skip every second point
	// and duplicate the last point.
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width; xx++ {
			pLatLon := [2]float64{
				la1 + float64(yy)*tbbHeight/float64(height-1),
				lo1 + float64(xx)*tbbWidth/float64(width-1),
			}
			// Get the height of the point.
			hei := getHeight(pLatLon) + 1000
			heightMap[(height-yy-1)*width+xx] = uint16(hei)
		}
	}

	buf := bytes.NewBuffer(nil)

	// Write the height map.
	if err := binary.Write(buf, byteorder, heightMap); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

// Get3DTile returns the quantized 3D tile for the given x, y, and zoom.
// WARNING: THIS DOES NOT WORK YET!!!!!
func (m *Map) Get3DTile(x, y, zoom int) *Tile3D {
	// Wrap the tile coordinates.
	x, y = wrapTileCoordinates(x, y, zoom)

	// Calculate the bounds of the tile.
	tbb := newTileBoundingBox(x, y, zoom)
	// Get all regions within the tile.
	la1, lo1, la2, lo2 := tbb.toLatLon()

	isLatLonInBounds := func(lat, lon float64) bool {
		return lat >= la1 && lat <= la2 && lon >= lo1 && lon <= lo2
	}

	// Collect min max elevation of all regions in the tile.
	var minElev, maxElev float64
	var regionsInBounds []int
	for i := 0; i < m.SphereMesh.numRegions; i++ {
		rLat := m.LatLon[i][0]
		rLon := m.LatLon[i][1]
		if isLatLonInBounds(rLat, rLon) {
			regionsInBounds = append(regionsInBounds, i)
			elev := m.Elevation[i]
			if elev < minElev {
				minElev = elev
			}
			if elev > maxElev {
				maxElev = elev
			}
		}
	}

	t3d := new3DTile(x, y, zoom)

	// Set the minimum and maximum heights of the tile.
	t3d.Header.MinimumHeight = float32(minElev)
	t3d.Header.MaximumHeight = float32(maxElev)

	// The three arrays contain the delta from the previous value that is then zig-zag
	// encoded in order to make small integers, regardless of their sign, use a small
	// number of bits.
	t3d.Vertex.VertexCount = uint32(len(regionsInBounds))
	t3d.Vertex.U = make([]uint16, t3d.Vertex.VertexCount)
	t3d.Vertex.V = make([]uint16, t3d.Vertex.VertexCount)
	t3d.Vertex.Height = make([]uint16, t3d.Vertex.VertexCount)

	uBuffer := make([]int64, t3d.Vertex.VertexCount)
	vBuffer := make([]int64, t3d.Vertex.VertexCount)
	heightBuffer := make([]int64, t3d.Vertex.VertexCount)
	for i, r := range regionsInBounds {
		// The horizontal coordinate of the vertex in the tile.
		// - When the u value is 0, the vertex is on the Western edge of the tile.
		// - When the value is 32767, the vertex is on the Eastern edge of the tile.
		// - For other values, the vertex's longitude is a linear interpolation between
		// the longitudes of the Western and Eastern edges of the tile.
		uBuffer[i] = int64((m.LatLon[r][1] - lo1) / (lo2 - lo1) * 32767)

		// The vertical coordinate of the vertex in the tile.
		// - When the v value is 0, the vertex is on the Southern edge of the tile.
		// - When the value is 32767, the vertex is on the Northern edge of the tile.
		// - For other values, the vertex's latitude is a linear interpolation between
		// the latitudes of the Southern and Northern edges of the tile.
		vBuffer[i] = int64((m.LatLon[r][0] - la1) / (la2 - la1) * 32767)

		// The height of the vertex in the tile.
		// - When the height value is 0, the vertex's height is equal to the minimum
		// height within the tile, as specified in the tile's header.
		// - When the value is 32767, the vertex's height is equal to the maximum height
		// within the tile.
		// - For other values, the vertex's height is a linear interpolation between the
		// minimum and maximum heights.
		heightBuffer[i] = int64((m.Elevation[r] - minElev) / (maxElev - minElev) * 32767)
	}

	// Zig-zag encode the values.
	for i := 0; i < int(t3d.Vertex.VertexCount); i++ {
		t3d.Vertex.U[i] = uint16(zigZagEncode(uBuffer[i]))
		t3d.Vertex.V[i] = uint16(zigZagEncode(vBuffer[i]))
		t3d.Vertex.Height[i] = uint16(zigZagEncode(heightBuffer[i]))
	}

	// Build up an region to index map.
	regionToIndex := make(map[int]int)
	for i, r := range regionsInBounds {
		regionToIndex[r] = i
	}

	// Now we have to figure out which triangles we have in the tile.
	var trianglesInBounds []int
	outTri := make([]int, 0, 3)
	for i := 0; i < m.SphereMesh.numTriangles; i++ {
		triLat := m.TriLatLon[i][0]
		triLon := m.TriLatLon[i][1]
		if isLatLonInBounds(triLat, triLon) {
			// Check if all three vertices are in the tile.
			allInBounds := true
			for _, r := range m.SphereMesh.t_circulate_r(outTri, i) {
				if _, ok := regionToIndex[r]; !ok {
					allInBounds = false
					break
				}
			}
			if allInBounds {
				trianglesInBounds = append(trianglesInBounds, i)
			}
		}
	}

	// The indices are used to specify which vertices are connected by triangles.
	// Each triangle is specified by three indices, which are used to look up the
	// corresponding vertices in the vertex array.
	t3d.Index.TriangleCount = uint32(len(trianglesInBounds))
	t3d.Index.Indices = make([]uint32, t3d.Index.TriangleCount*3)

	for i := 0; i < len(trianglesInBounds); i++ {
		tri := trianglesInBounds[i]
		// Get the three regions that make up the triangle.
		regionIndices := m.SphereMesh.t_circulate_r(outTri, tri)
		// Get the index of the triangle in the tile.
		triIndex := i * 3
		// Get the indices of the three regions in the tile.
		regionIndicesInTile := []int{
			regionToIndex[regionIndices[0]],
			regionToIndex[regionIndices[1]],
			regionToIndex[regionIndices[2]],
		}
		// Set the indices of the triangle.
		t3d.Index.Indices[triIndex] = uint32(regionIndicesInTile[0])
		t3d.Index.Indices[triIndex+1] = uint32(regionIndicesInTile[1])
		t3d.Index.Indices[triIndex+2] = uint32(regionIndicesInTile[2])
	}

	// Following the triangle indices is four more lists of indices.
	// These index lists enumerate the vertices that are on the edges
	// of the tile. It is helpful to know which vertices are on the edges
	// in order to add skirts to hide cracks between adjacent levels of detail.
	// Essentially we will add all the regions that do not have all their neighbors
	// in the tile and depending on where they are we will add them to the
	// appropriate edge list.

	centerLat := (la1 + la2) / 2
	centerLon := (lo1 + lo2) / 2

	for _, r := range regionsInBounds {
		// Get the neighbors of the region.
		neighbors := m.SphereMesh.r_circulate_r(nil, r)
		// Check if all the neighbors are in the tile.
		allInBounds := true
		for _, n := range neighbors {
			if _, ok := regionToIndex[n]; !ok {
				allInBounds = false
				break
			}
		}
		// If not all the neighbors are in the tile then we need to add the region
		// to the appropriate edge list.
		if !allInBounds {
			// Get the index of the region in the tile.
			regionIndex := regionToIndex[r]
			// Get the latitude and longitude of the region.
			//lat := m.LatLon[r][0]
			//lon := m.LatLon[r][1]
			// Check if the region is on the West edge.
			// For this we have to determine if the missing neighbor is to the
			// West of the region.
			for _, n := range neighbors {
				if _, ok := regionToIndex[n]; !ok {
					// Get the latitude and longitude of the neighbor.
					latN := m.LatLon[n][0]
					lonN := m.LatLon[n][1]

					deltaLat := latN - centerLat
					deltaLon := lonN - centerLon

					// Check which delta is larger.
					if math.Abs(deltaLat) > math.Abs(deltaLon) {
						// The delta latitude is larger so the region is on the North or South edge.
						if latN < centerLat {
							t3d.Edge.SouthIndices = append(t3d.Edge.SouthIndices, uint32(regionIndex))
							break
						} else {
							t3d.Edge.NorthIndices = append(t3d.Edge.NorthIndices, uint32(regionIndex))
							break
						}
					} else {
						// The delta longitude is larger so the region is on the East or West edge.
						if lonN < centerLon {
							t3d.Edge.WestIndices = append(t3d.Edge.WestIndices, uint32(regionIndex))
							break
						} else {
							t3d.Edge.EastIndices = append(t3d.Edge.EastIndices, uint32(regionIndex))
							break
						}
					}
				}
			}
		}
	}

	// Update the edge counts.
	t3d.Edge.WestVertexCount = uint32(len(t3d.Edge.WestIndices))
	t3d.Edge.SouthVertexCount = uint32(len(t3d.Edge.SouthIndices))
	t3d.Edge.EastVertexCount = uint32(len(t3d.Edge.EastIndices))
	t3d.Edge.NorthVertexCount = uint32(len(t3d.Edge.NorthIndices))

	return t3d
}

func zigZagEncode(value int64) int64 {
	return (value << 1) ^ (value >> 63)
}

func decodeZigZags(uBuffer, vBuffer, heightBuffer []uint16, vertexCount int) ([]int64, []int64, []int64) {
	var u int64
	var v int64
	var height int64

	uOut := make([]int64, vertexCount)
	for i, v := range uBuffer {
		uOut[i] = int64(v)
	}
	vOut := make([]int64, vertexCount)
	for i, v := range vBuffer {
		vOut[i] = int64(v)
	}
	heightOut := make([]int64, vertexCount)
	for i, v := range heightBuffer {
		heightOut[i] = int64(v)
	}

	zigZagDecode := func(value int64) int64 {
		return (value >> 1) ^ (-(value & 1))
	}

	for i := 0; i < vertexCount; i++ {
		u += zigZagDecode(uOut[i])
		v += zigZagDecode(vOut[i])
		height += zigZagDecode(heightOut[i])

		uOut[i] = u
		vOut[i] = v
		heightOut[i] = height
	}

	return uOut, vOut, heightOut
}
