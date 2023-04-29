package genworldvoronoi

import (
	"encoding/binary"
	"io"
)

// writeTo writes the mesh to the given writer.
func (m *SphereMesh) writeTo(w io.Writer) error {
	// Write the number of XYZ coordinates.
	if err := binary.Write(w, byteorder, uint64(len(m.XYZ))); err != nil {
		return err
	}

	// Write the XYZ coordinates.
	if err := binary.Write(w, byteorder, m.XYZ); err != nil {
		return err
	}

	// Write the number of LatLon coordinates.
	if err := binary.Write(w, byteorder, uint64(len(m.LatLon))); err != nil {
		return err
	}

	// Write the LatLon coordinates.
	if err := binary.Write(w, byteorder, m.LatLon); err != nil {
		return err
	}

	// Write the number of triangle XYZ coordinates.
	if err := binary.Write(w, byteorder, uint64(len(m.TriXYZ))); err != nil {
		return err
	}

	// Write the triangle XYZ coordinates.
	if err := binary.Write(w, byteorder, m.TriXYZ); err != nil {
		return err
	}

	// Write the number of triangle LatLon coordinates.
	if err := binary.Write(w, byteorder, uint64(len(m.TriLatLon))); err != nil {
		return err
	}

	// Write the triangle LatLon coordinates.
	if err := binary.Write(w, byteorder, m.TriLatLon); err != nil {
		return err
	}

	// Write the triangle mesh.
	return m.TriangleMesh.writeTo(w)
}

// readSphereMesh reads a sphere mesh from the given reader.
func readSphereMesh(r io.Reader) (*SphereMesh, error) {
	// Read the number of XYZ coordinates.
	var numXYZ uint64
	if err := binary.Read(r, byteorder, &numXYZ); err != nil {
		return nil, err
	}

	// Read the XYZ coordinates.
	xyz := make([]float64, numXYZ)
	if err := binary.Read(r, byteorder, xyz); err != nil {
		return nil, err
	}

	// Read the number of LatLon coordinates.
	var numLatLon uint64
	if err := binary.Read(r, byteorder, &numLatLon); err != nil {
		return nil, err
	}

	// Read the LatLon coordinates.
	latLon := make([][2]float64, numLatLon)
	if err := binary.Read(r, byteorder, latLon); err != nil {
		return nil, err
	}

	// Read the number of triangle XYZ coordinates.
	var numTriXYZ uint64
	if err := binary.Read(r, byteorder, &numTriXYZ); err != nil {
		return nil, err
	}

	// Read the triangle XYZ coordinates.
	triXYZ := make([]float64, numTriXYZ)
	if err := binary.Read(r, byteorder, triXYZ); err != nil {
		return nil, err
	}

	// Read the number of triangle LatLon coordinates.
	var numTriLatLon uint64
	if err := binary.Read(r, byteorder, &numTriLatLon); err != nil {
		return nil, err
	}

	// Read the triangle LatLon coordinates.
	triLatLon := make([][2]float64, numTriLatLon)
	if err := binary.Read(r, byteorder, triLatLon); err != nil {
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
