package lerloc

import (
	"Thesis/bits"
	"Thesis/locators/rloc"
	"Thesis/trie/zft"
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// Benchmark LocalExactRangeLocator construction
func BenchmarkLocalExactRangeLocatorBuild(b *testing.B) {
	rloc.InitBenchKeys()

	for _, bitLen := range rloc.BenchBitLengths {
		for _, count := range rloc.BenchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := rloc.BenchKeys[bitLen][count]

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					lerl, err := NewLocalExactRangeLocator(keys)
					if err != nil {
						b.Fatalf("Failed to build LocalExactRangeLocator: %v", err)
					}

					if lerl == nil {
						b.Fatal("Failed to build LocalExactRangeLocator")
					}

					// Report memory metrics
					size := lerl.ByteSize()
					widths := lerl.TypeWidths()
					b.ReportMetric(float64(widths.E), "E_bits")
					b.ReportMetric(float64(widths.S), "S_bits")
					b.ReportMetric(float64(widths.I), "I_bits")
					b.ReportMetric(float64(size), "total_bytes")
					b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
				}
			})
		}
	}
}

// Benchmark LocalExactRangeLocator query performance
func BenchmarkLocalExactRangeLocatorWeakPrefixSearch(b *testing.B) {
	rloc.InitBenchKeys()

	for _, bitLen := range rloc.BenchBitLengths {
		for _, count := range rloc.BenchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := rloc.BenchKeys[bitLen][count]
				lerl, err := NewLocalExactRangeLocator(keys)
				if err != nil {
					b.Fatalf("Failed to build LocalExactRangeLocator: %v", err)
				}

				if lerl == nil {
					b.Fatal("Failed to build LocalExactRangeLocator")
				}

				// Generate query prefixes from existing keys
				queryPrefixes := make([]bits.BitString, 0, len(keys))
				for _, key := range keys {
					if key.Size() > 2 {
						// Use prefix of varying lengths
						prefixLen := 1 + (int(key.Size()) / 3)
						prefix := key.Prefix(prefixLen)
						queryPrefixes = append(queryPrefixes, prefix)
					} else {
						queryPrefixes = append(queryPrefixes, key)
					}
				}

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					prefix := queryPrefixes[i%len(queryPrefixes)]
					_, _, err := lerl.WeakPrefixSearch(prefix)
					if err != nil {
						b.Fatalf("WeakPrefixSearch failed: %v", err)
					}
				}
			})
		}
	}
}

// Benchmark memory usage comparison
func BenchmarkMemoryComparison(b *testing.B) {
	rloc.InitBenchKeys()

	for _, bitLen := range rloc.BenchBitLengths {
		for _, count := range rloc.BenchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := rloc.BenchKeys[bitLen][count]

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					// Build both structures
					zt := zft.Build(keys)
					rl, err := rloc.NewRangeLocator(zt)
					if err != nil {
						b.Fatalf("Failed to build RangeLocator: %v", err)
					}
					lerl, err := NewLocalExactRangeLocator(keys)
					if err != nil {
						b.Fatalf("Failed to build LocalExactRangeLocator: %v", err)
					}

					if rl == nil || lerl == nil {
						b.Fatal("Failed to build structures")
					}

					// Calculate sizes
					rlSize := rl.ByteSize()
					lerlSize := lerl.ByteSize()

					// Calculate average string length
					totalLen := 0
					for _, key := range keys {
						totalLen += int(key.Size())
					}
					avgLen := float64(totalLen) / float64(len(keys))

					// Report metrics
					rlWidths := rl.TypeWidths()
					lerlWidths := lerl.TypeWidths()
					b.ReportMetric(float64(rlWidths.E), "rl_E_bits")
					b.ReportMetric(float64(rlWidths.S), "rl_S_bits")
					b.ReportMetric(float64(rlWidths.I), "rl_I_bits")
					b.ReportMetric(float64(lerlWidths.E), "lerl_E_bits")
					b.ReportMetric(float64(lerlWidths.S), "lerl_S_bits")
					b.ReportMetric(float64(lerlWidths.I), "lerl_I_bits")
					b.ReportMetric(float64(rlSize), "rl_total_bytes")
					b.ReportMetric(float64(lerlSize), "lerl_total_bytes")
					b.ReportMetric(float64(rlSize)*8/float64(count), "rl_bits_per_key")
					b.ReportMetric(float64(lerlSize)*8/float64(count), "lerl_bits_per_key")
					b.ReportMetric(avgLen, "avg_string_length")
					b.ReportMetric(float64(lerlSize)/float64(rlSize), "lerl_vs_rl_ratio")
				}
			})
		}
	}
}

// Benchmark empty prefix queries (should return all keys)
func BenchmarkEmptyPrefixQuery(b *testing.B) {
	rloc.InitBenchKeys()

	for _, bitLen := range rloc.BenchBitLengths {
		for _, count := range rloc.BenchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := rloc.BenchKeys[bitLen][count]
				lerl, err := NewLocalExactRangeLocator(keys)
				if err != nil {
					b.Fatalf("Failed to build LocalExactRangeLocator: %v", err)
				}

				if lerl == nil {
					b.Fatal("Failed to build LocalExactRangeLocator")
				}

				emptyPrefix := bits.NewFromText("")

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					start, end, err := lerl.WeakPrefixSearch(emptyPrefix)
					if err != nil {
						b.Fatalf("Empty prefix search failed: %v", err)
					}
					if start != 0 || end != len(keys) {
						b.Fatalf("Expected [0, %d), got [%d, %d)", len(keys), start, end)
					}
				}
			})
		}
	}
}

