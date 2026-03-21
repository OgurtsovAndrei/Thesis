package succinct_bit_vector

import (
	"math/rand"
	"testing"

	orig "github.com/hillbig/rsdic"
	fork "Thesis/succinct_bit_vector/rsdic"
)

const benchSize = 1000000

func setupOrig(n int, density float32) *orig.RSDic {
	rsd := orig.New()
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < n; i++ {
		rsd.PushBack(rng.Float32() < density)
	}
	return rsd
}

func setupFork(n int, density float32) *fork.RSDic {
	rsd := fork.New()
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < n; i++ {
		rsd.PushBack(rng.Float32() < density)
	}
	return rsd
}

// --- Dense (50%) ---

func BenchmarkBit_Dense_Orig(b *testing.B) {
	rsd := setupOrig(benchSize, 0.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Bit(uint64(rand.Int31n(int32(benchSize))))
	}
}

func BenchmarkBit_Dense_Fork(b *testing.B) {
	rsd := setupFork(benchSize, 0.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Bit(uint64(rand.Int31n(int32(benchSize))))
	}
}

func BenchmarkRank_Dense_Orig(b *testing.B) {
	rsd := setupOrig(benchSize, 0.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Rank(uint64(rand.Int31n(int32(benchSize))), true)
	}
}

func BenchmarkRank_Dense_Fork(b *testing.B) {
	rsd := setupFork(benchSize, 0.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Rank(uint64(rand.Int31n(int32(benchSize))), true)
	}
}

func BenchmarkSelect_Dense_Orig(b *testing.B) {
	rsd := setupOrig(benchSize, 0.5)
	oneNum := rsd.OneNum()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Select(uint64(rand.Int31n(int32(oneNum))), true)
	}
}

func BenchmarkSelect_Dense_Fork(b *testing.B) {
	rsd := setupFork(benchSize, 0.5)
	oneNum := rsd.OneNum()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Select(uint64(rand.Int31n(int32(oneNum))), true)
	}
}

// --- Sparse (1%) ---

func BenchmarkBit_Sparse_Orig(b *testing.B) {
	rsd := setupOrig(benchSize, 0.01)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Bit(uint64(rand.Int31n(int32(benchSize))))
	}
}

func BenchmarkBit_Sparse_Fork(b *testing.B) {
	rsd := setupFork(benchSize, 0.01)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Bit(uint64(rand.Int31n(int32(benchSize))))
	}
}

func BenchmarkRank_Sparse_Orig(b *testing.B) {
	rsd := setupOrig(benchSize, 0.01)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Rank(uint64(rand.Int31n(int32(benchSize))), true)
	}
}

func BenchmarkRank_Sparse_Fork(b *testing.B) {
	rsd := setupFork(benchSize, 0.01)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Rank(uint64(rand.Int31n(int32(benchSize))), true)
	}
}

func BenchmarkSelect_Sparse_Orig(b *testing.B) {
	rsd := setupOrig(benchSize, 0.01)
	oneNum := rsd.OneNum()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Select(uint64(rand.Int31n(int32(oneNum))), true)
	}
}

func BenchmarkSelect_Sparse_Fork(b *testing.B) {
	rsd := setupFork(benchSize, 0.01)
	oneNum := rsd.OneNum()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Select(uint64(rand.Int31n(int32(oneNum))), true)
	}
}

// --- 33% density (matches D2 in ERE) ---

func BenchmarkBit_D2_Orig(b *testing.B) {
	rsd := setupOrig(benchSize, 0.33)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Bit(uint64(rand.Int31n(int32(benchSize))))
	}
}

func BenchmarkBit_D2_Fork(b *testing.B) {
	rsd := setupFork(benchSize, 0.33)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Bit(uint64(rand.Int31n(int32(benchSize))))
	}
}

func BenchmarkRank_D2_Orig(b *testing.B) {
	rsd := setupOrig(benchSize, 0.33)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Rank(uint64(rand.Int31n(int32(benchSize))), true)
	}
}

func BenchmarkRank_D2_Fork(b *testing.B) {
	rsd := setupFork(benchSize, 0.33)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Rank(uint64(rand.Int31n(int32(benchSize))), true)
	}
}

func BenchmarkSelect_D2_Orig(b *testing.B) {
	rsd := setupOrig(benchSize, 0.33)
	oneNum := rsd.OneNum()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Select(uint64(rand.Int31n(int32(oneNum))), true)
	}
}

func BenchmarkSelect_D2_Fork(b *testing.B) {
	rsd := setupFork(benchSize, 0.33)
	oneNum := rsd.OneNum()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsd.Select(uint64(rand.Int31n(int32(oneNum))), true)
	}
}
