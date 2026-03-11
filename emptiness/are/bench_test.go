package are

import (
	"Thesis/bits"
	"math/rand"
	"sort"
	"testing"
)

func BenchmarkARE_PerformanceDegradation_Large(b *testing.B) {
	n := 1 << 20 // 1,048,576 keys
	epsilon := 0.001

	// Pre-generate keys for Uniform
	rngU := rand.New(rand.NewSource(42))
	keysUniform := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		keysUniform[i] = bits.NewFromUint64(rngU.Uint64())
	}
	sort.Slice(keysUniform, func(i, j int) bool { return keysUniform[i].Compare(keysUniform[j]) < 0 })
	filterUniform, _ := NewApproximateRangeEmptiness(keysUniform, epsilon)

	// Pre-generate keys for Heavy Bucket
	rngH := rand.New(rand.NewSource(42))
	kInternal := 21 // ceil(log2(2*n)) for 2^20 is 21
	fixedPrefix := rngH.Uint64() << (64 - kInternal)
	keysHeavy := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		suffix := rngH.Uint64() & ((1 << (64 - kInternal)) - 1)
		keysHeavy[i] = bits.NewFromUint64(fixedPrefix | suffix)
	}
	sort.Slice(keysHeavy, func(i, j int) bool { return keysHeavy[i].Compare(keysHeavy[j]) < 0 })
	filterHeavy, _ := NewApproximateRangeEmptiness(keysHeavy, epsilon)

	b.Run("Uniform_N20", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Query a random key from the set to minimize impact of sequential access
			idx := i % n
			k := keysUniform[idx]
			filterUniform.IsEmpty(k, k)
		}
	})

	b.Run("HeavyBucket_N20", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			idx := i % n
			k := keysHeavy[idx]
			filterHeavy.IsEmpty(k, k)
		}
	})
}
