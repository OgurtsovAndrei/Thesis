package hash

import (
	"Thesis/bits"
	"sort"
)

// PairwiseHash computes a 2-universal hash using the standard multiply-shift method.
// It maps a 64-bit input to a K-bit output: h(x) = (a*x + b) >> (64 - K).
func PairwiseHash(x, a, b uint64, K uint32) uint64 {
	return (a*x + b) >> (64 - K)
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

// SortAndDedupUint64 sorts the slice in place and removes consecutive duplicates,
// returning a sub-slice of the input backing array.
func SortAndDedupUint64(keys []uint64) []uint64 {
	if len(keys) == 0 {
		return keys
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	w := 1
	for i := 1; i < len(keys); i++ {
		if keys[i] != keys[i-1] {
			keys[w] = keys[i]
			w++
		}
	}
	return keys[:w]
}
