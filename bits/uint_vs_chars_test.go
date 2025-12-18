// Package bits performance benchmarks for trie.BitString comparison
//
// This file contains benchmarks comparing trie.BitString with our implementations.
// Most other benchmarks have been moved to the bench/ directory and organized by operation type.
//
// Comparison of BitString implementations:
// 1. Uint64BitString - Fast bit access using uint64 operations (≤64 bits)
// 2. CharBitString - Character-based bit string operations
// 3. Uint64ArrayBitString - Array of uint64s for arbitrary length bitstrings
// 4. trie.BitString - From siongui/go-succinct-data-structure-trie (provides Count/Rank)

package bits

import (
	"testing"

	trie "github.com/siongui/go-succinct-data-structure-trie/reference"
)

func BenchmarkTrieBitString_Init(b *testing.B) {
	data := randomBase64String(16) // 16 base64 chars = ~96 bits

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs := &trie.BitString{}
		bs.Init(data)
	}
}

func BenchmarkTrieBitString_Get(b *testing.B) {
	data := randomBase64String(16) // 16 base64 chars = ~96 bits
	bs := &trie.BitString{}
	bs.Init(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Get 1 bit at random position
		pos := uint(i % 96)
		bs.Get(pos, 1)
	}
}

func BenchmarkTrieBitString_Count(b *testing.B) {
	data := randomBase64String(16) // 16 base64 chars = ~96 bits
	bs := &trie.BitString{}
	bs.Init(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Count bits in range [0, 32)
		bs.Count(0, 32)
	}
}

func BenchmarkTrieBitString_Rank(b *testing.B) {
	data := randomBase64String(16) // 16 base64 chars = ~96 bits
	bs := &trie.BitString{}
	bs.Init(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Rank up to position
		pos := uint(i % 96)
		bs.Rank(pos)
	}
}

// --- Comparative benchmarks: Count/Rank operation ---

func BenchmarkCompare_Count_TrieBitString(b *testing.B) {
	input := randomBase64String(16) // 16 base64 chars = ~96 bits
	bs := &trie.BitString{}
	bs.Init(input)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Count bits in first half
		bs.Count(0, 48)
	}
}

// Note: Your BitString implementations don't have direct Count/Rank equivalents
// They focus on LCP, HasPrefix, Equal operations
