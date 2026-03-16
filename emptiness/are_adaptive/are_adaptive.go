package are_adaptive

import (
	"Thesis/bits"
	"Thesis/emptiness/ere"
	"fmt"
	"math"
	mbits "math/bits"
	"math/rand"
	"sort"
)

type AdaptiveApproximateRangeEmptiness struct {
	ere          *ere.ExactRangeEmptiness
	K            uint32
	RangeLen     uint64
	MinKey       bits.BitString
	TruncateBits uint32
	IsExactMode  bool
	n            int
	hashA        uint64
	hashB        uint64
}

// pairwiseHash computes a 2-universal hash: top K bits of (a*x + b) in 128-bit arithmetic.
func pairwiseHash(x, a, b uint64, K uint32) uint64 {
	hi, lo := mbits.Mul64(a, x)
	sumLo, carry := mbits.Add64(lo, b, 0)
	_ = sumLo
	sumHi := hi + carry
	return sumHi >> (64 - K)
}

// hashBlockIndex hashes a block index BitString to a K-bit uint64.
func hashBlockIndex(block bits.BitString, a, b uint64, K uint32) uint64 {
	var blockVal uint64
	if block.SizeBits() <= 64 {
		blockVal = block.TrieUint64()
	} else {
		blockVal = block.HashWithSeed(0)
	}
	return pairwiseHash(blockVal, a, b, K)
}

// ExactModeViable reports whether exact mode would trigger for a segment
// with the given spread, without building the filter.
// spread is max(S) - min(S) in the original key space.
func ExactModeViable(spread uint64, rangeLen uint64, K uint32) bool {
	if K == 0 || K > 64 {
		return false
	}
	var M uint32
	if spread > 0 {
		M = uint32(64 - mbits.LeadingZeros64(spread))
	}
	return M <= K
}

func NewAdaptiveARE(keys []bits.BitString, rangeLen uint64, epsilon float64, t uint32) (*AdaptiveApproximateRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &AdaptiveApproximateRangeEmptiness{n: 0}, nil
	}

	effectiveRangeLen := (rangeLen >> t) + 1
	rTarget := float64(n) * float64(effectiveRangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		return nil, fmt.Errorf("required K=%d exceeds 64 bits. Increase truncation 't'", K)
	}

	return NewAdaptiveAREFromK(keys, rangeLen, K, t)
}

func NewAdaptiveAREFromK(keys []bits.BitString, rangeLen uint64, K uint32, t uint32) (*AdaptiveApproximateRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &AdaptiveApproximateRangeEmptiness{n: 0}, nil
	}
	if K > 64 {
		return nil, fmt.Errorf("required K=%d exceeds 64 bits. Increase truncation 't'", K)
	}

	// 1. Find Min and Max keys (by Compare = trie order)
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

	// 2. Calculate data spread M (after subtraction and truncation)
	spread := maxKey.Sub(minKey).ShiftRight(t)

	var M uint32
	if spread.SizeBits() <= 64 {
		spreadVal := spread.TrieUint64()
		if spreadVal > 0 {
			M = uint32(64 - mbits.LeadingZeros64(spreadVal))
		}
	} else {
		M = 65 // multi-word spread → always SODA mode (K ≤ 64)
	}

	// 3. Adaptive: Exact vs Approximate
	isExactMode := (M <= K)
	finalUniverseBits := K
	if isExactMode {
		finalUniverseBits = M
	}

	rng := rand.New(rand.NewSource(int64(n) ^ int64(rangeLen)))
	hashA := rng.Uint64() | 1
	hashB := rng.Uint64()

	hashedKeys := make([]bits.BitString, n)
	for i, x := range keys {
		xPrime := x.Sub(minKey).ShiftRight(t)

		if isExactMode {
			hashedKeys[i] = bits.NewFromTrieUint64(xPrime.TrieUint64(), M)
		} else {
			W := xPrime.SizeBits()
			var block bits.BitString
			var offsetVal uint64

			rMask := (uint64(1) << K) - 1
			if K == 64 {
				rMask = ^uint64(0)
			}

			if W > K {
				block = xPrime.Prefix(int(W - K))
				offsetVal = xPrime.Suffix(K).TrieUint64()
			} else {
				block = bits.NewBitString(0)
				offsetVal = xPrime.TrieUint64()
			}

			u := hashBlockIndex(block, hashA, hashB, K)
			hx := (u + offsetVal) & rMask
			hashedKeys[i] = bits.NewFromTrieUint64(hx, K)
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

	return &AdaptiveApproximateRangeEmptiness{
		ere:          ereFilter,
		K:            finalUniverseBits,
		RangeLen:     rangeLen,
		MinKey:       minKey,
		TruncateBits: t,
		IsExactMode:  isExactMode,
		n:            n,
		hashA:        hashA,
		hashB:        hashB,
	}, nil
}

func (are *AdaptiveApproximateRangeEmptiness) IsEmpty(a, b bits.BitString) bool {
	if are.n == 0 || a.Compare(b) > 0 {
		return true
	}

	// Normalize
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
		// Convert normalized values to integer representation in the M-bit universe
		aBS := bits.NewFromTrieUint64(aPrime.TrieUint64(), are.K)
		bBS := bits.NewFromTrieUint64(bPrime.TrieUint64(), are.K)
		return are.ere.IsEmpty(aBS, bBS)
	}
	return are.sodaIsEmpty(aPrime, bPrime)
}

func (are *AdaptiveApproximateRangeEmptiness) sodaIsEmpty(a, b bits.BitString) bool {
	rMask := (uint64(1) << are.K) - 1
	if are.K == 64 {
		rMask = ^uint64(0)
	}

	W := a.SizeBits()
	var blockA, blockB bits.BitString
	var offA, offB uint64

	if W > are.K {
		prefixLen := int(W - are.K)
		blockA = a.Prefix(prefixLen)
		blockB = b.Prefix(prefixLen)
		offA = a.Suffix(are.K).TrieUint64()
		offB = b.Suffix(are.K).TrieUint64()
	} else {
		blockA = bits.NewBitString(0)
		blockB = bits.NewBitString(0)
		offA = a.TrieUint64()
		offB = b.TrieUint64()
	}

	toBS := func(val uint64) bits.BitString {
		return bits.NewFromTrieUint64(val, are.K)
	}

	if blockA.Equal(blockB) {
		u := hashBlockIndex(blockA, are.hashA, are.hashB, are.K)
		hA := (u + offA) & rMask
		hB := (u + offB) & rMask

		if hA <= hB {
			return are.ere.IsEmpty(toBS(hA), toBS(hB))
		}
		if !are.ere.IsEmpty(toBS(hA), toBS(rMask)) {
			return false
		}
		return are.ere.IsEmpty(toBS(0), toBS(hB))
	}

	// Multi-block: check suffix of first block
	uA := hashBlockIndex(blockA, are.hashA, are.hashB, are.K)
	hAStart := (uA + offA) & rMask
	hAEnd := (uA + rMask) & rMask
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
	if !are.ere.IsEmpty(toBS(0), toBS(rMask)) {
		return false
	}

	// Prefix of last block
	uB := hashBlockIndex(blockB, are.hashA, are.hashB, are.K)
	hBStart := (uB + 0) & rMask
	hBEnd := (uB + offB) & rMask
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

func (are *AdaptiveApproximateRangeEmptiness) SizeInBits() uint64 {
	if are.ere == nil {
		return 0
	}
	return are.ere.SizeInBits()
}
