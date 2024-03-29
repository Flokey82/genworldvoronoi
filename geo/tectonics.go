package geo

import (
	"container/heap"
	"container/list"
	"math"
	"sort"

	"github.com/Flokey82/genideas/genfibonaccisphere"
	"github.com/Flokey82/genworldvoronoi/noise"
	"github.com/Flokey82/genworldvoronoi/various"
	"github.com/Flokey82/go_gens/vectors"
)

const (
	tecTypeFibonacci = iota
	tecTypeNoise
	tecTypeRandom
)

// generatePlates generates a number of plate seed points and starts growing the plates
// starting from those seeds in a random order.
func (m *Geo) generatePlates() {
	m.ResetRand()
	mesh := m.SphereMesh
	regPlate := make([]int, mesh.NumRegions)
	for i := range regPlate {
		regPlate[i] = -1
	}

	//useAlternativePlates := false

	// Pick random regions as seed points for plate generation and for plate vectors.
	numPlates := min(m.NumPlates, m.NumPoints)
	// randRegsLatLon := m.PickRandomRegions2(numPlates * 2)
	randRegs := m.PickRandomRegions(numPlates*2, true)

	// Shuffle the regions.
	m.Rand.Shuffle(len(randRegs), func(i, j int) {
		randRegs[i], randRegs[j] = randRegs[j], randRegs[i]
	})

	// The first numPlates regions are the seed points for the plates.
	plateRegs := randRegs[:numPlates]

	// The remaining regions are used to calculate the plate vectors.
	vectorRegs := randRegs[numPlates:]

	// In Breadth First Search (BFS) the queue will be all elements in
	// queue[queue_out ... queue.length-1]. Pushing onto the queue
	// adds an element to the end, increasing queue.length. Popping
	// from the queue removes an element from the beginning by
	// increasing queue_out.

	mode := tecTypeRandom

	switch mode {
	case tecTypeFibonacci:
		seedLatLons := make([][2]float64, numPlates)
		for i, r := range plateRegs {
			seedLatLons[i] = m.LatLon[r]
		}
		s := genfibonaccisphere.NewSphereWithContinents(seedLatLons, 12345)

		for n := 0; n < m.NumRegions; n++ {
			idx := s.FindIndexToPoint(m.LatLon[n][0], m.LatLon[n][1])
			regPlate[n] = plateRegs[idx]
		}
	case tecTypeNoise:
		// To add variety, use a random search instead of a breadth first
		// search. The frontier of elements to be expanded is still
		// queue[queue_out ... queue.length-1], but pick a random element
		// to pop instead of the earliest one. Do this by swapping
		// queue[pos] and queue[queue_out].
		outReg := make([]int, 0, 6)

		// Set up a new noise generator.
		noise := noise.NewNoise(5, 2.0/3.0, m.Seed)

		// Use a priority queue with noise value and distance to seed point as priority.
		var queue AscPriorityQueue
		heap.Init(&queue)

		plateRegionCount := make(map[int]int)
		// Initialize the queue with the seed regions.
		// TODO: Start with the noise value of the true LAT/LON coordinates of the seed regions
		// (the lat/lon that we used initially to find the closest region).
		for _, r := range plateRegs {
			heap.Push(&queue, &QueueEntry{
				Destination: r,
				Score:       0,
				Origin:      r,
			})
		}

		// Work through the queue.
		for queue.Len() > 0 {
			// Pop the region with the highest priority.
			currentItem := heap.Pop(&queue).(*QueueEntry)
			currentReg := currentItem.Destination

			// If the region is already assigned to a plate, skip it.
			if regPlate[currentReg] != -1 {
				continue
			}

			// Assign the region to the current plate.
			regPlate[currentReg] = currentItem.Origin
			plateRegionCount[currentItem.Origin]++

			// Get the Neighbors of the current region.
			outReg = mesh.R_circulate_r(outReg, currentReg)

			// Iterate through the neighbors.
			for _, nbReg := range outReg {
				// If the neighbor is already assigned to a plate, skip it.
				if regPlate[nbReg] != -1 {
					continue
				}

				// Calculate the priority of the neighbor.
				// The priority is the distance to the seed point plus some noise.
				// This is to ensure that the plate growth is not uniform.
				score := noise.Eval3(m.XYZ[3*nbReg], m.XYZ[3*nbReg+1], m.XYZ[3*nbReg+2])
				score *= float64(plateRegionCount[currentItem.Origin])
				// score *= m.GetDistance(currentItem.Origin, nbReg)

				// Push the neighbor onto the queue.
				heap.Push(&queue, &QueueEntry{
					Destination: nbReg,
					Score:       score,
					Origin:      currentItem.Origin,
				})
			}
		}
	case tecTypeRandom:
		outReg := make([]int, 0, 6)

		// Assign seed regions to plates.
		var queue []int
		for _, r := range plateRegs {
			queue = append(queue, r)
			regPlate[r] = r
		}

		// TODO: How can we make the growth consistent across different mesh resolutions?
		for queueOut := 0; queueOut < len(queue); queueOut++ {
			pos := queueOut + m.Rand.Intn(len(queue)-queueOut)
			currentReg := queue[pos]
			queue[pos] = queue[queueOut]
			outReg = mesh.R_circulate_r(outReg, currentReg)
			for _, nbReg := range outReg {
				if regPlate[nbReg] == -1 {
					regPlate[nbReg] = regPlate[currentReg]
					queue = append(queue, nbReg)
				}
			}
		}
	}

	// Assign a random movement vector for each plate
	// and normalize it.
	// TODO: Use raw lat/lon to calculate the plate vectors.
	regXYZ := m.XYZ
	plateVectors := make([]vectors.Vec3, mesh.NumRegions)
	for i, centerReg := range plateRegs {
		// Get the random region that we use to calculate the vector.
		distReg := vectorRegs[i]

		// Get the vector from the center region to the random region.
		p0 := various.ConvToVec3(regXYZ[3*centerReg : 3*centerReg+3])
		p1 := various.ConvToVec3(regXYZ[3*distReg : 3*distReg+3])
		plateVectors[centerReg] = vectors.Sub3(p1, p0).Normalize()
	}

	m.PlateRegs = plateRegs
	m.RegionToPlate = regPlate
	m.PlateToVector = plateVectors
}

