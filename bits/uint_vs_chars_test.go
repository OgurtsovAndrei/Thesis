// Package bits performance benchmarks
//
// Comparison of BitString implementations:
// 1. Uint64BitString (your implementation) - Fast bit access using uint64 operations (uint64 input)
// 2. CharBitString (your implementation) - Character-based bit string operations (text input)
// 3. trie.BitString - From siongui/go-succinct-data-structure-trie (base64 input)
//
// Data Formats:
// - Uint64BitString: Uses NewUint64BitString(value, length) with uint64 values (≤64 bits)
// - CharBitString: Uses NewCharBitString() with ASCII text (8 chars = 64 bits)
// - trie.BitString: Uses base64 encoded strings (16 chars = ~96 bits)
//
// Benchmark Results (Apple M4) - Updated with correct constructors:
//
// Access Performance:
// - CharBitString:       ~0.35 ns/op (fastest - character-based access)
// - Uint64BitString:     ~0.99 ns/op (direct uint64 bit operations)
// - trie.BitString:      ~6.4 ns/op  (18x slower, but provides Count/Rank)
//
// Initialization Performance:
// - Uint64BitString:     ~0.23 ns/op (fastest - direct uint64 creation)
// - CharBitString:       ~0.23 ns/op (fast text processing)
// - trie.BitString:      ~0.23 ns/op (all very fast initialization)

package bits

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	trie "github.com/siongui/go-succinct-data-structure-trie/reference"
)

// randomTextString generates a random ASCII text string for CharBitString
func randomTextString(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		// Generate printable ASCII characters (32-126)
		char := byte(32 + r.Intn(95))
		sb.WriteByte(char)
	}
	return sb.String()
}

// randomUint64 generates a random uint64 value for Uint64BitString
func randomUint64() uint64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Uint64()
}

// randomBinaryString generates a random string of '0's and '1's (for legacy tests).
func randomBinaryString(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		if r.Intn(2) == 0 {
			sb.WriteByte('0')
		} else {
			sb.WriteByte('1')
		}
	}
	return sb.String()
}

// randomBase64String generates a random base64 string for trie.BitString
func randomBase64String(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var sb strings.Builder
	sb.Grow(length)
	for i := 0; i < length; i++ {
		sb.WriteByte(base64Chars[r.Intn(len(base64Chars))])
	}
	return sb.String()
}

// --- LCP Benchmarks ---

func BenchmarkRandom_LCP_Uint64(b *testing.B) {
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

func BenchmarkRandom_LCP_Char(b *testing.B) {
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

// --- At Benchmarks ---

func BenchmarkRandom_At_Uint64(b *testing.B) {
	input := randomBinaryString(64)
	bs := NewUint64FromBinaryText(input)
	limit := uint32(64)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.At(uint32(i) % limit)
	}
}

func BenchmarkRandom_At_Char(b *testing.B) {
	input := randomBinaryString(64)
	bs := NewCharFromBinary(input)
	limit := uint32(64)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.At(uint32(i) % limit)
	}
}

// --- Equal Benchmarks ---

func BenchmarkRandom_Equal_Uint64(b *testing.B) {
	s1 := randomBinaryString(64)
	// Equal compares exact values, so we test the positive case (identical)
	// which is often the "slowest" path for naive algorithms as they check everything.
	bs1 := NewUint64FromBinaryText(s1)
	bs2 := NewUint64FromBinaryText(s1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.Equal(bs2)
	}
}

func BenchmarkRandom_Equal_Char(b *testing.B) {
	s1 := randomBinaryString(64)
	bs1 := NewCharFromBinary(s1)
	bs2 := NewCharFromBinary(s1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.Equal(bs2)
	}
}

// --- HasPrefix Benchmarks ---

func BenchmarkRandom_HasPrefix_Uint64(b *testing.B) {
	s1 := randomBinaryString(64)
	sPrefix := s1[:32] // 32-bit prefix

	bs1 := NewUint64FromBinaryText(s1)
	bsPrefix := NewUint64FromBinaryText(sPrefix)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.HasPrefix(bsPrefix)
	}
}

func BenchmarkRandom_HasPrefix_Char(b *testing.B) {
	s1 := randomBinaryString(64)
	sPrefix := s1[:32]

	bs1 := NewCharFromBinary(s1)
	bsPrefix := NewCharFromBinary(sPrefix)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs1.HasPrefix(bsPrefix)
	}
}

// --- Benchmarks comparing with trie.BitString ---

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

// --- Comparative benchmarks: Access/At operation ---

func BenchmarkCompare_Access_Uint64(b *testing.B) {
	value := randomUint64()
	bs := NewUint64BitString(value, 64) // 64-bit uint64
	limit := uint32(64)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.At(uint32(i) % limit)
	}
}

func BenchmarkCompare_Access_Char(b *testing.B) {
	input := randomTextString(8) // 8 chars = 64 bits
	bs := NewCharBitString(input)
	limit := uint32(64)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.At(uint32(i) % limit)
	}
}

func BenchmarkCompare_Access_TrieBitString(b *testing.B) {
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

// --- Performance comparison for different string sizes ---

func benchmarkAccessBySize(b *testing.B, size int) {
	// Generate appropriate data for each implementation
	textInput := randomTextString(size / 8)     // size in bits -> size/8 chars
	base64Input := randomBase64String(size / 6) // ~6 bits per base64 char
	uint64Value := randomUint64()

	// Your implementations
	var bsUint64 Uint64BitString
	var bsChar CharBitString

	if size <= 64 { // Uint64 limited to 64 bits
		bsUint64 = NewUint64BitString(uint64Value, int8(size))
	}
	bsChar = NewCharBitString(textInput)

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

// --- Initialization benchmarks by size ---

func benchmarkInitBySize(b *testing.B, size int) {
	// Generate appropriate data for each implementation
	textInput := randomTextString(size / 8)     // size in bits -> size/8 chars
	base64Input := randomBase64String(size / 6) // ~6 bits per base64 char

	b.Run("Uint64", func(b *testing.B) {
		if size > 64 {
			b.Skip("size > 64")
		}
		value := randomUint64() // Generate once outside the loop
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewUint64BitString(value, int8(size))
		}
	})

	b.Run("Char", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewCharBitString(textInput)
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
