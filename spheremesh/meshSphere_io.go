package spheremesh

import (
	"io"

	"github.com/Flokey82/genworldvoronoi/various"
)

// WriteTo writes the mesh to the given writer.
func (m *SphereMesh) WriteTo(w io.Writer) error {
	// Write the XYZ coordinates.
	if err := various.WriteFloatSlice(w, m.XYZ); err != nil {
		return err
	}

	// Write the LatLon coordinates.
	if err := various.Write2FloatSlice(w, m.LatLon); err != nil {
		return err
	}

	// Write the triangle XYZ coordinates.
	if err := various.WriteFloatSlice(w, m.TriXYZ); err != nil {
		return err
	}

	// Write the triangle LatLon coordinates.
	if err := various.Write2FloatSlice(w, m.TriLatLon); err != nil {
		return err
	}

	// Write the triangle mesh.
	return m.TriangleMesh.writeTo(w)
}

// ReadSphereMesh reads a sphere mesh from the given reader.
func ReadSphereMesh(r io.Reader) (*SphereMesh, error) {
	// Read the XYZ coordinates.
	xyz, err := various.ReadFloatSlice(r)
	if err != nil {
		return nil, err
	}

	// Read the LatLon coordinates.
	latLon, err := various.Read2FloatSlice(r)
	if err != nil {
		return nil, err
	}

	// Read the triangle XYZ coordinates.
	triXYZ, err := various.ReadFloatSlice(r)
	if err != nil {
		return nil, err
	}

	// Read the triangle LatLon coordinates.
	triLatLon, err := various.Read2FloatSlice(r)
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
		RegQuadTree:  NewQuadTreeFromLatLon(latLon),
		TriQuadTree:  NewQuadTreeFromLatLon(triLatLon),
	}, nil
}
