package are_dbscan

// adversarial_fpr_test.go — adversarial strategies targeting HybridScanARE FPR guarantees.
//
// Each strategy is designed to exploit a specific structural weakness:
//   A1: P5-gap bypass — 95%+ gaps huge, 4% tiny → truncSafe says "safe" for the wrong region
//   A2: Sub-minClusterSize clusters dissolved → fallback has multi-modal distribution
//   A3: Cluster boundary straddle — queries where a < clusterMin <= clusterMax < b
//   A4: Borderline DBSCAN density — keys spaced so they oscillate cluster ↔ fallback
//   A5: Phantom overlap bomb — many tight keys + one very distant key → huge spread, tiny phantomSize threshold fails
//   A6: Spread distribution with queries targeting the wide gaps exactly
//   A7 (bonus): Keys all identical after truncation (dedup collapse in trunc)

import (
	"math/rand"
	"sort"
	"testing"

	"Thesis/testutils"
)

const (
	advFPRN          = 50_000
	advFPRQueryCount = 300_000
	advFPRSlack      = 5.0 // tolerate up to 5×epsilon before flagging
)

// measureFPRScan measures FPR for HybridScanARE on a set of empty queries.
// It skips any query that contains at least one key (ground truth non-empty).
func measureFPRScan(t *testing.T, keys []uint64, queries [][2]uint64, filter *HybridScanARE) float64 {
	t.Helper()
	return testutils.MeasureFPR(keys, queries, filter.IsEmpty)
}

// buildScanFilter builds HybridScanARE and fatals on error.
func buildScanFilter(t *testing.T, keys []uint64, rangeLen uint64, epsilon float64) *HybridScanARE {
	t.Helper()
	f, err := NewHybridScanARE(keys, 64, Config{K: kFromEps(len(keys), rangeLen, epsilon)})
	if err != nil {
		t.Fatalf("NewHybridScanARE build failed: %v", err)
	}
	return f
}

// uniformQueries generates n random queries [a, a+rangeLen-1] over the full uint64 space.
func uniformQueries(n int, rangeLen uint64, seed int64) [][2]uint64 {
	rng := rand.New(rand.NewSource(seed))
	qs := make([][2]uint64, n)
	for i := range qs {
		a := rng.Uint64()
		qs[i] = [2]uint64{a, a + rangeLen - 1}
	}
	return qs
}

