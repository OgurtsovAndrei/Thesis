package are_hybrid

import (
	"Thesis/bits"
	"math/rand"
	"sort"
	"testing"
)

func trieBS(val uint64) bits.BitString {
	return bits.NewFromTrieUint64(val, 64)
}

// makeSortedBS converts sorted uint64 slice to BitString slice.
func makeSortedBS(vals []uint64) []bits.BitString {
	bs := make([]bits.BitString, len(vals))
	for i, v := range vals {
		bs[i] = trieBS(v)
	}
	return bs
}

func TestDetectClusters_Basic(t *testing.T) {
	// 3 tight clusters of 50 keys each + 50 scattered keys = 200 total
	var keys []uint64

	// Cluster 1: around 10000
	for i := 0; i < 50; i++ {
		keys = append(keys, 10000+uint64(i))
	}
	// Cluster 2: around 1_000_000
	for i := 0; i < 50; i++ {
		keys = append(keys, 1_000_000+uint64(i))
	}
	// Cluster 3: around 100_000_000
	for i := 0; i < 50; i++ {
		keys = append(keys, 100_000_000+uint64(i))
	}
	// Scattered: spread across a huge range
	rng := rand.New(rand.NewSource(42))
	for len(keys) < 200 {
		keys = append(keys, rng.Uint64())
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	// Deduplicate
	j := 0
	for i := 1; i < len(keys); i++ {
		if keys[i] != keys[j] {
			j++
			keys[j] = keys[i]
		}
	}
	keys = keys[:j+1]

	bs := makeSortedBS(keys)
	clusters, fallback := detectClusters(bs, 10, 0.05)

	t.Logf("Found %d clusters, %d fallback keys (total %d)", len(clusters), len(fallback), len(keys))
	for i, c := range clusters {
		t.Logf("  Cluster %d: %d keys, range [%d, %d]", i, len(c.keys), c.minKey, c.maxKey)
	}

	// The 3 tight clusters (10K, 1M, 100M) are very close relative to random uint64 keys,
	// so they may be merged into 1. That's valid — SODA handles merged clusters fine.
	if len(clusters) < 1 {
		t.Errorf("expected at least 1 cluster, got %d", len(clusters))
	}

	// All 150 cluster keys must be in clusters (not fallback)
	totalClusterKeys := 0
	for _, c := range clusters {
		totalClusterKeys += len(c.keys)
	}
	if totalClusterKeys < 150 {
		t.Errorf("expected at least 150 keys in clusters, got %d", totalClusterKeys)
	}

	// Every key must be in exactly one place
	totalAssigned := len(fallback)
	for _, c := range clusters {
		totalAssigned += len(c.keys)
	}
	if totalAssigned != len(keys) {
		t.Errorf("key count mismatch: assigned %d, total %d", totalAssigned, len(keys))
	}
}

func TestDetectClusters_SmallClusterPruned(t *testing.T) {
	var keys []uint64

	// Small cluster: 3 keys
	for i := 0; i < 3; i++ {
		keys = append(keys, 1000+uint64(i))
	}
	// Big cluster: 100 keys
	for i := 0; i < 100; i++ {
		keys = append(keys, 1_000_000+uint64(i))
	}
	// Scattered: 97 keys
	rng := rand.New(rand.NewSource(77))
	for len(keys) < 200 {
		keys = append(keys, rng.Uint64())
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	bs := makeSortedBS(keys)
	clusters, fallback := detectClusters(bs, 10, 0.05) // threshold = 5% of 200 = 10

	// Small cluster of 3 should have been pruned to fallback
	for _, c := range clusters {
		if len(c.keys) < 10 {
			t.Errorf("cluster with %d keys should have been pruned (threshold=10)", len(c.keys))
		}
	}

	totalAssigned := len(fallback)
	for _, c := range clusters {
		totalAssigned += len(c.keys)
	}
	if totalAssigned != len(keys) {
		t.Errorf("key count mismatch: assigned %d, total %d", totalAssigned, len(keys))
	}

	t.Logf("Clusters: %d, Fallback: %d", len(clusters), len(fallback))
}

func TestDetectClusters_AllUniform(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	seen := make(map[uint64]bool)
	var keys []uint64
	for len(keys) < 1000 {
		v := rng.Uint64()
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	bs := makeSortedBS(keys)
	clusters, fallback := detectClusters(bs, 10, 0.05)

	t.Logf("Uniform: %d clusters, %d fallback", len(clusters), len(fallback))

	totalAssigned := len(fallback)
	for _, c := range clusters {
		totalAssigned += len(c.keys)
	}
	if totalAssigned != 1000 {
		t.Errorf("key count mismatch: assigned %d, expected 1000", totalAssigned)
	}
}

func TestHybridARE_Empty(t *testing.T) {
	h, err := NewHybridARE(nil, 100, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if !h.IsEmpty(trieBS(0), trieBS(1000)) {
		t.Error("empty filter should return true for any query")
	}
}

func TestHybridARE_NoFalseNegatives(t *testing.T) {
	// Generate cluster distribution
	rng := rand.New(rand.NewSource(99))
	keys := generateTestClusterKeys(5000, 5, 0.15, rng)

	bs := makeSortedBS(keys)
	h, err := NewHybridARE(bs, 100, 0.01)
	if err != nil {
		t.Fatal(err)
	}

	nc, nf, nt := h.Stats()
	t.Logf("Stats: %d clusters, %d fallback, %d total", nc, nf, nt)

	// Every inserted key must NOT be reported as empty
	for i, k := range keys {
		a := trieBS(k)
		if h.IsEmpty(a, a) {
			t.Fatalf("false negative at key index %d (val=%d)", i, k)
		}
	}
}

func TestHybridARE_FPR_Bounded(t *testing.T) {
	const (
		n          = 10000
		rangeLen   = uint64(100)
		queryCount = 100_000
		eps        = 0.01
	)

	rng := rand.New(rand.NewSource(99))
	keys := generateTestClusterKeys(n, 5, 0.15, rng)

	bs := makeSortedBS(keys)
	h, err := NewHybridARE(bs, rangeLen, eps)
	if err != nil {
		t.Fatal(err)
	}

	nc, nf, nt := h.Stats()
	t.Logf("Stats: %d clusters, %d fallback, %d total", nc, nf, nt)

	// Generate queries
	qrng := rand.New(rand.NewSource(12345))
	fp, total := 0, 0
	for i := 0; i < queryCount; i++ {
		a := qrng.Uint64()
		b := a + rangeLen - 1
		if b < a {
			continue // overflow
		}
		// Ground truth
		idx := sort.Search(len(keys), func(j int) bool { return keys[j] >= a })
		if idx < len(keys) && keys[idx] <= b {
			continue // non-empty range
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
	t.Logf("FPR: %.6f (target ε=%.3f), tested %d empty queries", fpr, eps, total)

	// Allow 3x epsilon as margin for statistical variance
	if fpr > 3*eps {
		t.Errorf("FPR %.6f exceeds 3*epsilon=%.3f", fpr, 3*eps)
	}
}

// generateTestClusterKeys creates n keys: unifFrac uniform + rest from Gaussian clusters.
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
					continue // overflow
				}
			} else {
				neg := uint64(-offset)
				if neg > center {
					continue // underflow
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
