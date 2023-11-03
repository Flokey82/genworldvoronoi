package geo

import (
	"container/heap"
	"log"
	"math"
	"math/rand"
	"sort"

	"github.com/Flokey82/genworldvoronoi/noise"
	"github.com/Flokey82/genworldvoronoi/spheremesh"
	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/geoquad"
	"github.com/Flokey82/go_gens/utils"
	"github.com/Flokey82/go_gens/vectors"
)

type BaseObject struct {
	Seed                   int64        // Seed for random number generators
	Rand                   *rand.Rand   // Rand initialized with above seed
	noise                  *noise.Noise // Opensimplex noise initialized with above seed
	*spheremesh.SphereMesh              // Triangle mesh containing the sphere information

	// Elevation related stuff
	Elevation         []float64       // Point / region elevation
	RegionCompression map[int]float64 // Point / region compression factor

	// Derived elevation related stuff
	Downhill         []int        // Point / region mapping to its lowest neighbor
	Landmasses       []int        // Point / region mapping of regions that are part of the same landmass
	LandmassSize     map[int]int  // Landmass ID to size mapping
	RegionIsMountain map[int]bool // Point / region is a mountain
	RegionIsVolcano  map[int]bool // Point / region is a volcano

	// Moisture related stuff
	Moisture          []float64    // Point / region moisture
	Rainfall          []float64    // Point / region rainfall
	Flux              []float64    // Point / region hydrology: throughflow of rainfall
	Waterbodies       []int        // Point / region mapping of pool to waterbody ID
	WaterbodySize     map[int]int  // Waterbody ID to size mapping
	LakeSize          map[int]int  // Lake ID to size mapping
	RegionIsWaterfall map[int]bool // Point / region is a waterfall

	// Temperature related stuff
	OceanTemperature []float64   // Ocean temperatures (yearly average)
	AirTemperature   []float64   // Air temperatures (yearly average)
	BiomeRegions     []int       // Point / region mapping of regions with the same biome
	BiomeRegionSize  map[int]int // Biome region ID to size mapping

	// Triangle stuff (purely derived from regions)
	TriElevation []float64 // Triangle elevation
	TriMoisture  []float64 // Triangle moisture

	// Currently unused:
	TriPool         []float64 // Triangle water pool depth
	TriFlow         []float64 // Triangle flow intensity (rainfall)
	TriDownflowSide []int     // Triangle mapping to side through which water flows downhill.
	OrderTri        []int     // Triangles in uphill order of elevation.
	SideFlow        []float64 // Flow intensity through sides

	// Currently unused:
	Waterpool []float64 // Point / region hydrology: water pool depth
	Drainage  []int     // Point / region mapping of pool to its drainage region
}

func newBaseObject(seed int64, mesh *spheremesh.SphereMesh) *BaseObject {
	return &BaseObject{
		Seed:              seed,
		Rand:              rand.New(rand.NewSource(seed)),
		noise:             noise.NewNoise(6, 2.0/3.0, seed),
		SphereMesh:        mesh,
		Elevation:         make([]float64, mesh.NumRegions),
		RegionCompression: make(map[int]float64),
		Moisture:          make([]float64, mesh.NumRegions),
		Flux:              make([]float64, mesh.NumRegions),
		Waterpool:         make([]float64, mesh.NumRegions),
		Rainfall:          make([]float64, mesh.NumRegions),
		OceanTemperature:  make([]float64, mesh.NumRegions),
		AirTemperature:    make([]float64, mesh.NumRegions),
		Downhill:          make([]int, mesh.NumRegions),
		Drainage:          make([]int, mesh.NumRegions),
		Waterbodies:       make([]int, mesh.NumRegions),
		WaterbodySize:     make(map[int]int),
		BiomeRegions:      make([]int, mesh.NumRegions),
		BiomeRegionSize:   make(map[int]int),
		Landmasses:        make([]int, mesh.NumRegions),
		LandmassSize:      make(map[int]int),
		LakeSize:          make(map[int]int),
		RegionIsMountain:  make(map[int]bool),
		RegionIsVolcano:   make(map[int]bool),
		RegionIsWaterfall: make(map[int]bool),
		TriPool:           make([]float64, mesh.NumTriangles),
		TriElevation:      make([]float64, mesh.NumTriangles),
		TriMoisture:       make([]float64, mesh.NumTriangles),
		TriDownflowSide:   make([]int, mesh.NumTriangles),
		OrderTri:          make([]int, mesh.NumTriangles),
		TriFlow:           make([]float64, mesh.NumTriangles),
		SideFlow:          make([]float64, mesh.NumSides),
	}
}

// ResetRand resets the random number generator to its initial state.
func (m *BaseObject) ResetRand() {
	m.Rand.Seed(m.Seed)
}

