package ere_one_d

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

const (
	testRuns           = 1_000
	minN               = 100
	maxExtraN          = 5000
	fixedBitLen        = 64
	rangeQueriesPerRun = 10
)

// setupBenchData generates a sorted set of random uint64 keys and the ERE structure.
func setupBenchData(rng *rand.Rand, n, bl int) ([]uint64, *ExactRangeEmptiness, error) {
	keySet := make(map[uint64]bool)
	sortedKeys := make([]uint64, 0, n)
	for len(keySet) < n {
		val := rng.Uint64()
		if !keySet[val] {
			keySet[val] = true
			sortedKeys = append(sortedKeys, val)
		}
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i] < sortedKeys[j]
	})

	ere, err := NewExactRangeEmptiness(sortedKeys, uint32(bl))
	return sortedKeys, ere, err
}

func TestERE_Property_PointInclusion(t *testing.T) {
	t.Parallel()
	runParallelERE(t, func(t *testing.T, rng *rand.Rand, keys []uint64, ere *ExactRangeEmptiness) {
		for j := 0; j < 50; j++ {
			key := keys[rng.Intn(len(keys))]
			if ere.IsEmpty(key, key) {
				t.Errorf("Key %d not found", key)
			}
		}
	})
}

func TestERE_Property_TightOverhang(t *testing.T) {
	t.Parallel()
	runParallelERE(t, func(t *testing.T, rng *rand.Rand, keys []uint64, ere *ExactRangeEmptiness) {
		for j := 0; j < 50; j++ {
			key := keys[rng.Intn(len(keys))]
			if key > 0 {
				if ere.IsEmpty(key-1, key) {
					t.Errorf("Range [%d, %d] should contain key", key-1, key)
				}
			}
			if key < ^uint64(0) {
				if ere.IsEmpty(key, key+1) {
					t.Errorf("Range [%d, %d] should contain key", key, key+1)
				}
			}
		}
	})
}

func TestERE_Property_SpanningRanges(t *testing.T) {
	t.Parallel()
	runParallelERE(t, func(t *testing.T, rng *rand.Rand, keys []uint64, ere *ExactRangeEmptiness) {
		n := len(keys)
		for j := 0; j < 20; j++ {
			idx1 := rng.Intn(n - 10)
			idx2 := idx1 + 1 + rng.Intn(min(n-idx1-1, 100))
			a, b := keys[idx1], keys[idx2]
			if ere.IsEmpty(a, b) {
				t.Errorf("Spanning range [%d, %d] failed", a, b)
			}
		}
	})
}

func TestERE_Property_BlockBoundaries(t *testing.T) {
	t.Parallel()
	runParallelERE(t, func(t *testing.T, rng *rand.Rand, keys []uint64, ere *ExactRangeEmptiness) {
		found := 0
		for block := uint64(0); block < uint64(ere.numBlocks-1) && found < 5; block++ {
			startB, endB := ere.getBlockRange(block)
			startNext, endNext := ere.getBlockRange(block + 1)
			if startB < endB && startNext < endNext {
				a, b := keys[endB-1], keys[startNext]
				if ere.IsEmpty(a, b) {
					t.Errorf("Boundary range [%d, %d] failed", a, b)
				}
				found++
			}
		}
	})
}

func TestERE_Property_MassiveSpan(t *testing.T) {
	t.Parallel()
	runParallelERE(t, func(t *testing.T, rng *rand.Rand, keys []uint64, ere *ExactRangeEmptiness) {
		if ere.IsEmpty(keys[0], keys[len(keys)-1]) {
			t.Errorf("Massive span failed")
		}
	})
}

func TestERE_Property_HeavyBucket(t *testing.T) {
	t.Parallel()
	for i := 0; i < 100; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter/%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 400)))
			n := 500

			// Force same 10-bit prefix for all keys
			// This matches k = 10 for n=500 (floor(log2(500)) = 8, but we pick k=10 explicitly
			// to force a heavy bucket by using the actual k derived in the constructor).
			k := uint32(10)
			fixedPrefix := (rng.Uint64() & ((1 << k) - 1)) << (64 - k)

			keySet := make(map[uint64]bool)
			sortedKeys := make([]uint64, 0, n)
			for len(keySet) < n {
				// Random suffix (54 bits)
				suffix := rng.Uint64() & ((1 << (64 - k)) - 1)
				val := fixedPrefix | suffix
				if !keySet[val] {
					keySet[val] = true
					sortedKeys = append(sortedKeys, val)
				}
			}
			sort.Slice(sortedKeys, func(i, j int) bool {
				return sortedKeys[i] < sortedKeys[j]
			})

			ere, err := NewExactRangeEmptiness(sortedKeys, 64)
			if err != nil {
				t.Fatalf("Failed to build ERE: %v", err)
			}

			// Verify point queries
			for j := 0; j < 100; j++ {
				key := sortedKeys[rng.Intn(n)]
				if ere.IsEmpty(key, key) {
					t.Errorf("HeavyBucket: Key %d not found", key)
				}
			}
		})
	}
}

// TestERE_Property_LinearMatchesBinary verifies LinearIsEmpty and IsEmpty agree.
func TestERE_Property_LinearMatchesBinary(t *testing.T) {
	t.Parallel()
	runParallelERE(t, func(t *testing.T, rng *rand.Rand, keys []uint64, ere *ExactRangeEmptiness) {
		for j := 0; j < 100; j++ {
			lo := rng.Uint64()
			hi := lo + uint64(rng.Intn(10000))
			got := ere.LinearIsEmpty(lo, hi)
			want := ere.IsEmpty(lo, hi)
			if got != want {
				t.Fatalf("LinearIsEmpty(%d,%d)=%v != IsEmpty=%v", lo, hi, got, want)
			}
		}
	})
}

func runParallelERE(t *testing.T, testFn func(t *testing.T, rng *rand.Rand, keys []uint64, ere *ExactRangeEmptiness)) {
	for i := 0; i < testRuns; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter/%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 100)))
			keys, ere, err := setupBenchData(rng, minN+rng.Intn(maxExtraN), fixedBitLen)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, ere)
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
