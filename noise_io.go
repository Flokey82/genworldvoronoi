package genworldvoronoi

import (
	"encoding/binary"
	"io"

	"github.com/ojrac/opensimplex-go"
)

func (n *Noise) writeTo(w io.Writer) error {
	// Write the number of octaves, persistence, and amplitudes, as well as the
	// seed. From this, we can reconstruct the noise function.
	if err := binary.Write(w, byteorder, int64(n.Octaves)); err != nil {
		return err
	}
	if err := binary.Write(w, byteorder, n.Persistence); err != nil {
		return err
	}
	if err := binary.Write(w, byteorder, n.Seed); err != nil {
		return err
	}

	// Write the amplitudes.
	if err := writeFloatSlice(w, n.Amplitudes); err != nil {
		return err
	}
	return nil
}

func readNoise(r io.Reader) (*Noise, error) {
	n := &Noise{}

	// Read the number of octaves, persistence, and amplitudes, as well as the
	// seed. From this, we can reconstruct the noise function.
	if err := binary.Read(r, byteorder, &n.Octaves); err != nil {
		return nil, err
	}
	if err := binary.Read(r, byteorder, &n.Persistence); err != nil {
		return nil, err
	}
	if err := binary.Read(r, byteorder, &n.Seed); err != nil {
		return nil, err
	}

	// Read the amplitudes.
	amps, err := readFloatSlice(r)
	if err != nil {
		return nil, err
	}
	n.Amplitudes = amps

	n.OS = opensimplex.NewNormalized(n.Seed)

	return n, nil
}
