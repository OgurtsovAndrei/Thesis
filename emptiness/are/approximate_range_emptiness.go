package are

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
	exact      *ere.ExactRangeEmptiness
	K          uint32
	minKey     bits.BitString
	maxKey     bits.BitString
	spreadBits uint32 // bit-width of (maxKey - minKey); 0 means single-key set
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

	// Keys must be sorted by Compare; minKey is keys[0], maxKey is keys[n-1].
	minKey := keys[0]
	maxKey := keys[n-1]

	// Compute spread = maxKey - minKey and its significant bit-width.
	spread := maxKey.Sub(minKey)
	spreadVal := spread.TrieUint64()
	var spreadBits uint32
	if spreadVal > 0 {
		spreadBits = uint32(64 - mbits.LeadingZeros64(spreadVal))
	}

	// Normalize and truncate to K bits.
	// Each normalized value is left-shifted so its spreadBits significant bits
	// sit at the top of a 64-bit integer, then Prefix(K) takes the top K of those.
	truncatedKeys := make([]bits.BitString, 0, n)
	var lastKey bits.BitString
	for i, k := range keys {
		trunc := normalizeToK(k, minKey, spreadBits, K)

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
		exact:      exact,
		K:          K,
		minKey:     minKey,
		maxKey:     maxKey,
		spreadBits: spreadBits,
	}, nil
}

// normalizeToK maps key into a K-bit prefix by:
//  1. Subtracting minKey (so the minimum maps to 0)
//  2. Left-shifting the spread to fill the top spreadBits bits of a 64-bit integer
//  3. Taking the top K bits as a K-bit BitString
func normalizeToK(key, minKey bits.BitString, spreadBits, K uint32) bits.BitString {
	normVal := key.Sub(minKey).TrieUint64()
	// Shift left so spreadBits significant bits fill from the MSB of a 64-bit uint.
	// Then the top K bits of that 64-bit value become our K-bit prefix.
	var shifted uint64
	if spreadBits > 0 && spreadBits < 64 {
		shifted = normVal << (64 - spreadBits)
	} else {
		shifted = normVal // spreadBits==0 (all identical) or spreadBits==64
	}
	return bits.NewFromTrieUint64(shifted>>uint32(64-K), K)
}

func (are *ApproximateRangeEmptiness) IsEmpty(a, b bits.BitString) bool {
	if are.exact == nil {
		return true
	}

	// If b < minKey, the query range is entirely below all stored keys.
	if b.Compare(are.minKey) < 0 {
		return true
	}

	// Normalize a: clamp to minKey if below it (normalized value = 0).
	var truncA bits.BitString
	if a.Compare(are.minKey) < 0 {
		truncA = bits.NewBitString(are.K)
	} else {
		truncA = normalizeToK(a, are.minKey, are.spreadBits, are.K)
	}

	// Normalize b: clamp to maxKey if above it.
	var truncB bits.BitString
	if b.Compare(are.maxKey) > 0 {
		truncB = normalizeToK(are.maxKey, are.minKey, are.spreadBits, are.K)
	} else {
		truncB = normalizeToK(b, are.minKey, are.spreadBits, are.K)
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
