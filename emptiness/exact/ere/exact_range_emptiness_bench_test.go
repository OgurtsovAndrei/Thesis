package ere

import (
	"Thesis/testutils"
	"fmt"
	"testing"
)

var BenchKeyCounts = testutils.DefaultBenchKeyCounts

func getBenchKeysUint64(count int) []uint64 {
	bs := testutils.GetBenchKeys(64, count)
	keys := make([]uint64, len(bs))
	for i, k := range bs {
		keys[i] = k.TrieUint64()
	}
	return keys
}

func BenchmarkExactRangeEmptiness_Build(b *testing.B) {
	for _, count := range BenchKeyCounts {
		b.Run(fmt.Sprintf("KeySize=64/Keys=%d", count), func(b *testing.B) {
			keys := getBenchKeysUint64(count)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ere, err := NewExactRangeEmptiness(keys, 64)
				if err != nil {
					b.Fatalf("Build failed: %v", err)
				}
				size := ere.ByteSize()
				b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
			}
		})
	}
}

func BenchmarkExactRangeEmptiness_Query(b *testing.B) {
	for _, count := range BenchKeyCounts {
		b.Run(fmt.Sprintf("KeySize=64/Keys=%d", count), func(b *testing.B) {
			keys := getBenchKeysUint64(count)
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
