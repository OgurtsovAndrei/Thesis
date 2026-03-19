package are_soda_hash

import (
	"Thesis/bits"
	"Thesis/emptiness/are_trunc"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func BenchmarkARE_Comparison(b *testing.B) {
	n := 100000
	epsilon := 0.001
	L := uint64(1000)

	rng := rand.New(rand.NewSource(42))
	keys := make([]uint64, n)
	bsKeys := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		val := rng.Uint64()
		keys[i] = val
		bsKeys[i] = bits.NewFromUint64(val)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	sort.Slice(bsKeys, func(i, j int) bool { return bsKeys[i].Compare(bsKeys[j]) < 0 })

	filterTrunc, _ := are_trunc.NewTruncARE(bsKeys, epsilon)
	filterSoda, _ := NewSodaARE(keys, L, epsilon)

	fmt.Printf("\n--- Space Analysis (N=%d, eps=%f, L=%d) ---\n", n, epsilon, L)
	fmt.Printf("ARE (Truncation): %.2f bits/key\n", float64(filterTrunc.SizeInBits())/float64(n))
	fmt.Printf("ARE (SODA Hash):  %.2f bits/key\n", float64(filterSoda.SizeInBits())/float64(n))
	fmt.Printf("--------------------------------------------\n")

	b.Run("Truncation_Uniform", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			k := bsKeys[i%n]
			filterTrunc.IsEmpty(k, k)
		}
	})

	b.Run("SodaHash_Uniform", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			k := keys[i%n]
			filterSoda.IsEmpty(k, k)
		}
	})
}
