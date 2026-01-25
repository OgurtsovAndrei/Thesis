// Package bits New Operations benchmarks
//
// This file contains benchmarks for new BitString operations added for performance optimization:
// 1. TrimTrailingZeros - Remove trailing zero bits from BitString
// 2. AppendBit - Add a single bit to the end of a BitString
// 3. IsAllOnes - Check if all bits in BitString are set to 1
// 4. Successor - Compute next BitString in lexicographic order

package bits

import (
	"fmt"
	"testing"
)

var opsSizes = []int{8, 16, 32, 64, 128, 256, 512, 1024}

// --- TrimTrailingZeros Benchmarks ---

func BenchmarkTrimTrailingZeros_Uint64(b *testing.B) {
	for _, size := range opsSizes {
		if size > 64 {
			continue // Uint64BitString limited to 64 bits
		}
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			// Create bitstring with trailing zeros (50% chance per bit)
			s := randomBinaryString(size)
			// Ensure some trailing zeros by replacing last few bits
			if size >= 4 {
				s = s[:size-4] + "0000"
			}
			bs := NewUint64FromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.TrimTrailingZeros()
			}
		})
	}
}

func BenchmarkTrimTrailingZeros_Char(b *testing.B) {
	for _, size := range opsSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			if size >= 4 {
				s = s[:size-4] + "0000"
			}
			bs := NewCharFromBinary(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.TrimTrailingZeros()
			}
		})
	}
}

func BenchmarkTrimTrailingZeros_Uint64Array(b *testing.B) {
	for _, size := range opsSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			if size >= 4 {
				s = s[:size-4] + "0000"
			}
			bs := NewUint64ArrayFromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.TrimTrailingZeros()
			}
		})
	}
}

// --- AppendBit Benchmarks ---

func BenchmarkAppendBit_Uint64(b *testing.B) {
	for _, size := range opsSizes {
		if size >= 64 {
			continue // Cannot append to full Uint64BitString
		}
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs := NewUint64FromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.AppendBit(true)
			}
		})
	}
}

func BenchmarkAppendBit_Char(b *testing.B) {
	for _, size := range opsSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs := NewCharFromBinary(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.AppendBit(true)
			}
		})
	}
}

func BenchmarkAppendBit_Uint64Array(b *testing.B) {
	for _, size := range opsSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			bs := NewUint64ArrayFromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.AppendBit(true)
			}
		})
	}
}

// --- IsAllOnes Benchmarks ---

func BenchmarkIsAllOnes_Uint64(b *testing.B) {
	for _, size := range opsSizes {
		if size > 64 {
			continue // Uint64BitString limited to 64 bits
		}
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			// Create all-ones bitstring for optimal case
			s := ""
			for i := 0; i < size; i++ {
				s += "1"
			}
			bs := NewUint64FromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.IsAllOnes()
			}
		})
	}
}

func BenchmarkIsAllOnes_Char(b *testing.B) {
	for _, size := range opsSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := ""
			for i := 0; i < size; i++ {
				s += "1"
			}
			bs := NewCharFromBinary(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.IsAllOnes()
			}
		})
	}
}

func BenchmarkIsAllOnes_Uint64Array(b *testing.B) {
	for _, size := range opsSizes {
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := ""
			for i := 0; i < size; i++ {
				s += "1"
			}
			bs := NewUint64ArrayFromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.IsAllOnes()
			}
		})
	}
}

// --- Successor Benchmarks ---

func BenchmarkSuccessor_Uint64(b *testing.B) {
	for _, size := range opsSizes {
		if size >= 64 {
			continue // Need room for potential overflow
		}
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			// Ensure it's not all-ones to avoid overflow issues
			if size > 0 {
				s = "0" + s[1:]
			}
			bs := NewUint64FromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.Successor()
			}
		})
	}
}

func BenchmarkSuccessor_Char(b *testing.B) {
	for _, size := range opsSizes {
		if size >= 64 {
			continue // Char delegates to Uint64BitString which has 64-bit limit
		}
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			s := randomBinaryString(size)
			if size > 0 {
				s = "0" + s[1:]
			}
			bs := NewCharFromBinary(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.Successor()
			}
		})
	}
}

