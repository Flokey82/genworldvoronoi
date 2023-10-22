package various

import (
	"encoding/binary"
	"io"
	"sort"
)

var byteorder = binary.LittleEndian

func WriteMapIntInt(w io.Writer, m map[int]int) error {
	if err := binary.Write(w, byteorder, int64(len(m))); err != nil {
		return err
	}
	// Get the sorted keys.
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Write the keys and values.
	for _, k := range keys {
		if err := binary.Write(w, byteorder, int64(k)); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, int64(m[k])); err != nil {
			return err
		}
	}
	return nil
}

func ReadMapIntInt(r io.Reader) (map[int]int, error) {
	m := make(map[int]int)
	var num int64
	if err := binary.Read(r, byteorder, &num); err != nil {
		return nil, err
	}
	for i := 0; i < int(num); i++ {
		var k, v int64
		if err := binary.Read(r, byteorder, &k); err != nil {
			return nil, err
		}
		if err := binary.Read(r, byteorder, &v); err != nil {
			return nil, err
		}
		m[int(k)] = int(v)
	}
	return m, nil
}

func WriteMapIntFloat64(w io.Writer, m map[int]float64) error {
	if err := binary.Write(w, byteorder, int64(len(m))); err != nil {
		return err
	}
	// Get the sorted keys.
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Write the keys and values.
	for _, k := range keys {
		if err := binary.Write(w, byteorder, int64(k)); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, m[k]); err != nil {
			return err
		}
	}
	return nil
}

func ReadMapIntFloat64(r io.Reader) (map[int]float64, error) {
	m := make(map[int]float64)
	var num int64
	if err := binary.Read(r, byteorder, &num); err != nil {
		return nil, err
	}
	for i := 0; i < int(num); i++ {
		var k int64
		var v float64
		if err := binary.Read(r, byteorder, &k); err != nil {
			return nil, err
		}
		if err := binary.Read(r, byteorder, &v); err != nil {
			return nil, err
		}
		m[int(k)] = v
	}
	return m, nil
}

func WriteMapIntBool(w io.Writer, m map[int]bool) error {
	if err := binary.Write(w, byteorder, int64(len(m))); err != nil {
		return err
	}
	// Get the sorted keys.
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Write the keys and values.
	for _, k := range keys {
		if err := binary.Write(w, byteorder, int64(k)); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, m[k]); err != nil {
			return err
		}
	}
	return nil
}

func ReadMapIntBool(r io.Reader) (map[int]bool, error) {
	m := make(map[int]bool)
	var num int64
	if err := binary.Read(r, byteorder, &num); err != nil {
		return nil, err
	}
	for i := 0; i < int(num); i++ {
		var k int64
		var v bool
		if err := binary.Read(r, byteorder, &k); err != nil {
			return nil, err
		}
		if err := binary.Read(r, byteorder, &v); err != nil {
			return nil, err
		}
		m[int(k)] = v
	}
	return m, nil
}

func WriteFloatSlice(w io.Writer, s []float64) error {
	if err := binary.Write(w, byteorder, int64(len(s))); err != nil {
		return err
	}
	for _, v := range s {
		if err := binary.Write(w, byteorder, v); err != nil {
			return err
		}
	}
	return nil
}

func ReadFloatSlice(r io.Reader) ([]float64, error) {
	var num int64
	if err := binary.Read(r, byteorder, &num); err != nil {
		return nil, err
	}
	s := make([]float64, num)
	for i := 0; i < int(num); i++ {
		if err := binary.Read(r, byteorder, &s[i]); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func Write2FloatSlice(w io.Writer, s [][2]float64) error {
	if err := binary.Write(w, byteorder, int64(len(s))); err != nil {
		return err
	}
	for _, v := range s {
		if err := binary.Write(w, byteorder, v[0]); err != nil {
			return err
		}
		if err := binary.Write(w, byteorder, v[1]); err != nil {
			return err
		}
	}
	return nil
}

func Read2FloatSlice(r io.Reader) ([][2]float64, error) {
	var num int64
	if err := binary.Read(r, byteorder, &num); err != nil {
		return nil, err
	}
	s := make([][2]float64, num)
	for i := 0; i < int(num); i++ {
		if err := binary.Read(r, byteorder, &s[i][0]); err != nil {
			return nil, err
		}
		if err := binary.Read(r, byteorder, &s[i][1]); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func WriteIntSlice(w io.Writer, s []int) error {
	if err := binary.Write(w, byteorder, int64(len(s))); err != nil {
		return err
	}
	for _, v := range s {
		if err := binary.Write(w, byteorder, int64(v)); err != nil {
			return err
		}
	}
	return nil
}

func ReadIntSlice(r io.Reader) ([]int, error) {
	var num int64
	if err := binary.Read(r, byteorder, &num); err != nil {
		return nil, err
	}
	s := make([]int, num)
	for i := 0; i < int(num); i++ {
		if err := binary.Read(r, byteorder, &s[i]); err != nil {
			return nil, err
		}
	}
	return s, nil
}
