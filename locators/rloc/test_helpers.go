package rloc

import (
	"Thesis/bits"
	"math"
	"math/rand"
	"sort"
	"sync"
)

var (
	BenchKeyCounts  = []int{1 << 5, 1 << 8, 1 << 10, 1 << 13, 1 << 15, 1 << 18}
	BenchBitLengths = []int{64, 128, 256, 512, 1024}
	BenchKeys       map[int]map[int][]bits.BitString // [bitLength][keyCount]
	benchOnce       sync.Once
)

func InitBenchKeys() {
	benchOnce.Do(func() {
		BenchKeys = make(map[int]map[int][]bits.BitString)
		for _, bitLen := range BenchBitLengths {
			BenchKeys[bitLen] = make(map[int][]bits.BitString)
			for _, count := range BenchKeyCounts {
				rawKeys := buildUniqueStrKeys(count, bitLen)

				bsKeys := make([]bits.BitString, count)
				for i, k := range rawKeys {
					bsKeys[i] = bits.NewFromText(k)
				}

				sort.Sort(BitStringSorter(bsKeys))

				BenchKeys[bitLen][count] = bsKeys
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

type BitStringSorter []bits.BitString

func (b BitStringSorter) Len() int           { return len(b) }
func (b BitStringSorter) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b BitStringSorter) Less(i, j int) bool { return b[i].Compare(b[j]) < 0 }

func GenUniqueBitStrings(seed int64, maxKeys int, maxBitLen int) []bits.BitString {
	r := rand.New(rand.NewSource(seed))

	numKeys := r.Intn(maxKeys) + 1
	minSize := int(math.Log2(float64(maxKeys))) + 1
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

func GenUniqueBitStringsDebug(seed int64, maxKeys int, maxBitLen int) []bits.BitString {
	// Re-using same logic for now
	return GenUniqueBitStrings(seed, maxKeys, maxBitLen)
}

func FindRange(keys []bits.BitString, prefix bits.BitString) (int, int) {
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
