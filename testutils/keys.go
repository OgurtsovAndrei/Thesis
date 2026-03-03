package testutils

import (
	"Thesis/bits"
	"math/rand"
	"sort"
	"sync"
)

var (
	DefaultBenchKeyCounts  = []int{1 << 5, 1 << 8, 1 << 10, 1 << 13, 1 << 15, 1 << 18}
	DefaultBenchBitLengths = []int{64, 256, 1024, 4096}

	benchKeys map[int]map[int][]bits.BitString // [bitLength][keyCount]
	benchMu   sync.Mutex
	benchOnce sync.Once
)

func initBenchKeys() {
	benchOnce.Do(func() {
		benchKeys = make(map[int]map[int][]bits.BitString)
	})
}

// GetBenchKeys возвращает набор ключей заданной длины и количества.
// Генерирует их лениво и кэширует.
func GetBenchKeys(bitLen int, count int) []bits.BitString {
	initBenchKeys()

	benchMu.Lock()
	defer benchMu.Unlock()

	if benchKeys[bitLen] == nil {
		benchKeys[bitLen] = make(map[int][]bits.BitString)
	}

	if keys, ok := benchKeys[bitLen][count]; ok {
		return keys
	}

	rawKeys := buildUniqueStrKeys(count, bitLen)
	bsKeys := make([]bits.BitString, count)
	for i, k := range rawKeys {
		bsKeys[i] = bits.NewFromText(k)
	}

	sort.Sort(bitStringSorter(bsKeys))
	benchKeys[bitLen][count] = bsKeys
	return bsKeys
}

// GetBenchKeysAsStrings возвращает набор ключей в виде строк.
func GetBenchKeysAsStrings(bitLen int, count int) []string {
	keys := GetBenchKeys(bitLen, count)
	strs := make([]string, len(keys))
	for i, k := range keys {
		strs[i] = string(k.Data())
	}
	return strs
}

func buildUniqueStrKeys(size int, bitLength int) []string {
	keys := make([]string, size)
	unique := make(map[string]bool, size)
	byteLength := (bitLength + 7) / 8

	r := rand.New(rand.NewSource(42))
	for i := 0; i < size; i++ {
		for {
			b := make([]byte, byteLength)
			r.Read(b)
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
