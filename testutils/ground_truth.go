package testutils

import "sort"

// GroundTruth returns true if the range [a, b] contains no keys from the sorted slice.
func GroundTruth(sortedKeys []uint64, a, b uint64) bool {
	idx := sort.Search(len(sortedKeys), func(i int) bool { return sortedKeys[i] >= a })
	return idx == len(sortedKeys) || sortedKeys[idx] > b
}
