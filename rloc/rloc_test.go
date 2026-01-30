package rloc

import (
	"Thesis/bits"
	"Thesis/zfasttrie"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"
)

const (
	testRuns  = 10000
	maxKeys   = 256
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
				// Save the failing case for manual debugging
				saveFailingCase(t, keys, seed, err)
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
	// Keys are in lexicographic (Compare) order, as per paper requirement
	// that ranges represent lexicographic ranks
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

type FailingCase struct {
	Seed      int64    `json:"seed"`
	Error     string   `json:"error"`
	NumKeys   int      `json:"num_keys"`
	KeysData  [][]byte `json:"keys_data"`  // Raw byte data
	KeysSizes []uint32 `json:"keys_sizes"` // Bit sizes
}

func saveFailingCase(t *testing.T, keys []bits.BitString, seed int64, err error) {
	failCase := FailingCase{
		Seed:      seed,
		Error:     err.Error(),
		NumKeys:   len(keys),
		KeysData:  make([][]byte, len(keys)),
		KeysSizes: make([]uint32, len(keys)),
	}

	for i, key := range keys {
		failCase.KeysData[i] = key.Data()
		failCase.KeysSizes[i] = key.Size()
	}

	filename := fmt.Sprintf("failing_case_seed_%d.json", seed)
	data, err := json.MarshalIndent(failCase, "", "  ")
	if err != nil {
		t.Logf("Failed to marshal failing case: %v", err)
		return
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		t.Logf("Failed to write failing case to %s: %v", filename, err)
		return
	}

	t.Logf("Saved failing case to %s", filename)
}
