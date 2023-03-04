package genworldvoronoi

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

// Each tile is a specially-encoded triangle mesh where vertices overlap their
// neighbors at tile edges. In other words, at the root, the eastern-most vertices
// in the western tile have the same longitude as the western-most vertices in the
// eastern tile.
//
// Terrain tiles are served gzipped. Once extracted, tiles are little-endian, binary data.
type Tile3D struct {
	Header QuantizedMeshHeader
	Vertex VertexData
	Index  IndexData
	Edge   EdgeIndices
}

func new3DTile(x, y, zoom int) *Tile3D {
	t := &Tile3D{}
	t.Header.CenterX, t.Header.CenterY, t.Header.CenterZ = tile2xyz(x, y, zoom)
	t.Header.BoundingSphereCenterX, t.Header.BoundingSphereCenterY, t.Header.BoundingSphereCenterZ = t.Header.CenterX, t.Header.CenterY, t.Header.CenterZ
	t.Header.BoundingSphereRadius = 6378137.0
	t.Header.HorizonOcclusionPointX, t.Header.HorizonOcclusionPointY, t.Header.HorizonOcclusionPointZ = t.Header.CenterX, t.Header.CenterY, t.Header.CenterZ
	return t
}

func (t *Tile3D) Write(w *bytes.Buffer) error {
	if err := binary.Write(w, byteorder, t.Header); err != nil {
		return err
	}
	if err := t.Vertex.Write(w); err != nil {
		return err
	}
	// Make sure to add padding bytes if necessary so we are 2 byte aligned if
	// we use 16 bit indices and 4 byte aligned if we use 32 bit indices.
	if t.Vertex.VertexCount > 65535 {
		if rem := len(w.Bytes()) % 4; rem != 0 {
			w.Write(make([]byte, 4-rem))
		}
	} else {
		if rem := len(w.Bytes()) % 2; rem != 0 {
			w.Write(make([]byte, 2-rem))
		}
	}
	if err := t.Index.Write(w, t.Vertex.VertexCount); err != nil {
		return err
	}
	if err := t.Edge.Write(w, t.Vertex.VertexCount); err != nil {
		return err
	}
	return nil
}

func tile2xyz(x, y, zoom int) (float64, float64, float64) {
	n := math.Pow(2, float64(zoom))
	lon := float64(x)/n*360.0 - 180.0
	lat := math.Atan(math.Sinh(math.Pi * (1 - 2*float64(y)/n)))
	return lon, lat, 0
}

type QuantizedMeshHeader struct {
	// The center of the tile in Earth-centered Fixed coordinates.
	CenterX float64
	CenterY float64
	CenterZ float64

	// The minimum and maximum heights in the area covered by this tile.
	// The minimum may be lower and the maximum may be higher than
	// the height of any vertex in this tile in the case that the min/max vertex
	// was removed during mesh simplification, but these are the appropriate
	// values to use for analysis or visualization.
	MinimumHeight float32
	MaximumHeight float32

	// The tile’s bounding sphere.  The X,Y,Z coordinates are again expressed
	// in Earth-centered Fixed coordinates, and the radius is in meters.
	BoundingSphereCenterX float64
	BoundingSphereCenterY float64
	BoundingSphereCenterZ float64
	BoundingSphereRadius  float64

	// The horizon occlusion point, expressed in the ellipsoid-scaled Earth-centered Fixed frame.
	// If this point is below the horizon, the entire tile is below the horizon.
	// See http://cesiumjs.org/2013/04/25/Horizon-culling/ for more information.
	HorizonOcclusionPointX float64
	HorizonOcclusionPointY float64
	HorizonOcclusionPointZ float64
}

// The vertexCount field indicates the size of the three arrays that follow.
// The three arrays contain the delta from the previous value that is then zig-zag
// encoded in order to make small integers, regardless of their sign, use a small
// number of bits.
type VertexData struct {
	VertexCount uint32
	// The horizontal coordinate of the vertex in the tile.
	// - When the u value is 0, the vertex is on the Western edge of the tile.
	// - When the value is 32767, the vertex is on the Eastern edge of the tile.
	// - For other values, the vertex's longitude is a linear interpolation between
	// the longitudes of the Western and Eastern edges of the tile.
	U []uint16 // [vertexCount]uint16
	// The vertical coordinate of the vertex in the tile.
	// - When the v value is 0, the vertex is on the Southern edge of the tile.
	// - When the value is 32767, the vertex is on the Northern edge of the tile.
	// - For other values, the vertex's latitude is a linear interpolation between
	// the latitudes of the Southern and Nothern edges of the tile.
	V []uint16 // [vertexCount]uint16
	// The height of the vertex in the tile.
	// - When the height value is 0, the vertex's height is equal to the minimum
	// height within the tile, as specified in the tile's header.
	// - When the value is 32767, the vertex's height is equal to the maximum height
	// within the tile.
	// - For other values, the vertex's height is a linear interpolation between the
	// minimum and maximum heights.
	Height []uint16 // [vertexCount]uint16
}