// assignOceanPlates randomly assigns approx. 50% of the plates as ocean plates.
func (m *Geo) assignOceanPlates() {
	m.ResetRand()
	m.PlateIsOcean = make(map[int]bool)
	if m.GeoConfig.OceanPlatesAltSelection {
		numOceanPlates := int(math.Ceil(float64(m.NumPlates) * m.GeoConfig.OceanPlatesFraction))
		for i, idx := range m.Rand.Perm(len(m.PlateRegs)) {
			if i >= numOceanPlates {
				break
			}
			// TODO: either make tiny plates non-ocean, or make sure tiny plates don't create seeds for rivers
			m.PlateIsOcean[m.PlateRegs[idx]] = true
		}
	} else {
		for _, r := range m.PlateRegs {
			if m.Rand.Float64() < m.GeoConfig.OceanPlatesFraction {
				// TODO: either make tiny plates non-ocean, or make sure tiny plates don't create seeds for rivers
				m.PlateIsOcean[r] = true
			}
		}
	}
}

// Calculate the collision measure, which is the amount
// that any neighbor's plate vector is pushing against
// the current plate vector.
const collisionThreshold = 0.75

// findCollisions iterates through all regions and finds the regions whose neighbor points
// belong to a different plate. This subset of points is than moved using their respective
// tectonic plate vector and if they approach each other to an extent where they exceed the
// collision threshold, a collision is noted. Depending on the type of plates involved in a
// collision, they produce certain effects like forming a coastline, mountains, etc.
//
// FIXME: The smaller the distance of the cells, the more likely a plate moves past the neighbor plate.
// This causes all kinds of issues.
func (m *Geo) findCollisions() ([]int, []int, []int, map[int]float64) {
	// Use either the largest or smallest compression value.
	useLargestCompression := true

	plateIsOcean := m.PlateIsOcean
	regPlate := m.RegionToPlate
	plateVectors := m.PlateToVector
	numRegions := m.SphereMesh.NumRegions
	compressionReg := make(map[int]float64)

	// Initialize the compression measure to either the largest or smallest
	// possible float64 value.
	inf := math.Inf(1)
	if useLargestCompression {
		inf = math.Inf(-1)
	}

	const deltaTime = 1e-11 // simulate movement

	// For each region, I want to know how much it's being compressed
	// into an adjacent region. The "compression" is the change in
	// distance as the two regions move. I'm looking for the adjacent
	// region from a different plate that pushes most into this one
	var mountainRegs, coastlineRegs, oceanRegs []int
	rOut := make([]int, 0, 6)
	var bestReg int
	var bestCompression float64
	for currentReg := 0; currentReg < numRegions; currentReg++ {
		bestCompression = inf
		bestReg = -1
		rOut = m.SphereMesh.R_circulate_r(rOut, currentReg)
		for _, nbReg := range rOut {
			if regPlate[currentReg] != regPlate[nbReg] {
				// sometimes I regret storing xyz in a compact array...
				currentPos := various.ConvToVec3(m.XYZ[3*currentReg : 3*currentReg+3])
				neighborPos := various.ConvToVec3(m.XYZ[3*nbReg : 3*nbReg+3])

				// simulate movement for deltaTime seconds
				distanceBefore := vectors.Dist3(currentPos, neighborPos)

				plateVec := plateVectors[regPlate[currentReg]].Mul(deltaTime)
				a := vectors.Add3(currentPos, plateVec)

				plateVecNeighbor := plateVectors[regPlate[nbReg]].Mul(deltaTime)
				b := vectors.Add3(neighborPos, plateVecNeighbor)

				distanceAfter := vectors.Dist3(a, b)

				// how much closer did these regions get to each other?
				compression := distanceBefore - distanceAfter

				// Sum up the compression for this region.
				// NOTE: Note sure if this actually makes sense.
				compressionReg[nbReg] += compression

				// keep track of the adjacent region that gets closest.
				// NOTE: changed from compression < bestCompression
				if (compression > bestCompression) == useLargestCompression {
					bestReg = nbReg
					bestCompression = compression
				}
			}
		}

		// Check if we have a ocean region.
		if m.RegionToPlate[currentReg] == currentReg && m.PlateIsOcean[currentReg] {
			oceanRegs = append(oceanRegs, currentReg)
		}

		// Check if we have a collision candidate.
		if bestReg == -1 {
			continue
		}

		compressionReg[currentReg] = bestCompression

		// at this point, bestCompression tells us how much closer
		// we are getting to the region that's pushing into us the most.
		collided := bestCompression > collisionThreshold*deltaTime

		enablePlateCheck := true
		if enablePlateCheck {
			currentPlate := m.RegionToPlate[currentReg]
			bestPlate := m.RegionToPlate[bestReg]
			if plateIsOcean[currentPlate] && plateIsOcean[bestPlate] {
				// If both plates are ocean plates and they collide, a coastline is produced.
				if collided {
					coastlineRegs = append(coastlineRegs, currentReg)
				}
			} else if !plateIsOcean[currentPlate] && !plateIsOcean[bestPlate] {
				// If both plates are non-ocean plates and they collide, mountains are formed.
				if collided {
					mountainRegs = append(mountainRegs, currentReg)
				}
			} else {
				// If the plates are of different types, a collision results in a mountain and
				// drifting apart results in a coastline being defined.
				if collided {
					// If one plate is ocean, mountains only fold up on the non-ocean plate.
					if !plateIsOcean[currentPlate] {
						mountainRegs = append(mountainRegs, currentReg)
					}
				} else {
					// This is incorrect, since can't be certain that we are drifting apart
					// without checking if we have actually a negative compression.
					// I leave this in here, because it just looks cool.
					coastlineRegs = append(coastlineRegs, currentReg)
				}
			}
		} else {
			// If both plates collide, mountains are formed.
			if collided {
				mountainRegs = append(mountainRegs, currentReg)
			}
		}
	}
	return mountainRegs, coastlineRegs, oceanRegs, compressionReg
}

