package are_hybrid_scan

import (
	"Thesis/bits"
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

func setupHybridScanData(rng *rand.Rand, n, bl int) ([]bits.BitString, *HybridScanARE, error) {
	keySet := make(map[string]bool)
	sortedKeys := make([]bits.BitString, 0, n)
	for len(sortedKeys) < n {
		bs := randomBitString(rng, bl)
		str := string(bs.Data())
		if !keySet[str] {
			keySet[str] = true
			sortedKeys = append(sortedKeys, bs)
		}
	}
	sort.Slice(sortedKeys, func(i, j int) bool {
		return sortedKeys[i].Compare(sortedKeys[j]) < 0
	})

	filter, err := NewHybridScanARE(sortedKeys, propRangeLen, propTargetEpsilon)
	return sortedKeys, filter, err
}

func TestHybridScanARE_Property_PointInclusion(t *testing.T) {
	t.Parallel()
	runParallelHybridScan(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *HybridScanARE) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if filter.IsEmpty(key, key) {
				t.Errorf("Key %v not found", key)
			}
		}
	})
}

func TestHybridScanARE_Property_TightOverhang(t *testing.T) {
	t.Parallel()
	runParallelHybridScan(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *HybridScanARE) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if !key.IsAllZeros() {
				prev := key.Predecessor()
				if filter.IsEmpty(prev, key) {
					t.Errorf("Range [%v, %v] failed", prev, key)
				}
			}
			if !key.IsAllOnes() {
				next := key.Successor()
				if filter.IsEmpty(key, next) {
					t.Errorf("Range [%v, %v] failed", key, next)
				}
			}
		}
	})
}

func TestHybridScanARE_Property_SpanningRanges(t *testing.T) {
	t.Parallel()
	runParallelHybridScan(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *HybridScanARE) {
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

func TestHybridScanARE_Property_MassiveSpan(t *testing.T) {
	t.Parallel()
	runParallelHybridScan(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *HybridScanARE) {
		if filter.IsEmpty(keys[0], keys[len(keys)-1]) {
			t.Errorf("Massive span failed")
		}
	})
}

func runParallelHybridScan(t *testing.T, testFn func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *HybridScanARE)) {
	bitLens := []int{64, 128, 256, 512}
	for i := 0; i < propTestRuns; i++ {
		i := i
		bl := bitLens[i%len(bitLens)]
		t.Run(fmt.Sprintf("BitLen%d/Iter%d", bl, i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 200)))
			keys, filter, err := setupHybridScanData(rng, propMinN+rng.Intn(propMaxExtraN), bl)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, filter)
		})
	}
}

func setupHybridScanDataClustered(rng *rand.Rand, n int) ([]bits.BitString, *HybridScanARE, error) {
	keys64, _ := testutils.GenerateClusterDistribution(n, 5, 0.15, rng)
	keysBS := make([]bits.BitString, len(keys64))
	for i, k := range keys64 {
		keysBS[i] = bits.NewFromTrieUint64(k, 64)
	}
	filter, err := NewHybridScanARE(keysBS, propRangeLen, propTargetEpsilon)
	return keysBS, filter, err
}

func runParallelHybridScanClustered(t *testing.T, testFn func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *HybridScanARE)) {
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
	runParallelHybridScanClustered(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *HybridScanARE) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if filter.IsEmpty(key, key) {
				t.Errorf("Key %v not found", key)
			}
		}
	})
}

func TestHybridScanARE_Property_SpanningRanges_Clustered(t *testing.T) {
	t.Parallel()
	runParallelHybridScanClustered(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *HybridScanARE) {
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

func TestHybridScanARE_Property_MassiveSpan_Clustered(t *testing.T) {
	t.Parallel()
	runParallelHybridScanClustered(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *HybridScanARE) {
		if filter.IsEmpty(keys[0], keys[len(keys)-1]) {
			t.Errorf("Massive span failed")
		}
	})
}

func randomBitString(rng *rand.Rand, bitLen int) bits.BitString {
	byteLen := (bitLen + 7) / 8
	data := make([]byte, byteLen)
	rng.Read(data)
	return bits.NewFromDataAndSize(data, uint32(bitLen))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
