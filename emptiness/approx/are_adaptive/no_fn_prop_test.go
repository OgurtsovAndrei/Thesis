package are_adaptive

import (
	"Thesis/testutils"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

const (
	propTestRuns     = 200
	propMinN         = 100
	propMaxExtraN    = 5000
	propTruncateBits = 0
	// Equivalent to (rangeLen=100, eps=0.001) for n up to ~5100:
	// K = ceil(log2(n*(L+1)/eps)) ≈ 30 bits at the upper end.
	propK = uint32(30)
)

func setupAdaptiveData(rng *rand.Rand, n int) ([]uint64, *AdaptiveARE, error) {
	keySet := make(map[uint64]bool)
	sortedKeys := make([]uint64, 0, n)
	for len(sortedKeys) < n {
		k := rng.Uint64()
		if !keySet[k] {
			keySet[k] = true
			sortedKeys = append(sortedKeys, k)
		}
	}
	sort.Slice(sortedKeys, func(i, j int) bool { return sortedKeys[i] < sortedKeys[j] })

	cfg := Config{K: propK, Threshold: propTruncateBits}
	filter, err := NewAdaptiveARE(sortedKeys, 64, cfg)
	return sortedKeys, filter, err
}

func setupAdaptiveDataClustered(rng *rand.Rand, n int) ([]uint64, *AdaptiveARE, error) {
	keys, _ := testutils.GenerateClusterDistribution(n, 5, 0.15, rng)
	cfg := Config{K: propK, Threshold: propTruncateBits}
	filter, err := NewAdaptiveARE(keys, 64, cfg)
	return keys, filter, err
}

func runParallelAdaptive(t *testing.T, testFn func(t *testing.T, rng *rand.Rand, keys []uint64, filter *AdaptiveARE)) {
	for i := 0; i < propTestRuns; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 200)))
			keys, filter, err := setupAdaptiveData(rng, propMinN+rng.Intn(propMaxExtraN))
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, filter)
		})
	}
}

func runParallelAdaptiveClustered(t *testing.T, testFn func(t *testing.T, rng *rand.Rand, keys []uint64, filter *AdaptiveARE)) {
	const clusterTestRuns = 200
	for i := 0; i < clusterTestRuns; i++ {
		i := i
		t.Run(fmt.Sprintf("Clustered/Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 9000)))
			n := propMinN + rng.Intn(propMaxExtraN)
			keys, filter, err := setupAdaptiveDataClustered(rng, n)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, filter)
		})
	}
}

func TestAdaptive_Property_PointInclusion(t *testing.T) {
	t.Parallel()
	runParallelAdaptive(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *AdaptiveARE) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if filter.IsEmpty(key, key) {
				t.Errorf("Key %v not found", key)
			}
		}
	})
}

func TestAdaptive_Property_TightOverhang(t *testing.T) {
	t.Parallel()
	runParallelAdaptive(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *AdaptiveARE) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if key != 0 {
				prev := key - 1
				if filter.IsEmpty(prev, key) {
					t.Errorf("Range [%v, %v] failed", prev, key)
				}
			}
			if key != ^uint64(0) {
				next := key + 1
				if filter.IsEmpty(key, next) {
					t.Errorf("Range [%v, %v] failed", key, next)
				}
			}
		}
	})
}

func TestAdaptive_Property_SpanningRanges(t *testing.T) {
	t.Parallel()
	runParallelAdaptive(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *AdaptiveARE) {
		n := len(keys)
		for j := 0; j < 10; j++ {
			idx1 := rng.Intn(n - 5)
			idx2 := idx1 + 1 + rng.Intn(minInt(n-idx1-1, 50))
			a, b := keys[idx1], keys[idx2]
			if filter.IsEmpty(a, b) {
				t.Errorf("Spanning range [%v, %v] failed", a, b)
			}
		}
	})
}

func TestAdaptive_Property_MassiveSpan(t *testing.T) {
	t.Parallel()
	runParallelAdaptive(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *AdaptiveARE) {
		if filter.IsEmpty(keys[0], keys[len(keys)-1]) {
			t.Errorf("Massive span failed")
		}
	})
}

func TestAdaptive_Property_PointInclusion_Clustered(t *testing.T) {
	t.Parallel()
	runParallelAdaptiveClustered(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *AdaptiveARE) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if filter.IsEmpty(key, key) {
				t.Errorf("Key %v not found", key)
			}
		}
	})
}

func TestAdaptive_Property_SpanningRanges_Clustered(t *testing.T) {
	t.Parallel()
	runParallelAdaptiveClustered(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *AdaptiveARE) {
		n := len(keys)
		for j := 0; j < 10; j++ {
			idx1 := rng.Intn(n - 5)
			idx2 := idx1 + 1 + rng.Intn(minInt(n-idx1-1, 50))
			a, b := keys[idx1], keys[idx2]
			if filter.IsEmpty(a, b) {
				t.Errorf("Spanning range [%v, %v] failed", a, b)
			}
		}
	})
}

func TestAdaptive_Property_MassiveSpan_Clustered(t *testing.T) {
	t.Parallel()
	runParallelAdaptiveClustered(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *AdaptiveARE) {
		if filter.IsEmpty(keys[0], keys[len(keys)-1]) {
			t.Errorf("Massive span failed")
		}
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
