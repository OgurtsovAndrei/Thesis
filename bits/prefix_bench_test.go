// Package bits HasPrefix benchmarks
//
// This file contains benchmarks for prefix checking operations (HasPrefix method)
// across all BitString implementations:
// 1. Uint64BitString - Bit mask-based prefix checking
// 2. CharBitString - GetLCPLength-based prefix checking
// 3. Uint64ArrayBitString - Word-level prefix checking with masks

package bits

import (
	"testing"
)

// --- HasPrefix Benchmarks ---

func BenchmarkHasPrefix_Uint64(b *testing.B) {
	s1 := randomBinaryString(64)
	sPrefix := s1[:32] // 32-bit prefix

	bs1 := NewUint64FromBinaryText(s1)
	bsPrefix := NewUint64FromBinaryText(sPrefix)

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bs1.HasPrefix(bsPrefix)
		}
	})
}

func BenchmarkHasPrefix_Char(b *testing.B) {
	s1 := randomBinaryString(64)
	sPrefix := s1[:32]

	bs1 := NewCharFromBinary(s1)
	bsPrefix := NewCharFromBinary(sPrefix)

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bs1.HasPrefix(bsPrefix)
		}
	})
}

func BenchmarkHasPrefix_Uint64Array(b *testing.B) {
	s1 := randomBinaryString(128) // Test with 128 bits
	sPrefix := s1[:64]            // 64-bit prefix

	bs1 := NewUint64ArrayFromBinaryText(s1)
	bsPrefix := NewUint64ArrayFromBinaryText(sPrefix)

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bs1.HasPrefix(bsPrefix)
		}
	})
}

// --- HasPrefix Same Type vs Cross Type ---

func BenchmarkHasPrefix_Uint64Array_SameType(b *testing.B) {
	s1 := randomBinaryString(256)
	sPrefix := s1[:128] // Half as prefix
	bs1 := NewUint64ArrayFromBinaryText(s1)
	bsPrefix := NewUint64ArrayFromBinaryText(sPrefix)

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bs1.HasPrefix(bsPrefix)
		}
	})
}

func BenchmarkHasPrefix_Uint64Array_CrossType(b *testing.B) {
	s1 := randomBinaryString(256)
	sPrefix := s1[:128] // Half as prefix
	bs1 := NewUint64ArrayFromBinaryText(s1)
	bsPrefix := NewCharFromBinary(sPrefix) // Different type

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bs1.HasPrefix(bsPrefix)
		}
	})
}

// --- HasPrefix with different prefix sizes ---

func BenchmarkHasPrefix_SmallPrefix(b *testing.B) {
	s1 := randomBinaryString(256)
	sPrefix := s1[:32] // Small prefix (32 bits)

	bs1Char := NewCharFromBinary(s1)
	bsPrefixChar := NewCharFromBinary(sPrefix)
	bs1Array := NewUint64ArrayFromBinaryText(s1)
	bsPrefixArray := NewUint64ArrayFromBinaryText(sPrefix)

	b.Run("Char", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs1Char.HasPrefix(bsPrefixChar)
		}
	})

	b.Run("Uint64Array", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs1Array.HasPrefix(bsPrefixArray)
		}
	})
}

func BenchmarkHasPrefix_MediumPrefix(b *testing.B) {
	s1 := randomBinaryString(512)
	sPrefix := s1[:256] // Medium prefix (256 bits)

	bs1Char := NewCharFromBinary(s1)
	bsPrefixChar := NewCharFromBinary(sPrefix)
	bs1Array := NewUint64ArrayFromBinaryText(s1)
	bsPrefixArray := NewUint64ArrayFromBinaryText(sPrefix)

	b.Run("Char", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs1Char.HasPrefix(bsPrefixChar)
		}
	})

	b.Run("Uint64Array", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs1Array.HasPrefix(bsPrefixArray)
		}
	})
}

func BenchmarkHasPrefix_LargePrefix(b *testing.B) {
	s1 := randomBinaryString(1024)
	sPrefix := s1[:512] // Large prefix (512 bits)

	bs1Char := NewCharFromBinary(s1)
	bsPrefixChar := NewCharFromBinary(sPrefix)
	bs1Array := NewUint64ArrayFromBinaryText(s1)
	bsPrefixArray := NewUint64ArrayFromBinaryText(sPrefix)

	b.Run("Char", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs1Char.HasPrefix(bsPrefixChar)
		}
	})

	b.Run("Uint64Array", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs1Array.HasPrefix(bsPrefixArray)
		}
	})
}

// --- Size-based HasPrefix benchmarks ---

func benchmarkHasPrefixBySize(b *testing.B, size int) {
	s1 := randomBinaryString(size)
	sPrefix := s1[:size/2] // Half as prefix

	// Create bitstrings for each implementation
	var bs1Uint64, bsPrefixUint64 Uint64BitString
	var bs1Char, bsPrefixChar CharBitString
	var bs1Uint64Array, bsPrefixUint64Array Uint64ArrayBitString

	if size <= 64 {
		bs1Uint64 = NewUint64FromBinaryText(s1)
		bsPrefixUint64 = NewUint64FromBinaryText(sPrefix)
	}
	bs1Char = NewCharFromBinary(s1)
	bsPrefixChar = NewCharFromBinary(sPrefix)
	bs1Uint64Array = NewUint64ArrayFromBinaryText(s1)
	bsPrefixUint64Array = NewUint64ArrayFromBinaryText(sPrefix)

	if size <= 64 {
		b.Run("Uint64", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bs1Uint64.HasPrefix(bsPrefixUint64)
			}
		})
	}

	b.Run("Char", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs1Char.HasPrefix(bsPrefixChar)
		}
	})

	b.Run("Uint64Array", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs1Uint64Array.HasPrefix(bsPrefixUint64Array)
		}
	})
}

func BenchmarkHasPrefix_Size64(b *testing.B)   { benchmarkHasPrefixBySize(b, 64) }
func BenchmarkHasPrefix_Size128(b *testing.B)  { benchmarkHasPrefixBySize(b, 128) }
func BenchmarkHasPrefix_Size256(b *testing.B)  { benchmarkHasPrefixBySize(b, 256) }
func BenchmarkHasPrefix_Size512(b *testing.B)  { benchmarkHasPrefixBySize(b, 512) }
func BenchmarkHasPrefix_Size1024(b *testing.B) { benchmarkHasPrefixBySize(b, 1024) }
