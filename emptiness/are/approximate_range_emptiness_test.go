package are

import (
	"Thesis/bits"
	"Thesis/testutils"
	"fmt"
	"testing"
)

func BenchmarkApproximateRangeEmptiness_Grid(b *testing.B) {
	counts := []int{1 << 18, 1 << 20, 1 << 22, 1 << 24}
	bitLens := []int{64, 128, 256, 512, 1024}
	epsilons := []float64{0.001} // 0.1% FP rate

	for _, epsilon := range epsilons {
		for _, bitLen := range bitLens {
			for _, count := range counts {
				name := fmt.Sprintf("Eps=%v/KeySize=%d/Keys=%d", epsilon, bitLen, count)
				keys := testutils.GetBenchKeys(bitLen, count)

				b.Run(name+"/Build", func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						are, _ := NewApproximateRangeEmptiness(keys, epsilon)
						if i == 0 {
							size := are.ByteSize()
							b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
						}
					}
				})

				are, _ := NewApproximateRangeEmptiness(keys, epsilon)
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
						are.IsEmpty(queryA[idx], queryB[idx])
					}
				})
			}
		}
	}
}
