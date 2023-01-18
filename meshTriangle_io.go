package genworldvoronoi

import (
	"encoding/binary"
	"io"
)

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
	if err := writeIntSlice(w, tm.Triangles); err != nil {
		return err
	}

	// Write the halfedges
	if err := writeIntSlice(w, tm.Halfedges); err != nil {
		return err
	}

	// Write the region-in-side index
	if err := writeIntSlice(w, tm.RegInSide); err != nil {
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
	tri, err := readIntSlice(r)
	if err != nil {
		return nil, err
	}
	tm.Triangles = tri

	// Read the halfedges
	he, err := readIntSlice(r)
	if err != nil {
		return nil, err
	}
	tm.Halfedges = he

	// Read the region-in-side index
	ris, err := readIntSlice(r)
	if err != nil {
		return nil, err
	}
	tm.RegInSide = ris

	return tm, nil
}
