package rloc

import (
	"Thesis/bits"
	"math/rand"
	"sort"
	"sync"
)

var (
	benchKeyCounts  = []int{1 << 5, 1 << 8, 1 << 10, 1 << 13, 1 << 15, 1 << 18}
	benchBitLengths = []int{64, 128, 256, 512, 1024}
	benchKeys       map[int]map[int][]bits.BitString // [bitLength][keyCount]
	benchOnce       sync.Once
)

func initBenchKeys() {
	benchOnce.Do(func() {
		benchKeys = make(map[int]map[int][]bits.BitString)
		for _, bitLen := range benchBitLengths {
			benchKeys[bitLen] = make(map[int][]bits.BitString)
			for _, count := range benchKeyCounts {
				rawKeys := buildUniqueStrKeys(count, bitLen)

				bsKeys := make([]bits.BitString, count)
				for i, k := range rawKeys {
					bsKeys[i] = bits.NewFromText(k)
				}

				sort.Sort(bitStringSorter(bsKeys))

				benchKeys[bitLen][count] = bsKeys
			}
		}
	})
}

func buildUniqueStrKeys(size int, bitLength int) []string {
	keys := make([]string, size)
	unique := make(map[string]bool, size)
	byteLength := (bitLength + 7) / 8 // Round up to nearest byte

	for i := 0; i < size; i++ {
		for {
			b := make([]byte, byteLength)
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
