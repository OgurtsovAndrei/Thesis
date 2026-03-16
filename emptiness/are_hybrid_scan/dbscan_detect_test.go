package are_hybrid_scan

import (
	"Thesis/bits"
	"math/rand"
	"sort"
	"testing"
)

func trieBS(val uint64) bits.BitString {
	return bits.NewFromTrieUint64(val, 64)
}

func makeSortedBS(vals []uint64) []bits.BitString {
	bs := make([]bits.BitString, len(vals))
	for i, v := range vals {
		bs[i] = trieBS(v)
	}
	return bs
}

func dedup(keys []uint64) []uint64 {
	if len(keys) == 0 {
		return keys
	}
	j := 0
	for i := 1; i < len(keys); i++ {
		if keys[i] != keys[j] {
			j++
			keys[j] = keys[i]
		}
	}
	return keys[:j+1]
}

// testEps computes eps = c * L / epsilon with default test parameters.
func testEps(rangeLen uint64, epsilon float64) uint64 {
	return uint64(float64(rangeLen) / epsilon * float64(epsMultiplier))
}

func TestDBSCAN_Sequential(t *testing.T) {
	const n = 1000
	const gap = uint64(100)
	keys := make([]uint64, n)
	for i := range keys {
		keys[i] = 1000 + uint64(i)*gap
	}

	bs := makeSortedBS(keys)
	// rangeLen=100, epsilon=0.01 -> eps = 100/0.01*10 = 100_000
	eps := testEps(100, 0.01)
	minPts := 256
	clusters, fallback := detectClustersDBSCAN(bs, eps, minPts)

	t.Logf("Sequential: eps=%d, %d clusters, %d fallback keys", eps, len(clusters), len(fallback))
	for i, c := range clusters {
		t.Logf("  Cluster %d: %d keys", i, len(c.keys))
	}

	if len(clusters) != 1 {
		t.Errorf("expected exactly 1 cluster for sequential data, got %d", len(clusters))
	}

	totalClusterKeys := 0
	for _, c := range clusters {
		totalClusterKeys += len(c.keys)
	}
	if totalClusterKeys < n*9/10 {
		t.Errorf("expected at least %d keys in clusters for sequential data, got %d", n*9/10, totalClusterKeys)
	}

	verifyTotalAssignment(t, clusters, fallback, n)
}

func TestDBSCAN_ClusteredData(t *testing.T) {
	var keys []uint64
	clusterCenters := []uint64{10_000, 1_000_000, 100_000_000, 5_000_000_000, 800_000_000_000}
	for _, center := range clusterCenters {
		for i := 0; i < 300; i++ {
			keys = append(keys, center+uint64(i))
		}
	}
	rng := rand.New(rand.NewSource(42))
	for len(keys) < 1700 {
		keys = append(keys, rng.Uint64())
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	keys = dedup(keys)

	bs := makeSortedBS(keys)
	eps := testEps(100, 0.01)
	minPts := 256
	clusters, fallback := detectClustersDBSCAN(bs, eps, minPts)

	t.Logf("Clustered: eps=%d, %d clusters, %d fallback keys (total %d)", eps, len(clusters), len(fallback), len(keys))
	for i, c := range clusters {
		t.Logf("  Cluster %d: %d keys, range [%d, %d]", i, len(c.keys), c.minKey, c.maxKey)
	}

	if len(clusters) < 1 {
		t.Errorf("expected at least 1 cluster, got %d", len(clusters))
	}

	totalClusterKeys := 0
	for _, c := range clusters {
		totalClusterKeys += len(c.keys)
	}
	if totalClusterKeys < 1500 {
		t.Errorf("expected at least 1500 keys in clusters (from 1500 tight keys), got %d", totalClusterKeys)
	}

	verifyTotalAssignment(t, clusters, fallback, len(keys))
}

func TestDBSCAN_UniformRandom(t *testing.T) {
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
	// eps = 100/0.01*10 = 100_000, avg gap for 1000 uniform uint64 keys ~ 2^47
	// So no clusters should form — everything goes to fallback (trunc).
	eps := testEps(100, 0.01)
	minPts := 256
	clusters, fallback := detectClustersDBSCAN(bs, eps, minPts)

	t.Logf("Uniform random: eps=%d, %d clusters, %d fallback keys", eps, len(clusters), len(fallback))

	totalClusterKeys := 0
	for _, c := range clusters {
		totalClusterKeys += len(c.keys)
	}
	t.Logf("  cluster keys: %d, fallback keys: %d", totalClusterKeys, len(fallback))

	// For uniform 64-bit keys, essentially all should be fallback.
	if len(fallback) < 900 {
		t.Errorf("expected most keys in fallback for uniform random, got only %d", len(fallback))
	}

	verifyTotalAssignment(t, clusters, fallback, 1000)
}

func TestDBSCAN_SingleClusterPlusScattered(t *testing.T) {
	var keys []uint64
	for i := 0; i < 500; i++ {
		keys = append(keys, 500_000+uint64(i))
	}
	rng := rand.New(rand.NewSource(99))
	for len(keys) < 600 {
		keys = append(keys, rng.Uint64())
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	keys = dedup(keys)

	bs := makeSortedBS(keys)
	eps := testEps(100, 0.01)
	minPts := 256
	clusters, fallback := detectClustersDBSCAN(bs, eps, minPts)

	t.Logf("Single cluster + scattered: eps=%d, %d clusters, %d fallback keys (total %d)", eps, len(clusters), len(fallback), len(keys))
	for i, c := range clusters {
		t.Logf("  Cluster %d: %d keys, range [%d, %d]", i, len(c.keys), c.minKey, c.maxKey)
	}

	if len(clusters) < 1 {
		t.Errorf("expected at least 1 cluster, got %d", len(clusters))
	}

	foundTight := false
	for _, c := range clusters {
		if c.minKey <= 500_000 && c.maxKey >= 500_499 {
			foundTight = true
		}
	}
	if !foundTight {
		t.Error("tight cluster [500000..500499] not detected")
	}

	verifyTotalAssignment(t, clusters, fallback, len(keys))
}

func TestDBSCAN_SmallN_BelowMinPts(t *testing.T) {
	keys := make([]uint64, 100)
	for i := range keys {
		keys[i] = uint64(i)
	}

	bs := makeSortedBS(keys)
	clusters, fallback := detectClustersDBSCAN(bs, 10, 256)

	if len(clusters) != 0 {
		t.Errorf("expected 0 clusters for n < minPts, got %d", len(clusters))
	}
	if len(fallback) != 100 {
		t.Errorf("expected 100 fallback keys, got %d", len(fallback))
	}
}

func verifyTotalAssignment(t *testing.T, clusters []clusterSegment, fallback []bits.BitString, expectedTotal int) {
	t.Helper()
	total := len(fallback)
	for _, c := range clusters {
		total += len(c.keys)
	}
	if total != expectedTotal {
		t.Errorf("key count mismatch: assigned %d, expected %d", total, expectedTotal)
	}
}
