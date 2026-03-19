package are_soda_hash

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"Thesis/testutils"
)

func TestSODA_FPR_Accuracy(t *testing.T) {
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
					keys := make([]uint64, n)
					seen := make(map[uint64]bool, n)
					for i := 0; i < n; {
						v := rng.Uint64()
						if !seen[v] {
							seen[v] = true
							keys[i] = v
							i++
						}
					}
					sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

					f, err := NewSodaARE(keys, rangeLen, eps)
					require.NoError(t, err)

					qrng := rand.New(rand.NewSource(123))
					queries := make([][2]uint64, numQueries)
					for i := range queries {
						a := qrng.Uint64()
						queries[i] = [2]uint64{a, a + rangeLen - 1}
					}

					fpr := testutils.MeasureFPR(keys, queries, f.IsEmpty)
					t.Logf("uniform: FPR=%.5f (eps=%.4f, L=%d)", fpr, eps, rangeLen)
					require.Less(t, fpr, eps*3, "uniform: FPR should be within 3x of target epsilon")
				})

				// --- Clustered distribution ---
				t.Run("clustered", func(t *testing.T) {
					rng := rand.New(rand.NewSource(77))
					keys, clusters := testutils.GenerateClusterDistribution(n, 5, 0.15, rng)

					f, err := NewSodaARE(keys, rangeLen, eps)
					require.NoError(t, err)

					qrng := rand.New(rand.NewSource(456))
					queries := testutils.GenerateClusterQueries(numQueries, clusters, 0.15, rangeLen, qrng)

					fpr := testutils.MeasureFPR(keys, queries, f.IsEmpty)
					t.Logf("clustered: FPR=%.5f (eps=%.4f, L=%d)", fpr, eps, rangeLen)
					require.Less(t, fpr, eps*3, "clustered: FPR should be within 3x of target epsilon")
				})

				// --- Sequential distribution ---
				t.Run("sequential", func(t *testing.T) {
					const base = uint64(1000)
					const gap = uint64(1_000_000)
					keys := make([]uint64, n)
					for i := range keys {
						keys[i] = base + uint64(i)*gap
					}

					f, err := NewSodaARE(keys, rangeLen, eps)
					require.NoError(t, err)

					qrng := rand.New(rand.NewSource(789))
					queries := make([][2]uint64, numQueries)
					for i := range queries {
						a := qrng.Uint64()
						queries[i] = [2]uint64{a, a + rangeLen - 1}
					}

					fpr := testutils.MeasureFPR(keys, queries, f.IsEmpty)
					t.Logf("sequential: FPR=%.5f (eps=%.4f, L=%d)", fpr, eps, rangeLen)
					require.Less(t, fpr, eps*3, "sequential: FPR should be within 3x of target epsilon")
				})
			})
		}
	}
}
