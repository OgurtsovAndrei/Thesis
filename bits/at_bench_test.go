// Package bits At/Access benchmarks
//
// This file contains benchmarks for bit access operations (At method)
// across all BitString implementations:
// 1. Uint64BitString - Direct bit mask operations
// 2. CharBitString - Byte indexing + bit mask
// 3. Uint64ArrayBitString - Word indexing + bit mask
// 4. trie.BitString - For comparison

package bits

import (
	"testing"

	trie "github.com/siongui/go-succinct-data-structure-trie/reference"
)

// --- At Benchmarks ---

func BenchmarkAt_Uint64(b *testing.B) {
	input := randomBinaryString(64)
	bs := NewUint64FromBinaryText(input)
	limit := uint32(64)

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			_ = bs.At(uint32(counter) % limit)
			counter++
		}
	})
}

func BenchmarkAt_Char(b *testing.B) {
	input := randomBinaryString(64)
	bs := NewCharFromBinary(input)
	limit := uint32(64)

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			_ = bs.At(uint32(counter) % limit)
			counter++
		}
	})
}

func BenchmarkAt_Uint64Array(b *testing.B) {
	input := randomBinaryString(128) // Test with 128 bits
	bs := NewUint64ArrayFromBinaryText(input)
	limit := uint32(128)

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			_ = bs.At(uint32(counter) % limit)
			counter++
		}
	})
}

// --- Comparative Access Benchmarks ---

func BenchmarkAccess_Uint64(b *testing.B) {
	value := randomUint64()
	bs := NewUint64FromUint64(value, 64) // 64-bit uint64
	limit := uint32(64)

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			_ = bs.At(uint32(counter) % limit)
			counter++
		}
	})
}

func BenchmarkAccess_Char(b *testing.B) {
	input := randomTextString(8) // 8 chars = 64 bits
	bs := NewCharFromText(input)
	limit := uint32(64)

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			_ = bs.At(uint32(counter) % limit)
			counter++
		}
	})
}

func BenchmarkAccess_Uint64Array(b *testing.B) {
	input := randomBinaryString(128) // 128 bits = 2 uint64 words
	bs := NewUint64ArrayFromBinaryText(input)
	limit := uint32(128)

	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			_ = bs.At(uint32(counter) % limit)
			counter++
		}
	})
}

func BenchmarkAccess_TrieBitString(b *testing.B) {
	input := randomBase64String(16) // 16 base64 chars = ~96 bits
	bs := &trie.BitString{}
	bs.Init(input)
	limit := 96

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pos := uint(i % limit)
		_ = bs.Get(pos, 1) > 0 // Get 1 bit and check if it's 1
	}
}

// --- Size-based Access benchmarks ---

func benchmarkAccessBySize(b *testing.B, size int) {
	// Generate appropriate data for each implementation
	textInput := randomTextString(size / 8)     // size in bits -> size/8 chars
	binaryInput := randomBinaryString(size)     // binary string for array
	base64Input := randomBase64String(size / 6) // ~6 bits per base64 char
	uint64Value := randomUint64()

	// Your implementations
	var bsUint64 Uint64BitString
	var bsChar CharBitString
	var bsUint64Array Uint64ArrayBitString

	if size <= 64 { // Uint64 limited to 64 bits
		bsUint64 = NewUint64FromUint64(uint64Value, int8(size))
	}
	bsChar = NewCharFromText(textInput)
	bsUint64Array = NewUint64ArrayFromBinaryText(binaryInput) // Works with any size

	// Trie BitString
	bsTrie := &trie.BitString{}
	bsTrie.Init(base64Input)

	b.Run("Uint64", func(b *testing.B) {
		if size > 64 {
			b.Skip("size > 64")
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos := uint32(i % size)
			_ = bsUint64.At(pos)
		}
	})

	b.Run("Char", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos := uint32(i % size)
			_ = bsChar.At(pos)
		}
	})

	b.Run("Uint64Array", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos := uint32(i % size)
			_ = bsUint64Array.At(pos)
		}
	})

	b.Run("TrieBitString", func(b *testing.B) {
		// Calculate actual bit length from base64 string length
		trieBitSize := len(base64Input) * 6 // ~6 bits per base64 char
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pos := uint(i % trieBitSize)
			_ = bsTrie.Get(pos, 1) > 0
		}
	})
}

func BenchmarkAccess_Size32(b *testing.B)   { benchmarkAccessBySize(b, 32) }
func BenchmarkAccess_Size64(b *testing.B)   { benchmarkAccessBySize(b, 64) }
func BenchmarkAccess_Size128(b *testing.B)  { benchmarkAccessBySize(b, 128) }
func BenchmarkAccess_Size256(b *testing.B)  { benchmarkAccessBySize(b, 256) }
func BenchmarkAccess_Size512(b *testing.B)  { benchmarkAccessBySize(b, 512) }
func BenchmarkAccess_Size1024(b *testing.B) { benchmarkAccessBySize(b, 1024) }
