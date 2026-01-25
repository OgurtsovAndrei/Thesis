// Package bits Initialization benchmarks
//
// This file contains benchmarks for BitString creation/initialization
// across all implementations:
// 1. Uint64BitString - Direct value assignment
// 2. CharBitString - PrettyString-based initialization
// 3. Uint64ArrayBitString - Array allocation and bit parsing
// 4. trie.BitString - For comparison

package bits

import (
	"testing"

	trie "github.com/siongui/go-succinct-data-structure-trie/reference"
)

// --- Basic Initialization Benchmarks ---

func BenchmarkInit_Uint64BitString(b *testing.B) {
	value := randomUint64() // Generate once outside the loop
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewUint64FromUint64(value, 64)
		}
	})
}

func BenchmarkInit_CharBitString(b *testing.B) {
	textInput := randomTextString(8) // 8 chars = 64 bits
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewCharFromText(textInput)
		}
	})
}

func BenchmarkInit_Uint64ArrayBitString(b *testing.B) {
	binaryInput := randomBinaryString(64)
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewUint64ArrayFromBinaryText(binaryInput)
		}
	})
}

func BenchmarkInit_TrieBitString(b *testing.B) {
	base64Input := randomBase64String(16) // 16 base64 chars = ~96 bits
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bs := &trie.BitString{}
			bs.Init(base64Input)
		}
	})
}

// --- Different construction methods ---

func BenchmarkInit_Uint64FromBinary(b *testing.B) {
	binaryInput := randomBinaryString(64)
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewUint64FromBinaryText(binaryInput)
		}
	})
}

func BenchmarkInit_CharFromBinary(b *testing.B) {
	binaryInput := randomBinaryString(64)
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewCharFromBinary(binaryInput)

		}
	})
}

func BenchmarkInit_Uint64ArrayFromData(b *testing.B) {
	data := make([]byte, 8)
	for i := range data {
		data[i] = byte(i * 33) // Some pattern
	}
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewUint64ArrFromDataAndSize(data, 64)
		}
	})
}

func BenchmarkInit_CharFromData(b *testing.B) {
	data := make([]byte, 8)
	for i := range data {
		data[i] = byte(i * 33) // Some pattern
	}
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewCharBitStringFromDataAndSize(data, 64)
		}
	})
}

func BenchmarkInit_Uint64FromData(b *testing.B) {
	data := make([]byte, 8)
	for i := range data {
		data[i] = byte(i * 33) // Some pattern
	}
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewUint64BitStringFromDataAndSize(data, 64)
		}
	})
}

// --- Size-based Initialization benchmarks ---

func benchmarkInitBySize(b *testing.B, size int) {
	// Generate appropriate data for each implementation
	textInput := randomTextString(size / 8)     // size in bits -> size/8 chars
	binaryInput := randomBinaryString(size)     // binary string for array
	base64Input := randomBase64String(size / 6) // ~6 bits per base64 char

	b.Run("Uint64", func(b *testing.B) {
		if size > 64 {
			b.Skip("size > 64")
		}
		value := randomUint64() // Generate once outside the loop
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewUint64FromUint64(value, int8(size))
		}
	})

	b.Run("Char", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewCharFromText(textInput)
		}
	})

	b.Run("Uint64Array", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewUint64ArrayFromBinaryText(binaryInput)
		}
	})

	b.Run("TrieBitString", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs := &trie.BitString{}
			bs.Init(base64Input)
		}
	})
}

func BenchmarkInit_Size32(b *testing.B)   { benchmarkInitBySize(b, 32) }
func BenchmarkInit_Size64(b *testing.B)   { benchmarkInitBySize(b, 64) }
func BenchmarkInit_Size128(b *testing.B)  { benchmarkInitBySize(b, 128) }
func BenchmarkInit_Size256(b *testing.B)  { benchmarkInitBySize(b, 256) }
func BenchmarkInit_Size512(b *testing.B)  { benchmarkInitBySize(b, 512) }
func BenchmarkInit_Size1024(b *testing.B) { benchmarkInitBySize(b, 1024) }

// --- Memory allocation patterns ---

func BenchmarkInit_Uint64Array_SmallAlloc(b *testing.B) {
	binaryInput := randomBinaryString(64) // Single word
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewUint64ArrayFromBinaryText(binaryInput)
		}
	})
}

func BenchmarkInit_Uint64Array_MediumAlloc(b *testing.B) {
	binaryInput := randomBinaryString(256) // 4 words
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewUint64ArrayFromBinaryText(binaryInput)
		}
	})
}

func BenchmarkInit_Uint64Array_LargeAlloc(b *testing.B) {
	binaryInput := randomBinaryString(1024) // 16 words
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewUint64ArrayFromBinaryText(binaryInput)
		}
	})
}

// --- Factory method benchmarks ---

func BenchmarkInit_FactoryNewBitString(b *testing.B) {
	textInput := "test"
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewFromText(textInput) // Uses current SelectedImpl (Uint64ArrayString)
		}
	})
}

func BenchmarkInit_FactoryNewFromUint64(b *testing.B) {
	value := randomUint64()
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewFromUint64(value) // Uses current SelectedImpl (Uint64ArrayString)
		}
	})
}

func BenchmarkInit_FactoryNewFromBinary(b *testing.B) {
	binaryInput := randomBinaryString(64)
	b.SetParallelism(benchmarkParallelism)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = NewFromBinary(binaryInput) // Uses current SelectedImpl (Uint64ArrayString)
		}
	})
}
