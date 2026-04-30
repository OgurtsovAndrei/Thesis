package are_trunc

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
)

func BenchmarkApproximateRangeEmptiness_Grid(b *testing.B) {
	counts := []int{1 << 18, 1 << 20, 1 << 22, 1 << 24}
	epsilon := 0.001

	for _, count := range counts {
		name := fmt.Sprintf("Eps=%v/Keys=%d", epsilon, count)

		rng := rand.New(rand.NewSource(42))
		seen := make(map[uint64]bool, count)
		keys := make([]uint64, 0, count)
		for len(keys) < count {
			v := rng.Uint64()
			if !seen[v] {
				seen[v] = true
				keys = append(keys, v)
			}
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

		K := uint32(math.Ceil(math.Log2(2.0 * float64(count) / epsilon)))
		b.Run(name+"/Build", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				are, _ := NewTruncARE(keys, 64, Config{K: K})
				if i == 0 {
					size := are.ByteSize()
					b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
				}
			}
		})

		are, _ := NewTruncARE(keys, 64, Config{K: K})
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
				are.IsEmpty(queryA[idx], queryB[idx])
			}
		})
	}
}
