package are_soda_hash

import (
	"Thesis/emptiness/exact/ere_one_d"
	internalhash "Thesis/emptiness/internal/hash"
	"fmt"
	"math"
	"math/rand"
)

type SodaARE struct {
	ere      *ere_one_d.ExactRangeEmptiness
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

	hashed := make([]uint64, n)
	for i, x := range keys {
		blockIdx := uint64(0)
		if K < 64 {
			blockIdx = x >> K
		}
		ux := internalhash.PairwiseHash(blockIdx, hashA, hashB, K)
		hx := (ux + x) & rMask
		hashed[i] = hx
	}

	uniqueHashed := internalhash.SortAndDedupUint64(hashed)

	ereFilter, err := ere_one_d.NewExactRangeEmptiness(uniqueHashed, K)
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

// NewSodaAREUint64 builds a SodaARE via the uint64 fast-path, avoiding the
// allocation of n bits.BitString values during construction. Semantics match
// NewSodaARE: input keys are not mutated.
func NewSodaAREUint64(keys []uint64, rangeLen uint64, epsilon float64) (*SodaARE, error) {
	n := len(keys)
	if n == 0 {
		return &SodaARE{n: 0, RangeLen: rangeLen}, nil
	}

	rTarget := float64(n) * float64(rangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
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

	hashed := make([]uint64, n)
	for i, x := range keys {
		blockIdx := uint64(0)
		if K < 64 {
			blockIdx = x >> K
		}
		ux := internalhash.PairwiseHash(blockIdx, hashA, hashB, K)
		hashed[i] = (ux + x) & rMask
	}

	uniqueHashed := internalhash.SortAndDedupUint64(hashed)

	ereFilter, err := ere_one_d.NewExactRangeEmptiness(uniqueHashed, K)
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

// NewSodaAREUint64InPlace is the destructive variant of NewSodaAREUint64. It
// hashes, sorts and deduplicates directly inside the caller-supplied keys
// slice, avoiding the per-key extra allocation. The input slice is consumed
// and mutated; callers must not rely on its post-call contents (neither
// values nor length-versus-capacity). This is the lowest-memory build path.
//
// Unlike NewSodaARE, K is supplied directly rather than derived from epsilon,
// matching NewSodaAREFromK.
func NewSodaAREUint64InPlace(keys []uint64, rangeLen uint64, K uint32) (*SodaARE, error) {
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

	for i, x := range keys {
		blockIdx := uint64(0)
		if K < 64 {
			blockIdx = x >> K
		}
		ux := internalhash.PairwiseHash(blockIdx, hashA, hashB, K)
		keys[i] = (ux + x) & rMask
	}

	uniqueHashed := internalhash.SortAndDedupUint64(keys)

	ereFilter, err := ere_one_d.NewExactRangeEmptiness(uniqueHashed, K)
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

	if blockA == blockB {
		u := internalhash.PairwiseHash(blockA, are.hashA, are.hashB, are.K)
		hA := (u + a) & rMask
		hB := (u + b) & rMask

		if hA <= hB {
			return are.ere.IsEmpty(hA, hB)
		}
		// Wrapped range [hA, rMask] U [0, hB]
		if !are.ere.IsEmpty(hA, rMask) {
			return false
		}
		return are.ere.IsEmpty(0, hB)
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
		if !are.ere.IsEmpty(hAStart, hAEnd) {
			return false
		}
	} else {
		if !are.ere.IsEmpty(hAStart, rMask) ||
			!are.ere.IsEmpty(0, hAEnd) {
			return false
		}
	}

	// Intermediate full blocks
	if blockB > blockA+1 {
		if !are.ere.IsEmpty(0, rMask) {
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
		if !are.ere.IsEmpty(hBStart, hBEnd) {
			return false
		}
	} else {
		if !are.ere.IsEmpty(hBStart, rMask) ||
			!are.ere.IsEmpty(0, hBEnd) {
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

func (are *SodaARE) EREStats() ere_one_d.Stats {
	if are.ere == nil {
		return ere_one_d.Stats{}
	}
	return are.ere.GetStats()
}

func (are *SodaARE) ERENonEmptyBlockSizes() []int {
	if are.ere == nil {
		return nil
	}
	return are.ere.NonEmptyBlockSizes()
}

func (are *SodaARE) ERE() *ere_one_d.ExactRangeEmptiness {
	return are.ere
}