// PropagateCompression propagates the compression values from the seed regions
// to all other regions.
func (m *BaseObject) PropagateCompression(compression map[int]float64) []float64 {
	// Get the min and max compression value so that we can
	// normalize the compression value, also we need to copy
	// the compression values into a slice so that we can
	// modify them and queue them up.
	cmp := make([]float64, m.SphereMesh.NumRegions)
	var cmpSeeds []int
	for r, comp := range compression {
		cmp[r] = comp
		cmpSeeds = append(cmpSeeds, r)
	}
	sort.Ints(cmpSeeds)

	// Queue up the seed regions, shuffle them so that we don't
	// always start with the same regions.
	queue := list.New()
	for _, r := range cmpSeeds {
		queue.PushBack(r)
	}

	// Normalize the compression values.
	minComp, maxComp := minMax(cmp)
	for r := range cmp {
		if cmp[r] > 0 {
			cmp[r] /= maxComp
		} else if cmp[r] < 0 {
			cmp[r] /= math.Abs(minComp)
		}
	}

	// Propagate the compression values.
	outRegs := make([]int, 0, 6)
	for queue.Len() > 0 {
		currentReg := queue.Remove(queue.Front()).(int)
		currentComp := cmp[currentReg]
		for _, nbReg := range m.SphereMesh.R_circulate_r(outRegs, currentReg) {
			// The compression value diminishes over distance.
			// This should be using the inverse square law, but
			// we use a linear function instead.
			distToNb := 1 + m.GetDistance(currentReg, nbReg)
			nbComp := currentComp / distToNb
			if cmp[nbReg] == 0 {
				cmp[nbReg] = nbComp
				queue.PushBack(nbReg)
			} else {
				// Average the compression values.
				// NOTE: I know this is not great, but it works.
				// ... otherwise we get real bad artifacts.
				cmp[nbReg] = (cmp[nbReg] + nbComp) / 2
			}
		}
	}

	// Normalize the compression values.
	minComp, maxComp = minMax(cmp)
	for r := range cmp {
		if cmp[r] > 0 {
			cmp[r] /= maxComp
		} else if cmp[r] < 0 {
			cmp[r] /= math.Abs(minComp)
		}
		// Apply a square falloff to the compression values.
		cmp[r] *= math.Abs(cmp[r])
	}
	return cmp
}