// PickRandomRegions picks n random points/regions from the given mesh.
func (m *BaseObject) PickRandomRegions(n int) []int {
	// Reset the random number generator.
	m.ResetRand()

	// Use lat/lon coordinates to pick random regions.
	// This will allow us to get stable results even if the number of mesh regions changes.
	// Like a result? Save the seed and you can get the same result again, even with higher
	// mesh resolutions.
	useLatLon := true

	// Pick n random regions.
	res := make([]int, 0, n)

	numRegs := m.SphereMesh.NumRegions
	if useLatLon {
		for len(res) < n && len(res) < numRegs {
			// Pick random regions based on their lat/lon coordinates.
			randLat, randLon := m.Rand.Float64()*180-90, m.Rand.Float64()*360-180
			// Find the closest region to the random lat/lon.
			reg, found := m.RegQuadTree.FindNearestNeighbor(geoquad.Point{Lat: randLat, Lon: randLon})
			if !found {
				continue
			}
			res = append(res, reg.Data.(int))
		}
	} else {
		for len(res) < n && len(res) < numRegs {
			res = append(res, m.Rand.Intn(numRegs))
		}
	}
	sort.Ints(res)
	return res
}

// assignTriValues averages out the values of the mesh points / regions and assigns them
// to the triangles of the mesh (or the triangle centroid).
func (m *BaseObject) assignTriValues() {
	rElevation := m.Elevation
	rMoisture := m.Moisture
	rPool := m.Waterpool
	tElevation := m.TriElevation
	tMoisture := m.TriMoisture
	tPool := m.TriPool
	mesh := m.SphereMesh
	numTriangles := mesh.NumTriangles

	for t := 0; t < numTriangles; t++ {
		s0 := 3 * t
		r1 := mesh.S_begin_r(s0)
		r2 := mesh.S_begin_r(s0 + 1)
		r3 := mesh.S_begin_r(s0 + 2)
		tPool[t] = (rPool[r1] + rPool[r2] + rPool[r3]) / 3
		tElevation[t] = (rElevation[r1] + rElevation[r2] + rElevation[r3]) / 3
		tMoisture[t] = (rMoisture[r1] + rMoisture[r2] + rMoisture[r3]) / 3
	}

	// This averages out rainfall to calculate moisture for triangles.
	// Note that this overrides the t_moisture calculated by averaging out r_moisture above.
	for t := 0; t < numTriangles; t++ {
		var moisture float64
		for i := 0; i < 3; i++ {
			s := 3*t + i
			r := mesh.S_begin_r(s)
			moisture += m.Rainfall[r] / 3
		}
		tMoisture[t] = moisture
	}
	m.TriElevation = tElevation
	m.TriPool = tPool
	m.TriMoisture = tMoisture
}

// AssignDownhill will populate r_downhill with a mapping of region to lowest neighbor region.
// NOTE: This is based on mewo2's terrain generation code
// See: https://github.com/mewo2/terrain
func (m *BaseObject) AssignDownhill(usePool bool) {
	m.Downhill = m.GetDownhill(usePool)
}

// GetDownhill will return a mapping of region to lowest neighbor region.
//
// If usePool is true, then the lowest neighbor will be calculated using
// the water pool depth plus the elevation of the region.
func (m *BaseObject) GetDownhill(usePool bool) []int {
	// Here we will map each region to the lowest neighbor.
	mesh := m.SphereMesh
	rDownhill := make([]int, mesh.NumRegions)

	chunkProcessor := func(start, end int) {
		outReg := make([]int, 0, 8)
		for r := start; r < end; r++ {
			lowestRegion := -1
			lowestElevation := m.Elevation[r]
			if usePool {
				lowestElevation += m.Waterpool[r]
			}
			for _, nbReg := range mesh.R_circulate_r(outReg, r) {
				elev := m.Elevation[nbReg]
				if usePool {
					elev += m.Waterpool[nbReg]
				}
				if elev < lowestElevation {
					lowestElevation = elev
					lowestRegion = nbReg
				}
			}
			rDownhill[r] = lowestRegion
		}
	}

	useGoRoutines := true
	if useGoRoutines {
		// Use goroutines to process the regions in chunks.
		various.KickOffChunkWorkers(mesh.NumRegions, chunkProcessor)
	} else {
		// Process the regions in a single chunk.
		chunkProcessor(0, mesh.NumRegions)
	}
	return rDownhill
}

// AssignDownflow starts with triangles that are considered "ocean" and works its way
// uphill to build a graph of child/parents that will allow us later to determine water
// flux and whatnot.
//
// NOTE: This is the original code that Amit uses in his procedural planets project.
// He uses triangle centroids for his river generation, where I prefer to use the regions
// directly.
func (m *BaseObject) AssignDownflow() {
	// Use a priority queue, starting with the ocean triangles and
	// moving upwards using elevation as the priority, to visit all
	// the land triangles.
	queue := make(AscPriorityQueue, 0)
	mesh := m.SphereMesh
	numTriangles := mesh.NumTriangles
	queueIn := 0
	for i := range m.TriDownflowSide {
		m.TriDownflowSide[i] = -999
	}
	heap.Init(&queue)

	// Part 1: ocean triangles get downslope assigned to the lowest neighbor.
	for t := 0; t < numTriangles; t++ {
		if m.TriElevation[t] < 0 {
			bestSide := -1
			bestElevation := m.TriElevation[t]
			for j := 0; j < 3; j++ {
				side := 3*t + j
				elevation := m.TriElevation[mesh.S_outer_t(side)]
				if elevation < bestElevation {
					bestSide = side
					bestElevation = elevation
				}
			}
			m.OrderTri[queueIn] = t
			queueIn++
			m.TriDownflowSide[t] = bestSide
			heap.Push(&queue, &QueueEntry{
				Destination: t,
				Score:       m.TriElevation[t],
				Index:       t,
			})
		}
	}

	// Part 2: land triangles get visited in elevation priority.
	for queueOut := 0; queueOut < numTriangles; queueOut++ {
		current_t := heap.Pop(&queue).(*QueueEntry).Destination
		for j := 0; j < 3; j++ {
			s := 3*current_t + j
			neighbor_t := mesh.S_outer_t(s) // uphill from current_t
			if m.TriDownflowSide[neighbor_t] == -999 && m.TriElevation[neighbor_t] >= 0.0 {
				m.TriDownflowSide[neighbor_t] = mesh.S_opposite_s(s)
				m.OrderTri[queueIn] = neighbor_t
				queueIn++
				heap.Push(&queue, &QueueEntry{
					Destination: neighbor_t,
					Score:       m.TriElevation[neighbor_t],
				})
			}
		}
	}
}

