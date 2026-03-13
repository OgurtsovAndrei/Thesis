package are_pgm

import (
	"Thesis/emptiness/are_soda_hash"
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

func TestPGMARE_FPR_SmallL_Uniform(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	n := 10000

	keys := make([]uint64, n)
	for i := range keys {
		keys[i] = rng.Uint64() >> 16
	}

	sorted := make([]uint64, n)
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	for _, rangeLen := range []uint64{128, 1024} {
		t.Logf("=== Uniform, rangeLen=%d ===", rangeLen)
		for _, pgmEps := range []int{32, 64, 128} {
			epsilon := 0.05
			filter, err := NewPGMApproximateRangeEmptiness(keys, rangeLen, epsilon, pgmEps)
			if err != nil {
				t.Fatalf("L=%d pgmEps=%d: %v", rangeLen, pgmEps, err)
			}

			fpr, trueEmpty := measureFPR(t, filter, sorted, func(rng *rand.Rand) (uint64, uint64) {
				a := rng.Uint64() >> 16
				b := a + rangeLen
				if b < a {
					b = ^uint64(0) >> 16
				}
				return a, b
			}, 999, 200000)

			bpk := float64(filter.TotalSizeInBits()) / float64(n)
			t.Logf("  pgmEps=%3d: K=%2d, FPR=%.4f (target %.4f), BPK=%.2f (ERE=%.2f + CDF=%.2f), trueEmpty=%d",
				pgmEps, filter.K, fpr, epsilon, bpk,
				float64(filter.SizeInBits())/float64(n),
				float64(filter.CDFSizeInBits())/float64(n),
				trueEmpty)
		}
	}
}

func TestPGMARE_FPR_SmallL_Cluster(t *testing.T) {
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

	sorted := make([]uint64, len(keys))
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	for _, rangeLen := range []uint64{128, 1024} {
		t.Logf("=== Cluster, rangeLen=%d ===", rangeLen)
		for _, pgmEps := range []int{32, 64, 128} {
			epsilon := 0.05
			filter, err := NewPGMApproximateRangeEmptiness(keys, rangeLen, epsilon, pgmEps)
			if err != nil {
				t.Fatalf("L=%d pgmEps=%d: %v", rangeLen, pgmEps, err)
			}

			fpr, trueEmpty := measureFPR(t, filter, sorted, func(rng *rand.Rand) (uint64, uint64) {
				center := centers[rng.Intn(len(centers))]
				a := safeGaussUint64(center, float64(1<<20), rng)
				b := a + rangeLen
				return a, b
			}, 789, 200000)

			bpk := float64(filter.TotalSizeInBits()) / float64(n)
			t.Logf("  pgmEps=%3d: K=%2d, FPR=%.4f (target %.4f), BPK=%.2f (ERE=%.2f + CDF=%.2f), trueEmpty=%d",
				pgmEps, filter.K, fpr, epsilon, bpk,
				float64(filter.SizeInBits())/float64(n),
				float64(filter.CDFSizeInBits())/float64(n),
				trueEmpty)
		}
	}
}

func TestPGMARE_FPR_SmallL_Cluster_Smoothing(t *testing.T) {
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

	sorted := make([]uint64, len(keys))
	copy(sorted, keys)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	rangeLen := uint64(128)
	pgmEps := 64

	for _, smooth := range []float64{0.0, 0.01, 0.05, 0.1, 0.2, 0.5, 1.0} {
		epsilon := 0.05
		filter, err := NewPGMApproximateRangeEmptinessSmooth(keys, rangeLen, epsilon, pgmEps, smooth)
		if err != nil {
			t.Fatalf("smooth=%.2f: %v", smooth, err)
		}

		fpr, trueEmpty := measureFPR(t, filter, sorted, func(rng *rand.Rand) (uint64, uint64) {
			center := centers[rng.Intn(len(centers))]
			a := safeGaussUint64(center, float64(1<<20), rng)
			b := a + rangeLen
			return a, b
		}, 789, 200000)

		bpk := float64(filter.TotalSizeInBits()) / float64(n)
		t.Logf("smooth=%.2f: K=%2d, FPR=%.4f (target %.4f), BPK=%.2f (ERE=%.2f + CDF=%.2f), trueEmpty=%d",
			smooth, filter.K, fpr, epsilon, bpk,
			float64(filter.SizeInBits())/float64(n),
			float64(filter.CDFSizeInBits())/float64(n),
			trueEmpty)
	}
}

type rangeFilter interface {
	IsEmpty(a, b uint64) bool
}

func measureFPRGeneric(sortedKeys []uint64, filter rangeFilter,
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

func TestPGMARE_vs_SODA(t *testing.T) {
	n := 10000
	epsilon := 0.05
	numQueries := 200000

	// Uniform keys
	rngU := rand.New(rand.NewSource(123))
	uniformKeys := make([]uint64, n)
	for i := range uniformKeys {
		uniformKeys[i] = rngU.Uint64() >> 16
	}
	sortedUniform := make([]uint64, n)
	copy(sortedUniform, uniformKeys)
	sort.Slice(sortedUniform, func(i, j int) bool { return sortedUniform[i] < sortedUniform[j] })

	// Cluster keys
	centers := []uint64{1 << 30, 1 << 35, 1 << 40, 1 << 42, 1 << 44}
	rngC := rand.New(rand.NewSource(456))
	clusterKeys := make([]uint64, 0, n)
	perCluster := n / len(centers)
	for _, c := range centers {
		for j := 0; j < perCluster; j++ {
			clusterKeys = append(clusterKeys, safeGaussUint64(c, float64(1<<20), rngC))
		}
	}
	sortedCluster := make([]uint64, len(clusterKeys))
	copy(sortedCluster, clusterKeys)
	sort.Slice(sortedCluster, func(i, j int) bool { return sortedCluster[i] < sortedCluster[j] })

	for _, rangeLen := range []uint64{128, 1024} {
		t.Logf("========== rangeLen=%d ==========", rangeLen)

		// --- SODA ---
		sodaU, err := are_soda_hash.NewApproximateRangeEmptinessSoda(uniformKeys, rangeLen, epsilon)
		if err != nil {
			t.Fatalf("SODA uniform: %v", err)
		}
		sodaC, err := are_soda_hash.NewApproximateRangeEmptinessSoda(clusterKeys, rangeLen, epsilon)
		if err != nil {
			t.Fatalf("SODA cluster: %v", err)
		}

		// --- CDF-ARE pgmEps=64 ---
		cdfU, err := NewPGMApproximateRangeEmptiness(uniformKeys, rangeLen, epsilon, 64)
		if err != nil {
			t.Fatalf("CDF uniform: %v", err)
		}
		cdfC, err := NewPGMApproximateRangeEmptiness(clusterKeys, rangeLen, epsilon, 64)
		if err != nil {
			t.Fatalf("CDF cluster: %v", err)
		}

		// Uniform queries
		uniformQuery := func(rng *rand.Rand) (uint64, uint64) {
			a := rng.Uint64() >> 16
			b := a + rangeLen
			if b < a {
				b = ^uint64(0) >> 16
			}
			return a, b
		}
		// Cluster queries (σ = σ_data)
		clusterQuery := func(rng *rand.Rand) (uint64, uint64) {
			center := centers[rng.Intn(len(centers))]
			a := safeGaussUint64(center, float64(1<<20), rng)
			b := a + rangeLen
			return a, b
		}

		// Uniform data, uniform queries
		fpr, te := measureFPRGeneric(sortedUniform, sodaU, uniformQuery, 999, numQueries)
		t.Logf("  SODA    uniform-data uniform-query: K=%2d, FPR=%.4f, BPK=%.2f, trueEmpty=%d",
			sodaU.K, fpr, float64(sodaU.SizeInBits())/float64(n), te)

		fpr, te = measureFPRGeneric(sortedUniform, cdfU, uniformQuery, 999, numQueries)
		t.Logf("  CDF-ARE uniform-data uniform-query: K=%2d, FPR=%.4f, BPK=%.2f (ERE=%.2f+CDF=%.2f), trueEmpty=%d",
			cdfU.K, fpr, float64(cdfU.TotalSizeInBits())/float64(n),
			float64(cdfU.SizeInBits())/float64(n), float64(cdfU.CDFSizeInBits())/float64(n), te)

		// Cluster data, cluster queries
		fpr, te = measureFPRGeneric(sortedCluster, sodaC, clusterQuery, 789, numQueries)
		t.Logf("  SODA    cluster-data cluster-query: K=%2d, FPR=%.4f, BPK=%.2f, trueEmpty=%d",
			sodaC.K, fpr, float64(sodaC.SizeInBits())/float64(n), te)

		fpr, te = measureFPRGeneric(sortedCluster, cdfC, clusterQuery, 789, numQueries)
		t.Logf("  CDF-ARE cluster-data cluster-query: K=%2d, FPR=%.4f, BPK=%.2f (ERE=%.2f+CDF=%.2f), trueEmpty=%d",
			cdfC.K, fpr, float64(cdfC.TotalSizeInBits())/float64(n),
			float64(cdfC.SizeInBits())/float64(n), float64(cdfC.CDFSizeInBits())/float64(n), te)
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