func BenchmarkSuccessor_Uint64Array(b *testing.B) {
	for _, size := range opsSizes {
		// For sizes <= 64, Uint64Array delegates to Uint64BitString which has overflow issues at 64-bit boundary
		// Only test the multi-word case (>64 bits) for this implementation
		if size <= 64 {
			continue
		}
		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			// Create a safe test string that won't overflow
			s := ""
			for i := 0; i < size-1; i++ {
				s += "0"
			}
			if size > 0 {
				s += "1" // Simple pattern: 00...001
			}
			bs := NewUint64ArrayFromBinaryText(s)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bs.Successor()
			}
		})
	}
}

// --- Combined Operation Benchmarks ---

func benchmarkNewOpsBySize(b *testing.B, size int) {
	// Create test data
	s := randomBinaryString(size)
	if size >= 4 {
		s = s[:size-4] + "0000" // Add trailing zeros for TrimTrailingZeros
	}
	if size > 0 {
		s = "0" + s[1:] // Ensure not all-ones for Successor
	}

	// Create bitstrings for each implementation
	var bsUint64 Uint64BitString
	var bsChar CharBitString
	var bsUint64Array Uint64ArrayBitString

	if size <= 64 {
		bsUint64 = NewUint64FromBinaryText(s)
	}
	bsChar = NewCharFromBinary(s)
	bsUint64Array = NewUint64ArrayFromBinaryText(s)

	if size <= 64 {
		b.Run("Uint64", func(b *testing.B) {
			b.Run("TrimTrailingZeros", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = bsUint64.TrimTrailingZeros()
				}
			})
			if size < 64 {
				b.Run("AppendBit", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = bsUint64.AppendBit(true)
					}
				})
			}
			b.Run("IsAllOnes", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = bsUint64.IsAllOnes()
				}
			})
			if size <= 63 {
				b.Run("Successor", func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = bsUint64.Successor()
					}
				})
			}
		})
	}

	b.Run("Char", func(b *testing.B) {
		b.Run("TrimTrailingZeros", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bsChar.TrimTrailingZeros()
			}
		})
		b.Run("AppendBit", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bsChar.AppendBit(true)
			}
		})
		b.Run("IsAllOnes", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bsChar.IsAllOnes()
			}
		})
		if size <= 63 {
			b.Run("Successor", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = bsChar.Successor()
				}
			})
		}
	})

	b.Run("Uint64Array", func(b *testing.B) {
		b.Run("TrimTrailingZeros", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bsUint64Array.TrimTrailingZeros()
			}
		})
		b.Run("AppendBit", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bsUint64Array.AppendBit(true)
			}
		})
		b.Run("IsAllOnes", func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bsUint64Array.IsAllOnes()
			}
		})
		if size > 64 {
			b.Run("Successor", func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = bsUint64Array.Successor()
				}
			})
		}
	})
}

func BenchmarkNewOps_Size8(b *testing.B)    { benchmarkNewOpsBySize(b, 8) }
func BenchmarkNewOps_Size16(b *testing.B)   { benchmarkNewOpsBySize(b, 16) }
func BenchmarkNewOps_Size32(b *testing.B)   { benchmarkNewOpsBySize(b, 32) }
func BenchmarkNewOps_Size64(b *testing.B)   { benchmarkNewOpsBySize(b, 64) }
func BenchmarkNewOps_Size128(b *testing.B)  { benchmarkNewOpsBySize(b, 128) }
func BenchmarkNewOps_Size256(b *testing.B)  { benchmarkNewOpsBySize(b, 256) }
func BenchmarkNewOps_Size512(b *testing.B)  { benchmarkNewOpsBySize(b, 512) }
func BenchmarkNewOps_Size1024(b *testing.B) { benchmarkNewOpsBySize(b, 1024) }

// --- Edge Case Benchmarks ---

func BenchmarkTrimTrailingZeros_AllZeros_Uint64(b *testing.B) {
	bs := NewUint64FromBinaryText("000000000000000000000000000000000000000000000000000000000000000")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.TrimTrailingZeros()
	}
}

func BenchmarkTrimTrailingZeros_NoTrailingZeros_Uint64(b *testing.B) {
	bs := NewUint64FromBinaryText("1111111111111111111111111111111111111111111111111111111111111111")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.TrimTrailingZeros()
	}
}

func BenchmarkSuccessor_EmptyString(b *testing.B) {
	bs := NewUint64FromBinaryText("")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.Successor()
	}
}

func BenchmarkIsAllOnes_EmptyString(b *testing.B) {
	bs := NewUint64FromBinaryText("")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.IsAllOnes()
	}
}

func BenchmarkAppendBit_EmptyString(b *testing.B) {
	bs := NewUint64FromBinaryText("")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.AppendBit(true)
	}
}