// GetDistance calculate the distance between two regions using
// the lat long and haversine.
func (m *BaseObject) GetDistance(r1, r2 int) float64 {
	return various.Haversine(m.LatLon[r1][0], m.LatLon[r1][1], m.LatLon[r2][0], m.LatLon[r2][1])
}

// GetRegNeighbors returns the neighbor regions of a region.
func (m *BaseObject) GetRegNeighbors(r int) []int {
	return m.SphereMesh.R_circulate_r(nil, r)
}

// GetLowestRegNeighbor returns the lowest neighbor region of a region.
func (m *BaseObject) GetLowestRegNeighbor(r int) int {
	lowestReg := -1
	lowestElev := 999.0
	rElev := m.Elevation[r]
	for _, nbReg := range m.GetRegNeighbors(r) {
		elev := m.Elevation[nbReg]
		if elev < lowestElev && elev < rElev {
			lowestElev = elev
			lowestReg = nbReg
		}
	}
	return lowestReg
}

// DirVecFromToRegs returns the direction vector from region r1 to r2.
func (m *BaseObject) DirVecFromToRegs(from, to int) [2]float64 {
	return various.CalcVecFromLatLong(m.LatLon[from][0], m.LatLon[from][1], m.LatLon[to][0], m.LatLon[to][1])
}

// GetClosestNeighbor returns the closest neighbor of r in the given direction.
func (m *BaseObject) GetClosestNeighbor(outregs []int, r int, vec [2]float64) int {
	if vec[0] == 0 && vec[1] == 0 {
		return r
	}
	lat := m.LatLon[r][0]
	lon := m.LatLon[r][1]
	lat, lon = various.AddVecToLatLong(lat, lon, vec)

	bestDist := math.Inf(1)
	bestR := -1

	neighbors := m.SphereMesh.R_circulate_r(outregs, r)
	for i := 0; i < len(neighbors); i++ {
		latLon2 := m.LatLon[neighbors[i]]
		dist := various.Haversine(lat, lon, latLon2[0], latLon2[1])
		if dist < bestDist {
			bestDist = dist
			bestR = neighbors[i]
		}
	}
	return bestR
}

// TestAreas essentially sums up the surface area of all the regions
// and prints the total.. which shows that we're pretty close to the
// surface area of a unit sphere. :) Yay!
func (m *BaseObject) TestAreas() {
	var tot float64
	numRegs := m.SphereMesh.NumRegions
	for i := 0; i < numRegs; i++ {
		a := m.GetRegArea(i)
		tot += a
		log.Println(a)
	}
	log.Println(tot)
}

// GetRegArea returns the surface area of a region on a unit sphere.
func (m *BaseObject) GetRegArea(r int) float64 {
	regLatLon := m.LatLon[r]
	tris := m.SphereMesh.R_circulate_t(make([]int, 0, 6), r)
	dists := make([]float64, len(tris))
	for i, tri := range tris {
		dLatLon := m.TriLatLon[tri]
		dists[i] = various.Haversine(regLatLon[0], regLatLon[1], dLatLon[0], dLatLon[1])
	}
	var area float64
	for ti0, t0 := range tris {
		ti1 := (ti0 + 1) % len(tris)
		t1 := tris[ti1]
		t0LatLon := m.TriLatLon[t0]
		t1LatLon := m.TriLatLon[t1]
		a := dists[ti0]
		b := dists[ti1]
		c := various.Haversine(t0LatLon[0], t0LatLon[1], t1LatLon[0], t1LatLon[1])
		area += various.HeronsTriArea(a, b, c)
	}
	return area
}

// GetSlope returns the region slope by averaging the slopes of the triangles
// around a given region.
//
// NOTE: This is based on mewo2's erosion code but uses rPolySlope instead of
// rSlope, which determines the slope based on all neighbors.
//
// See: https://github.com/mewo2/terrain
func (m *BaseObject) GetSlope() []float64 {
	// This will collect the slope for each region.
	slope := make([]float64, m.SphereMesh.NumRegions)

	// Get the downhill neighbors for all regions (ignoring water pools for now).
	dh := m.GetDownhill(false)

	chunkProcessor := func(start, end int) {
		outRegs := make([]int, 0, 8)
		for r := start; r < end; r++ {
			dhReg := dh[r]
			// Sinks have no slope, so we skip them.
			if dhReg < 0 {
				continue
			}

			// Get the slope vector.
			// The slope value we want is the length of the vector returned by rPolySlope.
			// NOTE: We use improved poly-slope code, which uses all neighbors for
			// the slope calculation.
			s := m.regPolySlope(outRegs, r)
			slope[r] = math.Sqrt(s[0]*s[0] + s[1]*s[1])
		}
	}

	useGoRoutines := true
	if useGoRoutines {
		// Use goroutines to process the regions in chunks.
		various.KickOffChunkWorkers(m.SphereMesh.NumRegions, chunkProcessor)
	} else {
		// Process all regions in a single chunk.
		chunkProcessor(0, m.SphereMesh.NumRegions)
	}
	return slope
}

