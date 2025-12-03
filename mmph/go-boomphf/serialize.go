package boomphf

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// The serialization format for the H structure is as follows (all values are LittleEndian):
//
// Header:
// - uint32: Count of levels (L) in h.b
//
// Data for h.b (L iterations):
// - uint32: Length of the current bitvector (N_i)
// - N_i * uint64: The actual uint64 blocks of the bitvector
//
// Data for h.ranks:
// - uint32: Count of levels (L) in h.ranks (should match the first header value)
//
// Data for h.ranks (L iterations):
// - uint32: Length of the current rank slice (R_i)
// - R_i * uint64: The actual uint64 rank values
//
// Total Size: Size() + 8 + L * 8 bytes (where L is the number of levels)

func (h *H) Serialize() ([]byte, error) {
	var buf []byte

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(h.b)))

	for _, bv := range h.b {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(bv)))
		for _, v := range bv {
			buf = binary.LittleEndian.AppendUint64(buf, v)
		}
	}

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(h.ranks)))

	for _, rankSlice := range h.ranks {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(rankSlice)))
		for _, v := range rankSlice {
			buf = binary.LittleEndian.AppendUint64(buf, v)
		}
	}

	return buf, nil
}

func Deserialize(data []byte, target *H) error {
	r := bytes.NewReader(data)

	var bLevels uint32
	if err := binary.Read(r, binary.LittleEndian, &bLevels); err != nil {
		return errors.New("failed to read b levels count")
	}

	target.b = make([]bitvector, bLevels)
	for i := uint32(0); i < bLevels; i++ {
		var bvLen uint32
		if err := binary.Read(r, binary.LittleEndian, &bvLen); err != nil {
			return errors.New("failed to read bitvector length")
		}

		bv := make(bitvector, bvLen)
		for j := uint32(0); j < bvLen; j++ {
			var val uint64
			if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
				return errors.New("failed to read bitvector data")
			}
			bv[j] = val
		}
		target.b[i] = bv
	}

	var rankLevels uint32
	if err := binary.Read(r, binary.LittleEndian, &rankLevels); err != nil {
		return errors.New("failed to read ranks levels count")
	}

	target.ranks = make([][]uint64, rankLevels)
	for i := uint32(0); i < rankLevels; i++ {
		var rankLen uint32
		if err := binary.Read(r, binary.LittleEndian, &rankLen); err != nil {
			return errors.New("failed to read rank slice length")
		}

		rankSlice := make([]uint64, rankLen)
		for j := uint32(0); j < rankLen; j++ {
			var val uint64
			if err := binary.Read(r, binary.LittleEndian, &val); err != nil {
				return errors.New("failed to read rank slice data")
			}
			rankSlice[j] = val
		}
		target.ranks[i] = rankSlice
	}

	if r.Len() != 0 {
		return errors.New("trailing data after successful deserialization")
	}

	return nil
}
