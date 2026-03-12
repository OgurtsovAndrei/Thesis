package are_optimized

import (
	"Thesis/bits"
	"Thesis/emptiness/ere"
	"fmt"
	"math"
	"sort"
)

type OptimizedApproximateRangeEmptiness struct {
	ere          *ere.ExactRangeEmptiness
	K            uint32
	RangeLen     uint64
	MinKey       bits.BitString
	TruncateBits uint32
	n            int
}

// hashBlock64 returns a seed for a block.
func hashBlock64(blockIdx uint64) uint64 {
	return blockIdx * 0x9E3779B185EBCA87
}

// hashBlockAny returns a seed for a block of arbitrary size.
func hashBlockAny(blockIdx bits.BitString) uint64 {
	return blockIdx.HashWithSeed(0x9E3779B185EBCA87)
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
	// Note: We use Prefix to match the expected behavior of ERE
	return bits.NewFromUint64(reversed).Prefix(int(K))
}

func NewOptimizedARE(keys []bits.BitString, rangeLen uint64, epsilon float64, t uint32) (*OptimizedApproximateRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &OptimizedApproximateRangeEmptiness{n: 0}, nil
	}

	// 1. Find MinKey for Normalization
	minKey := keys[0]
	for _, k := range keys {
		if k.Compare(minKey) < -1 {
			minKey = k
		}
	}

	// 2. Adjust RangeLen for truncation
	// If we truncate t bits, a range of length L in the original space 
	// spans at most ceil(L / 2^t) + 1 points in the truncated space.
	effectiveRangeLen := (rangeLen >> t) + 1

	// 3. Calculate K for SODA (r = 2^K)
	rTarget := float64(n) * float64(effectiveRangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		return nil, fmt.Errorf("K exceeds 64 bits: %d. Try increasing truncation t", K)
	}

	rMask := (uint64(1) << K) - 1
	if K == 64 {
		rMask = ^uint64(0)
	}

	hashedKeys := make([]bits.BitString, n)
	for i, x := range keys {
		// Normalization & Truncation: x' = (x - min) >> t
		xPrime := x.Sub(minKey).ShiftRight(t)
		
		// SODA Hashing
		// blockIdx = xPrime >> K
		var blockIdx bits.BitString
		var xPrimeUint64 uint64
		
		if xPrime.SizeBits() > K {
			blockIdx = xPrime.ShiftRight(K)
			// We take last K bits of xPrime for the 'offset' part of SODA
			// Since SODA is (u + x) mod r, we need x as uint64
			xPrimeUint64 = xPrime.Word(0)
		} else {
			blockIdx = bits.NewBitString(0)
			xPrimeUint64 = xPrime.Word(0)
		}

		u := hashBlockAny(blockIdx)
		hx := (u + xPrimeUint64) & rMask
		hashedKeys[i] = toKBitString(hx, K)
	}

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

	ereFilter, err := ere.NewExactRangeEmptiness(uniqueHashed, bits.NewBitString(K))
	if err != nil {
		return nil, err
	}

	return &OptimizedApproximateRangeEmptiness{
		ere:          ereFilter,
		K:            K,
		RangeLen:     rangeLen,
		MinKey:       minKey,
		TruncateBits: t,
		n:            n,
	}, nil
}

func (are *OptimizedApproximateRangeEmptiness) IsEmpty(a, b bits.BitString) bool {
	if are.n == 0 || a.Compare(b) > 0 {
		return true
	}

	// 1. Normalization & Truncation
	// a' = (a - min) >> t, b' = (b - min) >> t
	// If a < MinKey, we clamp to 0.
	var aPrime, bPrime bits.BitString
	if a.Compare(are.MinKey) < 0 {
		aPrime = bits.NewBitString(a.SizeBits()).ShiftRight(are.TruncateBits)
	} else {
		aPrime = a.Sub(are.MinKey).ShiftRight(are.TruncateBits)
	}
	
	if b.Compare(are.MinKey) < 0 {
		// Entire range is before our known data
		return true
	}
	bPrime = b.Sub(are.MinKey).ShiftRight(are.TruncateBits)

	// Now we perform SODA IsEmpty on aPrime, bPrime
	return are.sodaIsEmpty(aPrime, bPrime)
}

func (are *OptimizedApproximateRangeEmptiness) sodaIsEmpty(a, b bits.BitString) bool {
	rMask := (uint64(1) << are.K) - 1
	if are.K == 64 {
		rMask = ^uint64(0)
	}

	// Split into blocks of size 2^K
	blockA := bits.NewBitString(0)
	if a.SizeBits() > are.K {
		blockA = a.ShiftRight(are.K)
	}
	
	blockB := bits.NewBitString(0)
	if b.SizeBits() > are.K {
		blockB = b.ShiftRight(are.K)
	}

	if blockA.Equal(blockB) {
		u := hashBlockAny(blockA)
		hA := (u + are.extractLowK(a)) & rMask
		hB := (u + are.extractLowK(b)) & rMask
		
		if hA <= hB {
			return are.ere.IsEmpty(toKBitString(hA, are.K), toKBitString(hB, are.K))
		} else {
			if !are.ere.IsEmpty(toKBitString(hA, are.K), toKBitString(rMask, are.K)) {
				return false
			}
			return are.ere.IsEmpty(toKBitString(0, are.K), toKBitString(hB, are.K))
		}
	}

	// Multi-block: for simplicity and 0% FN, we check boundaries 
	// and assume intermediate blocks are non-empty if filter has keys.
	// Boundary A
	uA := hashBlockAny(blockA)
	hA_start := (uA + are.extractLowK(a)) & rMask
	hA_end := (uA + rMask) & rMask // Approximate end of block
	
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

	// Intermediate full blocks
	// If there's any gap between blockA and blockB, we return false (conservative)
	// because a full block maps to the entire hashed universe.
	if !are.ere.IsEmpty(toKBitString(0, are.K), toKBitString(rMask, are.K)) {
		return false
	}

	// Boundary B
	uB := hashBlockAny(blockB)
	hB_start := (uB + 0) & rMask
	hB_end := (uB + are.extractLowK(b)) & rMask
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

func (are *OptimizedApproximateRangeEmptiness) extractLowK(bs bits.BitString) uint64 {
	return bs.Word(0)
}

func (are *OptimizedApproximateRangeEmptiness) SizeInBits() uint64 {
	if are.ere == nil {
		return 0
	}
	return are.ere.SizeInBits()
}