// GetSteepness returns the steepness of every region to their downhill neighbor.
//
// NOTE: We define steepness as the angle to a region from its downhill neighbor
// expressed as a value between 0.0 to 1.0 (representing an angle from 0° to 90°).
func (m *BaseObject) GetSteepness() []float64 {
	// This will collect the steepness for each region.
	steeps := make([]float64, m.SphereMesh.NumRegions)

	// Get the downhill neighbors for all regions (ignoring water pools for now).
	dh := m.GetDownhill(false)

	chunkProcessor := func(start, end int) {
		for r := start; r < end; r++ {
			dhReg := dh[r]
			if dhReg < 0 {
				continue // Skip all sinks.
			}

			// In order to calculate the steepness value, we get the great arc distance
			// of each region and its downhill neighbor, as well as the elevation change.
			//
			//     __r            r
			//     | |\            \
			//     | | \            \
			// height|  \            \
			//     | |   \            \
			//     |_|____\dh[r]   ____\dh[r] <- we want to calculate this angle
			//       |dist|
			//
			// We calculate the angle (in radians) as follows:
			// angle = atan(height/dist)
			//
			// Finally, to get the steepness in a range of 0.0 ... 1.0:
			// steepness = angle * 2 / Pi

			// Calculate height difference between r and dh[r].
			hDiff := m.Elevation[r] - m.Elevation[dhReg]

			// Great arc distance between the lat/lon coordinates of r and dh[r].
			regLatLon := m.LatLon[r]
			dhRegLatLon := m.LatLon[dhReg]
			dist := various.Haversine(regLatLon[0], regLatLon[1], dhRegLatLon[0], dhRegLatLon[1])

			// Calculate the the angle (0°-90°) expressed as range from 0.0 to 1.0.
			steeps[r] = math.Atan(hDiff/dist) * 2 / math.Pi
		}
	}

	useGoRoutines := true
	// Use go routines to process a chunk of regions at a time.
	if useGoRoutines {
		various.KickOffChunkWorkers(m.SphereMesh.NumRegions, chunkProcessor)
	} else {
		chunkProcessor(0, m.SphereMesh.NumRegions)
	}
	return steeps
}

// regPolySlope calculates the slope of a region, taking in account all neighbors (which form a polygon).
func (m *BaseObject) regPolySlope(outRegs []int, i int) [2]float64 {
	// See: https://www.khronos.org/opengl/wiki/Calculating_a_Surface_Normal
	//
	// Begin Function CalculateSurfaceNormal (Input Polygon) Returns Vector
	//  Set Vertex Normal to (0, 0, 0)
	//
	//  Begin Cycle for Index in [0, Polygon.vertexNumber)
	//    Set Vertex Current to Polygon.verts[Index]
	//    Set Vertex Next    to Polygon.verts[(Index plus 1) mod Polygon.vertexNumber]
	//
	//    Set Normal.X to Sum of Normal.X and (multiply (Current.Z minus Next.Z) by (Current.Y plus Next.Y))
	//    Set Normal.Z to Sum of Normal.Z and (multiply (Current.Y minus Next.Y) by (Current.X plus Next.X))
	//    Set Normal.Y to Sum of Normal.Y and (multiply (Current.X minus Next.X) by (Current.Z plus Next.Z))
	//  End Cycle
	//
	//  Returning Normalize(Normal)
	// End Function

	// Get the origin vector of the center region.
	// We will rotate the points with this vector until the polygon is facing upwards.
	center := various.ConvToVec3(m.XYZ[i*3:]).Normalize()

	// Get the axis of rotation.
	axis := center.Cross(vectors.Up)

	// Calculate the angle of rotation.
	angle := math.Acos(vectors.Up.Dot(center) / (vectors.Up.Len() * center.Len()))

	var normal vectors.Vec3
	nbs := m.R_circulate_r(outRegs, i)
	for j, r := range nbs {
		jNext := nbs[(j+1)%len(nbs)]
		// Get the current and next vertex and scale the vector by the height factor
		// and elevation, then rotate the vector around the axis.
		current := various.ConvToVec3(m.XYZ[r*3:]).
			Rotate(axis, angle).
			Mul(1 + 0.1*m.Elevation[r])
		next := various.ConvToVec3(m.XYZ[jNext*3:]).
			Rotate(axis, angle).
			Mul(1 + 0.1*m.Elevation[jNext])
		normal.X += (current.Z - next.Z) * (current.Y + next.Y)
		normal.Y += (current.Y - next.Y) * (current.X + next.X)
		normal.Z += (current.X - next.X) * (current.Z + next.Z)
	}
	normal = normal.Normalize()
	return [2]float64{normal.X / -normal.Z, normal.Y / -normal.Z} // TODO: Normalize
}

