package are_soda_hash

import (
	"Thesis/bits"
	"Thesis/emptiness/ere"
	"fmt"
	"math"
	"sort"
)

type ApproximateRangeEmptinessSoda struct {
	ere      *ere.ExactRangeEmptiness
	K        uint32
	RangeLen uint64
	n        int
}

// hashBlock is a pairwise independent hash for blocks.
// We use a simple multiplicative hash with a large prime-like constant.
func hashBlock(blockIdx uint64) uint64 {
	return blockIdx * 0x9E3779B185EBCA87
}

func toKBitString(val uint64, K uint32) bits.BitString {
	if K == 0 {
		return bits.NewBitString(0)
	}
	var reversed uint64
	for i := uint32(0); i < K; i++ {
		if (val & (uint64(1) << (K - 1 - i))) != 0 {
			reversed |= (uint64(1) << i)
		}
	}
	return bits.NewFromUint64(reversed).Prefix(int(K))
}

func NewApproximateRangeEmptinessSoda(keys []uint64, rangeLen uint64, epsilon float64) (*ApproximateRangeEmptinessSoda, error) {
	n := len(keys)
	if n == 0 {
		return &ApproximateRangeEmptinessSoda{n: 0, RangeLen: rangeLen}, nil
	}

	// Calculate K where r = 2^K >= n * RangeLen / epsilon
	rTarget := float64(n) * float64(rangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		return nil, fmt.Errorf("K exceeds 64 bits: %d", K)
	}

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
		ux := hashBlock(blockIdx)
		hx := (ux + x) & rMask
		hashedKeys[i] = toKBitString(hx, K)
	}

	// ERE expects sorted keys
	sort.Slice(hashedKeys, func(i, j int) bool {
		return hashedKeys[i].Compare(hashedKeys[j]) < 0
	})
	
	uniqueHashed := make([]bits.BitString, 0, n)
	if len(hashedKeys) > 0 {
		uniqueHashed = append(uniqueHashed, hashedKeys[0])
		for i := 1; i < len(hashedKeys); i++ {
			if !hashedKeys[i].Equal(hashedKeys[i-1]) {
				uniqueHashed = append(uniqueHashed, hashedKeys[i])
			}
		}
	}

	universe := bits.NewBitString(K)
	ereFilter, err := ere.NewExactRangeEmptiness(uniqueHashed, universe)
	if err != nil {
		return nil, err
	}

	return &ApproximateRangeEmptinessSoda{
		ere:      ereFilter,
		K:        K,
		RangeLen: rangeLen,
		n:        n,
	}, nil
}

func (are *ApproximateRangeEmptinessSoda) IsEmpty(a, b uint64) bool {
	if are.n == 0 || a > b {
		return true
	}

	// SODA guarantees accuracy for ranges up to RangeLen.
	// For much larger ranges (approaching 2^K), FPR will degrade towards 100% 
	// because full blocks map to the entire hashed universe.
	
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
		// Case 1: Range is within a single SODA block
		u := hashBlock(blockA)
		hA := (u + a) & rMask
		hB := (u + b) & rMask
		
		// Note: SODA is a cyclic shift. If hA > hB, it wraps around r.
		if hA <= hB {
			return are.ere.IsEmpty(toKBitString(hA, are.K), toKBitString(hB, are.K))
		} else {
			// Wrapped range [hA, rMask] U [0, hB]
			if !are.ere.IsEmpty(toKBitString(hA, are.K), toKBitString(rMask, are.K)) {
				return false
			}
			return are.ere.IsEmpty(toKBitString(0, are.K), toKBitString(hB, are.K))
		}
	}

	// Case 2: Multi-block range
	// 2.1 Check suffix of the first block
	uA := hashBlock(blockA)
	var maxA uint64
	if are.K == 64 {
		maxA = ^uint64(0)
	} else {
		maxA = ((blockA + 1) << are.K) - 1
	}
	hA_start := (uA + a) & rMask
	hA_end := (uA + maxA) & rMask
	if hA_start <= hA_end {
		if !are.ere.IsEmpty(toKBitString(hA_start, are.K), toKBitString(hA_end, are.K)) {
			return false
		}
	} else {
		if !are.ere.IsEmpty(toKBitString(hA_start, are.K), toKBitString(rMask, are.K)) ||
			!are.ere.IsEmpty(toKBitString(0, are.K), toKBitString(hA_end, are.K)) {
			return false
		}
	}

	// 2.2 Check intermediate full blocks
	// Optimization: If there's any key in the filter, any full block query returns false.
	// This is because a full block always maps to the entire hashed universe [0, rMask].
	if blockB > blockA+1 {
		if !are.ere.IsEmpty(toKBitString(0, are.K), toKBitString(rMask, are.K)) {
			return false
		}
	}

	// 2.3 Check prefix of the last block
	uB := hashBlock(blockB)
	var minB uint64
	if are.K < 64 {
		minB = blockB << are.K
	}
	hB_start := (uB + minB) & rMask
	hB_end := (uB + b) & rMask
	if hB_start <= hB_end {
		if !are.ere.IsEmpty(toKBitString(hB_start, are.K), toKBitString(hB_end, are.K)) {
			return false
		}
	} else {
		if !are.ere.IsEmpty(toKBitString(hB_start, are.K), toKBitString(rMask, are.K)) ||
			!are.ere.IsEmpty(toKBitString(0, are.K), toKBitString(hB_end, are.K)) {
			return false
		}
	}

	return true
}

func (are *ApproximateRangeEmptinessSoda) SizeInBits() uint64 {
	if are.ere == nil {
		return 0
	}
	return are.ere.SizeInBits()
}
