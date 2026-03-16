package are_trunc

import (
	"Thesis/bits"
	"Thesis/emptiness/ere"
	"Thesis/utils"
	"fmt"
	"math"
	mbits "math/bits"
)

// ApproximateRangeEmptiness is a probabilistic data structure that answers 1D range emptiness
// queries with a guaranteed upper bound on the false positive probability (\epsilon).
// Uses prefix truncation with key normalization: keys are shifted relative to minKey so that
// the spread occupies all K bits effectively (avoids all-zero-prefix collapse for small-valued keys).
type ApproximateRangeEmptiness struct {
	exact       *ere.ExactRangeEmptiness
	K           uint32
	minKey      bits.BitString
	maxKey      bits.BitString
	spreadStart uint32 // trie position of first significant bit in (maxKey - minKey)
}

func NewApproximateRangeEmptiness(keys []bits.BitString, epsilon float64) (*ApproximateRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &ApproximateRangeEmptiness{exact: nil, K: 0}, nil
	}

	val := (2.0 * float64(n)) / epsilon
	K := uint32(math.Ceil(math.Log2(val)))
	if K == 0 {
		K = 1
	}

	return NewApproximateRangeEmptinessFromK(keys, K)
}

func NewApproximateRangeEmptinessFromK(keys []bits.BitString, K uint32) (*ApproximateRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &ApproximateRangeEmptiness{exact: nil, K: 0}, nil
	}
	if K == 0 {
		K = 1
	}

	minKey := keys[0]
	maxKey := keys[n-1]

	spread := maxKey.Sub(minKey)
	spreadStart := trieFirstSetBit(spread)

	truncatedKeys := make([]bits.BitString, 0, n)
	var lastKey bits.BitString
	for i, k := range keys {
		trunc := normalizeToK(k, minKey, spreadStart, K)

		if i == 0 || trunc.Compare(lastKey) > 0 {
			truncatedKeys = append(truncatedKeys, trunc)
			lastKey = trunc
		} else if trunc.Compare(lastKey) == 0 {
			continue
		} else {
			return nil, fmt.Errorf("keys must be sorted by Compare")
		}
	}

	universe := bits.NewBitString(K)
	exact, err := ere.NewExactRangeEmptiness(truncatedKeys, universe)
	if err != nil {
		return nil, err
	}

	return &ApproximateRangeEmptiness{
		exact:       exact,
		K:           K,
		minKey:      minKey,
		maxKey:      maxKey,
		spreadStart: spreadStart,
	}, nil
}

// trieFirstSetBit returns the trie position of the first set bit (MSB in trie order).
// Returns bs.SizeBits() if the value is zero.
func trieFirstSetBit(bs bits.BitString) uint32 {
	W := bs.SizeBits()
	numWords := (W + 63) / 64
	for i := uint32(0); i < numWords; i++ {
		w := bs.Word(i)
		if w != 0 {
			return i*64 + uint32(mbits.TrailingZeros64(w))
		}
	}
	return W
}

// normalizeToK maps key into a K-bit prefix by:
//  1. Subtracting minKey (so the minimum maps to 0)
//  2. Extracting K bits starting at spreadStart (first significant bit of the spread)
func normalizeToK(key, minKey bits.BitString, spreadStart, K uint32) bits.BitString {
	offset := key.Sub(minKey)
	return offset.BitRange(spreadStart, K)
}

func (are *ApproximateRangeEmptiness) IsEmpty(a, b bits.BitString) bool {
	if are.exact == nil {
		return true
	}

	if b.Compare(are.minKey) < 0 {
		return true
	}
	if a.Compare(are.maxKey) > 0 {
		return true
	}

	var truncA bits.BitString
	if a.Compare(are.minKey) < 0 {
		truncA = bits.NewBitString(are.K)
	} else {
		truncA = normalizeToK(a, are.minKey, are.spreadStart, are.K)
	}

	var truncB bits.BitString
	if b.Compare(are.maxKey) > 0 {
		truncB = normalizeToK(are.maxKey, are.minKey, are.spreadStart, are.K)
	} else {
		truncB = normalizeToK(b, are.minKey, are.spreadStart, are.K)
	}

	return are.exact.IsEmpty(truncA, truncB)
}

func (are *ApproximateRangeEmptiness) SizeInBits() uint64 {
	if are.exact == nil {
		return 0
	}
	return are.exact.SizeInBits()
}

func (are *ApproximateRangeEmptiness) ByteSize() int {
	if are == nil || are.exact == nil {
		return 0
	}
	return are.exact.ByteSize() + 8
}

func (are *ApproximateRangeEmptiness) MemDetailed() utils.MemReport {
	if are == nil || are.exact == nil {
		return utils.MemReport{Name: "ApproximateRangeEmptiness", TotalBytes: 0}
	}
	return are.exact.MemDetailed()
}
