package maps

import (
	"Thesis/bits"
	"fmt"
	"math/rand"
	"testing"
)

// generateRandomKeys is an optimized version for benchmarks
func generateRandomKeys(n, bitLen int) []bits.BitString {
	r := rand.New(rand.NewSource(42))
	keys := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		keys[i] = bits.NewFromBinary(randomBinaryString(r, bitLen))
	}
	return keys
}

func randomBinaryString(r *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		if r.Intn(2) == 0 {
			b[i] = '0'
		} else {
			b[i] = '1'
		}
	}
	return string(b)
}

func BenchmarkMaps(b *testing.B) {
	sizes := []int{1000, 10000, 100000}
	bitLens := []int{64, 256, 1024}

	for _, bitLen := range bitLens {
		for _, n := range sizes {
			keys := generateRandomKeys(n, bitLen)

			b.Run(fmt.Sprintf("BitMap_Put_N%d_L%d", n, bitLen), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					bm := NewBitMap[int]()
					for j, k := range keys {
						bm.Put(k, j)
					}
				}
			})

			b.Run(fmt.Sprintf("ArrayBitMap_Put_N%d_L%d", n, bitLen), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					bm := NewArrayBitMap[int]()
					for j, k := range keys {
						bm.Put(k, j)
					}
				}
			})

			bm := NewBitMap[int]()
			abm := NewArrayBitMap[int]()
			for j, k := range keys {
				bm.Put(k, j)
				abm.Put(k, j)
			}

			b.Run(fmt.Sprintf("BitMap_Get_N%d_L%d", n, bitLen), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = bm.Get(keys[i%n])
				}
			})

			b.Run(fmt.Sprintf("ArrayBitMap_Get_N%d_L%d", n, bitLen), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, _ = abm.Get(keys[i%n])
				}
			})
		}
	}
}