// RegSlope returns the x/y vector for a given region by averaging the
// x/y vectors of the neighbor triangle centers.
func (m *BaseObject) RegSlope(i int) [2]float64 {
	var res [2]float64
	var count int

	// Buffer for circulating r and t.
	outTri := make([]int, 0, 6)
	outReg := make([]int, 0, 6)

	// NOTE: This is way less accurate. In theory we'd need
	// to calculate the normal of a polygon.
	// See solution rSlope2.
	mesh := m.SphereMesh
	for _, t := range mesh.R_circulate_t(outTri, i) {
		slope := m.RegTriSlope(t, mesh.T_circulate_r(outReg, t))
		res[0] += slope[0]
		res[1] += slope[1]
		count++
	}
	res[0] /= float64(count)
	res[1] /= float64(count)
	return res
}

// RegTriSlope calculates the slope based on three regions.
//
// NOTE: This is based on mewo2's erosion code
// See: https://github.com/mewo2/terrain
//
// WARNING: This only takes in account 3 neighbors!!
// Our implementation however has at times more than 3!
func (m *BaseObject) RegTriSlope(t int, nbs []int) [2]float64 {
	// Skip if we don't have enough regions.
	if len(nbs) != 3 {
		return [2]float64{0, 0}
	}

	// I assume that this is what this code is based on...?
	//
	// See: https://www.khronos.org/opengl/wiki/Calculating_a_Surface_Normal
	//
	// Begin Function CalculateSurfaceNormal (Input Triangle) Returns Vector
	//
	//	Set Vector U to (Triangle.p2 minus Triangle.p1)
	//	Set Vector V to (Triangle.p3 minus Triangle.p1)
	//
	//	Set Normal.X to (multiply U.Z by V.Y) minus (multiply U.Y by V.Z)
	//	Set Normal.Z to (multiply U.Y by V.X) minus (multiply U.X by V.Y)
	//	Set Normal.Y to (multiply U.X by V.Z) minus (multiply U.Z by V.X)
	//
	//	Returning Normal
	//
	// End Function

	// Calculate the normal of the triangle.
	normal := m.RegTriNormal(t, nbs)

	// Calculate the baricentric coordinates of the triangle center.

	det := normal.Z // negative Z?
	return [2]float64{
		normal.X / det,
		normal.Y / det,
	}
}

// RegTriNormal calculates the surface normal of a triangle.
func (m *BaseObject) RegTriNormal(t int, nbs []int) vectors.Vec3 {
	// Rotate the points so that the triangle is facing upwards.
	// So we calculate the difference between the center vector and the
	// global up vector.
	// Then we rotate the points by the resulting difference vector.
	// This is done by calculating the cross product of the two vectors.
	// The cross product is the axis of rotation and the length of the
	// cross product is the angle of rotation.

	// Get the origin vector of the triangle center.
	// We will rotate the points with this vector until the triangle is facing upwards.
	center := various.ConvToVec3(m.TriXYZ[t*3:]).Normalize()

	// Get the axis to rotate the 'center' vector to the global up vector.
	axis := center.Cross(vectors.Up)

	// Calculate the angle of rotation.
	angle := math.Acos(vectors.Up.Dot(center) / (vectors.Up.Len() * center.Len()))

	// Get the three points of the triangle.
	p0 := various.ConvToVec3(m.XYZ[nbs[0]*3:])
	p1 := various.ConvToVec3(m.XYZ[nbs[1]*3:])
	p2 := various.ConvToVec3(m.XYZ[nbs[2]*3:])

	p0 = p0.Rotate(axis, angle).Mul(1 + 0.05*m.Elevation[nbs[0]])
	p1 = p1.Rotate(axis, angle).Mul(1 + 0.05*m.Elevation[nbs[1]])
	p2 = p2.Rotate(axis, angle).Mul(1 + 0.05*m.Elevation[nbs[2]])

	// Calculate the normal.
	return p1.Sub(p0).Cross(p2.Sub(p0)).Normalize()
}

// GetSinks returns all regions that do not have a downhill neighbor.
// If 'skipSinksBelowSea' is true, regions below sea level are excluded.
// If 'usePool' is true, water pool data is used to determine if the sink is a lake.
func (m *BaseObject) GetSinks(skipSinksBelowSea, usePool bool) []int {
	// Identify sinks above sea level.
	var regSinks []int
	for r, lowestReg := range m.GetDownhill(usePool) {
		if lowestReg == -1 && (!skipSinksBelowSea || m.Elevation[r] > 0) { // && m.r_drainage[r] < 0
			regSinks = append(regSinks, r)
		}
	}
	return regSinks
}

