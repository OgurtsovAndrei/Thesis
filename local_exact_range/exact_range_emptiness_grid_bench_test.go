package local_exact_range

import (
	"Thesis/bits"
	"Thesis/testutils"
	"fmt"
	"testing"
)

func BenchmarkExactRangeEmptinessGrid(b *testing.B) {
	counts := []int{1000000}
	bitLens := []int{64, 128, 256, 512}

	for _, bitLen := range bitLens {
		for _, count := range counts {
			name := fmt.Sprintf("KeySize=%d/Keys=%d", bitLen, count)
			keys := testutils.GetBenchKeys(bitLen, count)
			universe := bits.NewBitString(uint32(bitLen))

			b.Run(name+"/Build", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					ere, _ := NewExactRangeEmptiness(keys, universe)
					if i == 0 {
						size := ere.ByteSize()
						b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
					}
				}
			})

			ere, _ := NewExactRangeEmptiness(keys, universe)
			queryA := make([]bits.BitString, 100)
			queryB := make([]bits.BitString, 100)
			for i := 0; i < 100; i++ {
				idx1 := i * len(keys) / 100
				idx2 := (idx1 + 10) % len(keys)
				if idx1 > idx2 { idx1, idx2 = idx2, idx1 }
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
}
