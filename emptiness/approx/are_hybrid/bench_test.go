package are_hybrid

import (
	"Thesis/bits"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func BenchmarkHybridARE_Build(b *testing.B) {
	sizes := []int{1 << 14, 1 << 16, 1 << 18, 1 << 20}
	epsilons := []float64{0.01, 0.001}
	rangeLen := uint64(100)

	for _, n := range sizes {
		rng := rand.New(rand.NewSource(42))
		keysClustered := generateTestClusterKeys(n, 5, 0.15, rng)
		bsClustered := makeSortedBS(keysClustered)

		for _, eps := range epsilons {
			b.Run(
				fmt.Sprintf("Clustered_N%d_eps%.0e", n, eps),
				func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						NewHybridARE(bsClustered, rangeLen, eps)
					}
				},
			)
		}

		keysUniform := generateUniformKeys(n, rand.New(rand.NewSource(42)))
		bsUniform := makeSortedBS(keysUniform)

		for _, eps := range epsilons {
			b.Run(
				fmt.Sprintf("Uniform_N%d_eps%.0e", n, eps),
				func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						NewHybridARE(bsUniform, rangeLen, eps)
					}
				},
			)
		}
	}
}

func BenchmarkHybridARE_Query(b *testing.B) {
	const n = 1 << 20
	const rangeLen = uint64(100)
	const queryCount = 1 << 16

	rng := rand.New(rand.NewSource(42))
	keysClustered := generateTestClusterKeys(n, 5, 0.15, rng)
	bsClustered := makeSortedBS(keysClustered)

	keysUniform := generateUniformKeys(n, rand.New(rand.NewSource(42)))
	bsUniform := makeSortedBS(keysUniform)

	qrng := rand.New(rand.NewSource(12345))
	queries := make([][2]bits.BitString, queryCount)
	for i := range queries {
		a := qrng.Uint64()
		queries[i] = [2]bits.BitString{
			bits.NewFromTrieUint64(a, 64),
			bits.NewFromTrieUint64(a+rangeLen-1, 64),
		}
	}

	for _, tc := range []struct {
		name string
		bs   []bits.BitString
		eps  float64
	}{
		{"Clustered_eps1e-2", bsClustered, 0.01},
		{"Clustered_eps1e-3", bsClustered, 0.001},
		{"Uniform_eps1e-2", bsUniform, 0.01},
		{"Uniform_eps1e-3", bsUniform, 0.001},
	} {
		h, err := NewHybridARE(tc.bs, rangeLen, tc.eps)
		if err != nil {
			b.Fatalf("build %s: %v", tc.name, err)
		}
		nc, nf, nt := h.Stats()
		b.Logf("%s: %d clusters, %d fallback, %d total, %.1f BPK",
			tc.name, nc, nf, nt, float64(h.SizeInBits())/float64(nt))

		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				q := queries[i%queryCount]
				h.IsEmpty(q[0], q[1])
			}
		})
	}
}

func generateUniformKeys(n int, rng *rand.Rand) []uint64 {
	seen := make(map[uint64]bool, n)
	keys := make([]uint64, 0, n)
	for len(keys) < n {
		v := rng.Uint64()
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
