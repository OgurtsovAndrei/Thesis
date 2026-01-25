package rloc

import (
	"Thesis/bits"
	"Thesis/zfasttrie"
	"fmt"
	"testing"
)

// Benchmark RangeLocator construction
func BenchmarkRangeLocatorBuild(b *testing.B) {
	initBenchKeys()

	for _, count := range benchKeyCounts {
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keys := benchKeys[count]

			b.SetParallelism(benchmarkParallelism)
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				zt := zfasttrie.Build(keys)
				rl := NewRangeLocator(zt)

				if rl == nil {
					b.Fatal("Failed to build RangeLocator")
				}

				// Report memory metrics
				size := rl.ByteSize()
				b.ReportMetric(float64(size), "total_bytes")
				b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
			}
		})
	}
}

// Benchmark RangeLocator query performance
func BenchmarkRangeLocatorQuery(b *testing.B) {
	initBenchKeys()

	for _, count := range benchKeyCounts {
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keys := benchKeys[count]
			zt := zfasttrie.Build(keys)
			rl := NewRangeLocator(zt)

			if rl == nil {
				b.Fatal("Failed to build RangeLocator")
			}

			// Collect node extents from the trie
			var nodeExtents []bits.BitString
			it := zfasttrie.NewIterator(zt)
			for it.Next() {
				node := it.Node()
				if node != nil {
					nodeExtents = append(nodeExtents, node.Extent)
				}
			}

			if len(nodeExtents) == 0 {
				b.Skip("No trie nodes found")
			}

			b.SetParallelism(benchmarkParallelism)
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				extent := nodeExtents[i%len(nodeExtents)]
				_, _, err := rl.Query(extent)
				if err != nil {
					b.Fatalf("Query failed: %v", err)
				}
			}
		})
	}
}
