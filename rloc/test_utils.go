package rloc

import (
	"Thesis/bits"
	"math/rand"
	"sort"
	"sync"
)

const benchmarkParallelism = 4

var (
	benchKeyCounts = []int{1 << 5, 1 << 8, 1 << 10, 1 << 13, 1 << 15, 1 << 18, 1 << 20}
	benchBitLength = 64 // Fixed bit length for all keys
	benchKeys      map[int][]bits.BitString
	benchOnce      sync.Once
)

func initBenchKeys() {
	benchOnce.Do(func() {
		benchKeys = make(map[int][]bits.BitString)
		for _, count := range benchKeyCounts {
			rawKeys := buildUniqueStrKeys(count)

			bsKeys := make([]bits.BitString, count)
			for i, k := range rawKeys {
				bsKeys[i] = bits.NewFromText(k)
			}

			sort.Sort(bitStringSorter(bsKeys))

			benchKeys[count] = bsKeys
		}
	})
}

func buildUniqueStrKeys(size int) []string {
	keys := make([]string, size)
	unique := make(map[string]bool, size)

	for i := 0; i < size; i++ {
		for {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			s := string(b)
			if !unique[s] {
				keys[i] = s
				unique[s] = true
				break
			}
		}
	}
	return keys
}

type bitStringSorter []bits.BitString

func (b bitStringSorter) Len() int           { return len(b) }
func (b bitStringSorter) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b bitStringSorter) Less(i, j int) bool { return b[i].Compare(b[j]) < 0 }
