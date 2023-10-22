// Package genworldvoronoi is a port of redblobgames' amazing planet generator.
// See: https://www.redblobgames.com/x/1843-planet-generation
// And: https://github.com/redblobgames/1843-planet-generation
package genworldvoronoi

type Map struct {
	*Geo // Geography / geology
	*Civ // Civilization
	*Bio // Plants / animals / funghi

	// *TileCache
	// CoarseMeshes []*SphereMesh // Coarse meshes for each zoom level.
}

func NewMapFromConfig(seed int64, cfg *Config) (*Map, error) {
	if cfg == nil {
		cfg = NewConfig()
	}

	// Initialize the planet.
	geo, err := newGeo(seed, cfg.GeoConfig)
	if err != nil {
		return nil, err
	}

	// Initialize the map.
	m := &Map{
		Geo: geo,
		Civ: NewCiv(geo, cfg.CivConfig),
		Bio: newBio(geo, cfg.BioConfig),
	}
	m.generateMap()

	/*
		m.TileCache = NewTileCache(m.BaseObject)

		// Generate coarse meshes for LOD.
		m.CoarseMeshes = make([]*SphereMesh, 5)
		for i := range m.CoarseMeshes {
			if i == len(m.CoarseMeshes)-1 {
				m.CoarseMeshes[i] = m.SphereMesh
				continue
			}
			n := 1 << utils.Max(len(m.CoarseMeshes)-i-1, 0)
			log.Println("Generating coarse mesh for zoom level", i, "with", n, "iterations")
			mesh, err := m.MakeCoarseSphereMesh(n)
			if err != nil {
				return nil, err
			}
			m.CoarseMeshes[i] = mesh
		}*/
	return m, nil
}

func NewMap(seed int64, numPlates, numPoints, numVolcanoes int, jitter float64) (*Map, error) {
	cfg := NewConfig()
	cfg.NumPlates = numPlates
	cfg.NumPoints = numPoints
	cfg.NumVolcanoes = numVolcanoes
	cfg.Jitter = jitter

	return NewMapFromConfig(seed, cfg)
}

/*
func (m *Map) getCoarseForZoom(zoom int) (*SphereMesh, int) {
	n := 1 << utils.Max(len(m.CoarseMeshes)-zoom-1, 0)
	log.Println("Zoom level", zoom, "has", n, "iterations")
	return m.CoarseMeshes[utils.Min(zoom, len(m.CoarseMeshes)-1)], n
}
*/

func (m *Map) generateMap() {
	// Build geography / geology / climate.
	m.generateGeology()

	// Build civilization.
	m.generateCivilization()

	// Build plants / animals / funghi.
	m.generateBiology()
}

// Tick advances the map by one tick.
func (m *Map) Tick() {
	m.Geo.Tick()
	m.Bio.Tick()
	m.Civ.Tick()
}

/*
// TileCache is a cache for tiles stored as a quadtree.
//
// This cache is used to retrieve tiles for rendering.
// If a tile is not found in the cache, the closest parent is returned, and
// the tile is generated in the background.
type TileCache struct {
	*TileTree
}

func NewTileCache(bo *BaseObject) *TileCache {
	return &TileCache{
		TileTree: &TileTree{
			BaseObject: bo,
		},
	}
}

// TileTree is a quadtree of tiles.
type TileTree struct {
	Parent     *TileTree
	Children   [4]*TileTree
	X, Y, Zoom int
	*BaseObject
}

func (t *TileTree) Get(x, y, zoom int) *TileTree {
	x, y = wrapTileCoordinates(x, y, zoom)

	log.Println("Get tile:", x, y, zoom)
	// If the tile is not in the quadtree, return.
	if zoom < t.Zoom {
		return nil
	}

	// If the tile is exactly the one requested, return.
	if zoom == t.Zoom && x == t.X && y == t.Y {
		log.Println("Found tile in cache:", t.X, t.Y, t.Zoom)
		return t
	}

	// Check if the tile is a parent of the tile we are looking for.
	xAtCurrentTile := x >> uint(zoom-t.Zoom)
	yAtCurrentTile := y >> uint(zoom-t.Zoom)
	if xAtCurrentTile != t.X || yAtCurrentTile != t.Y {
		return nil
	}

	// Find the points in the child quadtrees.
	// Get the index of the child that will contain the tile that we are looking for.
	childIndex := 0
	xChild := x >> uint(zoom-t.Zoom-1)
	yChild := y >> uint(zoom-t.Zoom-1)

	if xChild%2 == 1 {
		childIndex += 1
	}
	if yChild%2 == 1 {
		childIndex += 2
	}

	// If the child is not nil, find the points in the child.
	if t.Children[childIndex] != nil {
		log.Println("Cache hit: Returning child tile", xChild, yChild, zoom+1)
		return t.Children[childIndex].Get(x, y, zoom)
	}

	log.Println("Cache miss: Generating tile", xChild, yChild, zoom+1)
	tbb := newTileBoundingBox(xChild, yChild, t.Zoom+1)
	la1, lo1, la2, lo2 := tbb.toLatLon()
	// Make sure we have some padding around the tile.
	la1 -= 0.1
	lo1 -= 0.1
	la2 += 0.1
	lo2 += 0.1
	res := t.BaseObject.getBoundingBoxRegions(la1, lo1, la2, lo2)
	childZoom := t.Zoom + 1
	var baseObject *BaseObject
	if childZoom%2 == 1 {
		ipl, err := t.BaseObject.interpolate(res.Regions)
		if err != nil {
			return nil
		}
		baseObject = &ipl.BaseObject
	} else {
		baseObject = t.BaseObject
	}

	// If the child is nil, return the current tile.
	// TODO: GENERATE THE TILE AND CACHE IT.
	t.Children[childIndex] = &TileTree{
		Parent:     t,
		X:          xChild,
		Y:          yChild,
		Zoom:       childZoom,
		BaseObject: baseObject,
	}
	return t.Children[childIndex].Get(x, y, zoom)
}
*/
