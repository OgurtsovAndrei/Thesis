package hash

import (
	"fmt"
	"testing"
)

// Benchmark matrix:
//
//   n ∈ {2^20, 2^22, 2^24, 2^26, 2^28}, K ∈ {20, 24, 28, 32, 36, 40}
//
// Inputs are pseudo-random uniform in [0, 2^K), matching the post-pairwise-
// hash distribution of the SODA build path.

var benchSizesLog2 = []int{20, 22, 24, 26, 28}
var benchKs = []uint32{20, 24, 28, 32, 36, 40}

type sortVariant struct {
	name string
	fn   func(keys []uint64, K uint32) []uint64
}

var sortVariants = []sortVariant{
	{"sortSlice", func(k []uint64, _ uint32) []uint64 { return SortAndDedupUint64(k) }},
	{"slicesSort", func(k []uint64, _ uint32) []uint64 { return SortAndDedupUint64Slices(k) }},
	{"radixFull", func(k []uint64, _ uint32) []uint64 { return SortAndDedupUint64Radix(k) }},
	{"radixK", func(k []uint64, K uint32) []uint64 { return SortAndDedupUint64RadixK(k, K) }},
	{"bitmap", func(k []uint64, K uint32) []uint64 { return SortAndDedupUint64Bitmap(k, K) }},
	{"americanFlag", func(k []uint64, K uint32) []uint64 { return SortAndDedupUint64AmericanFlag(k, K) }},
	{"msdBitmap", func(k []uint64, K uint32) []uint64 { return SortAndDedupUint64MSDBitmap(k, K) }},
	{"adaptive", func(k []uint64, K uint32) []uint64 { return SortAndDedupUint64Adaptive(k, K) }},
}

// runBench executes a destructive sort/dedup function with copy-per-iteration.
// b.StopTimer / b.StartTimer isolate the cost of the function under test.
func runBench(b *testing.B, master []uint64, K uint32, fn func([]uint64, uint32) []uint64) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		buf := make([]uint64, len(master))
		copy(buf, master)
		b.StartTimer()
		_ = fn(buf, K)
	}
}

func BenchmarkSortAndDedup(b *testing.B) {
	for _, lg := range benchSizesLog2 {
		n := 1 << lg
		for _, K := range benchKs {
			master := genUniformKeys(n, K, 0xC0FFEE)
			for _, v := range sortVariants {
				name := fmt.Sprintf("%s/n=2^%d/K=%d", v.name, lg, K)
				b.Run(name, func(b *testing.B) {
					runBench(b, master, K, v.fn)
				})
			}
		}
	}
}
