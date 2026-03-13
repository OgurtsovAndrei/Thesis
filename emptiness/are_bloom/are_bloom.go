package are_bloom

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/bits-and-blooms/bloom/v3"
)

// BloomARE is a baseline range emptiness filter that answers range queries
// by performing L individual Bloom filter point lookups.
type BloomARE struct {
	filter *bloom.BloomFilter
}

func NewBloomARE(keys []uint64, rangeLen uint64, epsilon float64) (*BloomARE, error) {
	if len(keys) == 0 {
		return &BloomARE{filter: bloom.NewWithEstimates(1, epsilon)}, nil
	}

	pointFPR := 1 - math.Pow(1-epsilon, 1.0/float64(rangeLen))
	if pointFPR <= 0 || math.IsNaN(pointFPR) {
		return nil, fmt.Errorf("bloom: point FPR underflow for ε=%g, L=%d", epsilon, rangeLen)
	}

	bf := bloom.NewWithEstimates(uint(len(keys)), pointFPR)
	var buf [8]byte
	for _, k := range keys {
		binary.LittleEndian.PutUint64(buf[:], k)
		bf.Add(buf[:])
	}
	return &BloomARE{filter: bf}, nil
}

func (b *BloomARE) IsEmpty(a, bEnd uint64) bool {
	var buf [8]byte
	for x := a; x <= bEnd; x++ {
		binary.LittleEndian.PutUint64(buf[:], x)
		if b.filter.Test(buf[:]) {
			return false
		}
		if x == math.MaxUint64 {
			break
		}
	}
	return true
}

func (b *BloomARE) SizeInBits() uint64 {
	return uint64(b.filter.Cap())
}