func (vd *VertexData) Write(w io.Writer) error {
	if err := binary.Write(w, byteorder, vd.VertexCount); err != nil {
		return err
	}
	if err := binary.Write(w, byteorder, vd.U); err != nil {
		return err
	}
	if err := binary.Write(w, byteorder, vd.V); err != nil {
		return err
	}
	if err := binary.Write(w, byteorder, vd.Height); err != nil {
		return err
	}
	return nil
}

// Immediately following the vertex data is the index data.
// Indices specify how the vertices are linked together into triangles.
// If tile has more than 65536 vertices, the tile uses the IndexData32 structure
// to encode indices. Otherwise, it uses the IndexData16 structure.
//
// To enforce proper byte alignment, padding is added before the IndexData to ensure
// 2 byte alignment for IndexData16 and 4 byte alignment for IndexData32.
//
// Indices are encoded using the high water mark encoding from webgl-loader.
// Each triplet of indices specifies one triangle to be rendered, in counter-clockwise winding order.
type IndexData struct {
	TriangleCount uint32

	// Either Indices16 or Indices32 is used, depending on the number of vertices.
	// Indices16 []uint16 // [triangleCount * 3]uint16
	Indices []uint32 // [triangleCount * 3]uint32
}

func (id *IndexData) Write(w io.Writer, vertexCount uint32) error {
	if err := binary.Write(w, byteorder, id.TriangleCount); err != nil {
		return err
	}
	if vertexCount > 65536 {
		if err := binary.Write(w, byteorder, id.Indices); err != nil {
			return err
		}
	} else {
		indices16 := make([]uint16, len(id.Indices))
		for i, v := range id.Indices {
			indices16[i] = uint16(v)
		}
		if err := binary.Write(w, byteorder, indices16); err != nil {
			return err
		}
	}
	return nil
}

// Following the triangle indices is four more lists of indices:
// Depending on the number of vertices, the tile uses either EdgeIndices16 or EdgeIndices32.
type EdgeIndices struct {
	WestVertexCount  uint32
	WestIndices      []uint32 // [westVertexCount]uint32
	SouthVertexCount uint32
	SouthIndices     []uint32 // [southVertexCount]uint32
	EastVertexCount  uint32
	EastIndices      []uint32 // [eastVertexCount]uint32
	NorthVertexCount uint32
	NorthIndices     []uint32 // [northVertexCount]uint32
}

func (ei *EdgeIndices) Write(w io.Writer, vertexCount uint32) error {
	if vertexCount > 65536 {
		if err := binary.Write(w, byteorder, ei.WestVertexCount); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, ei.WestIndices); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, ei.SouthVertexCount); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, ei.SouthIndices); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, ei.EastVertexCount); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, ei.EastIndices); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, ei.NorthVertexCount); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, ei.NorthIndices); err != nil {
			return err
		}
	} else {
		if err := binary.Write(w, byteorder, ei.WestVertexCount); err != nil {
			return err
		}
		indices16 := make([]uint16, len(ei.WestIndices))
		for i, v := range ei.WestIndices {
			indices16[i] = uint16(v)
		}
		if err := binary.Write(w, byteorder, indices16); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, ei.SouthVertexCount); err != nil {
			return err
		}
		indices16 = make([]uint16, len(ei.SouthIndices))
		for i, v := range ei.SouthIndices {
			indices16[i] = uint16(v)
		}
		if err := binary.Write(w, byteorder, indices16); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, ei.EastVertexCount); err != nil {
			return err
		}
		indices16 = make([]uint16, len(ei.EastIndices))
		for i, v := range ei.EastIndices {
			indices16[i] = uint16(v)
		}
		if err := binary.Write(w, byteorder, indices16); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, ei.NorthVertexCount); err != nil {
			return err
		}
		indices16 = make([]uint16, len(ei.NorthIndices))
		for i, v := range ei.NorthIndices {
			indices16[i] = uint16(v)
		}
		if err := binary.Write(w, byteorder, indices16); err != nil {
			return err
		}
	}
	return nil
}
