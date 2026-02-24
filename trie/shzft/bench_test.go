package shzft

import (
	"Thesis/trie/hzft"
	"fmt"
	"testing"
)

var benchKeyCounts = []int{1024, 8192, 32768, 131072}

func BenchmarkMemory(b *testing.B) {
	for _, l := range []int{64, 256, 1024} {
		for _, n := range benchKeyCounts {
			b.Run(fmt.Sprintf("HZFT_L=%d_N=%d", l, n), func(b *testing.B) {
				keys := genRandomKeys(n, l, 42)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					trie := hzft.NewHZFastTrie[uint16](keys)
					b.ReportMetric(float64(trie.ByteSize())*8.0/float64(n), "bits/key")
				}
			})

			b.Run(fmt.Sprintf("SHZFT_L=%d_N=%d", l, n), func(b *testing.B) {
				keys := genRandomKeys(n, l, 42)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					trie := NewSuccinctHZFastTrie(keys)
					b.ReportMetric(float64(trie.ByteSize())*8.0/float64(n), "bits/key")
				}
			})
		}
	}
}

func BenchmarkQuery(b *testing.B) {
	for _, l := range []int{64, 1024} {
		for _, n := range []int{32768} {
			keys := genRandomKeys(n, l, 42)
			hzftTrie := hzft.NewHZFastTrie[uint16](keys)
			shzftTrie := NewSuccinctHZFastTrie(keys)

			b.Run(fmt.Sprintf("HZFT_Query_L=%d_N=%d", l, n), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = hzftTrie.GetExistingPrefix(keys[i%n])
				}
			})

			b.Run(fmt.Sprintf("SHZFT_Query_L=%d_N=%d", l, n), func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = shzftTrie.GetExistingPrefix(keys[i%n])
				}
			})
		}
	}
}
