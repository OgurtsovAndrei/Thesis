package are_hybrid_scan

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"Thesis/bits"
	are_hybrid "Thesis/emptiness/are_hybrid"
	"Thesis/testutils"
)

const (
	advN          = 100_000
	advQueryCount = 200_000
)

type adversarialResult struct {
	distName    string
	fprHybrid   float64
	fprScan     float64
	bpkHybrid   float64
	bpkScan     float64
	clustHybrid [3]int // numClusters, fallbackKeys, totalKeys
	clustScan   [3]int
}

func buildBoth(t *testing.T, keys []uint64, rangeLen uint64, epsilon float64) (*are_hybrid.HybridARE, *HybridScanARE, []bits.BitString) {
	t.Helper()
	bs := makeSortedBS(keys)
	h, err := are_hybrid.NewHybridARE(bs, rangeLen, epsilon)
	if err != nil {
		t.Fatalf("HybridARE build: %v", err)
	}
	s, err := NewHybridScanARE(bs, rangeLen, epsilon)
	if err != nil {
		t.Fatalf("HybridScanARE build: %v", err)
	}
	return h, s, bs
}

func measureBoth(t *testing.T, keys []uint64, queries [][2]uint64, h *are_hybrid.HybridARE, s *HybridScanARE) (fprH, fprS float64) {
	t.Helper()
	isEmptyH := func(a, b uint64) bool {
		return h.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b))
	}
	isEmptyS := func(a, b uint64) bool {
		return s.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b))
	}
	fprH = testutils.MeasureFPR(keys, queries, isEmptyH)
	fprS = testutils.MeasureFPR(keys, queries, isEmptyS)
	return
}

func logResult(t *testing.T, r adversarialResult) {
	t.Helper()
	broken := ""
	if r.fprHybrid > 0.5 || r.fprScan > 0.5 {
		broken = " [BROKEN]"
	}
	t.Logf("%-40s  HybridARE: FPR=%.5f BPK=%.1f (cl=%d fb=%d tot=%d)  HybridScanARE: FPR=%.5f BPK=%.1f (cl=%d fb=%d tot=%d)%s",
		r.distName,
		r.fprHybrid, r.bpkHybrid,
		r.clustHybrid[0], r.clustHybrid[1], r.clustHybrid[2],
		r.fprScan, r.bpkScan,
		r.clustScan[0], r.clustScan[1], r.clustScan[2],
		broken,
	)
}

// strategy1: sequential keys 0,1,2,...,n-1 with near-key gap queries.
// Worst case for trunc fallback phantom overlap.
func strategy1_SequentialNearGap(t *testing.T) adversarialResult {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
	)

	keys := make([]uint64, advN)
	for i := range keys {
		keys[i] = uint64(i)
	}

	h, s, _ := buildBoth(t, keys, rangeLen, epsilon)

	// Targeted queries: gaps between consecutive keys. Since keys are 0..n-1,
	// any window [a, a+L-1] with a > n-1 is empty. We query just past the last key.
	queries := make([][2]uint64, advQueryCount)
	for i := range queries {
		// Start just after the key array
		a := uint64(advN) + uint64(i)*rangeLen
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fprH, fprS := measureBoth(t, keys, queries, h, s)
	nc, nf, nt := h.Stats()
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:    "S1: sequential_near_gap",
		fprHybrid:   fprH,
		fprScan:     fprS,
		bpkHybrid:   float64(h.SizeInBits()) / float64(nt),
		bpkScan:     float64(s.SizeInBits()) / float64(st),
		clustHybrid: [3]int{nc, nf, nt},
		clustScan:   [3]int{sc, sf, st},
	}
}