// Benchmark with different prefix lengths
func BenchmarkPrefixLengthVariation(b *testing.B) {
	rloc.InitBenchKeys()

	for _, bitLen := range rloc.BenchBitLengths {
		count := rloc.BenchKeyCounts[3] // Use medium-sized dataset
		keys := rloc.BenchKeys[bitLen][count]
		lerl, err := NewLocalExactRangeLocator(keys)
		if err != nil {
			b.Fatalf("Failed to build LocalExactRangeLocator: %v", err)
		}

		if lerl == nil {
			b.Fatal("Failed to build LocalExactRangeLocator")
		}

		// Test different prefix lengths
		prefixLengths := []int{1, 2, 4, 8, 10, 12, 16, 20, 24, 32, 48}

		for _, prefixLen := range prefixLengths {
			b.Run(fmt.Sprintf("KeySize=%d/PrefixLen=%d", bitLen, prefixLen), func(b *testing.B) {
				// Generate prefixes of specific length
				var queryPrefixes []bits.BitString
				for _, key := range keys {
					if int(key.Size()) > prefixLen {
						prefix := key.Prefix(prefixLen)
						queryPrefixes = append(queryPrefixes, prefix)
					}
				}

				if len(queryPrefixes) == 0 {
					b.Skip("No keys long enough for this prefix length")
				}

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					prefix := queryPrefixes[i%len(queryPrefixes)]
					_, _, err := lerl.WeakPrefixSearch(prefix)
					if err != nil {
						b.Fatalf("WeakPrefixSearch failed: %v", err)
					}
				}
			})
		}
	}
}

// Benchmark miss queries (prefixes not matching any key)
func BenchmarkMissQueries(b *testing.B) {
	rloc.InitBenchKeys()

	for _, bitLen := range rloc.BenchBitLengths {
		count := rloc.BenchKeyCounts[3] // Use medium-sized dataset
		keys := rloc.BenchKeys[bitLen][count]

		b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
			lerl, err := NewLocalExactRangeLocator(keys)
			if err != nil {
				b.Fatalf("Failed to build LocalExactRangeLocator: %v", err)
			}

			if lerl == nil {
				b.Fatal("Failed to build LocalExactRangeLocator")
			}

			// Generate prefixes that are unlikely to match
			r := rand.New(rand.NewSource(42))
			var missQueries []bits.BitString
			for i := 0; i < 100; i++ {
				// Generate random bit strings that are likely not prefixes
				queryBitLen := 8 + r.Intn(8)
				val := r.Uint64()
				if queryBitLen < 64 {
					val &= (1 << uint(queryBitLen)) - 1
				}
				// Flip some bits to make it even less likely to match
				val ^= (1 << uint(r.Intn(queryBitLen)))

				bs := bits.NewFromUint64(val)
				if int(bs.Size()) > queryBitLen {
					bs = bs.Prefix(queryBitLen)
				}
				missQueries = append(missQueries, bs)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				prefix := missQueries[i%len(missQueries)]
				start, end, err := lerl.WeakPrefixSearch(prefix)
				if err != nil {
					b.Fatalf("WeakPrefixSearch failed: %v", err)
				}
				// Miss queries should return [0,0) or small ranges
				_ = start
				_ = end
			}
		})
	}
}

// Benchmark scaling behavior
func BenchmarkScalingBehavior(b *testing.B) {
	rloc.InitBenchKeys()

	for _, bitLen := range rloc.BenchBitLengths {
		for _, count := range rloc.BenchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := rloc.BenchKeys[bitLen][count]

				lerl, err := NewLocalExactRangeLocator(keys)
				if err != nil {
					b.Fatalf("Failed to build LocalExactRangeLocator: %v", err)
				}

				if lerl == nil {
					b.Fatal("Failed to build LocalExactRangeLocator")
				}

				// Generate some queries
				var queryPrefixes []bits.BitString
				queryCount := int(math.Min(float64(len(keys)), 1000))
				for i := 0; i < queryCount; i++ {
					key := keys[i]
					if key.Size() > 2 {
						prefixLen := 1 + int(key.Size()/3)
						prefix := key.Prefix(prefixLen)
						queryPrefixes = append(queryPrefixes, prefix)
					}
				}

				size := lerl.ByteSize()
				b.ReportMetric(float64(size), "memory_bytes")
				b.ReportMetric(float64(size)/float64(count), "bytes_per_key")

				b.ReportAllocs()
				b.ResetTimer()

				// Use b.N for proper benchmarking
				for i := 0; i < b.N; i++ {
					prefix := queryPrefixes[i%len(queryPrefixes)]
					_, _, err := lerl.WeakPrefixSearch(prefix)
					if err != nil {
						b.Fatalf("Query failed: %v", err)
					}
				}
			})
		}
	}
}

// Benchmark detailed memory breakdown
func BenchmarkMemoryDetailed(b *testing.B) {
	rloc.InitBenchKeys()

	bitLen := 64 // Focus on 64-bit keys for the breakdown study
	for _, count := range rloc.BenchKeyCounts {
		b.Run(fmt.Sprintf("Keys=%d", count), func(b *testing.B) {
			keys := rloc.BenchKeys[bitLen][count]

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				lerl, err := NewLocalExactRangeLocator(keys)
				if err != nil {
					b.Fatalf("Failed to build LocalExactRangeLocator: %v", err)
				}

				// Log the detailed report for the analyzer
				if i == 0 {
					report := lerl.MemDetailed()
					b.Logf("JSON_MEM_REPORT: %s", report.JSON())
				}
			}
		})
	}
}
