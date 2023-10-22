package genworldvoronoi

import (
	"encoding/binary"
	"io"

	"github.com/Flokey82/genworldvoronoi/various"
)

var byteorder = binary.LittleEndian

func (tm *TriangleMesh) writeTo(w io.Writer) error {
	// Write the number of regions, sides, and triangles
	if err := binary.Write(w, byteorder, int64(tm.numRegions)); err != nil {
		return err
	}
	if err := binary.Write(w, byteorder, int64(tm.numSides)); err != nil {
		return err
	}
	if err := binary.Write(w, byteorder, int64(tm.numTriangles)); err != nil {
		return err
	}
	if err := binary.Write(w, byteorder, int64(tm.numHalfedges)); err != nil {
		return err
	}

	// Write the triangles
	if err := various.WriteIntSlice(w, tm.Triangles); err != nil {
		return err
	}

	// Write the halfedges
	if err := various.WriteIntSlice(w, tm.Halfedges); err != nil {
		return err
	}

	// Write the region-in-side index
	if err := various.WriteIntSlice(w, tm.RegInSide); err != nil {
		return err
	}

	return nil
}

func readTriangleMesh(r io.Reader) (*TriangleMesh, error) {
	tm := &TriangleMesh{}

	// Read the number of regions, sides, and triangles
	if err := binary.Read(r, byteorder, &tm.numRegions); err != nil {
		return nil, err
	}
	if err := binary.Read(r, byteorder, &tm.numSides); err != nil {
		return nil, err
	}
	if err := binary.Read(r, byteorder, &tm.numTriangles); err != nil {
		return nil, err
	}
	if err := binary.Read(r, byteorder, &tm.numHalfedges); err != nil {
		return nil, err
	}

	// Read the triangles
	tri, err := various.ReadIntSlice(r)
	if err != nil {
		return nil, err
	}
	tm.Triangles = tri

	// Read the halfedges
	he, err := various.ReadIntSlice(r)
	if err != nil {
		return nil, err
	}
	tm.Halfedges = he

	// Read the region-in-side index
	ris, err := various.ReadIntSlice(r)
	if err != nil {
		return nil, err
	}
	tm.RegInSide = ris

	return tm, nil
}
