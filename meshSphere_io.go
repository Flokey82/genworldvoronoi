package genworldvoronoi

import (
	"io"
)

// writeTo writes the mesh to the given writer.
func (m *SphereMesh) writeTo(w io.Writer) error {
	// Write the XYZ coordinates.
	if err := writeFloatSlice(w, m.XYZ); err != nil {
		return err
	}

	// Write the LatLon coordinates.
	if err := write2FloatSlice(w, m.LatLon); err != nil {
		return err
	}

	// Write the triangle XYZ coordinates.
	if err := writeFloatSlice(w, m.TriXYZ); err != nil {
		return err
	}

	// Write the triangle LatLon coordinates.
	if err := write2FloatSlice(w, m.TriLatLon); err != nil {
		return err
	}

	// Write the triangle mesh.
	return m.TriangleMesh.writeTo(w)
}

// readSphereMesh reads a sphere mesh from the given reader.
func readSphereMesh(r io.Reader) (*SphereMesh, error) {
	// Read the XYZ coordinates.
	xyz, err := readFloatSlice(r)
	if err != nil {
		return nil, err
	}

	// Read the LatLon coordinates.
	latLon, err := read2FloatSlice(r)
	if err != nil {
		return nil, err
	}

	// Read the triangle XYZ coordinates.
	triXYZ, err := readFloatSlice(r)
	if err != nil {
		return nil, err
	}

	// Read the triangle LatLon coordinates.
	triLatLon, err := read2FloatSlice(r)
	if err != nil {
		return nil, err
	}

	// Read the triangle mesh.
	tri, err := readTriangleMesh(r)
	if err != nil {
		return nil, err
	}

	return &SphereMesh{
		TriangleMesh: tri,
		XYZ:          xyz,
		LatLon:       latLon,
		TriXYZ:       triXYZ,
		TriLatLon:    triLatLon,
		regQuadTree:  newQuadTreeFromLatLon(latLon),
		triQuadTree:  newQuadTreeFromLatLon(triLatLon),
	}, nil
}