// ---
// A1: P5-gap bypass
// ---
func TestAdversarialFPR_A1_P5GapBypass(t *testing.T) {
	const (
		rangeLen = uint64(1000)
		epsilon  = 0.01
		n        = advFPRN
	)

	tightCount := n * 19 / 20

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)

	for i := 0; i < tightCount; i++ {
		v := uint64(i)
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}

	rng := rand.New(rand.NewSource(11111))
	base := uint64(1) << 50
	for len(keys) < n {
		v := base + rng.Uint64()%(uint64(1)<<50)
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	filter := buildScanFilter(t, keys, rangeLen, epsilon)
	nc, nf, nt := filter.Stats()
	t.Logf("A1: clusters=%d fallback=%d total=%d", nc, nf, nt)

	queries := make([][2]uint64, advFPRQueryCount)
	qrng := rand.New(rand.NewSource(22222))
	for i := range queries {
		a := qrng.Uint64() % (uint64(tightCount) + rangeLen)
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fpr := measureFPRScan(t, keys, queries, filter)
	t.Logf("A1 FPR=%.6f target=%.4f (5× limit = %.4f)", fpr, epsilon, advFPRSlack*epsilon)

	if fpr > advFPRSlack*epsilon {
		t.Errorf("A1 BROKEN: FPR %.6f > %.4f (5×ε)", fpr, advFPRSlack*epsilon)
	}
}

// ---
// A2: Sub-minClusterSize cluster dissolution
// ---
func TestAdversarialFPR_A2_SmallClusterDissolution(t *testing.T) {
	const (
		rangeLen      = uint64(100)
		epsilon       = 0.01
		numMiniClust  = 30
		keysPerMini   = 255 // just below minClusterSize=256
		interClustGap = uint64(10_000_000)
	)
	n := numMiniClust * keysPerMini

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)

	for c := 0; c < numMiniClust; c++ {
		base := uint64(c) * interClustGap
		for i := 0; i < keysPerMini; i++ {
			v := base + uint64(i)
			if !seen[v] {
				seen[v] = true
				keys = append(keys, v)
			}
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	filter := buildScanFilter(t, keys, rangeLen, epsilon)
	nc, nf, nt := filter.Stats()
	t.Logf("A2: clusters=%d fallback=%d total=%d (all should be fallback, clusters < 256)", nc, nf, nt)

	rng := rand.New(rand.NewSource(33333))
	queries := make([][2]uint64, advFPRQueryCount)
	for i := range queries {
		clust := rng.Intn(numMiniClust - 1)
		mid := uint64(clust)*interClustGap + interClustGap/2
		a := mid - rangeLen/2
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fpr := measureFPRScan(t, keys, queries, filter)
	t.Logf("A2 FPR=%.6f target=%.4f (5× limit = %.4f)", fpr, epsilon, advFPRSlack*epsilon)

	if fpr > advFPRSlack*epsilon {
		t.Errorf("A2 BROKEN: FPR %.6f > %.4f (5×ε)", fpr, advFPRSlack*epsilon)
	}
}

// ---
// A3: Cluster boundary straddle queries
// ---
func TestAdversarialFPR_A3_ClusterBoundaryStraddle(t *testing.T) {
	const (
		rangeLen      = uint64(500)
		epsilon       = 0.01
		numClusters   = 5
		keysPerClust  = 5000
		clusterWidth  = uint64(20_000)
		interClustGap = uint64(200_000_000)
	)
	n := numClusters * keysPerClust

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)

	for c := 0; c < numClusters; c++ {
		base := uint64(c) * (clusterWidth + interClustGap)
		for i := 0; i < keysPerClust; i++ {
			v := base + uint64(i)*2
			if !seen[v] {
				seen[v] = true
				keys = append(keys, v)
			}
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	filter := buildScanFilter(t, keys, rangeLen, epsilon)
	nc, nf, nt := filter.Stats()
	t.Logf("A3: clusters=%d fallback=%d total=%d", nc, nf, nt)

	const safeGap = interClustGap - rangeLen - 1
	queries := make([][2]uint64, advFPRQueryCount)
	rng := rand.New(rand.NewSource(55555))
	for i := range queries {
		c := rng.Intn(numClusters)
		clusterBase := uint64(c) * (clusterWidth + interClustGap)
		clusterMax := clusterBase + uint64(keysPerClust-1)*2

		offset := rng.Uint64() % safeGap
		a := clusterMax + 1 + offset
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fpr := measureFPRScan(t, keys, queries, filter)
	t.Logf("A3 FPR=%.6f target=%.4f (5× limit = %.4f)", fpr, epsilon, advFPRSlack*epsilon)

	if fpr > advFPRSlack*epsilon {
		t.Errorf("A3 BROKEN: FPR %.6f > %.4f (5×ε)", fpr, advFPRSlack*epsilon)
	}
}

// ---
// A4: Borderline DBSCAN density — half in, half out
// ---
func TestAdversarialFPR_A4_BorderlineDBSCANDensity(t *testing.T) {
	const (
		rangeLen = uint64(1000)
		epsilon  = 0.01
		n        = advFPRN
	)

	dbscanEps := uint64(float64(epsMultiplier) * float64(rangeLen) / epsilon)
	gap := dbscanEps / uint64(dbscanMinPts-1)
	if gap == 0 {
		gap = 1
	}

	keys := make([]uint64, n)
	base := uint64(1_000_000_000)
	for i := range keys {
		keys[i] = base + uint64(i)*gap
	}

	filter := buildScanFilter(t, keys, rangeLen, epsilon)
	nc, nf, nt := filter.Stats()
	t.Logf("A4: clusters=%d fallback=%d total=%d dbscanEps=%d gap=%d", nc, nf, nt, dbscanEps, gap)

	queries := uniformQueries(advFPRQueryCount, rangeLen, 66666)
	fpr := measureFPRScan(t, keys, queries, filter)
	t.Logf("A4 FPR=%.6f target=%.4f (5× limit = %.4f)", fpr, epsilon, advFPRSlack*epsilon)

	if fpr > advFPRSlack*epsilon {
		t.Errorf("A4 BROKEN: FPR %.6f > %.4f (5×ε)", fpr, advFPRSlack*epsilon)
	}
}

// ---
// A5: Phantom overlap bomb — one extreme outlier inflates spread
// ---
func TestAdversarialFPR_A5_OutlierInflatedSpread(t *testing.T) {
	const (
		rangeLen   = uint64(100)
		epsilon    = 0.01
		n          = advFPRN
		tightCount = n - 1
	)

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)

	for i := 0; i < tightCount; i++ {
		v := uint64(i) * 2
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}

	outlier := uint64(1) << 62
	keys = append(keys, outlier)
	seen[outlier] = true

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	filter := buildScanFilter(t, keys, rangeLen, epsilon)
	nc, nf, nt := filter.Stats()
	t.Logf("A5: clusters=%d fallback=%d total=%d", nc, nf, nt)

	rng := rand.New(rand.NewSource(77777))
	queries := make([][2]uint64, advFPRQueryCount)
	for i := range queries {
		a := rng.Uint64() % (uint64(tightCount) * 2)
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fpr := measureFPRScan(t, keys, queries, filter)
	t.Logf("A5 FPR=%.6f target=%.4f (5× limit = %.4f)", fpr, epsilon, advFPRSlack*epsilon)

	if fpr > advFPRSlack*epsilon {
		t.Errorf("A5 BROKEN: FPR %.6f > %.4f (5×ε)", fpr, advFPRSlack*epsilon)
	}
}

// ---
// A6: Spread distribution with targeted gap queries
// ---
func TestAdversarialFPR_A6_ExponentialSpacingTruncOverlap(t *testing.T) {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
		n        = advFPRN
	)

	hugeGap := rangeLen * 10_000
	smallGap := uint64(1)

	smallCount := n / 20
	largeCount := n - smallCount

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)

	for i := 0; i < smallCount; i++ {
		v := uint64(i) * smallGap
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}

	lastKey := uint64(smallCount-1) * smallGap
	for i := 0; i < largeCount; i++ {
		v := lastKey + uint64(i+1)*hugeGap
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	filter := buildScanFilter(t, keys, rangeLen, epsilon)
	nc, nf, nt := filter.Stats()
	t.Logf("A6: clusters=%d fallback=%d total=%d smallGap=%d hugeGap=%d", nc, nf, nt, smallGap, hugeGap)

	rng := rand.New(rand.NewSource(88888))
	queries := make([][2]uint64, advFPRQueryCount)
	for i := range queries {
		idx := smallCount + rng.Intn(largeCount-1)
		keyA := lastKey + uint64(idx-smallCount+1)*hugeGap
		keyB := lastKey + uint64(idx-smallCount+2)*hugeGap
		if keyB <= keyA+rangeLen {
			a := rng.Uint64()
			queries[i] = [2]uint64{a, a + rangeLen - 1}
			continue
		}
		gapInterior := keyA + 1
		a := gapInterior + rng.Uint64()%(keyB-keyA-rangeLen)
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fpr := measureFPRScan(t, keys, queries, filter)
	t.Logf("A6 FPR=%.6f target=%.4f (5× limit = %.4f)", fpr, epsilon, advFPRSlack*epsilon)

	if fpr > advFPRSlack*epsilon {
		t.Errorf("A6 BROKEN: FPR %.6f > %.4f (5×ε)", fpr, advFPRSlack*epsilon)
	}
}

// ---
// A7: Degenerate truncation collapse — many distinct keys map to same K-bit prefix
// ---
func TestAdversarialFPR_A7_TruncCollapseSmallSpread(t *testing.T) {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
		spread   = uint64(1) << 16
		n        = 10_000
	)

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)
	rng := rand.New(rand.NewSource(99999))

	base := uint64(1) << 40
	for len(keys) < n {
		v := base + rng.Uint64()%spread
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	filter := buildScanFilter(t, keys, rangeLen, epsilon)
	nc, nf, nt := filter.Stats()
	t.Logf("A7: clusters=%d fallback=%d total=%d spread=%d", nc, nf, nt, spread)

	queries := make([][2]uint64, advFPRQueryCount)
	qrng := rand.New(rand.NewSource(10001))
	for i := range queries {
		a := base + qrng.Uint64()%spread
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fpr := measureFPRScan(t, keys, queries, filter)
	t.Logf("A7 FPR=%.6f target=%.4f (5× limit = %.4f)", fpr, epsilon, advFPRSlack*epsilon)

	if fpr > advFPRSlack*epsilon {
		t.Errorf("A7 BROKEN: FPR %.6f > %.4f (5×ε)", fpr, advFPRSlack*epsilon)
	}
}

// ---
// A8: Adversarial SODA hash collision via multi-block queries
// ---
func TestAdversarialFPR_A8_SODAMultiBlockQuery(t *testing.T) {
	const (
		rangeLen = uint64(1_000_000)
		epsilon  = 0.01
		n        = 20_000
	)

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)
	rng := rand.New(rand.NewSource(20202))

	for len(keys) < n {
		v := rng.Uint64() >> 4
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	filter := buildScanFilter(t, keys, rangeLen, epsilon)
	nc, nf, nt := filter.Stats()
	t.Logf("A8: clusters=%d fallback=%d total=%d rangeLen=%d", nc, nf, nt, rangeLen)

	queries := uniformQueries(advFPRQueryCount, rangeLen, 30303)
	fpr := measureFPRScan(t, keys, queries, filter)
	t.Logf("A8 FPR=%.6f target=%.4f (5× limit = %.4f)", fpr, epsilon, advFPRSlack*epsilon)

	if fpr > advFPRSlack*epsilon {
		t.Errorf("A8 BROKEN: FPR %.6f > %.4f (5×ε)", fpr, advFPRSlack*epsilon)
	}
}

// TestAdversarialFPR is the umbrella test that runs all adversarial strategies
// and prints a combined summary.
func TestAdversarialFPR(t *testing.T) {
	strategies := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"A1/P5GapBypass", TestAdversarialFPR_A1_P5GapBypass},
		{"A2/SmallClusterDissolution", TestAdversarialFPR_A2_SmallClusterDissolution},
		{"A3/ClusterBoundaryStraddle", TestAdversarialFPR_A3_ClusterBoundaryStraddle},
		{"A4/BorderlineDBSCANDensity", TestAdversarialFPR_A4_BorderlineDBSCANDensity},
		{"A5/OutlierInflatedSpread", TestAdversarialFPR_A5_OutlierInflatedSpread},
		{"A6/ExponentialSpacingTruncOverlap", TestAdversarialFPR_A6_ExponentialSpacingTruncOverlap},
		{"A7/TruncCollapseSmallSpread", TestAdversarialFPR_A7_TruncCollapseSmallSpread},
		{"A8/SODAMultiBlockQuery", TestAdversarialFPR_A8_SODAMultiBlockQuery},
	}

	for _, s := range strategies {
		s := s
		t.Run(s.name, func(t *testing.T) {
			s.fn(t)
		})
	}
}
