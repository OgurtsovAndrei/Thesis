package ere

import (
	"fmt"
	"testing"
)

func BenchmarkExactRangeEmptinessGrid(b *testing.B) {
	counts := []int{1000000}

	for _, count := range counts {
		name := fmt.Sprintf("KeySize=64/Keys=%d", count)
		keys := getBenchKeysUint64(count)

		b.Run(name+"/Build", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ere, _ := NewExactRangeEmptiness(keys, 64)
				if i == 0 {
					size := ere.ByteSize()
					b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
				}
			}
		})

		ere, _ := NewExactRangeEmptiness(keys, 64)
		queryA := make([]uint64, 100)
		queryB := make([]uint64, 100)
		for i := 0; i < 100; i++ {
			idx1 := i * len(keys) / 100
			idx2 := (idx1 + 10) % len(keys)
			if idx1 > idx2 {
				idx1, idx2 = idx2, idx1
			}
			queryA[i] = keys[idx1]
			queryB[i] = keys[idx2]
		}

		b.Run(name+"/Query", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx := i % 100
				ere.IsEmpty(queryA[idx], queryB[idx])
			}
		})
	}
}
