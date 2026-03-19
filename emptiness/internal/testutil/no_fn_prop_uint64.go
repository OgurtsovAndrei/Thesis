package testutil

import (
	"Thesis/testutils"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

// Uint64Checker is implemented by any range emptiness filter that operates on uint64 keys.
type Uint64Checker interface {
	IsEmpty(a, b uint64) bool
}

// BuildUint64Filter constructs a filter from uint64 keys and returns it as a Uint64Checker.
type BuildUint64Filter func(keys []uint64, rng *rand.Rand) (Uint64Checker, error)

// RunUint64NoFNProps runs the standard no-false-negative property tests for a uint64-keyed filter.
// maxQueryLen constrains the maximum query range length used in spanning-range tests.
func RunUint64NoFNProps(t *testing.T, testRuns, minN, maxExtraN int, maxQueryLen uint64, build BuildUint64Filter) {
	t.Helper()

	t.Run("PointInclusion", func(t *testing.T) {
		t.Parallel()
		runParallelUint64(t, testRuns, minN, maxExtraN, build, func(t *testing.T, rng *rand.Rand, keys []uint64, f Uint64Checker) {
			for j := 0; j < 20; j++ {
				key := keys[rng.Intn(len(keys))]
				if f.IsEmpty(key, key) {
					t.Errorf("Key %v not found", key)
				}
			}
		})
	})

	t.Run("TightOverhang", func(t *testing.T) {
		t.Parallel()
		runParallelUint64(t, testRuns, minN, maxExtraN, build, func(t *testing.T, rng *rand.Rand, keys []uint64, f Uint64Checker) {
			for j := 0; j < 20; j++ {
				key := keys[rng.Intn(len(keys))]
				if key > 0 {
					if f.IsEmpty(key-1, key) {
						t.Errorf("Range [%v, %v] failed", key-1, key)
					}
				}
				if key < ^uint64(0) {
					if f.IsEmpty(key, key+1) {
						t.Errorf("Range [%v, %v] failed", key, key+1)
					}
				}
			}
		})
	})

	t.Run("SpanningRanges", func(t *testing.T) {
		t.Parallel()
		runParallelUint64(t, testRuns, minN, maxExtraN, build, func(t *testing.T, rng *rand.Rand, keys []uint64, f Uint64Checker) {
			n := len(keys)
			for j := 0; j < 10; j++ {
				idx1 := rng.Intn(n - 5)
				idx2 := idx1 + 1 + rng.Intn(minInt(n-idx1-1, 50))
				a, b := keys[idx1], keys[idx2]
				if b-a > maxQueryLen {
					b = a + maxQueryLen
				}
				if b >= keys[idx1] && f.IsEmpty(a, b) {
					t.Errorf("Spanning range [%v, %v] failed", a, b)
				}
			}
		})
	})
}

// RunUint64NoFNPropsClustered runs the clustered variant of no-FN property tests for uint64-keyed filters.
func RunUint64NoFNPropsClustered(t *testing.T, clusterRuns, minN, maxExtraN int, maxQueryLen uint64, build BuildUint64Filter) {
	t.Helper()

	t.Run("PointInclusion_Clustered", func(t *testing.T) {
		t.Parallel()
		runParallelUint64Clustered(t, clusterRuns, minN, maxExtraN, build, func(t *testing.T, rng *rand.Rand, keys []uint64, f Uint64Checker) {
			for j := 0; j < 20; j++ {
				key := keys[rng.Intn(len(keys))]
				if f.IsEmpty(key, key) {
					t.Errorf("Key %v not found", key)
				}
			}
		})
	})

	t.Run("TightOverhang_Clustered", func(t *testing.T) {
		t.Parallel()
		runParallelUint64Clustered(t, clusterRuns, minN, maxExtraN, build, func(t *testing.T, rng *rand.Rand, keys []uint64, f Uint64Checker) {
			for j := 0; j < 20; j++ {
				key := keys[rng.Intn(len(keys))]
				if key > 0 {
					if f.IsEmpty(key-1, key) {
						t.Errorf("Range [%v, %v] failed", key-1, key)
					}
				}
				if key < ^uint64(0) {
					if f.IsEmpty(key, key+1) {
						t.Errorf("Range [%v, %v] failed", key, key+1)
					}
				}
			}
		})
	})

	t.Run("SpanningRanges_Clustered", func(t *testing.T) {
		t.Parallel()
		runParallelUint64Clustered(t, clusterRuns, minN, maxExtraN, build, func(t *testing.T, rng *rand.Rand, keys []uint64, f Uint64Checker) {
			n := len(keys)
			for j := 0; j < 10; j++ {
				idx1 := rng.Intn(n - 5)
				idx2 := idx1 + 1 + rng.Intn(minInt(n-idx1-1, 50))
				a, b := keys[idx1], keys[idx2]
				if b-a > maxQueryLen {
					b = a + maxQueryLen
				}
				if b >= keys[idx1] && f.IsEmpty(a, b) {
					t.Errorf("Spanning range [%v, %v] failed", a, b)
				}
			}
		})
	})
}

func runParallelUint64(
	t *testing.T,
	testRuns, minN, maxExtraN int,
	build BuildUint64Filter,
	testFn func(t *testing.T, rng *rand.Rand, keys []uint64, f Uint64Checker),
) {
	t.Helper()
	for i := 0; i < testRuns; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 400)))
			keys := generateRandomUint64Keys(rng, minN+rng.Intn(maxExtraN))
			f, err := build(keys, rng)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, f)
		})
	}
}

func runParallelUint64Clustered(
	t *testing.T,
	clusterRuns, minN, maxExtraN int,
	build BuildUint64Filter,
	testFn func(t *testing.T, rng *rand.Rand, keys []uint64, f Uint64Checker),
) {
	t.Helper()
	for i := 0; i < clusterRuns; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 700)))
			n := minN + rng.Intn(maxExtraN)
			keys, _ := testutils.GenerateClusterDistribution(n, 5, 0.15, rng)
			f, err := build(keys, rng)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, f)
		})
	}
}

func generateRandomUint64Keys(rng *rand.Rand, n int) []uint64 {
	keySet := make(map[uint64]bool)
	keys := make([]uint64, 0, n)
	for len(keys) < n {
		val := rng.Uint64()
		if !keySet[val] {
			keySet[val] = true
			keys = append(keys, val)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}
