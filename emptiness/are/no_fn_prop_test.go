package are

import (
	"Thesis/bits"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

const (
	testRuns      = 1_000
	minN          = 100
	maxExtraN     = 5000
	targetEpsilon = 0.001
)

func setupAREData(rng *rand.Rand, n, bl int) ([]bits.BitString, *ApproximateRangeEmptiness, error) {
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

	filter, err := NewApproximateRangeEmptiness(sortedKeys, targetEpsilon)
	return sortedKeys, filter, err
}

func TestARE_Property_PointInclusion(t *testing.T) {
	t.Parallel()
	runParallelARE(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *ApproximateRangeEmptiness) {
		for j := 0; j < 20; j++ {
			key := keys[rng.Intn(len(keys))]
			if filter.IsEmpty(key, key) {
				t.Errorf("Key %v not found", key)
			}
		}
	})
}

func TestARE_Property_TightOverhang(t *testing.T) {
	t.Parallel()
	runParallelARE(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *ApproximateRangeEmptiness) {
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

func TestARE_Property_SpanningRanges(t *testing.T) {
	t.Parallel()
	runParallelARE(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *ApproximateRangeEmptiness) {
		n := len(keys)
		for j := 0; j < 10; j++ {
			idx1 := rng.Intn(n - 5)
			idx2 := idx1 + 1 + rng.Intn(min(n-idx1-1, 50))
			a, b := keys[idx1], keys[idx2]
			if filter.IsEmpty(a, b) {
				t.Errorf("Spanning range [%v, %v] failed", a, b)
			}
		}
	})
}

func TestARE_Property_MassiveSpan(t *testing.T) {
	t.Parallel()
	runParallelARE(t, func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *ApproximateRangeEmptiness) {
		if filter.IsEmpty(keys[0], keys[len(keys)-1]) {
			t.Errorf("Massive span failed")
		}
	})
}

func runParallelARE(t *testing.T, testFn func(t *testing.T, rng *rand.Rand, keys []bits.BitString, filter *ApproximateRangeEmptiness)) {
	bitLens := []int{64, 128, 256, 512}
	for i := 0; i < testRuns; i++ {
		i := i
		bl := bitLens[i%len(bitLens)]
		t.Run(fmt.Sprintf("BitLen%d/Iter%d", bl, i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 200)))
			keys, filter, err := setupAREData(rng, minN+rng.Intn(maxExtraN), bl)
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			testFn(t, rng, keys, filter)
		})
	}
}

func randomBitString(rng *rand.Rand, bitLen int) bits.BitString {
	byteLen := (bitLen + 7) / 8
	data := make([]byte, byteLen)
	rng.Read(data)
	return bits.NewFromDataAndSize(data, uint32(bitLen))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
