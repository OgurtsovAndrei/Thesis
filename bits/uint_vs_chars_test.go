package bits

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

// randomBinaryString generates a random string of '0's and '1's.
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