// strategy2: arithmetic progression clusters — gap just above the DBSCAN eps threshold.
// Each cluster has keys in arithmetic progression; cluster detector may or may not fire.
func strategy2_ArithmeticClusters(t *testing.T) adversarialResult {
	const (
		rangeLen     = uint64(1000)
		epsilon      = 0.01
		numClusters  = 10
		keysPerClust = advN / numClusters
	)

	// eps threshold for DBSCAN: epsMultiplier * L / epsilon = 10 * 1000 / 0.01 = 1_000_000
	// gap just above this means each pair of consecutive keys is a borderline neighbor.
	dbscanEps := uint64(float64(epsMultiplier) * float64(rangeLen) / epsilon) // ~1_000_000
	gap := dbscanEps + dbscanEps/10                                            // 10% above threshold

	keys := make([]uint64, 0, advN)
	seen := make(map[uint64]bool, advN)
	rng := rand.New(rand.NewSource(31415))
	for c := 0; c < numClusters; c++ {
		center := (rng.Uint64() >> 2) + uint64(c)*gap*uint64(keysPerClust)*2
		for i := 0; i < keysPerClust; i++ {
			v := center + uint64(i)*gap
			if !seen[v] {
				seen[v] = true
				keys = append(keys, v)
			}
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	h, s, _ := buildBoth(t, keys, rangeLen, epsilon)

	// Targeted: query the inter-cluster gaps (midpoints between cluster boundaries).
	queries := make([][2]uint64, 0, advQueryCount)
	qrng := rand.New(rand.NewSource(27182))
	for len(queries) < advQueryCount {
		a := qrng.Uint64()
		queries = append(queries, [2]uint64{a, a + rangeLen - 1})
	}

	fprH, fprS := measureBoth(t, keys, queries, h, s)
	nc, nf, nt := h.Stats()
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:    "S2: arithmetic_clusters_borderline_eps",
		fprHybrid:   fprH,
		fprScan:     fprS,
		bpkHybrid:   float64(h.SizeInBits()) / float64(nt),
		bpkScan:     float64(s.SizeInBits()) / float64(st),
		clustHybrid: [3]int{nc, nf, nt},
		clustScan:   [3]int{sc, sf, st},
	}
}

// strategy3: bimodal — half tight (gap=1), half widely spread.
// Query the spread region specifically to stress trunc fallback.
func strategy3_BimodalSpreadRegion(t *testing.T) adversarialResult {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
		half     = advN / 2
	)

	keys := make([]uint64, 0, advN)
	seen := make(map[uint64]bool, advN)

	// Tight half: keys 0..half-1
	for i := 0; i < half; i++ {
		v := uint64(i)
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}

	// Spread half: uniform random in [2^48, 2^60)
	rng := rand.New(rand.NewSource(161803))
	spreadBase := uint64(1) << 48
	spreadRange := (uint64(1) << 60) - spreadBase
	for len(keys) < advN {
		v := spreadBase + rng.Uint64()%spreadRange
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	h, s, _ := buildBoth(t, keys, rangeLen, epsilon)

	// Query the spread region (avoid tight cluster entirely)
	queries := make([][2]uint64, advQueryCount)
	qrng := rand.New(rand.NewSource(11235))
	for i := range queries {
		a := spreadBase + qrng.Uint64()%spreadRange
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fprH, fprS := measureBoth(t, keys, queries, h, s)
	nc, nf, nt := h.Stats()
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:    "S3: bimodal_spread_region",
		fprHybrid:   fprH,
		fprScan:     fprS,
		bpkHybrid:   float64(h.SizeInBits()) / float64(nt),
		bpkScan:     float64(s.SizeInBits()) / float64(st),
		clustHybrid: [3]int{nc, nf, nt},
		clustScan:   [3]int{sc, sf, st},
	}
}

// strategy4: targeted midpoint queries between consecutive keys.
// For each pair (keys[i], keys[i+1]), query the midpoint [mid, mid+L-1].
// This is the densest possible gap distribution.
func strategy4_TargetedMidpoints(t *testing.T) adversarialResult {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
	)

	rng := rand.New(rand.NewSource(57721))
	seen := make(map[uint64]bool, advN)
	keys := make([]uint64, 0, advN)
	for len(keys) < advN {
		v := rng.Uint64() >> 4 // keep in [0, 2^60)
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	h, s, _ := buildBoth(t, keys, rangeLen, epsilon)

	// Build targeted queries from consecutive midpoints.
	// We cycle through consecutive pairs if needed to reach advQueryCount.
	queries := make([][2]uint64, 0, advQueryCount)
	for len(queries) < advQueryCount {
		i := len(queries) % (len(keys) - 1)
		lo, hi := keys[i], keys[i+1]
		if hi-lo < 2 {
			// No gap between consecutive keys, skip
			queries = append(queries, [2]uint64{lo + 1, lo + 1}) // degenerate, will be filtered
			continue
		}
		mid := lo + (hi-lo)/2
		a := mid
		b := a + rangeLen - 1
		if b > hi-1 {
			b = hi - 1
		}
		if b < a {
			queries = append(queries, [2]uint64{a, a})
		} else {
			queries = append(queries, [2]uint64{a, b})
		}
	}

	fprH, fprS := measureBoth(t, keys, queries, h, s)
	nc, nf, nt := h.Stats()
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:    "S4: targeted_midpoints_uniform",
		fprHybrid:   fprH,
		fprScan:     fprS,
		bpkHybrid:   float64(h.SizeInBits()) / float64(nt),
		bpkScan:     float64(s.SizeInBits()) / float64(st),
		clustHybrid: [3]int{nc, nf, nt},
		clustScan:   [3]int{sc, sf, st},
	}
}

