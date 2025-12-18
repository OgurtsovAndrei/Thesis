// Package bits Equal/Eq benchmarks
//
// This file contains benchmarks for equality operations (Equal and Eq methods)
// across all BitString implementations:
// 1. Uint64BitString - Direct uint64 value comparison
// 2. CharBitString - String-based comparison
// 3. Uint64ArrayBitString - Word-by-word comparison

package bits

import (
	"fmt"
	"testing"
)

var equalSizes = []int{64, 128, 256, 512, 1024}

// --- Equal Benchmarks ---

func BenchmarkEqual_Uint64(b *testing.B) {
	for _, size := range equalSizes {
		if size > 64 {
			continue // Uint64BitString limited to 64 bits
		}
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs1 := NewUint64FromBinaryText(s)
			bs2 := NewUint64FromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bs1.Equal(bs2)
			}
		})
	}
}

func BenchmarkEqual_Char(b *testing.B) {
	for _, size := range equalSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs1 := NewCharFromBinary(s)
			bs2 := NewCharFromBinary(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bs1.Equal(bs2)
			}
		})
	}
}

func BenchmarkEqual_Uint64Array(b *testing.B) {
	for _, size := range equalSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs1 := NewUint64ArrayFromBinaryText(s)
			bs2 := NewUint64ArrayFromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bs1.Equal(bs2)
			}
		})
	}
}

// --- Eq Benchmarks (should be same as Equal) ---

func BenchmarkEq_Uint64(b *testing.B) {
	for _, size := range equalSizes {
		if size > 64 {
			continue
		}
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs1 := NewUint64FromBinaryText(s)
			bs2 := NewUint64FromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs1.Eq(bs2)
			}
		})
	}
}

func BenchmarkEq_Char(b *testing.B) {
	for _, size := range equalSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs1 := NewCharFromBinary(s)
			bs2 := NewCharFromBinary(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs1.Eq(bs2)
			}
		})
	}
}

func BenchmarkEq_Uint64Array(b *testing.B) {
	for _, size := range equalSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs1 := NewUint64ArrayFromBinaryText(s)
			bs2 := NewUint64ArrayFromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs1.Eq(bs2)
			}
		})
	}
}

// --- Equal Same Type vs Cross Type ---

func BenchmarkEqual_Uint64Array_SameType(b *testing.B) {
	s := randomBinaryString(256)
	bs1 := NewUint64ArrayFromBinaryText(s)
	bs2 := NewUint64ArrayFromBinaryText(s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.Equal(bs2)
	}
}

func BenchmarkEqual_Uint64Array_CrossType(b *testing.B) {
	s := randomBinaryString(256)
	bs1 := NewUint64ArrayFromBinaryText(s)
	bs2 := NewCharFromBinary(s) // Different type

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.Equal(bs2)
	}
}
