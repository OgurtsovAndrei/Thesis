package local_exact_range

import (
	"Thesis/bits"
	"Thesis/testutils"
	"fmt"
	"testing"
)

var (
	BenchKeyCounts  = testutils.DefaultBenchKeyCounts
	BenchBitLengths = testutils.DefaultBenchBitLengths
)

func BenchmarkExactRangeEmptiness_Build(b *testing.B) {
	for _, bitLen := range BenchBitLengths {
		for _, count := range BenchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := testutils.GetBenchKeys(bitLen, count)

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					universe := bits.NewBitString(uint32(bitLen))
					ere, err := NewExactRangeEmptiness(keys, universe)
					if err != nil {
						b.Fatalf("Build failed: %v", err)
					}
					// Report memory metrics
					size := ere.ByteSize()
					b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
				}
			})
		}
	}
}

func BenchmarkExactRangeEmptiness_Query(b *testing.B) {
	for _, bitLen := range BenchBitLengths {
		for _, count := range BenchKeyCounts {
			b.Run(fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count), func(b *testing.B) {
				keys := testutils.GetBenchKeys(bitLen, count)
				universe := bits.NewBitString(uint32(bitLen))
				ere, _ := NewExactRangeEmptiness(keys, universe)

				// Generate some query ranges from keys
				queryA := make([]bits.BitString, 100)
				queryB := make([]bits.BitString, 100)
				for i := 0; i < 100; i++ {
					idx1 := i * len(keys) / 100
					idx2 := (i + 1) * len(keys) / 100
					if idx2 >= len(keys) {
						idx2 = len(keys) - 1
					}
					queryA[i] = keys[idx1]
					queryB[i] = keys[idx2]
				}

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					idx := i % 100
					ere.IsEmpty(queryA[idx], queryB[idx])
				}
			})
		}
	}
}