// strategy5: keys spaced at exactly DBSCAN eps — borderline detection.
func strategy5_ExactDBSCANEps(t *testing.T) adversarialResult {
	const (
		rangeLen = uint64(1000)
		epsilon  = 0.01
	)

	// DBSCAN eps = epsMultiplier * rangeLen / epsilon
	dbscanEpsF := float64(epsMultiplier) * float64(rangeLen) / epsilon
	gap := uint64(dbscanEpsF) // exactly at eps boundary

	const base = uint64(1_000_000)
	keys := make([]uint64, advN)
	for i := range keys {
		keys[i] = base + uint64(i)*gap
	}

	h, s, _ := buildBoth(t, keys, rangeLen, epsilon)

	rng := rand.New(rand.NewSource(14142))
	queries := make([][2]uint64, advQueryCount)
	for i := range queries {
		a := rng.Uint64()
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fprH, fprS := measureBoth(t, keys, queries, h, s)
	nc, nf, nt := h.Stats()
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:    "S5: exact_dbscan_eps_boundary",
		fprHybrid:   fprH,
		fprScan:     fprS,
		bpkHybrid:   float64(h.SizeInBits()) / float64(nt),
		bpkScan:     float64(s.SizeInBits()) / float64(st),
		clustHybrid: [3]int{nc, nf, nt},
		clustScan:   [3]int{sc, sf, st},
	}
}

// strategy6: extreme parameters — L=10000, eps=0.001.
func strategy6_HighLLowEps(t *testing.T) adversarialResult {
	const (
		rangeLen = uint64(10_000)
		epsilon  = 0.001
	)

	rng := rand.New(rand.NewSource(22360))
	seen := make(map[uint64]bool, advN)
	keys := make([]uint64, 0, advN)
	for len(keys) < advN {
		v := rng.Uint64()
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	h, s, _ := buildBoth(t, keys, rangeLen, epsilon)

	qrng := rand.New(rand.NewSource(73205))
	queries := make([][2]uint64, advQueryCount)
	for i := range queries {
		a := qrng.Uint64()
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fprH, fprS := measureBoth(t, keys, queries, h, s)
	nc, nf, nt := h.Stats()
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:    "S6: high_L10000_eps0.001",
		fprHybrid:   fprH,
		fprScan:     fprS,
		bpkHybrid:   float64(h.SizeInBits()) / float64(nt),
		bpkScan:     float64(s.SizeInBits()) / float64(st),
		clustHybrid: [3]int{nc, nf, nt},
		clustScan:   [3]int{sc, sf, st},
	}
}

// strategy7: all keys in [0, 2^20], queries uniform over [0, 2^60].
// Very dense data in a tiny range; most queries miss it, but the ones that
// hit the dense range exercise the filters at maximum density.
func strategy7_DenseTinyRange(t *testing.T) adversarialResult {
	const (
		rangeLen  = uint64(100)
		epsilon   = 0.01
		keySpace  = uint64(1 << 20)
		queryHigh = uint64(1) << 60
	)

	seen := make(map[uint64]bool, advN)
	keys := make([]uint64, 0, advN)
	rng := rand.New(rand.NewSource(31622))
	for len(keys) < advN {
		v := rng.Uint64() % keySpace
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	h, s, _ := buildBoth(t, keys, rangeLen, epsilon)

	// Mixed: half queries in the dense range, half uniform to stress both paths.
	qrng := rand.New(rand.NewSource(41421))
	queries := make([][2]uint64, advQueryCount)
	for i := range queries {
		var a uint64
		if i%2 == 0 {
			a = qrng.Uint64() % keySpace
		} else {
			a = qrng.Uint64() % queryHigh
		}
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fprH, fprS := measureBoth(t, keys, queries, h, s)
	nc, nf, nt := h.Stats()
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:    "S7: dense_tiny_range_mixed_queries",
		fprHybrid:   fprH,
		fprScan:     fprS,
		bpkHybrid:   float64(h.SizeInBits()) / float64(nt),
		bpkScan:     float64(s.SizeInBits()) / float64(st),
		clustHybrid: [3]int{nc, nf, nt},
		clustScan:   [3]int{sc, sf, st},
	}
}

// strategy8: sequential keys but queries targeting inter-key gaps directly.
// For sequential keys 0,1,...,n-1 with gap=1 there are no gaps — use gap=2
// so every odd number is a gap. Query [odd, odd+L-1].
func strategy8_SequentialGap2(t *testing.T) adversarialResult {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
	)

	keys := make([]uint64, advN)
	for i := range keys {
		keys[i] = uint64(i) * 2 // 0, 2, 4, ..., 2*(n-1)
	}

	h, s, _ := buildBoth(t, keys, rangeLen, epsilon)

	// Queries starting at odd positions — always a gap at the start.
	queries := make([][2]uint64, advQueryCount)
	for i := range queries {
		// Odd start within the key range
		a := uint64(1 + (i%(advN-1))*2)
		b := a + rangeLen - 1
		// Cap b to just before the next even key
		nextKey := a + 1 // next even is a+1
		if b >= nextKey {
			b = nextKey - 1
		}
		if b < a {
			b = a
		}
		queries[i] = [2]uint64{a, b}
	}

	fprH, fprS := measureBoth(t, keys, queries, h, s)
	nc, nf, nt := h.Stats()
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:    "S8: sequential_gap2_odd_queries",
		fprHybrid:   fprH,
		fprScan:     fprS,
		bpkHybrid:   float64(h.SizeInBits()) / float64(nt),
		bpkScan:     float64(s.SizeInBits()) / float64(st),
		clustHybrid: [3]int{nc, nf, nt},
		clustScan:   [3]int{sc, sf, st},
	}
}

