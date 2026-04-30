package are_hybrid_scan

import (
	"Thesis/testutils"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

const (
	propTestRuns      = 1_000
	propMinN          = 100
	propMaxExtraN     = 5000
	propTargetEpsilon = 0.001
	propRangeLen      = uint64(100)
)

func setupHybridScanData(rng *rand.Rand, n int) ([]uint64, *HybridScanARE, error) {
	seen := make(map[uint64]bool)
	sortedKeys := make([]uint64, 0, n)
	for len(sortedKeys) < n {
		v := rng.Uint64()
		if !seen[v] {
			seen[v] = true
			sortedKeys = append(sortedKeys, v)
		}
	}
	sort.Slice(sortedKeys, func(i, j int) bool { return sortedKeys[i] < sortedKeys[j] })
	filter, err := NewHybridScanARE(sortedKeys, 64, Config{RangeLen: float64(propRangeLen), Eps: propTargetEpsilon})
	return sortedKeys, filter, err
}

func TestHybridScanARE_Property_PointInclusion(t *testing.T) {
	t.Parallel()
	runParallelHybridScan(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *HybridScanARE) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if filter.IsEmpty(key, key) {
				t.Errorf("Key %d not found", key)
			}
		}
	})
}

func TestHybridScanARE_Property_TightOverhang(t *testing.T) {
	t.Parallel()
	runParallelHybridScan(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *HybridScanARE) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if key > 0 {
				prev := key - 1
				if filter.IsEmpty(prev, key) {
					t.Errorf("Range [%d, %d] failed", prev, key)
				}
			}
			if key < ^uint64(0) {
				next := key + 1
				if filter.IsEmpty(key, next) {
					t.Errorf("Range [%d, %d] failed", key, next)
				}
			}
		}
	})
}

func TestHybridScanARE_Property_SpanningRanges(t *testing.T) {
	t.Parallel()
	runParallelHybridScan(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *HybridScanARE) {
		n := len(keys)
		for j := 0; j < 10; j++ {
			idx1 := rng.Intn(n - 5)
			idx2 := idx1 + 1 + rng.Intn(minInt(n-idx1-1, 50))
			a, b := keys[idx1], keys[idx2]
			if filter.IsEmpty(a, b) {
				t.Errorf("Spanning range [%d, %d] failed", a, b)
			}
		}
	})
}

func TestHybridScanARE_Property_MassiveSpan(t *testing.T) {
	t.Parallel()
	runParallelHybridScan(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *HybridScanARE) {
		if filter.IsEmpty(keys[0], keys[len(keys)-1]) {
			t.Errorf("Massive span failed")
		}
	})
}

func runParallelHybridScan(t *testing.T, testFn func(t *testing.T, rng *rand.Rand, keys []uint64, filter *HybridScanARE)) {
	for i := 0; i < propTestRuns; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 200)))
			n := propMinN + rng.Intn(propMaxExtraN)
			keys, filter, err := setupHybridScanData(rng, n)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, filter)
		})
	}
}

func setupHybridScanDataClustered(rng *rand.Rand, n int) ([]uint64, *HybridScanARE, error) {
	keys64, _ := testutils.GenerateClusterDistribution(n, 5, 0.15, rng)
	sort.Slice(keys64, func(i, j int) bool { return keys64[i] < keys64[j] })
	filter, err := NewHybridScanARE(keys64, 64, Config{RangeLen: float64(propRangeLen), Eps: propTargetEpsilon})
	return keys64, filter, err
}

func runParallelHybridScanClustered(t *testing.T, testFn func(t *testing.T, rng *rand.Rand, keys []uint64, filter *HybridScanARE)) {
	const clusterTestRuns = 200
	for i := 0; i < clusterTestRuns; i++ {
		i := i
		t.Run(fmt.Sprintf("Clustered/Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 9000)))
			n := propMinN + rng.Intn(propMaxExtraN)
			keys, filter, err := setupHybridScanDataClustered(rng, n)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, filter)
		})
	}
}

func TestHybridScanARE_Property_PointInclusion_Clustered(t *testing.T) {
	t.Parallel()
	runParallelHybridScanClustered(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *HybridScanARE) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if filter.IsEmpty(key, key) {
				t.Errorf("Key %d not found", key)
			}
		}
	})
}

func TestHybridScanARE_Property_SpanningRanges_Clustered(t *testing.T) {
	t.Parallel()
	runParallelHybridScanClustered(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *HybridScanARE) {
		n := len(keys)
		for j := 0; j < 10; j++ {
			idx1 := rng.Intn(n - 5)
			idx2 := idx1 + 1 + rng.Intn(minInt(n-idx1-1, 50))
			a, b := keys[idx1], keys[idx2]
			if filter.IsEmpty(a, b) {
				t.Errorf("Spanning range [%d, %d] failed", a, b)
			}
		}
	})
}

func TestHybridScanARE_Property_MassiveSpan_Clustered(t *testing.T) {
	t.Parallel()
	runParallelHybridScanClustered(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *HybridScanARE) {
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
