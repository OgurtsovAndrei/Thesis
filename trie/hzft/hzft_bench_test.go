package hzft

import (
	"Thesis/bits"
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

				sort.Slice(bsKeys, func(i, j int) bool {
					return bsKeys[i].Compare(bsKeys[j]) < 0
				})

				benchKeys[bitLen][count] = bsKeys
			}
		}
	})
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

// Benchmark HZFT construction (old/heavy - builds full ZFT first)
func BenchmarkHZFTBuild_Heavy(b *testing.B) {
	initBenchKeys()

	for _, bitLen := range benchBitLengths {
		for _, count := range benchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := benchKeys[bitLen][count]

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					// Old path: Build full ZFT first
					hzft, err := NewHZFastTrieFromIteratorHeavy[uint32](bits.NewSliceBitStringIterator(keys))
					if err != nil {
						b.Fatalf("Failed to build heavy HZFT: %v", err)
					}
					if hzft == nil {
						b.Fatal("Failed to build heavy HZFT")
					}
				}
			})
		}
	}
}

// Benchmark HZFT construction (new/streaming - processes keys on-the-fly)
func BenchmarkHZFTBuild_Streaming(b *testing.B) {
	initBenchKeys()

	for _, bitLen := range benchBitLengths {
		for _, count := range benchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := benchKeys[bitLen][count]

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					hzft, err := NewHZFastTrieFromIteratorStreaming[uint32](bits.NewSliceBitStringIterator(keys))
					if err != nil {
						b.Fatalf("Failed to build streaming HZFT: %v", err)
					}
					if hzft == nil {
						b.Fatal("Failed to build streaming HZFT")
					}
				}
			})
		}
	}
}
