// Package bits LCP (Longest Common Prefix) benchmarks
//
// This file contains benchmarks specifically for GetLCPLength operations
// across all BitString implementations:
// 1. Uint64BitString - XOR + TrailingZeros64 for fast LCP
// 2. CharBitString - Byte-wise comparison with TrailingZeros8
// 3. Uint64ArrayBitString - Word-wise comparison with TrailingZeros64

package bits

import (
	"testing"
)

// --- LCP Benchmarks ---

func BenchmarkLCP_Uint64(b *testing.B) {
	s1 := randomBinaryString(64)
	s2 := s1[:63]
	// Make s2 differ at the very last bit to force full traversal
	if s1[63] == '0' {
		s2 += "1"
	} else {
		s2 += "0"
	}

	bs1 := NewUint64FromBinaryText(s1)
	bs2 := NewUint64FromBinaryText(s2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.GetLCPLength(bs2)
	}
}

func BenchmarkLCP_Char(b *testing.B) {
	s1 := randomBinaryString(64)
	s2 := s1[:63]
	if s1[63] == '0' {
		s2 += "1"
	} else {
		s2 += "0"
	}

	bs1 := NewCharFromBinary(s1)
	bs2 := NewCharFromBinary(s2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.GetLCPLength(bs2)
	}
}

func BenchmarkLCP_Uint64Array(b *testing.B) {
	s1 := randomBinaryString(128) // Test with 128 bits to show multi-word capability
	s2 := s1[:127]
	if s1[127] == '0' {
		s2 += "1"
	} else {
		s2 += "0"
	}

	bs1 := NewUint64ArrayFromBinaryText(s1)
	bs2 := NewUint64ArrayFromBinaryText(s2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.GetLCPLength(bs2)
	}
}

// --- LCP Same Type vs Cross Type ---

func BenchmarkLCP_Uint64Array_SameType(b *testing.B) {
	s1 := randomBinaryString(256)
	s2 := randomBinaryString(256)
	bs1 := NewUint64ArrayFromBinaryText(s1)
	bs2 := NewUint64ArrayFromBinaryText(s2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.GetLCPLength(bs2)
	}
}

func BenchmarkLCP_Uint64Array_CrossType(b *testing.B) {
	s1 := randomBinaryString(256)
	s2 := randomBinaryString(256)
	bs1 := NewUint64ArrayFromBinaryText(s1)
	bs2 := NewCharFromBinary(s2) // Different type

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.GetLCPLength(bs2)
	}
}

// --- Size-based LCP benchmarks ---

func benchmarkLCPBySize(b *testing.B, size int) {
	s1 := randomBinaryString(size)
	s2 := s1[:size-1]
	if s1[size-1] == '0' {
		s2 += "1"
	} else {
		s2 += "0"
	}

	// Create bitstrings for each implementation
	var bs1Uint64, bs2Uint64 Uint64BitString
	var bs1Char, bs2Char CharBitString
	var bs1Uint64Array, bs2Uint64Array Uint64ArrayBitString

	if size <= 64 {
		bs1Uint64 = NewUint64FromBinaryText(s1)
		bs2Uint64 = NewUint64FromBinaryText(s2)
	}
	bs1Char = NewCharFromBinary(s1)
	bs2Char = NewCharFromBinary(s2)
	bs1Uint64Array = NewUint64ArrayFromBinaryText(s1)
	bs2Uint64Array = NewUint64ArrayFromBinaryText(s2)

	if size <= 64 {
		b.Run("Uint64", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bs1Uint64.GetLCPLength(bs2Uint64)
			}
		})
	}

	b.Run("Char", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs1Char.GetLCPLength(bs2Char)
		}
	})

	b.Run("Uint64Array", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bs1Uint64Array.GetLCPLength(bs2Uint64Array)
		}
	})
}

func BenchmarkLCP_Size64(b *testing.B)   { benchmarkLCPBySize(b, 64) }
func BenchmarkLCP_Size128(b *testing.B)  { benchmarkLCPBySize(b, 128) }
func BenchmarkLCP_Size256(b *testing.B)  { benchmarkLCPBySize(b, 256) }
func BenchmarkLCP_Size512(b *testing.B)  { benchmarkLCPBySize(b, 512) }
func BenchmarkLCP_Size1024(b *testing.B) { benchmarkLCPBySize(b, 1024) }
