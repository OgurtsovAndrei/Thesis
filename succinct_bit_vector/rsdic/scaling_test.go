package rsdic

import (
	"fmt"
	"math/rand"
	"testing"
)

// buildRSDic constructs an RSDic of N random bits at ~50% density and returns
// it together with the count of one-bits.
func buildRSDic(n uint64, seed int64) (*RSDic, uint64) {
	rng := rand.New(rand.NewSource(seed))
	rs := New()
	var ones uint64
	for i := uint64(0); i < n; i++ {
		if rng.Intn(2) == 1 {
			rs.PushBack(true)
			ones++
		} else {
			rs.PushBack(false)
		}
	}
	return rs, ones
}

var scalingSizes = []uint64{
	1 << 20,
	1 << 22,
	1 << 24,
	1 << 26,
	1 << 28,
}

const kIters = 200_000

// BenchmarkSelectScaling measures Select1 latency as a function of bitvector size.
func BenchmarkSelectScaling(b *testing.B) {
	for _, n := range scalingSizes {
		n := n
		b.Run(fmt.Sprintf("N=2^%d", log2(n)), func(b *testing.B) {
			rs, ones := buildRSDic(n, 42)
			if ones == 0 {
				b.Fatalf("no ones in bitvector of size %d", n)
			}
			rng := rand.New(rand.NewSource(7))
			ranks := make([]uint64, kIters)
			for i := range ranks {
				ranks[i] = uint64(rng.Int63n(int64(ones)))
			}
			bytes := float64(rs.AllocSize())
			b.ResetTimer()
			var sink uint64
			for i := 0; i < b.N; i++ {
				sink ^= rs.Select1(ranks[i%kIters])
			}
			b.StopTimer()
			if sink == 0xDEADBEEF {
				b.Log("sink trick")
			}
			b.ReportMetric(bytes/(1024*1024), "bitvec_MB")
		})
	}
}

// BenchmarkRankScaling measures Rank(pos, true) latency as a function of bitvector size.
func BenchmarkRankScaling(b *testing.B) {
	for _, n := range scalingSizes {
		n := n
		b.Run(fmt.Sprintf("N=2^%d", log2(n)), func(b *testing.B) {
			rs, _ := buildRSDic(n, 42)
			rng := rand.New(rand.NewSource(7))
			positions := make([]uint64, kIters)
			for i := range positions {
				positions[i] = uint64(rng.Int63n(int64(n)))
			}
			bytes := float64(rs.AllocSize())
			b.ResetTimer()
			var sink uint64
			for i := 0; i < b.N; i++ {
				sink ^= rs.Rank(positions[i%kIters], true)
			}
			b.StopTimer()
			if sink == 0xDEADBEEF {
				b.Log("sink trick")
			}
			b.ReportMetric(bytes/(1024*1024), "bitvec_MB")
		})
	}
}

// BenchmarkSelectInterleavedRank measures the cost of a Select(k) followed
// immediately by Select(k+1) — the access pattern used by ERE.getBlockRange.
// One b.N iteration corresponds to one such *pair* of calls.
func BenchmarkSelectInterleavedRank(b *testing.B) {
	for _, n := range scalingSizes {
		n := n
		b.Run(fmt.Sprintf("N=2^%d", log2(n)), func(b *testing.B) {
			rs, ones := buildRSDic(n, 42)
			if ones < 2 {
				b.Fatalf("insufficient ones in bitvector of size %d", n)
			}
			rng := rand.New(rand.NewSource(7))
			ranks := make([]uint64, kIters)
			for i := range ranks {
				ranks[i] = uint64(rng.Int63n(int64(ones - 1)))
			}
			bytes := float64(rs.AllocSize())
			b.ResetTimer()
			var sink uint64
			for i := 0; i < b.N; i++ {
				r := ranks[i%kIters]
				sink ^= rs.Select1(r)
				sink ^= rs.Select1(r + 1)
			}
			b.StopTimer()
			if sink == 0xDEADBEEF {
				b.Log("sink trick")
			}
			b.ReportMetric(bytes/(1024*1024), "bitvec_MB")
		})
	}
}

// BenchmarkSelect1Pair measures the cost of Select1Pair(k) — single call that
// returns both Select1(k) and Select1(k+1). One b.N iteration corresponds to
// one Select1Pair call, directly comparable with BenchmarkSelectInterleavedRank.
func BenchmarkSelect1Pair(b *testing.B) {
	for _, n := range scalingSizes {
		n := n
		b.Run(fmt.Sprintf("N=2^%d", log2(n)), func(b *testing.B) {
			rs, ones := buildRSDic(n, 42)
			if ones < 2 {
				b.Fatalf("insufficient ones in bitvector of size %d", n)
			}
			rng := rand.New(rand.NewSource(7))
			ranks := make([]uint64, kIters)
			for i := range ranks {
				ranks[i] = uint64(rng.Int63n(int64(ones - 1)))
			}
			bytes := float64(rs.AllocSize())
			b.ResetTimer()
			var sink uint64
			for i := 0; i < b.N; i++ {
				a, c := rs.Select1Pair(ranks[i%kIters])
				sink ^= a
				sink ^= c
			}
			b.StopTimer()
			if sink == 0xDEADBEEF {
				b.Log("sink trick")
			}
			b.ReportMetric(bytes/(1024*1024), "bitvec_MB")
		})
	}
}

func log2(x uint64) int {
	r := 0
	for x > 1 {
		x >>= 1
		r++
	}
	return r
}
