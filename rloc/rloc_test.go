package rloc

import (
	"Thesis/bits"
	"Thesis/zfasttrie"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
	"time"
)

const (
	testRuns  = 1000
	maxKeys   = 1024
	maxBitLen = 16
)

func TestRangeLocator_Correctness(t *testing.T) {
	for run := 0; run < testRuns; run++ {
		t.Run(fmt.Sprintf("run=%d", run), func(t *testing.T) {
			t.Parallel()
			seed := time.Now().UnixNano()
			keys := genUniqueBitStrings(seed)

			zt := zfasttrie.Build(keys)
			rl, err := NewRangeLocator(zt)
			if err != nil {
				t.Fatalf("NewRangeLocator failed (seed: %d): %v", seed, err)
			}

			it := zfasttrie.NewIterator(zt)
			for it.Next() {
				node := it.Node()
				if node == nil {
					continue
				}

				start, end, err := rl.Query(node.Extent)
				if err != nil {
					t.Fatalf("Query failed for existing node (seed: %d): %v", seed, err)
				}

				expectedStart, expectedEnd := findRange(keys, node.Extent)

				if start != expectedStart || end != expectedEnd {
					t.Errorf("Mismatch for node %s (seed: %d). Got: [%d, %d), Exp: [%d, %d)",
						node.Extent.PrettyString(), seed, start, end, expectedStart, expectedEnd)
					t.FailNow()
				}
			}
		})
	}
}

func genUniqueBitStrings(seed int64) []bits.BitString {
	r := rand.New(rand.NewSource(seed))

	numKeys := r.Intn(maxKeys) + 1
	minSize := int(math.Log2(maxKeys)) + 1
	bitLen := minSize + r.Intn(maxBitLen-minSize)

	uniqueUints := make(map[uint64]bool)
	mask := uint64(0)
	if bitLen == 64 {
		mask = 0xFFFFFFFFFFFFFFFF
	} else {
		mask = (uint64(1) << uint(bitLen)) - 1
	}

	for len(uniqueUints) < numKeys {
		uniqueUints[r.Uint64()&mask] = true
	}

	keys := make([]bits.BitString, 0, len(uniqueUints))
	for val := range uniqueUints {
		bs := bits.NewFromUint64(val)
		if uint32(bitLen) < bs.Size() {
			bs = bs.Prefix(bitLen)
		}
		keys = append(keys, bs)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Compare(keys[j]) < 0
	})
	return keys
}

func findRange(keys []bits.BitString, prefix bits.BitString) (int, int) {
	// Use BitString.Compare() instead of string comparison for better performance
	start := sort.Search(len(keys), func(i int) bool {
		return keys[i].Compare(prefix) >= 0
	})

	end := start
	for end < len(keys) {
		if !keys[end].HasPrefix(prefix) {
			break
		}
		end++
	}
	return start, end
}
