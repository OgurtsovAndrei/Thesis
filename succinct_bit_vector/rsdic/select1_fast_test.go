package rsdic

import (
	"fmt"
	"math/rand"
	"testing"
)

// TestSelect1FastEquivalence verifies that Select1Fast returns the same
// position as the original Select1 across a variety of bitvector
// patterns and ranks.
func TestSelect1FastEquivalence(t *testing.T) {
	patterns := []struct {
		name string
		gen  func(n int, rng *rand.Rand) []bool
	}{
		{"uniform50", func(n int, rng *rand.Rand) []bool {
			b := make([]bool, n)
			for i := range b {
				b[i] = rng.Intn(2) == 1
			}
			return b
		}},
		{"sparse", func(n int, rng *rand.Rand) []bool {
			b := make([]bool, n)
			for i := range b {
				b[i] = rng.Intn(100) == 0
			}
			return b
		}},
		{"clustered_unary", func(n int, rng *rand.Rand) []bool {
			// Models the ERE encoding: 0^|B_i| 1 for ~1024 blocks, where
			// 30% of blocks have size ~22000 zeros and the rest are empty.
			out := make([]bool, 0, n)
			numBlocks := 1024
			for i := 0; i < numBlocks && len(out) < n; i++ {
				size := 0
				if rng.Intn(10) < 3 {
					size = 22000 + rng.Intn(5000)
				}
				for j := 0; j < size && len(out) < n; j++ {
					out = append(out, false)
				}
				if len(out) < n {
					out = append(out, true)
				}
			}
			for len(out) < n {
				out = append(out, false)
			}
			return out
		}},
	}

	for _, p := range patterns {
		p := p
		t.Run(p.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(42))
			bits := p.gen(1<<18, rng)
			rs := New()
			for _, b := range bits {
				rs.PushBack(b)
			}
			ones := rs.OneNum()
			if ones == 0 {
				t.Skip("no ones in bitvector")
			}
			for i := uint64(0); i < ones; i++ {
				want := rs.Select1(i)
				got := rs.Select1Fast(i)
				if got != want {
					t.Fatalf("rank=%d: Select1=%d, Select1Fast=%d", i, want, got)
				}
			}
		})
	}
}

// BenchmarkSelect1VsFastClustered models the SODA-degenerate rank stream:
// build a clustered-unary bitvector, then call Select1 (and Select1Fast)
// on small ranks (0..numBlocks-1) — the same access pattern that
// ERE.getBlockRange produces.
func BenchmarkSelect1VsFastClustered(b *testing.B) {
	const (
		numBlocks       = 1024
		populatedRatio  = 30 // percent
		populatedSize   = 22000
		ranksPerVariant = 200_000
	)

	rng := rand.New(rand.NewSource(42))
	rs := New()
	for i := 0; i < numBlocks; i++ {
		size := 0
		if rng.Intn(100) < populatedRatio {
			size = populatedSize
		}
		for j := 0; j < size; j++ {
			rs.PushBack(false)
		}
		rs.PushBack(true)
	}
	ones := rs.OneNum()
	b.Logf("rsdic: %d bits, %d ones", rs.Num(), ones)

	rng2 := rand.New(rand.NewSource(7))
	ranks := make([]uint64, ranksPerVariant)
	for i := range ranks {
		ranks[i] = uint64(rng2.Intn(numBlocks))
	}

	b.Run("Original_Select1", func(b *testing.B) {
		var sink uint64
		for i := 0; i < b.N; i++ {
			sink ^= rs.Select1(ranks[i%ranksPerVariant])
		}
		if sink == 0xDEADBEEF {
			b.Log("sink trick")
		}
	})
	b.Run("Select1Fast", func(b *testing.B) {
		var sink uint64
		for i := 0; i < b.N; i++ {
			sink ^= rs.Select1Fast(ranks[i%ranksPerVariant])
		}
		if sink == 0xDEADBEEF {
			b.Log("sink trick")
		}
	})
}

// BenchmarkSelect1VsFastUniform: same comparison with uniform-density
// bitvector and uniform random ranks. Expectation: both implementations
// are within a few ns — the original isn't pathological here.
func BenchmarkSelect1VsFastUniform(b *testing.B) {
	const n = 1 << 24
	rng := rand.New(rand.NewSource(42))
	rs := New()
	for i := 0; i < n; i++ {
		rs.PushBack(rng.Intn(2) == 1)
	}
	ones := rs.OneNum()
	rng2 := rand.New(rand.NewSource(7))
	ranks := make([]uint64, 200_000)
	for i := range ranks {
		ranks[i] = uint64(rng2.Int63n(int64(ones)))
	}

	for _, kind := range []string{"Original", "Fast"} {
		kind := kind
		b.Run(fmt.Sprintf("%s/Select1", kind), func(b *testing.B) {
			var sink uint64
			for i := 0; i < b.N; i++ {
				if kind == "Original" {
					sink ^= rs.Select1(ranks[i%len(ranks)])
				} else {
					sink ^= rs.Select1Fast(ranks[i%len(ranks)])
				}
			}
			if sink == 0xDEADBEEF {
				b.Log("sink trick")
			}
		})
	}
}