func TestAdversarial(t *testing.T) {
	type strategyFn func(t *testing.T) adversarialResult

	strategies := []struct {
		name string
		fn   strategyFn
	}{
		{"S1/sequential_near_gap", strategy1_SequentialNearGap},
		{"S2/arithmetic_clusters_borderline_eps", strategy2_ArithmeticClusters},
		{"S3/bimodal_spread_region", strategy3_BimodalSpreadRegion},
		{"S4/targeted_midpoints_uniform", strategy4_TargetedMidpoints},
		{"S5/exact_dbscan_eps_boundary", strategy5_ExactDBSCANEps},
		{"S6/high_L_low_eps", strategy6_HighLLowEps},
		{"S7/dense_tiny_range", strategy7_DenseTinyRange},
		{"S8/sequential_gap2_odd_queries", strategy8_SequentialGap2},
	}

	results := make([]adversarialResult, 0, len(strategies))

	for _, tc := range strategies {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r := tc.fn(t)
			results = append(results, r)
			logResult(t, r)
		})
	}

	t.Log("")
	t.Log("=== ADVERSARIAL SUMMARY ===")
	t.Logf("%-40s  %10s  %10s  %8s  %8s", "Distribution", "FPR Hybrid", "FPR Scan", "BPK H", "BPK S")
	t.Log(fmt.Sprintf("%-40s  %10s  %10s  %8s  %8s", "---", "---", "---", "---", "---"))
	anyBroken := false
	for _, r := range results {
		marker := ""
		if r.fprHybrid > 0.5 || r.fprScan > 0.5 {
			marker = " <<< BROKEN (FPR > 0.5)"
			anyBroken = true
		} else if r.fprHybrid > 0.1 || r.fprScan > 0.1 {
			marker = " *** HIGH FPR"
		}
		t.Logf("%-40s  %10.5f  %10.5f  %8.1f  %8.1f%s",
			r.distName, r.fprHybrid, r.fprScan, r.bpkHybrid, r.bpkScan, marker)
	}
	if anyBroken {
		t.Log("BROKEN: at least one filter produced FPR > 0.5 — see rows marked above")
	} else {
		t.Log("No filter broken (FPR <= 0.5 on all strategies)")
	}
}
