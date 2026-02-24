package trie

import (
	"Thesis/bits"
	"Thesis/locators/lerloc"
	"Thesis/locators/rloc"
	"Thesis/trie/azft"
	"Thesis/trie/zft"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"testing"
)

var (
	benchKeyCounts  = []int{1 << 10, 1 << 13, 1 << 15}
	benchBitLengths = []int{64, 128, 256}
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
	byteLength := (bitLength + 7) / 8

	r := rand.New(rand.NewSource(42)) // Fixed seed for reproducibility
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

// Benchmark ZFT construction
func BenchmarkZFTBuild(b *testing.B) {
	initBenchKeys()

	for _, bitLen := range benchBitLengths {
		for _, count := range benchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := benchKeys[bitLen][count]

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					zt := zft.Build(keys)
					if zt == nil {
						b.Fatal("Failed to build ZFastTrie")
					}
				}
			})
		}
	}
}

// Benchmark AZFT construction
func BenchmarkAZFTBuild(b *testing.B) {
	initBenchKeys()

	for _, bitLen := range benchBitLengths {
		for _, count := range benchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := benchKeys[bitLen][count]

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					// Build AZFT (now all use streaming internally)
					azft, err := azft.NewApproxZFastTrie[uint16, uint32, uint32](keys)
					if err != nil {
						b.Fatalf("Failed to build AZFT: %v", err)
					}
					if azft == nil {
						b.Fatal("Failed to build AZFT")
					}
				}
			})
		}
	}
}


// Benchmark RangeLocator construction
func BenchmarkRangeLocatorBuild(b *testing.B) {
	initBenchKeys()

	for _, bitLen := range benchBitLengths {
		for _, count := range benchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := benchKeys[bitLen][count]

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					zt := zft.Build(keys)
					rl, err := rloc.NewRangeLocator(zt)
					if err != nil {
						b.Fatalf("NewRangeLocator failed: %v", err)
					}
					if rl == nil {
						b.Fatal("Failed to build RangeLocator")
					}
				}
			})
		}
	}
}

// Benchmark LocalExactRangeLocator construction
func BenchmarkLocalExactRangeLocatorBuild(b *testing.B) {
	initBenchKeys()

	for _, bitLen := range benchBitLengths {
		for _, count := range benchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := benchKeys[bitLen][count]

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					lerl, err := lerloc.NewLocalExactRangeLocator(keys)
					if err != nil {
						b.Fatalf("Failed to build LocalExactRangeLocator: %v", err)
					}
					if lerl == nil {
						b.Fatal("Failed to build LocalExactRangeLocator")
					}
				}
			})
		}
	}
}