// assignRegionElevation finds collisions between plate regions and assigns
// elevation for each point on the sphere accordingly, which will result in
// mountains, coastlines, etc.
// To ensure variation, opensimplex noise is used to break up any uniformity.
func (m *Geo) assignRegionElevation() {
	// TODO: Use collision values to determine intensity of generated landscape features.
	m.Mountain_r, m.Coastline_r, m.Ocean_r, m.RegionCompression = m.findCollisions()

	// Sort mountains by compression.
	sort.Slice(m.Mountain_r, func(i, j int) bool {
		return m.RegionCompression[m.Mountain_r[i]] > m.RegionCompression[m.Mountain_r[j]]
	})

	// Take note of all mountains.
	// Since they are sorted by compression, we can use the first m.NumVolcanoes
	// as volcanoes.
	var gotVolcanoes int
	for _, r := range m.Mountain_r {
		m.RegionIsMountain[r] = true
		if gotVolcanoes < m.NumVolcanoes {
			m.RegionIsVolcano[r] = true
			gotVolcanoes++
		}
	}

	// Distance field generation.
	// I do not quite know how that works, but it is based on:
	// See: https://www.redblobgames.com/x/1728-elevation-control/
	stopReg := make(map[int]bool)
	for _, r := range m.Mountain_r {
		stopReg[r] = true
	}
	for _, r := range m.Coastline_r {
		stopReg[r] = true
	}
	for _, r := range m.Ocean_r {
		stopReg[r] = true
	}

	// Calculate distance fields.
	// Graph distance from mountains (stops at ocean regions).
	rDistanceA := m.AssignDistanceField(m.Mountain_r, convToMap(m.Ocean_r))
	// Graph distance from ocean (stops at coastline regions).
	rDistanceB := m.AssignDistanceField(m.Ocean_r, convToMap(m.Coastline_r))
	// Graph distance from coastline (stops at all other regions).
	rDistanceC := m.AssignDistanceField(m.Coastline_r, stopReg)

	// Propagate the compression values.
	compPerReg := m.PropagateCompression(m.RegionCompression)

	// This code below calculates the height of a given region based on a linear
	// interpolation of the three distance values above.
	//
	// Ideally, we would use some form of noise using the distance to a mountain / faultline
	// to generate a more natural looking landscape with mountain ridges resulting from the
	// folding of the plates.
	//
	// Since we want a "wave" like appearance, we could use one dimensional noise based on the
	// distance to the faultline with some variation for a more natural look.
	const epsilon = 1e-7
	r_xyz := m.XYZ

	// Exponent for interpolation.
	// n = 1 is a linear interpolation
	// n = 2 is a square interpolation
	// n = 0.5 is a square root interpolation
	na := 1.0 / 1.0
	nb := 1.0 / 1.0
	nc := 1.0 / 1.0
	for r := 0; r < m.SphereMesh.NumRegions; r++ {
		a := math.Pow(rDistanceA[r], na) + epsilon // Distance from mountains
		b := math.Pow(rDistanceB[r], nb) + epsilon // Distance from oceans
		c := math.Pow(rDistanceC[r], nc) + epsilon // Distance from coastline
		if m.PlateIsOcean[m.RegionToPlate[r]] {
			// Ocean plates are slightly lower than other plates.
			m.Elevation[r] = -0.1
		}
		if math.IsInf(rDistanceA[r], 0) && math.IsInf(rDistanceB[r], 0) {
			// If the distance from mountains and oceans is unset (infinity),
			// we increase the elevation by 0.1 since we wouldn't be able to
			// calculate the harmonic mean.
			m.Elevation[r] += 0.1
		} else {
			// The height is calculated as weighted harmonic mean of the
			// three distance values.
			f := (1/a - 1/b) / (1/a + 1/b + 1/c)

			// Average with plate compression to get some
			// variation in the landscape.
			f = (f + compPerReg[r]) * 0.5

			// Apply a square falloff to the elevaltion values.
			// f *= math.Abs(f)
			m.Elevation[r] += f
		}
	}

	/*
		// Add a cosine based on the distance to the closest mountain.
		// This is to simulate the effect of the mountain ridges.
		// NOTE: This looks very unnatural. :(
		for r := 0; r < m.SphereMesh.NumRegions; r++ {
			if m.Elevation[r] < 0 {
				continue
			}
			// Get the distance to the closest mountain.
			minDist := math.Inf(1)
			for _, r2 := range m.mountain_r {
				dist := m.GetDistance(r, r2)
				if dist < minDist {
					minDist = dist
				}
			}

			// Add a cosine based on the distance to the closest mountain.
			v := (math.Cos(minDist*math.Pi*128) + 1) / 2
			randAmount := m.noise.Eval3(r_xyz[3*r], r_xyz[3*r+1], r_xyz[3*r+2])
			m.Elevation[r] *= 0.5 + (0.5 * (1 - randAmount)) + 0.5*v*v*randAmount
		}
	*/

	// Apply noise to the elevation values.
	if m.GeoConfig.MultiplyNoise {
		for r := 0; r < m.SphereMesh.NumRegions; r++ {
			m.Elevation[r] *= m.noise.Eval3(r_xyz[3*r], r_xyz[3*r+1], r_xyz[3*r+2])
		}
	} else {
		for r := 0; r < m.SphereMesh.NumRegions; r++ {
			m.Elevation[r] += m.noise.Eval3(r_xyz[3*r], r_xyz[3*r+1], r_xyz[3*r+2])*2 - 1 // Noise from -1.0 to 1.0
		}
	}

	// Normalize the elevation values to the range -1.0 - 1.0
	// TODO: Protect against division by zero.
	if m.GeoConfig.NormalizeElevation {
		minElevation, maxElevation := minMax(m.Elevation)
		for r := 0; r < m.SphereMesh.NumRegions; r++ {
			if m.Elevation[r] < 0 {
				m.Elevation[r] /= math.Abs(minElevation)
			} else {
				m.Elevation[r] /= maxElevation
			}
		}
	}

	// Apply a square falloff to the elevation values above sea level.
	if m.GeoConfig.TectonicFalloff {
		for r := range m.Elevation {
			if m.Elevation[r] > 0 {
				m.Elevation[r] *= m.Elevation[r]
			}
		}
	}
}
