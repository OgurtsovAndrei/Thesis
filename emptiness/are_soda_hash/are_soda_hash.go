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

	// Interval length is (b - a + 1). SODA guarantees for length <= RangeLen.
	if b-a >= are.RangeLen {
		return false
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
		// One interval
		u := hashBlock(blockA)
		hA := (u + a) & rMask
		hB := (u + b) & rMask
		
		bsA := toKBitString(hA, are.K)
		bsB := toKBitString(hB, are.K)
		return are.ere.IsEmpty(bsA, bsB)
	} else {
		// Two intervals
		uA := hashBlock(blockA)
		// Calculate max element of blockA avoiding overflow
		var maxA uint64
		if are.K == 64 {
			maxA = ^uint64(0)
		} else {
			nextBlock := blockA + 1
			if nextBlock == 1<<(64-are.K) {
				maxA = ^uint64(0)
			} else {
				maxA = (nextBlock << are.K) - 1
			}
		}
		
		hA_start := (uA + a) & rMask
		hA_end := (uA + maxA) & rMask
		bsA_start := toKBitString(hA_start, are.K)
		bsA_end := toKBitString(hA_end, are.K)
		if !are.ere.IsEmpty(bsA_start, bsA_end) {
			return false
		}

		uB := hashBlock(blockB)
		var minB uint64
		if are.K < 64 {
			minB = blockB << are.K
		}
		hB_start := (uB + minB) & rMask
		hB_end := (uB + b) & rMask
		bsB_start := toKBitString(hB_start, are.K)
		bsB_end := toKBitString(hB_end, are.K)
		if !are.ere.IsEmpty(bsB_start, bsB_end) {
			return false
		}

		return true
	}
}

func (are *ApproximateRangeEmptinessSoda) SizeInBits() uint64 {
	if are.ere == nil {
		return 0
	}
	return are.ere.SizeInBits()
}
