package testutils

import "Thesis/bits"

// TrieBS converts a uint64 to a 64-bit BitString in trie order.
func TrieBS(val uint64) bits.BitString {
	return bits.NewFromTrieUint64(val, 64)
}
