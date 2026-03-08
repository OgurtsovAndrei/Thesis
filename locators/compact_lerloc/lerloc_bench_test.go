package compact_lerloc

import (
	"Thesis/bits"
	"Thesis/locators/lerloc"
	"Thesis/locators/rloc"
	"Thesis/utils"
	"fmt"
	"testing"
)

func BenchmarkCompactLocalExactRangeLocator_Build(b *testing.B) {
	rloc.InitBenchKeys()

	for _, bitLen := range rloc.BenchBitLengths {
		for _, count := range rloc.BenchKeyCounts {
			if count > 65536 { continue } // Skip very large sets for now
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := rloc.GetBenchKeys(bitLen, count)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					lerl, err := NewAutoCompactLocalExactRangeLocator(keys)
					if err != nil {
						b.Fatalf("Failed to build locator: %v", err)
					}

					// Report memory metrics
					size := lerl.ByteSize()
					b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
				}
			})
		}
	}
}

func BenchmarkCompactLocalExactRangeLocator_Query(b *testing.B) {
	rloc.InitBenchKeys()

	for _, bitLen := range rloc.BenchBitLengths {
		for _, count := range rloc.BenchKeyCounts {
			if count > 65536 { continue }
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := rloc.GetBenchKeys(bitLen, count)
				lerl, _ := NewAutoCompactLocalExactRangeLocator(keys)

				queryPrefixes := generateQueryPrefixes(keys)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					prefix := queryPrefixes[i%len(queryPrefixes)]
					_, _, _ = lerl.WeakPrefixSearch(prefix)
				}
			})
		}
	}
}

func generateQueryPrefixes(keys []bits.BitString) []bits.BitString {
	queryPrefixes := make([]bits.BitString, 0, len(keys))
	for _, key := range keys {
		if key.Size() > 2 {
			prefixLen := 1 + (int(key.Size()) / 3)
			queryPrefixes = append(queryPrefixes, key.Prefix(prefixLen))
		} else {
			queryPrefixes = append(queryPrefixes, key)
		}
	}
	return queryPrefixes
}

func BenchmarkMemoryComparison(b *testing.B) {
	rloc.InitBenchKeys()

	bitLen := 64
	for _, count := range rloc.BenchKeyCounts {
		if count > 65536 { continue }
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keys := rloc.GetBenchKeys(bitLen, count)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				original, _ := lerloc.NewLocalExactRangeLocator(keys)
				compact, _ := NewAutoCompactLocalExactRangeLocator(keys)

				b.ReportMetric(float64(original.ByteSize())*8/float64(count), "orig_bits_key")
				b.ReportMetric(float64(compact.ByteSize())*8/float64(count), "lemon_bits_key")
				b.ReportMetric(float64(original.ByteSize())/float64(compact.ByteSize()), "improvement_ratio")
			}
		})
	}
}

func BenchmarkMemoryDetailed(b *testing.B) {
	rloc.InitBenchKeys()

	bitLen := 64
	for _, count := range rloc.BenchKeyCounts {
		if count > 65536 { continue }
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keys := rloc.GetBenchKeys(bitLen, count)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				lerl, _ := NewAutoCompactLocalExactRangeLocator(keys)
				if i == 0 {
					// Use a helper or just Log it
					report := lerl.(interface{ MemDetailed() utils.MemReport }).MemDetailed()
					b.Logf("JSON_MEM_REPORT: %s", report.JSON())
				}
			}
		})
	}
}
