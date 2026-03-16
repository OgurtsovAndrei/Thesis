package are_hybrid

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"Thesis/testutils"
)

func TestHybridARE_FPR_Accuracy(t *testing.T) {
	const n = 10000
	const numQueries = 200_000

	epsilons := []float64{0.1, 0.01, 0.001}
	rangeLens := []uint64{1, 10, 100, 1000}

	for _, eps := range epsilons {
		for _, rangeLen := range rangeLens {
			eps := eps
			rangeLen := rangeLen
			t.Run(fmt.Sprintf("eps=%.3f/L=%d", eps, rangeLen), func(t *testing.T) {
				// --- Uniform distribution ---
				t.Run("uniform", func(t *testing.T) {
					rng := rand.New(rand.NewSource(42))
					sortedU64 := make([]uint64, n)
					seen := make(map[uint64]bool, n)
					for i := 0; i < n; {
						v := rng.Uint64()
						if !seen[v] {
							seen[v] = true
							sortedU64[i] = v
							i++
						}
					}
					sort.Slice(sortedU64, func(i, j int) bool { return sortedU64[i] < sortedU64[j] })

					bsSlice := makeSortedBS(sortedU64)
					filter, err := NewHybridARE(bsSlice, rangeLen, eps)
					require.NoError(t, err)

					qrng := rand.New(rand.NewSource(123))
					queries := make([][2]uint64, numQueries)
					for i := range queries {
						a := qrng.Uint64()
						queries[i] = [2]uint64{a, a + rangeLen - 1}
					}

					isEmpty := func(a, b uint64) bool {
						return filter.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b))
					}
					fpr := testutils.MeasureFPR(sortedU64, queries, isEmpty)
					t.Logf("uniform: FPR=%.5f (eps=%.4f, L=%d)", fpr, eps, rangeLen)
					require.Less(t, fpr, eps*3, "Hybrid FPR too high for uniform distribution")
				})

				// --- Clustered distribution ---
				t.Run("clustered", func(t *testing.T) {
					rng := rand.New(rand.NewSource(77))
					sortedU64, clusters := testutils.GenerateClusterDistribution(n, 5, 0.15, rng)

					bsSlice := makeSortedBS(sortedU64)
					filter, err := NewHybridARE(bsSlice, rangeLen, eps)
					require.NoError(t, err)

					qrng := rand.New(rand.NewSource(456))
					queries := testutils.GenerateClusterQueries(numQueries, clusters, 0.15, rangeLen, qrng)

					isEmpty := func(a, b uint64) bool {
						return filter.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b))
					}
					fpr := testutils.MeasureFPR(sortedU64, queries, isEmpty)
					t.Logf("clustered: FPR=%.5f (eps=%.4f, L=%d)", fpr, eps, rangeLen)
					require.Less(t, fpr, eps*3, "Hybrid FPR too high for clustered distribution")
				})

				// --- Sequential distribution ---
				t.Run("sequential", func(t *testing.T) {
					t.Skip("sequential distribution not yet supported by cluster detector")

					const base = uint64(1000)
					const gap = uint64(1_000_000)
					sortedU64 := make([]uint64, n)
					for i := range sortedU64 {
						sortedU64[i] = base + uint64(i)*gap
					}

					bsSlice := makeSortedBS(sortedU64)
					filter, err := NewHybridARE(bsSlice, rangeLen, eps)
					require.NoError(t, err)

					qrng := rand.New(rand.NewSource(789))
					queries := make([][2]uint64, numQueries)
					for i := range queries {
						a := qrng.Uint64()
						queries[i] = [2]uint64{a, a + rangeLen - 1}
					}

					isEmpty := func(a, b uint64) bool {
						return filter.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b))
					}
					fpr := testutils.MeasureFPR(sortedU64, queries, isEmpty)
					t.Logf("sequential: FPR=%.5f (eps=%.4f, L=%d)", fpr, eps, rangeLen)
					require.Less(t, fpr, eps*3, "Hybrid FPR too high for sequential distribution")
				})
			})
		}
	}
}
