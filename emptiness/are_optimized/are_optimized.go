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
	IsExactMode  bool
	n            int
}

// hashBlockAny returns a seed for a block of arbitrary size.
func hashBlockAny(blockIdx bits.BitString) uint64 {
	return blockIdx.HashWithSeed(0x9E3779B185EBCA87)
}

func toKBitString(val uint64, K uint32) bits.BitString {
	if K == 0 {
		return bits.NewBitString(0)
	}
	// Important: toKBitString in our system means converting to BitString
	// and taking exactly K bits as a prefix for ERE.
	var reversed uint64
	for i := uint32(0); i < K; i++ {
		if (val & (uint64(1) << (K - 1 - i))) != 0 {
			reversed |= (uint64(1) << i)
		}
	}
	return bits.NewFromUint64(reversed).Prefix(int(K))
}

func toMBitString(val bits.BitString, M uint32) bits.BitString {
	// For Exact mode, we don't reverse bits, we just take the prefix.
	// But ERE expects MSB-first logic in its trie. 
	// Our BitString already stores words in Little-endian, but Prefix(M) takes 
	// the first M bits (indices 0 to M-1).
	return val.Prefix(int(M))
}

func NewOptimizedARE(keys []bits.BitString, rangeLen uint64, epsilon float64, t uint32) (*OptimizedApproximateRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &OptimizedApproximateRangeEmptiness{n: 0}, nil
	}

	// 1. Find Min and Max keys
	minKey := keys[0]
	maxKey := keys[0]
	for _, k := range keys {
		if k.Compare(minKey) < 0 {
			minKey = k
		}
		if k.Compare(maxKey) > 0 {
			maxKey = k
		}
	}

	// 2. Calculate actual data spread M (after truncation)
	spread := maxKey.Sub(minKey).ShiftRight(t)
	M := spread.BitLength()

	// 3. Calculate target K for SODA robustness (r = 2^K)
	effectiveRangeLen := (rangeLen >> t) + 1
	rTarget := float64(n) * float64(effectiveRangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		return nil, fmt.Errorf("required K=%d exceeds 64 bits. Increase truncation 't'", K)
	}

	// 4. Adaptive Choice: Exact vs Approximate
	isExactMode := (M <= K)
	
	// If exact, we only need M bits to distinguish keys perfectly.
	// If approximate, we need K bits to bound FPR.
	finalUniverseBits := K
	if isExactMode {
		finalUniverseBits = M
	}

	hashedKeys := make([]bits.BitString, n)
	for i, x := range keys {
		xPrime := x.Sub(minKey).ShiftRight(t)
		
		if isExactMode {
			// Exact Mode: No hashing, just use xPrime as is.
			hashedKeys[i] = toMBitString(xPrime, M)
		} else {
			// SODA Mode: (hash(blockIdx) + offset) mod 2^K
			rMask := (uint64(1) << K) - 1
			if K == 64 {
				rMask = ^uint64(0)
			}

			var blockIdx bits.BitString
			var offset uint64
			if xPrime.SizeBits() > K {
				blockIdx = xPrime.ShiftRight(K)
				offset = xPrime.Word(0) // low bits
			} else {
				blockIdx = bits.NewBitString(0)
				offset = xPrime.Word(0)
			}
			
			u := hashBlockAny(blockIdx)
			hx := (u + offset) & rMask
			hashedKeys[i] = toKBitString(hx, K)
		}
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

	ereFilter, err := ere.NewExactRangeEmptiness(uniqueHashed, bits.NewBitString(finalUniverseBits))
	if err != nil {
		return nil, err
	}

	return &OptimizedApproximateRangeEmptiness{
		ere:          ereFilter,
		K:            finalUniverseBits,
		RangeLen:     rangeLen,
		MinKey:       minKey,
		TruncateBits: t,
		IsExactMode:  isExactMode,
		n:            n,
	}, nil
}

func (are *OptimizedApproximateRangeEmptiness) IsEmpty(a, b bits.BitString) bool {
	if are.n == 0 || a.Compare(b) > 0 {
		return true
	}

	// Normalize and truncate
	var aPrime, bPrime bits.BitString
	if a.Compare(are.MinKey) < 0 {
		aPrime = bits.NewBitString(a.SizeBits()).ShiftRight(are.TruncateBits)
	} else {
		aPrime = a.Sub(are.MinKey).ShiftRight(are.TruncateBits)
	}
	
	if b.Compare(are.MinKey) < 0 {
		return true
	}
	bPrime = b.Sub(are.MinKey).ShiftRight(are.TruncateBits)

	if are.IsExactMode {
		// Exact Mode: Just a direct ERE query on M bits
		return are.ere.IsEmpty(toMBitString(aPrime, are.K), toMBitString(bPrime, are.K))
	} else {
		// SODA Mode
		return are.sodaIsEmpty(aPrime, bPrime)
	}
}

func (are *OptimizedApproximateRangeEmptiness) sodaIsEmpty(a, b bits.BitString) bool {
	rMask := (uint64(1) << are.K) - 1
	if are.K == 64 {
		rMask = ^uint64(0)
	}

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
		hA := (u + a.Word(0)) & rMask
		hB := (u + b.Word(0)) & rMask
		
		if hA <= hB {
			return are.ere.IsEmpty(toKBitString(hA, are.K), toKBitString(hB, are.K))
		} else {
			if !are.ere.IsEmpty(toKBitString(hA, are.K), toKBitString(rMask, are.K)) {
				return false
			}
			return are.ere.IsEmpty(toKBitString(0, are.K), toKBitString(hB, are.K))
		}
	}

	// Multi-block conservative check
	uA := hashBlockAny(blockA)
	hA_start := (uA + a.Word(0)) & rMask
	hA_end := (uA + rMask) & rMask
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

	if !are.ere.IsEmpty(toKBitString(0, are.K), toKBitString(rMask, are.K)) {
		return false
	}

	uB := hashBlockAny(blockB)
	hB_start := (uB + 0) & rMask
	hB_end := (uB + b.Word(0)) & rMask
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

func (are *OptimizedApproximateRangeEmptiness) SizeInBits() uint64 {
	if are.ere == nil {
		return 0
	}
	return are.ere.SizeInBits()
}
