package are_hybrid_scan

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"Thesis/testutils"
)

func TestHybridScanARE_Empty(t *testing.T) {
	h, err := NewHybridScanARE(nil, 100, 0.01)
	require.NoError(t, err)
	if !h.IsEmpty(trieBS(0), trieBS(1000)) {
		t.Error("empty filter should return true for any query")
	}
}

func TestHybridScanARE_SingleKey(t *testing.T) {
	bs := makeSortedBS([]uint64{42})
	h, err := NewHybridScanARE(bs, 100, 0.01)
	require.NoError(t, err)

	if h.IsEmpty(trieBS(42), trieBS(42)) {
		t.Error("false negative on single key")
	}
}

func TestHybridScanARE_NoFalseNegatives(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	keys := generateTestClusterKeys(5000, 5, 0.15, rng)

	bs := makeSortedBS(keys)
	h, err := NewHybridScanARE(bs, 100, 0.01)
	require.NoError(t, err)

	nc, nf, nt := h.Stats()
	t.Logf("Stats: %d clusters, %d fallback, %d total", nc, nf, nt)

	for i, k := range keys {
		a := trieBS(k)
		if h.IsEmpty(a, a) {
			t.Fatalf("false negative at key index %d (val=%d)", i, k)
		}
	}
}

func TestHybridScanARE_NoFalseNegatives_Sequential(t *testing.T) {
	const n = 2000
	keys := make([]uint64, n)
	for i := range keys {
		keys[i] = 1000 + uint64(i)*100
	}

	bs := makeSortedBS(keys)
	h, err := NewHybridScanARE(bs, 100, 0.01)
	require.NoError(t, err)

	nc, nf, nt := h.Stats()
	t.Logf("Stats: %d clusters, %d fallback, %d total", nc, nf, nt)

	for i, k := range keys {
		a := trieBS(k)
		if h.IsEmpty(a, a) {
			t.Fatalf("false negative at key index %d (val=%d)", i, k)
		}
	}
}

func TestHybridScanARE_FPR_Bounded(t *testing.T) {
	const (
		n          = 10000
		rangeLen   = uint64(100)
		queryCount = 100_000
		eps        = 0.01
	)

	rng := rand.New(rand.NewSource(99))
	keys := generateTestClusterKeys(n, 5, 0.15, rng)

	bs := makeSortedBS(keys)
	h, err := NewHybridScanARE(bs, rangeLen, eps)
	require.NoError(t, err)

	nc, nf, nt := h.Stats()
	t.Logf("Stats: %d clusters, %d fallback, %d total", nc, nf, nt)

	qrng := rand.New(rand.NewSource(12345))
	fp, total := 0, 0
	for i := 0; i < queryCount; i++ {
		a := qrng.Uint64()
		b := a + rangeLen - 1
		if b < a {
			continue
		}
		idx := sort.Search(len(keys), func(j int) bool { return keys[j] >= a })
		if idx < len(keys) && keys[idx] <= b {
			continue
		}
		total++
		if !h.IsEmpty(trieBS(a), trieBS(b)) {
			fp++
		}
	}

	if total == 0 {
		t.Skip("no empty queries generated")
	}

	fpr := float64(fp) / float64(total)
	t.Logf("FPR: %.6f (target eps=%.3f), tested %d empty queries", fpr, eps, total)

	if fpr > 3*eps {
		t.Errorf("FPR %.6f exceeds 3*epsilon=%.3f", fpr, 3*eps)
	}
}

