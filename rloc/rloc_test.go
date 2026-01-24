package rloc

import (
	"Thesis/bits"
	"Thesis/zfasttrie"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"
)

const (
	testRuns  = 100
	maxKeys   = 200
	maxBitLen = 63
)

func TestRangeLocator_Correctness(t *testing.T) {
	for run := 0; run < testRuns; run++ {
		fmt.Println("Iteration", run)
		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))

		numKeys := r.Intn(maxKeys) + 1
		bitLen := r.Intn(maxBitLen) + 1

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
				bs = bits.NewBitStringPrefix(bs, uint32(bitLen))
			}
			keys = append(keys, bs)
		}

		sort.Slice(keys, func(i, j int) bool {
			return keys[i].Compare(keys[j]) < 0
		})

		zt := zfasttrie.Build(keys)
		rl := NewRangeLocator(zt)

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
					toBinary(node.Extent), seed, start, end, expectedStart, expectedEnd)
				t.FailNow()
			}
		}
	}
}

func findRange(keys []bits.BitString, prefix bits.BitString) (int, int) {
	pStr := toBinary(prefix)
	start := sort.Search(len(keys), func(i int) bool {
		return toBinary(keys[i]) >= pStr
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
