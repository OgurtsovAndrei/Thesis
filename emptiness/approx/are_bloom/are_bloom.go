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

	return NewBloomAREFromPointFPR(keys, pointFPR)
}

// NewBloomAREFromPointFPR builds the bloom filter for a target per-point
// FPR, directly. The structure is L-independent: a query [a, a+L-1] is
// answered by L successive bloom probes on the same filter, so range
// FPR = 1 - (1-pointFPR)^L is computed implicitly at query time, never
// baked into the filter.
//
// Use this constructor when the bench wants to reuse one Bloom across
// many L values; pass the per-point FPR you want, not the per-range eps.
func NewBloomAREFromPointFPR(keys []uint64, pointFPR float64) (*BloomARE, error) {
	if pointFPR <= 0 || pointFPR >= 1 || math.IsNaN(pointFPR) {
		return nil, fmt.Errorf("bloom: invalid pointFPR %g", pointFPR)
	}
	if len(keys) == 0 {
		return &BloomARE{filter: bloom.NewWithEstimates(1, pointFPR)}, nil
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
