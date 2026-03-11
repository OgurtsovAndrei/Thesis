package are_soda_hash

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

const (
	testRuns      = 100
	minN          = 100
	maxExtraN     = 1000
	targetEpsilon = 0.001
	maxQueryLen   = 1000
)

func setupSodaData(rng *rand.Rand, n int) ([]uint64, *ApproximateRangeEmptinessSoda, error) {
	keySet := make(map[uint64]bool)
	sortedKeys := make([]uint64, 0, n)
	for len(sortedKeys) < n {
		val := rng.Uint64()
		if !keySet[val] {
			keySet[val] = true
			sortedKeys = append(sortedKeys, val)
		}
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i] < sortedKeys[j]
	})

	filter, err := NewApproximateRangeEmptinessSoda(sortedKeys, maxQueryLen, targetEpsilon)
	return sortedKeys, filter, err
}

func TestSODA_Property_PointInclusion(t *testing.T) {
	t.Parallel()
	runParallelSoda(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *ApproximateRangeEmptinessSoda) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if filter.IsEmpty(key, key) {
				t.Errorf("Key %v not found", key)
			}
		}
	})
}

func TestSODA_Property_TightOverhang(t *testing.T) {
	t.Parallel()
	runParallelSoda(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *ApproximateRangeEmptinessSoda) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if key > 0 {
				if filter.IsEmpty(key-1, key) {
					t.Errorf("Range [%v, %v] failed", key-1, key)
				}
			}
			if key < ^uint64(0) {
				if filter.IsEmpty(key, key+1) {
					t.Errorf("Range [%v, %v] failed", key, key+1)
				}
			}
		}
	})
}

func TestSODA_Property_SpanningRanges(t *testing.T) {
	t.Parallel()
	runParallelSoda(t, func(t *testing.T, rng *rand.Rand, keys []uint64, filter *ApproximateRangeEmptinessSoda) {
		n := len(keys)
		for j := 0; j < 10; j++ {
			idx1 := rng.Intn(n - 5)
			idx2 := idx1 + 1 + rng.Intn(min(n-idx1-1, 50))
			a, b := keys[idx1], keys[idx2]
			if b - a > maxQueryLen {
				b = a + maxQueryLen // constrain query length
			}
			if b >= keys[idx1] && filter.IsEmpty(a, b) {
				t.Errorf("Spanning range [%v, %v] failed", a, b)
			}
		}
	})
}

func runParallelSoda(t *testing.T, testFn func(t *testing.T, rng *rand.Rand, keys []uint64, filter *ApproximateRangeEmptinessSoda)) {
	for i := 0; i < testRuns; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 400)))
			keys, filter, err := setupSodaData(rng, minN+rng.Intn(maxExtraN))
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, filter)
		})
	}
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
