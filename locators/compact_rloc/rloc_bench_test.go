package compact_rloc

import (
	"Thesis/locators/rloc"
	"Thesis/trie/zft"
	"fmt"
	"testing"
)

func BenchmarkCompactRangeLocator_MemoryAndBuild(b *testing.B) {
	for _, bitLen := range rloc.BenchBitLengths {
		for _, n := range rloc.BenchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, n), func(b *testing.B) {
				keys := rloc.GetBenchKeys(bitLen, n)
				zt := zft.Build(keys)

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					rl, err := NewCompactRangeLocator(zt)
					if err != nil {
						b.Fatalf("build failed: %v", err)
					}
					b.ReportMetric(float64(rl.ByteSize())*8/float64(n), "bits/key")
				}
			})
		}
	}
}

func BenchmarkCompactRangeLocator_Query(b *testing.B) {
	for _, bitLen := range rloc.BenchBitLengths {
		for _, n := range rloc.BenchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, n), func(b *testing.B) {
				keys := rloc.GetBenchKeys(bitLen, n)
				zt := zft.Build(keys)

				rl, err := NewCompactRangeLocator(zt)
				if err != nil {
					b.Fatalf("build failed: %v", err)
				}

				// Pre-collect nodes for querying
				var queries []zft.NodeInfo
				it := zft.NewIterator(zt)
				for it.Next() {
					if it.Node() != nil {
						queries = append(queries, *it.Node())
					}
				}

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					node := queries[i%len(queries)]
					_, _, _ = rl.Query(node.Extent)
				}
			})
		}
	}
}
