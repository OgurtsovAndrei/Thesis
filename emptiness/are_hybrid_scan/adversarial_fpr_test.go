package are_hybrid_scan

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
	isEmpty := func(a, b uint64) bool {
		return filter.IsEmpty(trieBS(a), trieBS(b))
	}
	return testutils.MeasureFPR(keys, queries, isEmpty)
}

// buildScanFilter builds HybridScanARE and fatals on error.
func buildScanFilter(t *testing.T, keys []uint64, rangeLen uint64, epsilon float64) *HybridScanARE {
	t.Helper()
	bs := makeSortedBS(keys)
	f, err := NewHybridScanARE(bs, rangeLen, epsilon)
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
//
// Weak point: truncSafe uses the 5th-percentile gap as proxy for min gap.
// If exactly 5% of gaps are tiny (≈ 1) but 95% are huge, the P5 index
// (= len(gaps)/20) falls on a TINY gap → truncSafe returns false → adaptive used.
// The attack: make the P5 gap just barely above phantomSize so truncSafe returns
// TRUE, but many gaps above P5 have actual gaps << phantomSize.
//
// Construction: 95% of keys tightly packed at spacing 1, 5% randomly placed
// far away. After sorting, the smallest 5% of gaps come from the tight block.
// The largest gaps (from far-placed keys) are >> phantomSize. If phantomSize
// is derived from the spread (max−min >> gap between tight keys), many tight
// gaps collapse to the same truncated value → phantom overlap within the tight region.
// ---
func TestAdversarialFPR_A1_P5GapBypass(t *testing.T) {
	const (
		rangeLen = uint64(1000)
		epsilon  = 0.01
		n        = advFPRN
	)

	// 95% tight (gap=1), 5% randomly scattered very far.
	tightCount := n * 19 / 20

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)

	// Tight block near 0 with unit gaps.
	for i := 0; i < tightCount; i++ {
		v := uint64(i)
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}

	// Scattered keys pushed far out — beyond 2^50 so spread is huge.
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

	// Queries specifically in the tight region — many phantom-size windows overlap.
	queries := make([][2]uint64, advFPRQueryCount)
	qrng := rand.New(rand.NewSource(22222))
	for i := range queries {
		// Query within the tight region (0..tightCount).
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
//
// Weak point: minClusterSize = 256. Clusters with < 256 keys are dissolved into
// fallback. If we create many tight mini-clusters of size 255, they all go to
// fallback. The fallback then has a multi-modal distribution: many separate
// dense islands. truncSafe sees the overall spread (covering all islands) and may
// still approve trunc — but phantom overlap now bridges the wide inter-island gaps.
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

	// Queries targeting the wide inter-cluster gaps.
	rng := rand.New(rand.NewSource(33333))
	queries := make([][2]uint64, advFPRQueryCount)
	for i := range queries {
		// Pick midpoint between two consecutive mini-clusters.
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
//
// Weak point: when a query [a, b] straddles a cluster boundary (a < cluster.minKey
// and b > cluster.maxKey), IsEmpty scans BOTH the cluster filter AND the fallback.
// A false positive fires if EITHER returns non-empty. If both the cluster filter
// has phantom overlap near its boundary AND the fallback has phantom overlap near
// the same region, the combined path has higher effective FPR.
// ---
func TestAdversarialFPR_A3_ClusterBoundaryStraddle(t *testing.T) {
	const (
		rangeLen      = uint64(500)
		epsilon       = 0.01
		numClusters   = 5
		keysPerClust  = 5000
		clusterWidth  = uint64(20_000)      // wide enough for 5000 distinct keys
		interClustGap = uint64(200_000_000) // large gap between clusters
	)
	n := numClusters * keysPerClust

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)

	// Well-separated tight clusters.
	for c := 0; c < numClusters; c++ {
		base := uint64(c) * (clusterWidth + interClustGap)
		// Use consecutive keys within each cluster — guaranteed unique and bounded.
		for i := 0; i < keysPerClust; i++ {
			v := base + uint64(i)*2 // stride 2 to leave small gaps
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

	// Queries that land in the inter-cluster gap just outside each cluster's boundary.
	// This exercises the path where the cluster filter is NOT invoked (a > clusterMax)
	// but the fallback still processes the query.
	const safeGap = interClustGap - rangeLen - 1 // safe modulus
	queries := make([][2]uint64, advFPRQueryCount)
	rng := rand.New(rand.NewSource(55555))
	for i := range queries {
		c := rng.Intn(numClusters)
		clusterBase := uint64(c) * (clusterWidth + interClustGap)
		clusterMax := clusterBase + uint64(keysPerClust-1)*2

		// Right-side gap: starts just after cluster ends.
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
//
// Weak point: DBSCAN eps = 10 * L / ε. Keys spaced at exactly eps/minPts so
// some windows have exactly minPts-1 neighbors (noise) and others have minPts
// (core). The assignment fluctuates based on exact positioning. Some keys end
// up in clusters, others in fallback, creating a split-brain state where queries
// near the cluster-fallback seam are handled by both filters simultaneously with
// uncorrelated phantom overlaps.
// ---
func TestAdversarialFPR_A4_BorderlineDBSCANDensity(t *testing.T) {
	const (
		rangeLen = uint64(1000)
		epsilon  = 0.01
		n        = advFPRN
	)

	// dbscanEps = 10 * L / ε = 10 * 1000 / 0.01 = 1_000_000
	// Space keys at exactly dbscanEps / (minPts - 1) = 1_000_000 / 9 ≈ 111_111
	// This puts minPts-1 keys in the eps window → borderline core/noise.
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
//
// Weak point: phantom_size = spread / 2^K. If spread is dominated by a single
// outlier key, phantom_size becomes huge. Many inter-key gaps that are smaller
// than phantom_size collide after truncation, causing phantom overlaps.
// truncSafe uses P5 gap; if the tight region contains ≥5% of gaps all small,
// P5 is small too → truncSafe returns false → adaptive used. But the adaptive
// SODA mode has its own vulnerability when many keys hash to the same block.
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

	// Almost all keys tightly clustered at spacing 2 (leaves gap=1 between).
	for i := 0; i < tightCount; i++ {
		v := uint64(i) * 2
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}

	// Single extreme outlier at 2^62, inflating spread to ~2^62.
	outlier := uint64(1) << 62
	keys = append(keys, outlier)
	seen[outlier] = true

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	filter := buildScanFilter(t, keys, rangeLen, epsilon)
	nc, nf, nt := filter.Stats()
	t.Logf("A5: clusters=%d fallback=%d total=%d", nc, nf, nt)

	// Query the tight region where phantom overlap is worst.
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
//
// Weak point: spread distribution has huge consecutive gaps. When trunc is used,
// each key maps to the same handful of truncated values, creating phantom overlaps
// across the entire span. Queries in these wide gaps should be empty yet the trunc
// filter may fire if the gap width < phantom_size.
//
// Construction: keys at positions that are powers of 2 — exponentially spaced.
// The P5 gap (5th smallest) will be tiny (low-end keys dense) while the top 95%
// of gaps are enormous. If truncSafe sees a small P5, it rejects trunc → adaptive.
// But if we space them so P5 is just above phantom_size, trunc is used and the
// huge gaps suffer phantom overlap.
// ---
func TestAdversarialFPR_A6_ExponentialSpacingTruncOverlap(t *testing.T) {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
		n        = advFPRN
	)

	// Keys with mixed-density: the bottom 10% have unit spacing, top 90% have
	// exponentially growing gaps. This ensures P5 gap ≈ 1 (from the dense bottom)
	// so truncSafe returns false → adaptive used. But adaptive SODA mode can still
	// suffer if many keys collide to the same hash block.
	//
	// Alternative: space keys so P5 gap ≈ phantomSize + 1 (barely safe).
	// Then trunc is enabled, and the wide-gap queries see phantom overlaps.
	//
	// We compute phantomSize = spread / 2^K, then set P5 gap = phantomSize + 2.
	// The bottom 5% gaps are small, upper 95% are huge (= rangeLen * 10000).

	hugeGap := rangeLen * 10_000
	smallGap := uint64(1)

	// First n/20 consecutive keys have small gaps (to anchor P5).
	smallCount := n / 20
	largeCount := n - smallCount

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)

	// Dense region: spacing = smallGap.
	for i := 0; i < smallCount; i++ {
		v := uint64(i) * smallGap
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}

	// Sparse region: spacing = hugeGap.
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

	// Query the wide gaps between sparse keys.
	rng := rand.New(rand.NewSource(88888))
	queries := make([][2]uint64, advFPRQueryCount)
	for i := range queries {
		// Pick a random sparse-region index and query its interior gap.
		idx := smallCount + rng.Intn(largeCount-1)
		keyA := lastKey + uint64(idx-smallCount+1)*hugeGap
		keyB := lastKey + uint64(idx-smallCount+2)*hugeGap
		if keyB <= keyA+rangeLen {
			// Gap too small — fall back to uniform query.
			a := rng.Uint64()
			queries[i] = [2]uint64{a, a + rangeLen - 1}
			continue
		}
		// Start somewhere in the interior of this gap.
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
//
// Weak point: normalizeToK extracts K bits starting at spreadStart (first
// significant bit of spread). If spread is small relative to key values, many
// keys collapse to the same truncated value. The trunc filter then sees a
// universe with very few distinct values, so it fires on nearly every query that
// covers those values.
//
// Construction: keys all within a range of 2^K, spread = 2^K-1. All keys are
// distinct but their K-bit suffixes might not be unique if they cluster in a
// sub-range. We choose n=4000 keys within [0, 2^12) with K=12 → exact mode
// fires (M <= K) → no phantom overlap from trunc. But with larger n and small K:
// n * (L+1) / 2^K > 1 → FPR guarantee breaks.
//
// Specifically: use epsilon=0.5 (very loose), but construct so that the actual
// trunc/adaptive suffers more than 5× that value by exhausting the key universe.
// This tests the "degenerate" edge at the boundary of the epsilon formula.
// ---
func TestAdversarialFPR_A7_TruncCollapseSmallSpread(t *testing.T) {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
		// Use a small spread so all keys land in the same region.
		spread = uint64(1) << 16 // 2^16 keyspace
		n      = 10_000          // dense in a tiny space
	)

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)
	rng := rand.New(rand.NewSource(99999))

	// All keys in [base, base+spread).
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

	// Query within the same spread region — many queries are empty since keys
	// are dense but not continuous.
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
//
// Weak point: in SODA (adaptive) mode, when a query spans multiple hash blocks
// (blockA != blockB), the filter checks intermediate full-block ranges and may
// return false (non-empty) spuriously. If many keys happen to hash to the same
// K-bit value as the phantom block range, false positives accumulate.
//
// Construction: keys uniformly distributed in [0, 2^60), forcing SODA mode
// (spread >> K). Queries large enough to span multiple hash blocks. With
// small K (tight epsilon budget), hash collision probability ≈ n/2^K ≈ epsilon,
// so multi-block query FPR should be bounded but we stress it with many queries.
// ---
func TestAdversarialFPR_A8_SODAMultiBlockQuery(t *testing.T) {
	const (
		rangeLen = uint64(1_000_000) // large range → spans many hash blocks
		epsilon  = 0.01
		n        = 20_000
	)

	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)
	rng := rand.New(rand.NewSource(20202))

	// Uniform 60-bit keys → SODA mode likely (spread ≈ 2^60 >> K).
	for len(keys) < n {
		v := rng.Uint64() >> 4 // keep in [0, 2^60)
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
