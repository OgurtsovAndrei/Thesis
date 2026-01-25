// Package bits Hash benchmarks
//
// This file contains benchmarks for Hash method across all BitString implementations:
// 1. Uint64BitString - Direct uint64 value as hash (optimal)
// 2. CharBitString - FNV-1a hash over byte data
// 3. Uint64ArrayBitString - FNV-1a hash over uint64 words

package bits

import (
	"fmt"
	"testing"
)

var hashSizes = []int{64, 128, 256, 512, 1024}

// --- Hash Benchmarks ---

func BenchmarkHash_Uint64(b *testing.B) {
	for _, size := range hashSizes {
		if size > 64 {
			continue // Uint64BitString limited to 64 bits
		}
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs := NewUint64FromBinaryText(s)

			b.SetParallelism(benchmarkParallelism)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = bs.Hash()
				}
			})
		})
	}
}

func BenchmarkHash_Char(b *testing.B) {
	for _, size := range hashSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs := NewCharFromBinary(s)

			b.SetParallelism(benchmarkParallelism)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = bs.Hash()
				}
			})
		})
	}
}

func BenchmarkHash_Uint64Array(b *testing.B) {
	for _, size := range hashSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs := NewUint64ArrayFromBinaryText(s)

			b.SetParallelism(benchmarkParallelism)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = bs.Hash()
				}
			})
		})
	}
}