// FillSinks is an implementation of the algorithm described in
// https://www.researchgate.net/publication/240407597_A_fast_simple_and_versatile_algorithm_to_fill_the_depressions_of_digital_elevation_models
// and a partial port of the implementation in:
// https://github.com/Rob-Voss/Learninator/blob/master/js/lib/Terrain.js
//
// If randEpsilon is true, a randomized epsilon value is added to the elevation
// during each iteration. This is to prevent the algorithm from being too
// uniform.
//
// NOTE: This algorithm produces a too uniform result at the moment, resulting
// in very artificially looking rivers. It lacks some kind of variation like
// noise. It's very fast and less destructive than my other, home-grown algorithm.
// Maybe it's worth to combine the two in some way?
func (m *BaseObject) FillSinks(randEpsilon bool) []float64 {
	// Reset the RNG.
	m.ResetRand()

	inf := math.Inf(0)
	mesh := m.SphereMesh
	baseEpsilon := 1.0 / (float64(mesh.NumRegions) * 1000.0)
	newHeight := make([]float64, mesh.NumRegions)
	for i := range newHeight {
		if m.Elevation[i] <= 0 {
			// Set the elevation at or below sea level to the current
			// elevation.
			newHeight[i] = m.Elevation[i]
		} else {
			// Set the elevation above sea level to infinity.
			newHeight[i] = inf
		}
	}

	// Loop until no more changes are made.
	var epsilon float64
	outReg := make([]int, 0, 8)
	outPermRegs := make([]int, 0, len(m.Elevation))
	for {
		if randEpsilon {
			// Variation.
			//
			// In theory we could use noise or random values to slightly
			// alter epsilon here. It should still work, albeit a bit slower.
			// The idea is to make the algorithm less destructive and more
			// natural looking.
			//
			// NOTE: I've decided to use m.rand.Float64() instead of noise.
			epsilon = baseEpsilon * m.Rand.Float64()
		}
		changed := false

		// By shuffling the order in which we parse regions,
		// we ensure a more natural look.
		for _, r := range m.randPerm(outPermRegs, len(m.Elevation)) {
			// Skip all regions that have the same elevation as in
			// the current heightmap.
			if newHeight[r] == m.Elevation[r] {
				continue
			}

			// Iterate over all neighbors.
			// NOTE: This used to be in a random order, but that
			// had a high cost, so I dropped it for now.
			for _, nb := range mesh.R_circulate_r(outReg, r) {
				// Since we have set all inland regions to infinity,
				// we will only succeed here if the newHeight of the neighbor
				// is either below sea level or if the newHeight has already
				// been set AND if the elevation is higher than the neighbors.
				//
				// This means that we're working our way inland, starting from
				// the coast, comparing each region with the processed / set
				// neighbors (that aren't set to infinity) in the new heightmap
				// until we run out of regions that need change.
				if m.Elevation[r] >= newHeight[nb]+epsilon {
					newHeight[r] = m.Elevation[r]
					changed = true
					break
				}

				// If we reach this point, the neighbor in the new heightmap
				// is higher than the current elevation of 'r'.
				// This can mean two things. Either the neighbor is set to infinity
				// or the current elevation might indicate a sink.

				// So we check if the newHeight of r is larger than the
				// newHeight of the neighbor (plus epsilon), which will ensure that
				// the newHeight of neighbor is not set to infinity.
				//
				// Additionally we check if the newHeight of the neighbor
				// is higher than the current height of r, which ensures that if the
				// current elevation indicates a sink, we will fill up the sink to the
				// new neighbor height plus epsilon.
				//
				// TODO: Simplify this comment word salad.
				oh := newHeight[nb] + epsilon
				if newHeight[r] > oh && oh > m.Elevation[r] {
					newHeight[r] = oh
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}
	return newHeight
}

// AssignDistanceField calculates the distance from any point in seedRegs to all other points, but
// don't go past any point in stopReg.
func (m *BaseObject) AssignDistanceField(seedRegs []int, stopReg map[int]bool) []float64 {
	// Reset the random number generator.
	m.ResetRand()

	inf := math.Inf(0)
	mesh := m.SphereMesh
	numRegions := mesh.NumRegions

	// Initialize the distance values for all regions to +Inf.
	regDistance := make([]float64, numRegions)
	for i := range regDistance {
		regDistance[i] = inf
	}

	// Initialize the queue for the breadth first search with
	// the seed regions.
	queue := make([]int, len(seedRegs), numRegions)
	for i, r := range seedRegs {
		queue[i] = r
		regDistance[r] = 0
	}

	// Allocate a slice for the output of mesh.R_circulate_r.
	outRegs := make([]int, 0, 8)

	// Random search adapted from breadth first search.
	// TODO: Improve the queue. Currently this is growing unchecked.
	for queueOut := 0; queueOut < len(queue); queueOut++ {
		pos := queueOut + m.Rand.Intn(len(queue)-queueOut)
		currentReg := queue[pos]
		queue[pos] = queue[queueOut]
		for _, nbReg := range mesh.R_circulate_r(outRegs, currentReg) {
			if !math.IsInf(regDistance[nbReg], 0) || stopReg[nbReg] {
				continue
			}

			// If the current distance value for neighbor_r is unset (-1)
			// and if neighbor_r is not a "stop region", we set the distance
			// value to the distance value of current_r, incremented by 1.
			regDistance[nbReg] = regDistance[currentReg] + 1
			queue = append(queue, nbReg)
		}

		// If we have consumed over 1000000 elements in the queue,
		// we reset the queue to the remaining elements.
		if queueOut >= numRegions {
			n := copy(queue, queue[queueOut:])
			queue = queue[:n]
			queueOut = 0
		}
	}

	// TODO: possible enhancement: keep track of which seed is closest
	// to this point, so that we can assign variable mountain/ocean
	// elevation to each seed instead of them always being +1/-1
	return regDistance
}

// UpdateDistanceField updates the distance field for the given regions, given the new seed points.
func (m *BaseObject) UpdateDistanceField(regDistance []float64, seedRegs []int, stopReg map[int]bool) []float64 {
	// Reset the random number generator.
	m.ResetRand()
	mesh := m.SphereMesh
	numRegions := mesh.NumRegions
	queue := make([]int, len(seedRegs), numRegions)

	// TODO: Also check if a seed point has "disappeared" .If so, we
	// might need to recompute the distance field for all regions.
	for i, r := range seedRegs {
		// Check if the region distance in the current field is not 0,
		// which means that the region has not been previously used as
		// a seed region. If the region distance is not 0, we set it
		// to 0 and add it to the queue.
		if regDistance[r] != 0 {
			regDistance[r] = 0
			queue[i] = r
		}
	}

	// Allocate a slice for the output of mesh.R_circulate_r.
	outRegs := make([]int, 0, 8)

	// Random search adapted from breadth first search.
	// TODO: Improve the queue. Currently this is growing unchecked.
	for queueOut := 0; queueOut < len(queue); queueOut++ {
		pos := queueOut + m.Rand.Intn(len(queue)-queueOut)
		currentReg := queue[pos]
		queue[pos] = queue[queueOut]
		nextRegDistance := regDistance[currentReg] + 1
		for _, nbReg := range mesh.R_circulate_r(outRegs, currentReg) {
			// If the distance we expect to find for the neighbor region is
			// is larger than the current distance, we skip the neighbor
			// since we have already found a shorter path to a seed region.
			if regDistance[nbReg] <= nextRegDistance || stopReg[nbReg] {
				continue
			}

			// If the current distance value for neighbor_r is unset (-1)
			// and if neighbor_r is not a "stop region", we set the distance
			// value to the distance value of current_r, incremented by 1.
			regDistance[nbReg] = nextRegDistance
			queue = append(queue, nbReg)
		}

		// If we have consumed over 1000000 elements in the queue,
		// we reset the queue to the remaining elements.
		if queueOut >= numRegions {
			n := copy(queue, queue[queueOut:])
			queue = queue[:n]
			queueOut = 0
		}
	}

	return regDistance
}

type Interpolated struct {
	NumRegions int
	BaseObject
}

// Interpolate adds for each neighboring region pair one intermediate,
// interpolated region, increasing the "resolution" for the given regions.
func (m *BaseObject) Interpolate(regions []int) (*Interpolated, error) {
	var ipl Interpolated
	ipl.Seed = m.Seed
	ipl.Rand = rand.New(rand.NewSource(m.Seed))

	// Increase the resolution by one octave.
	ipl.noise = m.noise.PlusOneOctave()

	// Get all points within bounds.
	seen := make(map[[2]int]bool)

	// Carry over mountains, volcanoes and compression.
	regionIsMountain := make(map[int]bool)
	regionIsVolcano := make(map[int]bool)
	regionIsWaterfall := make(map[int]bool)
	regionCompression := make(map[int]float64)
	outRegs := make([]int, 0, 6)

	mesh := m.SphereMesh
	var xyz []float64
	for _, r := range regions {
		if m.RegionIsMountain[r] {
			regionIsMountain[ipl.NumRegions] = true
		}
		if m.RegionIsVolcano[r] {
			regionIsVolcano[ipl.NumRegions] = true
		}
		if m.RegionIsWaterfall[r] {
			regionIsWaterfall[ipl.NumRegions] = true
		}
		if m.RegionCompression[r] != 0 {
			regionCompression[ipl.NumRegions] = m.RegionCompression[r]
		}

		ipl.NumRegions++
		rxyz := m.XYZ[r*3 : (r*3)+3]
		xyz = append(xyz, rxyz...)
		ipl.Moisture = append(ipl.Moisture, m.Moisture[r])
		ipl.Rainfall = append(ipl.Rainfall, m.Rainfall[r])
		ipl.Flux = append(ipl.Flux, m.Flux[r])
		ipl.Waterpool = append(ipl.Waterpool, m.Waterpool[r])
		ipl.Elevation = append(ipl.Elevation, m.Elevation[r])
		ipl.OceanTemperature = append(ipl.OceanTemperature, m.OceanTemperature[r])
		ipl.AirTemperature = append(ipl.AirTemperature, m.AirTemperature[r])

		// Circulate_r all points and add midpoints.
		for _, nbReg := range mesh.R_circulate_r(outRegs, r) {
			// Check if we already added a midpoint for this edge.
			var check [2]int
			if r < nbReg {
				check[0] = r
				check[1] = nbReg
			} else {
				check[0] = nbReg
				check[1] = r
			}
			if seen[check] {
				continue
			}
			seen[check] = true

			// Generate midpoint and average values.
			rnxyz := m.XYZ[nbReg*3 : (nbReg*3)+3]
			mid := various.ConvToVec3([]float64{
				(rxyz[0] + rnxyz[0]) / 2,
				(rxyz[1] + rnxyz[1]) / 2,
				(rxyz[2] + rnxyz[2]) / 2,
			}).Normalize()
			xyz = append(xyz, mid.X, mid.Y, mid.Z)
			ipl.NumRegions++

			// Calculate diff and use noise to add variation.
			nvl := (ipl.noise.Eval3(mid.X, mid.Y, mid.Z) + 1) / 2
			diffElevation := m.Elevation[nbReg] - m.Elevation[r]
			diffMoisture := m.Moisture[nbReg] - m.Moisture[r]
			diffRainfall := m.Rainfall[nbReg] - m.Rainfall[r]
			diffFlux := m.Flux[nbReg] - m.Flux[r]
			diffPool := m.Waterpool[nbReg] - m.Waterpool[r]
			diffOceanTemp := m.OceanTemperature[nbReg] - m.OceanTemperature[r]
			diffAirTemp := m.AirTemperature[nbReg] - m.AirTemperature[r]

			// TODO: Add some better variation with the water pool and stuff.
			// TODO: Add flood fill, downhill and flux?
			// TODO: Average compression values?

			ipl.Elevation = append(ipl.Elevation, m.Elevation[r]+(diffElevation*nvl))
			ipl.Moisture = append(ipl.Moisture, m.Moisture[r]+(diffMoisture*nvl))
			ipl.Rainfall = append(ipl.Rainfall, m.Rainfall[r]+(diffRainfall*nvl))
			ipl.Flux = append(ipl.Flux, m.Flux[r]+(diffFlux*nvl))
			ipl.Waterpool = append(ipl.Waterpool, m.Waterpool[r]+(diffPool*nvl))
			ipl.OceanTemperature = append(ipl.OceanTemperature, m.OceanTemperature[r]+(diffOceanTemp*nvl))
			ipl.AirTemperature = append(ipl.AirTemperature, m.AirTemperature[r]+(diffAirTemp*nvl))
		}
	}

	// Convert the XYZ to lat/lon.
	latLon := make([][2]float64, 0, len(xyz)/3)
	for r := 0; r < len(xyz); r += 3 {
		// HACKY! Fix this properly!
		nla, nlo := various.LatLonFromVec3(various.ConvToVec3(xyz[r:r+3]).Normalize(), 1.0)
		latLon = append(latLon, [2]float64{nla, nlo})
	}

	// Create the sphere mesh.
	sphere, err := spheremesh.NewSphereMesh(latLon, xyz, false)
	if err != nil {
		return nil, err
	}
	ipl.SphereMesh = sphere

	// Update quadtrees.
	ipl.RegQuadTree = spheremesh.NewQuadTreeFromLatLon(ipl.SphereMesh.LatLon)
	ipl.TriQuadTree = spheremesh.NewQuadTreeFromLatLon(ipl.SphereMesh.TriLatLon)

	// Assign all the values.
	ipl.RegionIsMountain = regionIsMountain
	ipl.RegionIsVolcano = regionIsVolcano
	ipl.RegionIsWaterfall = regionIsWaterfall
	ipl.RegionCompression = regionCompression
	ipl.TriPool = make([]float64, sphere.NumTriangles)
	ipl.TriElevation = make([]float64, sphere.NumTriangles)
	ipl.TriMoisture = make([]float64, sphere.NumTriangles)
	ipl.TriDownflowSide = make([]int, sphere.NumTriangles)
	ipl.OrderTri = make([]int, sphere.NumTriangles)
	ipl.TriFlow = make([]float64, sphere.NumTriangles)
	ipl.SideFlow = make([]float64, sphere.NumSides)
	ipl.AssignDownhill(true)
	ipl.assignTriValues()
	ipl.AssignDownflow()
	ipl.AssignFlow()

	return &ipl, nil
}

// randPerm returns a random permutation of the given slice indices.
// This works like rand.Perm, but it reuses the same slice.
func (m *BaseObject) randPerm(perm []int, n int) []int {
	perm = perm[:utils.Min(cap(perm), n)]
	if diff := len(perm) - n; diff < 0 {
		perm = append(perm, make([]int, -diff)...)
	}
	// In the following loop, the iteration when i=0 always swaps m[0] with m[0].
	// A change to remove this useless iteration is to assign 1 to i in the init
	// statement. But Perm also effects r. Making this change will affect
	// the final state of r. So this change can't be made for compatibility
	// reasons for Go 1.
	for i := 0; i < n; i++ {
		j := m.Rand.Intn(i + 1)
		perm[i] = perm[j]
		perm[j] = i
	}
	return perm
}

// BoundingBoxResult contains the results of a bounding box query.
type BoundingBoxResult struct {
	Regions   []int // Regions withi the bounding box.
	Triangles []int // Triangles within the bounding box.
}

// getBoundingBoxRegions returns all regions and triangles within the given lat/lon bounding box.
//
// TODO: Add margin in order to also return regions/triangles that are partially
// within the bounding box.
func (m *BaseObject) getBoundingBoxRegions(lat1, lon1, lat2, lon2 float64) *BoundingBoxResult {
	r := &BoundingBoxResult{}
	// TODO: Add convenience function to check against bounding box.
	for i, ll := range m.LatLon {
		if l0, l1 := ll[0], ll[1]; l0 < lat1 || l0 >= lat2 || l1 < lon1 || l1 >= lon2 {
			continue
		}
		r.Regions = append(r.Regions, i)
	}
	for i, ll := range m.TriLatLon {
		if l0, l1 := ll[0], ll[1]; l0 < lat1 || l0 >= lat2 || l1 < lon1 || l1 >= lon2 {
			continue
		}
		r.Triangles = append(r.Triangles, i)
	}
	return r
}
