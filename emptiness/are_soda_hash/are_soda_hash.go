package are_soda_hash

import (
	"Thesis/bits"
	"Thesis/emptiness/ere"
	internalhash "Thesis/emptiness/internal/hash"
	"fmt"
	"math"
	"math/rand"
)

type SodaARE struct {
	ere      *ere.ExactRangeEmptiness
	K        uint32
	RangeLen uint64
	n        int
	hashA    uint64
	hashB    uint64
}

func NewSodaARE(keys []uint64, rangeLen uint64, epsilon float64) (*SodaARE, error) {
	n := len(keys)
	if n == 0 {
		return &SodaARE{n: 0, RangeLen: rangeLen}, nil
	}

	rTarget := float64(n) * float64(rangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		return nil, fmt.Errorf("K exceeds 64 bits: %d", K)
	}

	return NewSodaAREFromK(keys, rangeLen, K)
}

func NewSodaAREFromK(keys []uint64, rangeLen uint64, K uint32) (*SodaARE, error) {
	n := len(keys)
	if n == 0 {
		return &SodaARE{n: 0, RangeLen: rangeLen}, nil
	}
	if K > 64 {
		return nil, fmt.Errorf("K exceeds 64 bits: %d", K)
	}

	rng := rand.New(rand.NewSource(int64(n) ^ int64(rangeLen)))
	hashA := rng.Uint64() | 1 // odd
	hashB := rng.Uint64()

	rMask := ^uint64(0)
	if K < 64 {
		rMask = (uint64(1) << K) - 1
	}

	hashedKeys := make([]bits.BitString, n)
	for i, x := range keys {
		blockIdx := uint64(0)
		if K < 64 {
			blockIdx = x >> K
		}
		ux := internalhash.PairwiseHash(blockIdx, hashA, hashB, K)
		hx := (ux + x) & rMask
		hashedKeys[i] = bits.NewFromTrieUint64(hx, K)
	}

	uniqueHashed := internalhash.SortAndDedup(hashedKeys)

	universe := bits.NewBitString(K)
	ereFilter, err := ere.NewExactRangeEmptiness(uniqueHashed, universe)
	if err != nil {
		return nil, err
	}

	return &SodaARE{
		ere:      ereFilter,
		K:        K,
		RangeLen: rangeLen,
		n:        n,
		hashA:    hashA,
		hashB:    hashB,
	}, nil
}

func (are *SodaARE) IsEmpty(a, b uint64) bool {
	if are.n == 0 || a > b {
		return true
	}

	rMask := ^uint64(0)
	if are.K < 64 {
		rMask = (uint64(1) << are.K) - 1
	}

	blockA := uint64(0)
	if are.K < 64 {
		blockA = a >> are.K
	}

	blockB := uint64(0)
	if are.K < 64 {
		blockB = b >> are.K
	}

	toBS := func(val uint64) bits.BitString {
		return bits.NewFromTrieUint64(val, are.K)
	}

	if blockA == blockB {
		u := internalhash.PairwiseHash(blockA, are.hashA, are.hashB, are.K)
		hA := (u + a) & rMask
		hB := (u + b) & rMask

		if hA <= hB {
			return are.ere.IsEmpty(toBS(hA), toBS(hB))
		}
		// Wrapped range [hA, rMask] U [0, hB]
		if !are.ere.IsEmpty(toBS(hA), toBS(rMask)) {
			return false
		}
		return are.ere.IsEmpty(toBS(0), toBS(hB))
	}

	// Multi-block: check suffix of first block
	uA := internalhash.PairwiseHash(blockA, are.hashA, are.hashB, are.K)
	var maxA uint64
	if are.K == 64 {
		maxA = ^uint64(0)
	} else {
		maxA = ((blockA + 1) << are.K) - 1
	}
	hAStart := (uA + a) & rMask
	hAEnd := (uA + maxA) & rMask
	if hAStart <= hAEnd {
		if !are.ere.IsEmpty(toBS(hAStart), toBS(hAEnd)) {
			return false
		}
	} else {
		if !are.ere.IsEmpty(toBS(hAStart), toBS(rMask)) ||
			!are.ere.IsEmpty(toBS(0), toBS(hAEnd)) {
			return false
		}
	}

	// Intermediate full blocks
	if blockB > blockA+1 {
		if !are.ere.IsEmpty(toBS(0), toBS(rMask)) {
			return false
		}
	}

	// Prefix of last block
	uB := internalhash.PairwiseHash(blockB, are.hashA, are.hashB, are.K)
	var minB uint64
	if are.K < 64 {
		minB = blockB << are.K
	}
	hBStart := (uB + minB) & rMask
	hBEnd := (uB + b) & rMask
	if hBStart <= hBEnd {
		if !are.ere.IsEmpty(toBS(hBStart), toBS(hBEnd)) {
			return false
		}
	} else {
		if !are.ere.IsEmpty(toBS(hBStart), toBS(rMask)) ||
			!are.ere.IsEmpty(toBS(0), toBS(hBEnd)) {
			return false
		}
	}

	return true
}

func (are *SodaARE) SizeInBits() uint64 {
	if are.ere == nil {
		return 0
	}
	return are.ere.SizeInBits()
}