func TestHybridScanARE_FPR_Accuracy(t *testing.T) {
	const n = 10000
	const numQueries = 200_000

	epsilons := []float64{0.01, 0.001}
	rangeLens := []uint64{10, 100, 1000}

	for _, eps := range epsilons {
		for _, rangeLen := range rangeLens {
			eps := eps
			rangeLen := rangeLen
			t.Run(fmt.Sprintf("eps=%.3f/L=%d", eps, rangeLen), func(t *testing.T) {
				// Uniform: all keys go to trunc fallback (optimal for uniform).
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
					filter, err := NewHybridScanARE(bsSlice, rangeLen, eps)
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
					require.Less(t, fpr, eps*3, "HybridScan FPR too high for uniform distribution")
				})

				// Tight clusters: keys with gap=1, DBSCAN will find them.
				t.Run("tight_clustered", func(t *testing.T) {
					dbscanEps := uint64(float64(rangeLen) / eps * float64(epsMultiplier))
					rng := rand.New(rand.NewSource(77))

					sortedU64 := make([]uint64, 0, n)
					seen := make(map[uint64]bool, n)
					// 5 tight clusters of 1700 keys each = 8500 + 1500 uniform
					for c := 0; c < 5; c++ {
						center := rng.Uint64() >> 1 // avoid overflow
						for i := 0; i < 1700 && len(sortedU64) < 8500; i++ {
							v := center + uint64(i)
							if !seen[v] {
								seen[v] = true
								sortedU64 = append(sortedU64, v)
							}
						}
					}
					for len(sortedU64) < n {
						v := rng.Uint64()
						if !seen[v] {
							seen[v] = true
							sortedU64 = append(sortedU64, v)
						}
					}
					sort.Slice(sortedU64, func(i, j int) bool { return sortedU64[i] < sortedU64[j] })

					bsSlice := makeSortedBS(sortedU64)
					filter, err := NewHybridScanARE(bsSlice, rangeLen, eps)
					require.NoError(t, err)

					nc, nf, _ := filter.Stats()
					t.Logf("clusters=%d fallback=%d dbscanEps=%d", nc, nf, dbscanEps)

					qrng := rand.New(rand.NewSource(456))
					queries := make([][2]uint64, numQueries)
					for i := range queries {
						a := qrng.Uint64()
						queries[i] = [2]uint64{a, a + rangeLen - 1}
					}

					isEmpty := func(a, b uint64) bool {
						return filter.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b))
					}
					fpr := testutils.MeasureFPR(sortedU64, queries, isEmpty)
					t.Logf("tight_clustered: FPR=%.5f (eps=%.4f, L=%d)", fpr, eps, rangeLen)
					require.Less(t, fpr, eps*3, "HybridScan FPR too high for tight clustered distribution")
				})

				// Sequential: gap chosen so DBSCAN eps covers multiple keys.
				t.Run("sequential", func(t *testing.T) {
					dbscanEps := uint64(float64(rangeLen) / eps * float64(epsMultiplier))
					// Choose gap so that dbscanEps / gap >= minPts (256).
					// gap = dbscanEps / 512 ensures ~512 points per window.
					gap := dbscanEps / 512
					if gap == 0 {
						gap = 1
					}
					const base = uint64(1000)
					sortedU64 := make([]uint64, n)
					for i := range sortedU64 {
						sortedU64[i] = base + uint64(i)*gap
					}

					bsSlice := makeSortedBS(sortedU64)
					filter, err := NewHybridScanARE(bsSlice, rangeLen, eps)
					require.NoError(t, err)

					nc, nf, _ := filter.Stats()
					t.Logf("clusters=%d fallback=%d dbscanEps=%d gap=%d", nc, nf, dbscanEps, gap)

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
					require.Less(t, fpr, eps*3, "HybridScan FPR too high for sequential distribution")
				})
			})
		}
	}
}

func TestHybridScanARE_SizeInBits(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	keys := generateTestClusterKeys(5000, 5, 0.15, rng)

	bs := makeSortedBS(keys)
	h, err := NewHybridScanARE(bs, 100, 0.01)
	require.NoError(t, err)

	sizeBits := h.SizeInBits()
	bpk := float64(sizeBits) / float64(len(keys))
	t.Logf("Size: %d bits (%.2f BPK for %d keys)", sizeBits, bpk, len(keys))

	if sizeBits == 0 {
		t.Error("expected non-zero size")
	}
}

func generateTestClusterKeys(n int, numClusters int, unifFrac float64, rng *rand.Rand) []uint64 {
	seen := make(map[uint64]bool)
	keys := make([]uint64, 0, n)

	nUnif := int(float64(n) * unifFrac)
	for len(keys) < nUnif {
		v := rng.Uint64()
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}

	perCluster := (n - nUnif) / numClusters
	for c := 0; c < numClusters; c++ {
		center := rng.Uint64()
		stddev := float64(uint64(1) << (20 + rng.Intn(10)))
		generated := 0
		for generated < perCluster || (c == numClusters-1 && len(keys) < n) {
			offset := int64(rng.NormFloat64() * stddev)
			var v uint64
			if offset >= 0 {
				v = center + uint64(offset)
				if v < center {
					continue
				}
			} else {
				neg := uint64(-offset)
				if neg > center {
					continue
				}
				v = center - neg
			}
			if !seen[v] {
				seen[v] = true
				keys = append(keys, v)
				generated++
			}
			if len(keys) >= n {
				break
			}
		}
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
