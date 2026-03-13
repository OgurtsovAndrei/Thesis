package are_pgm

import (
	"Thesis/emptiness/are_soda_hash"
	"Thesis/testutils"
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

// generateQueries pre-generates a [][2]uint64 slice using a callback query generator.
func generateQueries(queryGen func(rng *rand.Rand) (uint64, uint64), seed int64, numQueries int) [][2]uint64 {
	rng := rand.New(rand.NewSource(seed))
	queries := make([][2]uint64, numQueries)
	for i := range queries {
		a, b := queryGen(rng)
		queries[i] = [2]uint64{a, b}
	}
	return queries
}

// countTrueEmpty counts how many queries represent truly empty ranges.
func countTrueEmpty(sortedKeys []uint64, queries [][2]uint64) int {
	count := 0
	for _, q := range queries {
		if q[1] >= q[0] && testutils.GroundTruth(sortedKeys, q[0], q[1]) {
			count++
		}
	}
	return count
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

		queries := generateQueries(func(rng *rand.Rand) (uint64, uint64) {
			a := rng.Uint64() >> 16
			b := a + rangeLen
			if b < a {
				b = ^uint64(0) >> 16
			}
			return a, b
		}, 999, 100000)

		fpr := testutils.MeasureFPR(sorted, queries, filter.IsEmpty)
		trueEmpty := countTrueEmpty(sorted, queries)
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
			keys = append(keys, testutils.SampleGaussian(c, float64(1<<20), rng))
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

		queries := generateQueries(func(rng *rand.Rand) (uint64, uint64) {
			center := centers[rng.Intn(len(centers))]
			a := testutils.SampleGaussian(center, float64(1<<22), rng)
			b := a + rangeLen
			return a, b
		}, 789, 100000)

		fpr := testutils.MeasureFPR(sorted, queries, filter.IsEmpty)
		trueEmpty := countTrueEmpty(sorted, queries)
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

			queries := generateQueries(func(rng *rand.Rand) (uint64, uint64) {
				a := rng.Uint64() >> 16
				b := a + rangeLen
				if b < a {
					b = ^uint64(0) >> 16
				}
				return a, b
			}, 999, 200000)

			fpr := testutils.MeasureFPR(sorted, queries, filter.IsEmpty)
			trueEmpty := countTrueEmpty(sorted, queries)
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
			keys = append(keys, testutils.SampleGaussian(c, float64(1<<20), rng))
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

			queries := generateQueries(func(rng *rand.Rand) (uint64, uint64) {
				center := centers[rng.Intn(len(centers))]
				a := testutils.SampleGaussian(center, float64(1<<20), rng)
				b := a + rangeLen
				return a, b
			}, 789, 200000)

			fpr := testutils.MeasureFPR(sorted, queries, filter.IsEmpty)
			trueEmpty := countTrueEmpty(sorted, queries)
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
			keys = append(keys, testutils.SampleGaussian(c, float64(1<<20), rng))
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

		queries := generateQueries(func(rng *rand.Rand) (uint64, uint64) {
			center := centers[rng.Intn(len(centers))]
			a := testutils.SampleGaussian(center, float64(1<<20), rng)
			b := a + rangeLen
			return a, b
		}, 789, 200000)

		fpr := testutils.MeasureFPR(sorted, queries, filter.IsEmpty)
		trueEmpty := countTrueEmpty(sorted, queries)
		bpk := float64(filter.TotalSizeInBits()) / float64(n)
		t.Logf("smooth=%.2f: K=%2d, FPR=%.4f (target %.4f), BPK=%.2f (ERE=%.2f + CDF=%.2f), trueEmpty=%d",
			smooth, filter.K, fpr, epsilon, bpk,
			float64(filter.SizeInBits())/float64(n),
			float64(filter.CDFSizeInBits())/float64(n),
			trueEmpty)
	}
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
			clusterKeys = append(clusterKeys, testutils.SampleGaussian(c, float64(1<<20), rngC))
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

		// Pre-generate queries
		uniformQueries := generateQueries(func(rng *rand.Rand) (uint64, uint64) {
			a := rng.Uint64() >> 16
			b := a + rangeLen
			if b < a {
				b = ^uint64(0) >> 16
			}
			return a, b
		}, 999, numQueries)

		clusterQueries := generateQueries(func(rng *rand.Rand) (uint64, uint64) {
			center := centers[rng.Intn(len(centers))]
			a := testutils.SampleGaussian(center, float64(1<<20), rng)
			b := a + rangeLen
			return a, b
		}, 789, numQueries)

		// Uniform data, uniform queries
		fpr := testutils.MeasureFPR(sortedUniform, uniformQueries, sodaU.IsEmpty)
		te := countTrueEmpty(sortedUniform, uniformQueries)
		t.Logf("  SODA    uniform-data uniform-query: K=%2d, FPR=%.4f, BPK=%.2f, trueEmpty=%d",
			sodaU.K, fpr, float64(sodaU.SizeInBits())/float64(n), te)

		fpr = testutils.MeasureFPR(sortedUniform, uniformQueries, cdfU.IsEmpty)
		t.Logf("  CDF-ARE uniform-data uniform-query: K=%2d, FPR=%.4f, BPK=%.2f (ERE=%.2f+CDF=%.2f), trueEmpty=%d",
			cdfU.K, fpr, float64(cdfU.TotalSizeInBits())/float64(n),
			float64(cdfU.SizeInBits())/float64(n), float64(cdfU.CDFSizeInBits())/float64(n), te)

		// Cluster data, cluster queries
		fpr = testutils.MeasureFPR(sortedCluster, clusterQueries, sodaC.IsEmpty)
		te = countTrueEmpty(sortedCluster, clusterQueries)
		t.Logf("  SODA    cluster-data cluster-query: K=%2d, FPR=%.4f, BPK=%.2f, trueEmpty=%d",
			sodaC.K, fpr, float64(sodaC.SizeInBits())/float64(n), te)

		fpr = testutils.MeasureFPR(sortedCluster, clusterQueries, cdfC.IsEmpty)
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
