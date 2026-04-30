package are_hybrid

import (
	"math/rand"
	"sort"
	"testing"
)

func TestDBSCAN_ClusteredData(t *testing.T) {
	// 5 tight clusters of 100 keys each + 100 uniform noise keys = 600 total
	var keys []uint64
	clusterCenters := []uint64{10_000, 1_000_000, 100_000_000, 5_000_000_000, 800_000_000_000}
	for _, center := range clusterCenters {
		for i := 0; i < 100; i++ {
			keys = append(keys, center+uint64(i))
		}
	}
	rng := rand.New(rand.NewSource(42))
	for len(keys) < 600 {
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
	clusters, fallback := detectClusters(bs, 0.95, 0.01)

	t.Logf("Found %d clusters, %d fallback keys (total %d)", len(clusters), len(fallback), len(keys))
	for i, c := range clusters {
		t.Logf("  Cluster %d: %d keys, range [%d, %d]", i, len(c.keys), c.minKey, c.maxKey)
	}

	if len(clusters) < 1 {
		t.Errorf("expected at least 1 cluster, got %d", len(clusters))
	}

	// All 500 tight cluster keys must end up in some cluster (not fallback).
	totalClusterKeys := 0
	for _, c := range clusters {
		totalClusterKeys += len(c.keys)
	}
	if totalClusterKeys < 500 {
		t.Errorf("expected at least 500 keys in clusters (from 500 tight keys), got %d", totalClusterKeys)
	}

	// Verify total assignment
	totalAssigned := len(fallback)
	for _, c := range clusters {
		totalAssigned += len(c.keys)
	}
	if totalAssigned != len(keys) {
		t.Errorf("key count mismatch: assigned %d, total %d", totalAssigned, len(keys))
	}
}

func TestDBSCAN_Sequential(t *testing.T) {
	// Evenly spaced keys: should be detected as one big cluster, NOT all fallback.
	// This is the key regression test: the old >= threshold broke on equal gaps.
	const n = 1000
	const gap = uint64(100)
	keys := make([]uint64, n)
	for i := range keys {
		keys[i] = 1000 + uint64(i)*gap
	}

	bs := makeSortedBS(keys)
	clusters, fallback := detectClusters(bs, 0.95, 0.01)

	t.Logf("Sequential: %d clusters, %d fallback keys", len(clusters), len(fallback))
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

	if len(fallback) > n/10 {
		t.Errorf("too many fallback keys for sequential data: %d (expected < %d)", len(fallback), n/10)
	}
}

func TestDBSCAN_UniformRandom(t *testing.T) {
	// Uniform random keys over the full uint64 range.
	// With P95 eps, most consecutive pairs are within eps, so most keys
	// end up in clusters. The test verifies total assignment consistency.
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
	clusters, fallback := detectClusters(bs, 0.95, 0.01)

	t.Logf("Uniform random: %d clusters, %d fallback keys", len(clusters), len(fallback))

	totalClusterKeys := 0
	for _, c := range clusters {
		totalClusterKeys += len(c.keys)
	}

	totalAssigned := len(fallback) + totalClusterKeys
	if totalAssigned != 1000 {
		t.Errorf("key count mismatch: assigned %d, expected 1000", totalAssigned)
	}

	t.Logf("  cluster keys: %d, fallback keys: %d", totalClusterKeys, len(fallback))
}

func TestDBSCAN_SingleClusterPlusScattered(t *testing.T) {
	// One tight cluster of 200 keys + 50 widely scattered keys.
	var keys []uint64
	for i := 0; i < 200; i++ {
		keys = append(keys, 500_000+uint64(i))
	}
	rng := rand.New(rand.NewSource(99))
	for len(keys) < 250 {
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
	clusters, fallback := detectClusters(bs, 0.95, 0.01)

	t.Logf("Single cluster + scattered: %d clusters, %d fallback keys (total %d)", len(clusters), len(fallback), len(keys))
	for i, c := range clusters {
		t.Logf("  Cluster %d: %d keys, range [%d, %d]", i, len(c.keys), c.minKey, c.maxKey)
	}

	if len(clusters) < 1 {
		t.Errorf("expected at least 1 cluster, got %d", len(clusters))
	}

	// The tight cluster (200 keys) should be detected as part of some cluster.
	foundTight := false
	for _, c := range clusters {
		if c.minKey <= 500_000 && c.maxKey >= 500_199 {
			foundTight = true
		}
	}
	if !foundTight {
		t.Error("tight cluster [500000..500199] not detected")
	}

	// Verify total assignment
	totalAssigned := len(fallback)
	for _, c := range clusters {
		totalAssigned += len(c.keys)
	}
	if totalAssigned != len(keys) {
		t.Errorf("key count mismatch: assigned %d, total %d", totalAssigned, len(keys))
	}
}
