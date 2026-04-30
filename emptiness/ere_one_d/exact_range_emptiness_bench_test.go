package ere_one_d

import (
	"fmt"
	"testing"
)

var BenchKeyCounts = []int{1 << 10, 1 << 13, 1 << 15, 1 << 18, 1 << 20}

func BenchmarkExactRangeEmptiness_Build(b *testing.B) {
	for _, count := range BenchKeyCounts {
		b.Run(fmt.Sprintf("KeySize=64/Keys=%d", count), func(b *testing.B) {
			keys := generateSortedUint64(count)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ere, err := NewExactRangeEmptiness(keys, 64)
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

func BenchmarkExactRangeEmptiness_Query(b *testing.B) {
	for _, count := range BenchKeyCounts {
		b.Run(fmt.Sprintf("KeySize=64/Keys=%d", count), func(b *testing.B) {
			keys := generateSortedUint64(count)
			ere, _ := NewExactRangeEmptiness(keys, 64)

			queryA := make([]uint64, 100)
			queryB := make([]uint64, 100)
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
