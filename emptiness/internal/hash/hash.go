package hash

import (
	"Thesis/bits"
	mbits "math/bits"
	"sort"
)

// PairwiseHash computes a 2-universal hash: top K bits of (a*x + b) in 128-bit arithmetic.
func PairwiseHash(x, a, b uint64, K uint32) uint64 {
	hi, lo := mbits.Mul64(a, x)
	sumLo, carry := mbits.Add64(lo, b, 0)
	_ = sumLo
	sumHi := hi + carry
	return sumHi >> (64 - K)
}

// SortAndDedup sorts a slice of BitStrings and removes duplicates, returning a new deduplicated slice.
func SortAndDedup(keys []bits.BitString) []bits.BitString {
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Compare(keys[j]) < 0
	})
	unique := make([]bits.BitString, 0, len(keys))
	if len(keys) > 0 {
		unique = append(unique, keys[0])
		for i := 1; i < len(keys); i++ {
			if !keys[i].Equal(keys[i-1]) {
				unique = append(unique, keys[i])
			}
		}
	}
	return unique
}
