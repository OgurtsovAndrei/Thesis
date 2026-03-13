package are_pgm

import (
	"math/rand"
	"sort"
	"testing"
)

func TestPGMARE_Basic(t *testing.T) {
	keys := []uint64{100, 200, 300, 400, 500}
	filter, err := NewPGMApproximateRangeEmptiness(keys, 50, 0.01, 4)
	if err != nil {
		t.Fatalf("NewPGMApproximateRangeEmptiness: %v", err)
	}

	for _, k := range keys {
		if filter.IsEmpty(k, k) {
			t.Errorf("IsEmpty(%d, %d) = true, want false (key exists)", k, k)
		}
	}

	if filter.IsEmpty(100, 500) {
		t.Error("IsEmpty(100, 500) = true, want false")
	}

	t.Logf("K=%d, ERE bits: %d, CDF bits: %d, Total bits: %d, BPK: %.2f",
		filter.K, filter.SizeInBits(), filter.CDFSizeInBits(), filter.TotalSizeInBits(),
		float64(filter.TotalSizeInBits())/float64(len(keys)))
}

func TestPGMARE_NoFalseNegatives(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	n := 5000
	keys := make([]uint64, n)
	for i := range keys {
		keys[i] = rng.Uint64() >> 16
	}

	rangeLen := uint64(1 << 20)
	filter, err := NewPGMApproximateRangeEmptiness(keys, rangeLen, 0.01, 64)
	if err != nil {
		t.Fatalf("NewPGMApproximateRangeEmptiness: %v", err)
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	fnCount := 0
	for _, k := range keys {
		if filter.IsEmpty(k, k) {
			fnCount++
		}
	}
	if fnCount > 0 {
		t.Errorf("false negatives: %d / %d", fnCount, n)
	}

	for i := 0; i < n-1; i += 100 {
		if filter.IsEmpty(keys[i], keys[i+1]) {
			t.Errorf("IsEmpty(%d, %d) = true, but keys[%d] and keys[%d] exist",
				keys[i], keys[i+1], i, i+1)
		}
	}
}

func safeGaussUint64(center uint64, stddev float64, rng *rand.Rand) uint64 {
	off := rng.NormFloat64() * stddev
	if off >= 0 {
		v := center + uint64(off)
		if v < center { // overflow
			return ^uint64(0)
		}
		return v
	}
	abs := uint64(-off)
	if abs > center {
		return 0
	}
	return center - abs
}

func measureFPR(t *testing.T, filter *PGMApproximateRangeEmptiness, sortedKeys []uint64,
	queryGen func(rng *rand.Rand) (uint64, uint64), seed int64, numQueries int) (fpr float64, trueEmptyCount int) {
	rng := rand.New(rand.NewSource(seed))
	n := len(sortedKeys)
	fp := 0
	trueEmpty := 0

	for q := 0; q < numQueries; q++ {
		a, b := queryGen(rng)
		if b < a {
			continue
		}

		idx := sort.Search(n, func(i int) bool { return sortedKeys[i] >= a })
		reallyEmpty := idx >= n || sortedKeys[idx] > b

		if reallyEmpty {
			trueEmpty++
			if !filter.IsEmpty(a, b) {
				fp++
			}
		}
	}

	if trueEmpty == 0 {
		return 0, 0
	}
	return float64(fp) / float64(trueEmpty), trueEmpty
}

func TestPGMARE_FPR_Uniform(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	n := 10000

	keys := make([]uint64, n)
	for i := range keys {
		keys[i] = rng.Uint64() >> 16
	}

	rangeLen := uint64(1 << 20)
	for _, pgmEps := range []int{8, 32, 64, 128} {
		epsilon := 0.05
		filter, err := NewPGMApproximateRangeEmptiness(keys, rangeLen, epsilon, pgmEps)
		if err != nil {
			t.Fatalf("pgmEps=%d: %v", pgmEps, err)
		}

		sorted := make([]uint64, n)
		copy(sorted, keys)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

		fpr, trueEmpty := measureFPR(t, filter, sorted, func(rng *rand.Rand) (uint64, uint64) {
			a := rng.Uint64() >> 16
			b := a + rangeLen
			if b < a {
				b = ^uint64(0) >> 16
			}
			return a, b
		}, 999, 100000)

		bpk := float64(filter.TotalSizeInBits()) / float64(n)

		t.Logf("Uniform pgmEps=%d: K=%d, FPR=%.4f (target %.4f), BPK=%.2f (ERE=%.2f + CDF=%.2f), trueEmpty=%d",
			pgmEps, filter.K, fpr, epsilon, bpk,
			float64(filter.SizeInBits())/float64(n),
			float64(filter.CDFSizeInBits())/float64(n),
			trueEmpty)
	}
}

func TestPGMARE_FPR_Cluster(t *testing.T) {
	rng := rand.New(rand.NewSource(456))
	n := 10000

	centers := []uint64{1 << 30, 1 << 35, 1 << 40, 1 << 42, 1 << 44}
	keys := make([]uint64, 0, n)
	perCluster := n / len(centers)
	for _, c := range centers {
		for j := 0; j < perCluster; j++ {
			keys = append(keys, safeGaussUint64(c, float64(1<<20), rng))
		}
	}

	rangeLen := uint64(1 << 20)
	sorted := make([]uint64, len(keys))
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	for _, pgmEps := range []int{8, 32, 64, 128} {
		epsilon := 0.05
		filter, err := NewPGMApproximateRangeEmptiness(keys, rangeLen, epsilon, pgmEps)
		if err != nil {
			t.Fatalf("pgmEps=%d: %v", pgmEps, err)
		}

		fpr, trueEmpty := measureFPR(t, filter, sorted, func(rng *rand.Rand) (uint64, uint64) {
			center := centers[rng.Intn(len(centers))]
			a := safeGaussUint64(center, float64(1<<22), rng)
			b := a + rangeLen
			return a, b
		}, 789, 100000)

		bpk := float64(filter.TotalSizeInBits()) / float64(n)

		t.Logf("Cluster pgmEps=%d: K=%d, FPR=%.4f (target %.4f), BPK=%.2f (ERE=%.2f + CDF=%.2f), trueEmpty=%d",
			pgmEps, filter.K, fpr, epsilon, bpk,
			float64(filter.SizeInBits())/float64(n),
			float64(filter.CDFSizeInBits())/float64(n),
			trueEmpty)
	}
}

func TestPGMARE_Empty(t *testing.T) {
	filter, err := NewPGMApproximateRangeEmptiness(nil, 100, 0.01, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filter.IsEmpty(0, 100) {
		t.Error("empty filter should always return true")
	}
}
