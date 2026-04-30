package rsdic

import (
	"math/rand"
	"testing"
)

// BenchmarkSelect1MixedBrackets interleaves queries across rsdics with
// very different bracket sizes (small + large) so the threshold branch
// in the adaptive Select1 sees both outcomes back-to-back. Compares
// pure-Linear, pure-Binary, and Adaptive.
//
// If the branch predictor is good, Adaptive ≈ best-of(Linear, Binary)
// per query → faster than either pure variant on average.
// If branch mispredicts add 3-15 ns/call, Adaptive may lose to whichever
// pure variant happens to be faster on average.
func BenchmarkSelect1MixedBrackets(b *testing.B) {
	const totalOnes = 1 << 14
	rsSmall := buildRSDicWithBracket(8, totalOnes)    // bracket ~ 8 (uniform-ish)
	rsLarge := buildRSDicWithBracket(4096, totalOnes) // bracket ~ 4096 (clustered)

	rng := rand.New(rand.NewSource(7))
	const kIters = 200_000
	type req struct {
		rs   *RSDic
		rank uint64
	}
	reqs := make([]req, kIters)
	for i := range reqs {
		if rng.Intn(2) == 0 {
			reqs[i] = req{rsSmall, uint64(rng.Int63n(int64(rsSmall.OneNum())))}
		} else {
			reqs[i] = req{rsLarge, uint64(rng.Int63n(int64(rsLarge.OneNum())))}
		}
	}

	b.Run("Linear", func(b *testing.B) {
		var sink uint64
		for i := 0; i < b.N; i++ {
			r := reqs[i%kIters]
			sink ^= r.rs.select1Linear(r.rank)
		}
		if sink == 0xDEADBEEF {
			b.Log("sink trick")
		}
	})
	b.Run("Binary", func(b *testing.B) {
		var sink uint64
		for i := 0; i < b.N; i++ {
			r := reqs[i%kIters]
			sink ^= r.rs.select1Binary(r.rank)
		}
		if sink == 0xDEADBEEF {
			b.Log("sink trick")
		}
	})
	b.Run("Adaptive", func(b *testing.B) {
		var sink uint64
		for i := 0; i < b.N; i++ {
			r := reqs[i%kIters]
			sink ^= r.rs.Select1(r.rank)
		}
		if sink == 0xDEADBEEF {
			b.Log("sink trick")
		}
	})
}

// BenchmarkSelect1AlternatingPattern uses a deterministic alternating
// pattern (small, large, small, large, ...) to maximally stress the
// branch predictor — every iteration the bracket size flips.
func BenchmarkSelect1AlternatingPattern(b *testing.B) {
	const totalOnes = 1 << 14
	rsSmall := buildRSDicWithBracket(8, totalOnes)
	rsLarge := buildRSDicWithBracket(4096, totalOnes)

	rng := rand.New(rand.NewSource(7))
	const kIters = 200_000
	type req struct {
		rs   *RSDic
		rank uint64
	}
	reqs := make([]req, kIters)
	for i := range reqs {
		if i%2 == 0 {
			reqs[i] = req{rsSmall, uint64(rng.Int63n(int64(rsSmall.OneNum())))}
		} else {
			reqs[i] = req{rsLarge, uint64(rng.Int63n(int64(rsLarge.OneNum())))}
		}
	}

	b.Run("Linear", func(b *testing.B) {
		var sink uint64
		for i := 0; i < b.N; i++ {
			r := reqs[i%kIters]
			sink ^= r.rs.select1Linear(r.rank)
		}
		if sink == 0xDEADBEEF {
			b.Log("sink trick")
		}
	})
	b.Run("Binary", func(b *testing.B) {
		var sink uint64
		for i := 0; i < b.N; i++ {
			r := reqs[i%kIters]
			sink ^= r.rs.select1Binary(r.rank)
		}
		if sink == 0xDEADBEEF {
			b.Log("sink trick")
		}
	})
	b.Run("Adaptive", func(b *testing.B) {
		var sink uint64
		for i := 0; i < b.N; i++ {
			r := reqs[i%kIters]
			sink ^= r.rs.Select1(r.rank)
		}
		if sink == 0xDEADBEEF {
			b.Log("sink trick")
		}
	})
}
