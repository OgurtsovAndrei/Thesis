package bits

import (
	"fmt"
	"math/rand"
	"testing"
)

// --- BENCHMARKS ---

func generateRandomBitString(maxSize int, r *rand.Rand) BitString {
	size := r.Intn(maxSize-1) + 1
	numWords := (size + 63) / 64
	data := make([]uint64, numWords)
	for i := 0; i < numWords; i++ {
		data[i] = r.Uint64()
	}
	// Mask the last word to ensure it's "clean" initially if needed, 
	// or leave it as is to simulate realistic creation.
	if size%64 != 0 {
		mask := (uint64(1) << (uint32(size) % 64)) - 1
		data[numWords-1] &= mask
	}
	return BitString{
		data:     data,
		sizeBits: uint32(size),
	}
}

func generatePool(count int, maxSize int) []BitString {
	r := rand.New(rand.NewSource(42))
	pool := make([]BitString, count)
	for i := 0; i < count; i++ {
		pool[i] = generateRandomBitString(maxSize, r)
	}
	return pool
}

var benchmarkSizes = []int{64, 256, 1024, 4096}

func BenchmarkPrefix(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool := generatePool(100, size)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bs := pool[i%100]
				targetSize := (i % int(bs.sizeBits)) + 1
				_ = bs.Prefix(targetSize)
			}
		})
	}
}

func BenchmarkHash(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			// Generate strings and then take prefix to ensure they have "junk bits"
			// for the 'After' version to handle.
			pool := generatePool(100, size+64)
			for i := range pool {
				targetSize := (rand.Intn(size-1) + 1)
				pool[i] = pool[i].Prefix(targetSize)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool[i%100].Hash()
			}
		})
	}
}

func BenchmarkEqual(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool1 := generatePool(100, size+64)
			pool2 := generatePool(100, size+64)
			// Equalize lengths and take prefixes
			for i := range pool1 {
				targetSize := (rand.Intn(size-1) + 1)
				pool1[i] = pool1[i].Prefix(targetSize)
				pool2[i] = pool2[i].Prefix(targetSize)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool1[i%100].Equal(pool2[i%100])
			}
		})
	}
}

func BenchmarkCompare(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool1 := generatePool(100, size+64)
			pool2 := generatePool(100, size+64)
			for i := range pool1 {
				len1 := rand.Intn(size-1) + 1
				len2 := rand.Intn(size-1) + 1
				pool1[i] = pool1[i].Prefix(len1)
				pool2[i] = pool2[i].Prefix(len2)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool1[i%100].Compare(pool2[i%100])
			}
		})
	}
}

func BenchmarkTrieCompare(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool1 := generatePool(100, size+64)
			pool2 := generatePool(100, size+64)
			for i := range pool1 {
				len1 := rand.Intn(size-1) + 1
				len2 := rand.Intn(size-1) + 1
				pool1[i] = pool1[i].Prefix(len1)
				pool2[i] = pool2[i].Prefix(len2)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool1[i%100].TrieCompare(pool2[i%100])
			}
		})
	}
}

func BenchmarkHasPrefix(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool1 := generatePool(100, size+64)
			pool2 := generatePool(100, size)
			for i := range pool1 {
				pSize := int(pool2[i].sizeBits)
				if pSize > int(pool1[i].sizeBits) {
					pSize = int(pool1[i].sizeBits)
				}
				pool2[i] = pool1[i].Prefix(pSize)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool1[i%100].HasPrefix(pool2[i%100])
			}
		})
	}
}

func BenchmarkTrimTrailingZeros(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool := generatePool(100, size+64)
			for i := range pool {
				pool[i] = pool[i].Prefix(size)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool[i%100].TrimTrailingZeros()
			}
		})
	}
}

func BenchmarkAppendBit(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool := generatePool(100, size+64)
			for i := range pool {
				pool[i] = pool[i].Prefix(size)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool[i%100].AppendBit(i%2 == 0)
			}
		})
	}
}

func BenchmarkIsAllOnes(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool := make([]BitString, 100)
			for i := 0; i < 100; i++ {
				s := rand.Intn(size-1) + 1
				bs := NewBitString(uint32(s))
				for j := range bs.data {
					bs.data[j] = ^uint64(0)
				}
				pool[i] = bs.Prefix(s)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool[i%100].IsAllOnes()
			}
		})
	}
}

func BenchmarkSuccessor(b *testing.B) {
	for _, size := range benchmarkSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			pool := generatePool(100, size+64)
			for i := range pool {
				pool[i] = pool[i].Prefix(size)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = pool[i%100].Successor()
			}
		})
	}
}
