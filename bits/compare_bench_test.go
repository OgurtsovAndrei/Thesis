// Package bits Compare benchmarks
//
// This file contains benchmarks for Compare method across all BitString implementations:
// 1. Uint64BitString - XOR + TrailingZeros64 for fast comparison
// 2. CharBitString - Byte-wise comparison with TrailingZeros8
// 3. Uint64ArrayBitString - Word-wise comparison with TrailingZeros64

package bits

import (
	"fmt"
	"testing"
)

var compareSizes = []int{64, 128, 256, 512, 1024}

// --- Compare Benchmarks ---

func BenchmarkCompare_Uint64(b *testing.B) {
	for _, size := range compareSizes {
		if size > 64 {
			continue // Uint64BitString limited to 64 bits
		}
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s1 := randomBinaryString(size)
			s2 := randomBinaryString(size)
			bs1 := NewUint64FromBinaryText(s1)
			bs2 := NewUint64FromBinaryText(s2)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs1.Compare(bs2)
			}
		})
	}
}

func BenchmarkCompare_Char(b *testing.B) {
	for _, size := range compareSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s1 := randomBinaryString(size)
			s2 := randomBinaryString(size)
			bs1 := NewCharFromBinary(s1)
			bs2 := NewCharFromBinary(s2)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs1.Compare(bs2)
			}
		})
	}
}

func BenchmarkCompare_Uint64Array(b *testing.B) {
	for _, size := range compareSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s1 := randomBinaryString(size)
			s2 := randomBinaryString(size)
			bs1 := NewUint64ArrayFromBinaryText(s1)
			bs2 := NewUint64ArrayFromBinaryText(s2)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs1.Compare(bs2)
			}
		})
	}
}

// --- Same Type vs Cross Type Benchmarks ---

func BenchmarkCompare_Uint64Array_SameType(b *testing.B) {
	s1 := randomBinaryString(256)
	s2 := randomBinaryString(256)
	bs1 := NewUint64ArrayFromBinaryText(s1)
	bs2 := NewUint64ArrayFromBinaryText(s2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs1.Compare(bs2)
	}
}

func BenchmarkCompare_Uint64Array_CrossType(b *testing.B) {
	s1 := randomBinaryString(256)
	s2 := randomBinaryString(256)
	bs1 := NewUint64ArrayFromBinaryText(s1)
	bs2 := NewCharFromBinary(s2) // Different type

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs1.Compare(bs2)
	}
}
