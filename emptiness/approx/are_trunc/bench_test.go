package are_trunc

import (
	"math/rand"
	"sort"
	"testing"
)

func BenchmarkARE_PerformanceDegradation_Large(b *testing.B) {
	n := 1 << 20
	epsilon := 0.001

	rngU := rand.New(rand.NewSource(42))
	keysUniform := make([]uint64, n)
	for i := 0; i < n; i++ {
		keysUniform[i] = rngU.Uint64()
	}
	sort.Slice(keysUniform, func(i, j int) bool { return keysUniform[i] < keysUniform[j] })
	filterUniform, _ := NewTruncARE(keysUniform, 64, Config{Eps: epsilon})

	rngH := rand.New(rand.NewSource(42))
	kInternal := 21
	fixedPrefix := rngH.Uint64() << (64 - kInternal)
	keysHeavy := make([]uint64, n)
	for i := 0; i < n; i++ {
		suffix := rngH.Uint64() & ((1 << (64 - kInternal)) - 1)
		keysHeavy[i] = fixedPrefix | suffix
	}
	sort.Slice(keysHeavy, func(i, j int) bool { return keysHeavy[i] < keysHeavy[j] })
	filterHeavy, _ := NewTruncARE(keysHeavy, 64, Config{Eps: epsilon})

	b.Run("Uniform_N20", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
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
