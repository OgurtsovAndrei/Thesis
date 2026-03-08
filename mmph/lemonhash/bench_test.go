package lemonhash

import (
	"Thesis/bits"
	"Thesis/testutils"
	"fmt"
	"sort"
	"testing"
)

var (
	benchKeyCounts = []int{1 << 5, 1 << 10, 1 << 15, 1 << 20, 1 << 24}
)

func BenchmarkMMPHBuild(b *testing.B) {
	for _, count := range benchKeyCounts {
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keysStr := testutils.GetBenchKeysAsStrings(64, count)
			keys := make([]bits.BitString, len(keysStr))
			for i, s := range keysStr {
				keys[i] = bits.NewFromText(s)
			}
			// Sort keys by byte representation to match C++ std::string order
			sort.Slice(keys, func(i, j int) bool {
				return string(keys[i].Data()) < string(keys[j].Data())
			})
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				table := New(keys)
				b.ReportMetric(float64(table.ByteSize())*8/float64(count), "bits/key_in_mem")
				b.ReportMetric(float64(table.ByteSize()), "bytes_in_mem")
			}
		})
	}
}

func BenchmarkMMPHQuery(b *testing.B) {
	for _, count := range benchKeyCounts {
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keysStr := testutils.GetBenchKeysAsStrings(64, count)
			keys := make([]bits.BitString, len(keysStr))
			for i, s := range keysStr {
				keys[i] = bits.NewFromText(s)
			}
			sort.Slice(keys, func(i, j int) bool {
				return string(keys[i].Data()) < string(keys[j].Data())
			})
			table := New(keys)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = table.Rank(keys[i%count])
			}
		})
	}
}
