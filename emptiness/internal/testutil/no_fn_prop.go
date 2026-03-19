package testutil

import (
	"Thesis/bits"
	"Thesis/testutils"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

// BitStringChecker is implemented by any range emptiness filter that operates on BitString keys.
type BitStringChecker interface {
	IsEmpty(a, b bits.BitString) bool
}

// BuildBitStringFilter constructs a filter from keys and returns it as a BitStringChecker.
type BuildBitStringFilter func(keys []bits.BitString, rng *rand.Rand) (BitStringChecker, error)

// RandomBitString generates a random BitString of the given bit length.
func RandomBitString(rng *rand.Rand, bitLen int) bits.BitString {
	byteLen := (bitLen + 7) / 8
	data := make([]byte, byteLen)
	rng.Read(data)
	return bits.NewFromDataAndSize(data, uint32(bitLen))
}

// RunBitStringNoFNProps runs the standard no-false-negative property tests for a BitString-keyed filter.
// testRuns controls how many random iterations run; minN and maxExtraN control key set sizes.
func RunBitStringNoFNProps(t *testing.T, testRuns, minN, maxExtraN int, build BuildBitStringFilter) {
	t.Helper()
	bitLens := []int{64, 128, 256, 512}

	t.Run("PointInclusion", func(t *testing.T) {
		t.Parallel()
		runParallel(t, testRuns, minN, maxExtraN, bitLens, build, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, f BitStringChecker) {
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
		runParallel(t, testRuns, minN, maxExtraN, bitLens, build, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, f BitStringChecker) {
			for j := 0; j < 20; j++ {
				key := keys[rng.Intn(len(keys))]
				if !key.IsAllZeros() {
					prev := key.Predecessor()
					if f.IsEmpty(prev, key) {
						t.Errorf("Range [%v, %v] failed", prev, key)
					}
				}
				if !key.IsAllOnes() {
					next := key.Successor()
					if f.IsEmpty(key, next) {
						t.Errorf("Range [%v, %v] failed", key, next)
					}
				}
			}
		})
	})

	t.Run("SpanningRanges", func(t *testing.T) {
		t.Parallel()
		runParallel(t, testRuns, minN, maxExtraN, bitLens, build, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, f BitStringChecker) {
			n := len(keys)
			for j := 0; j < 10; j++ {
				idx1 := rng.Intn(n - 5)
				idx2 := idx1 + 1 + rng.Intn(minInt(n-idx1-1, 50))
				a, b := keys[idx1], keys[idx2]
				if f.IsEmpty(a, b) {
					t.Errorf("Spanning range [%v, %v] failed", a, b)
				}
			}
		})
	})

	t.Run("MassiveSpan", func(t *testing.T) {
		t.Parallel()
		runParallel(t, testRuns, minN, maxExtraN, bitLens, build, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, f BitStringChecker) {
			if f.IsEmpty(keys[0], keys[len(keys)-1]) {
				t.Errorf("Massive span failed")
			}
		})
	})
}

// RunBitStringNoFNPropsClustered runs the clustered variant of no-FN property tests.
func RunBitStringNoFNPropsClustered(t *testing.T, clusterRuns, minN, maxExtraN int, build BuildBitStringFilter) {
	t.Helper()

	t.Run("PointInclusion_Clustered", func(t *testing.T) {
		t.Parallel()
		runParallelClustered(t, clusterRuns, minN, maxExtraN, build, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, f BitStringChecker) {
			for j := 0; j < 20; j++ {
				key := keys[rng.Intn(len(keys))]
				if f.IsEmpty(key, key) {
					t.Errorf("Key %v not found", key)
				}
			}
		})
	})

	t.Run("SpanningRanges_Clustered", func(t *testing.T) {
		t.Parallel()
		runParallelClustered(t, clusterRuns, minN, maxExtraN, build, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, f BitStringChecker) {
			n := len(keys)
			for j := 0; j < 10; j++ {
				idx1 := rng.Intn(n - 5)
				idx2 := idx1 + 1 + rng.Intn(minInt(n-idx1-1, 50))
				a, b := keys[idx1], keys[idx2]
				if f.IsEmpty(a, b) {
					t.Errorf("Spanning range [%v, %v] failed", a, b)
				}
			}
		})
	})

	t.Run("MassiveSpan_Clustered", func(t *testing.T) {
		t.Parallel()
		runParallelClustered(t, clusterRuns, minN, maxExtraN, build, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, f BitStringChecker) {
			if f.IsEmpty(keys[0], keys[len(keys)-1]) {
				t.Errorf("Massive span failed")
			}
		})
	})
}

func runParallel(
	t *testing.T,
	testRuns, minN, maxExtraN int,
	bitLens []int,
	build BuildBitStringFilter,
	testFn func(t *testing.T, rng *rand.Rand, keys []bits.BitString, f BitStringChecker),
) {
	t.Helper()
	for i := 0; i < testRuns; i++ {
		i := i
		bl := bitLens[i%len(bitLens)]
		t.Run(fmt.Sprintf("BitLen%d/Iter%d", bl, i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 200)))
			keys := generateRandomBitStrings(rng, minN+rng.Intn(maxExtraN), bl)
			f, err := build(keys, rng)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, f)
		})
	}
}

func runParallelClustered(
	t *testing.T,
	clusterRuns, minN, maxExtraN int,
	build BuildBitStringFilter,
	testFn func(t *testing.T, rng *rand.Rand, keys []bits.BitString, f BitStringChecker),
) {
	t.Helper()
	for i := 0; i < clusterRuns; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 9000)))
			n := minN + rng.Intn(maxExtraN)
			keys := generateClusteredBitStrings(rng, n)
			f, err := build(keys, rng)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, f)
		})
	}
}

func generateRandomBitStrings(rng *rand.Rand, n, bitLen int) []bits.BitString {
	keySet := make(map[string]bool)
	keys := make([]bits.BitString, 0, n)
	for len(keys) < n {
		bs := RandomBitString(rng, bitLen)
		str := string(bs.Data())
		if !keySet[str] {
			keySet[str] = true
			keys = append(keys, bs)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Compare(keys[j]) < 0
	})
	return keys
}

func generateClusteredBitStrings(rng *rand.Rand, n int) []bits.BitString {
	keys64, _ := testutils.GenerateClusterDistribution(n, 5, 0.15, rng)
	keys := make([]bits.BitString, len(keys64))
	for i, k := range keys64 {
		keys[i] = bits.NewFromTrieUint64(k, 64)
	}
	return keys
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
