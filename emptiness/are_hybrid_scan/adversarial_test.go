package are_hybrid_scan

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"Thesis/testutils"
)

const (
	advN          = 100_000
	advQueryCount = 200_000
)

type adversarialResult struct {
	distName string
	fprScan  float64
	bpkScan  float64
	clustScan [3]int
}

func buildScan(t *testing.T, keys []uint64, rangeLen uint64, epsilon float64) *HybridScanARE {
	t.Helper()
	s, err := NewHybridScanARE(keys, 64, Config{RangeLen: float64(rangeLen), Eps: epsilon})
	if err != nil {
		t.Fatalf("HybridScanARE build: %v", err)
	}
	return s
}

func measureScan(t *testing.T, keys []uint64, queries [][2]uint64, s *HybridScanARE) float64 {
	t.Helper()
	return testutils.MeasureFPR(keys, queries, s.IsEmpty)
}

func logResult(t *testing.T, r adversarialResult) {
	t.Helper()
	broken := ""
	if r.fprScan > 0.5 {
		broken = " [BROKEN]"
	}
	t.Logf("%-40s  HybridScanARE: FPR=%.5f BPK=%.1f (cl=%d fb=%d tot=%d)%s",
		r.distName,
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

	s := buildScan(t, keys, rangeLen, epsilon)

	queries := make([][2]uint64, advQueryCount)
	for i := range queries {
		a := uint64(advN) + uint64(i)*rangeLen
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fprS := measureScan(t, keys, queries, s)
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:  "S1: sequential_near_gap",
		fprScan:   fprS,
		bpkScan:   float64(s.SizeInBits()) / float64(st),
		clustScan: [3]int{sc, sf, st},
	}
}

// strategy2: arithmetic progression clusters — gap just above the DBSCAN eps threshold.
func strategy2_ArithmeticClusters(t *testing.T) adversarialResult {
	const (
		rangeLen     = uint64(1000)
		epsilon      = 0.01
		numClusters  = 10
		keysPerClust = advN / numClusters
	)

	dbscanEps := uint64(float64(epsMultiplier) * float64(rangeLen) / epsilon)
	gap := dbscanEps + dbscanEps/10

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

	s := buildScan(t, keys, rangeLen, epsilon)

	queries := make([][2]uint64, 0, advQueryCount)
	qrng := rand.New(rand.NewSource(27182))
	for len(queries) < advQueryCount {
		a := qrng.Uint64()
		queries = append(queries, [2]uint64{a, a + rangeLen - 1})
	}

	fprS := measureScan(t, keys, queries, s)
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:  "S2: arithmetic_clusters_borderline_eps",
		fprScan:   fprS,
		bpkScan:   float64(s.SizeInBits()) / float64(st),
		clustScan: [3]int{sc, sf, st},
	}
}

// strategy3: bimodal — half tight (gap=1), half widely spread.
func strategy3_BimodalSpreadRegion(t *testing.T) adversarialResult {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
		half     = advN / 2
	)

	keys := make([]uint64, 0, advN)
	seen := make(map[uint64]bool, advN)

	for i := 0; i < half; i++ {
		v := uint64(i)
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}

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

	s := buildScan(t, keys, rangeLen, epsilon)

	queries := make([][2]uint64, advQueryCount)
	qrng := rand.New(rand.NewSource(11235))
	for i := range queries {
		a := spreadBase + qrng.Uint64()%spreadRange
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fprS := measureScan(t, keys, queries, s)
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:  "S3: bimodal_spread_region",
		fprScan:   fprS,
		bpkScan:   float64(s.SizeInBits()) / float64(st),
		clustScan: [3]int{sc, sf, st},
	}
}

// strategy4: targeted midpoint queries between consecutive keys.
func strategy4_TargetedMidpoints(t *testing.T) adversarialResult {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
	)

	rng := rand.New(rand.NewSource(57721))
	seen := make(map[uint64]bool, advN)
	keys := make([]uint64, 0, advN)
	for len(keys) < advN {
		v := rng.Uint64() >> 4
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	s := buildScan(t, keys, rangeLen, epsilon)

	queries := make([][2]uint64, 0, advQueryCount)
	for len(queries) < advQueryCount {
		i := len(queries) % (len(keys) - 1)
		lo, hi := keys[i], keys[i+1]
		if hi-lo < 2 {
			queries = append(queries, [2]uint64{lo + 1, lo + 1})
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

	fprS := measureScan(t, keys, queries, s)
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:  "S4: targeted_midpoints_uniform",
		fprScan:   fprS,
		bpkScan:   float64(s.SizeInBits()) / float64(st),
		clustScan: [3]int{sc, sf, st},
	}
}

// strategy5: keys spaced at exactly DBSCAN eps — borderline detection.
func strategy5_ExactDBSCANEps(t *testing.T) adversarialResult {
	const (
		rangeLen = uint64(1000)
		epsilon  = 0.01
	)

	dbscanEpsF := float64(epsMultiplier) * float64(rangeLen) / epsilon
	gap := uint64(dbscanEpsF)

	const base = uint64(1_000_000)
	keys := make([]uint64, advN)
	for i := range keys {
		keys[i] = base + uint64(i)*gap
	}

	s := buildScan(t, keys, rangeLen, epsilon)

	rng := rand.New(rand.NewSource(14142))
	queries := make([][2]uint64, advQueryCount)
	for i := range queries {
		a := rng.Uint64()
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fprS := measureScan(t, keys, queries, s)
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:  "S5: exact_dbscan_eps_boundary",
		fprScan:   fprS,
		bpkScan:   float64(s.SizeInBits()) / float64(st),
		clustScan: [3]int{sc, sf, st},
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

	s := buildScan(t, keys, rangeLen, epsilon)

	qrng := rand.New(rand.NewSource(73205))
	queries := make([][2]uint64, advQueryCount)
	for i := range queries {
		a := qrng.Uint64()
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fprS := measureScan(t, keys, queries, s)
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:  "S6: high_L10000_eps0.001",
		fprScan:   fprS,
		bpkScan:   float64(s.SizeInBits()) / float64(st),
		clustScan: [3]int{sc, sf, st},
	}
}

// strategy7: all keys in [0, 2^20], queries uniform over [0, 2^60].
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

	s := buildScan(t, keys, rangeLen, epsilon)

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

	fprS := measureScan(t, keys, queries, s)
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:  "S7: dense_tiny_range_mixed_queries",
		fprScan:   fprS,
		bpkScan:   float64(s.SizeInBits()) / float64(st),
		clustScan: [3]int{sc, sf, st},
	}
}

// strategy8: sequential keys but queries targeting inter-key gaps directly.
func strategy8_SequentialGap2(t *testing.T) adversarialResult {
	const (
		rangeLen = uint64(100)
		epsilon  = 0.01
	)

	keys := make([]uint64, advN)
	for i := range keys {
		keys[i] = uint64(i) * 2
	}

	s := buildScan(t, keys, rangeLen, epsilon)

	queries := make([][2]uint64, advQueryCount)
	for i := range queries {
		a := uint64(1 + (i%(advN-1))*2)
		b := a + rangeLen - 1
		nextKey := a + 1
		if b >= nextKey {
			b = nextKey - 1
		}
		if b < a {
			b = a
		}
		queries[i] = [2]uint64{a, b}
	}

	fprS := measureScan(t, keys, queries, s)
	sc, sf, st := s.Stats()
	return adversarialResult{
		distName:  "S8: sequential_gap2_odd_queries",
		fprScan:   fprS,
		bpkScan:   float64(s.SizeInBits()) / float64(st),
		clustScan: [3]int{sc, sf, st},
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
	t.Logf("%-40s  %10s  %8s", "Distribution", "FPR Scan", "BPK S")
	t.Log(fmt.Sprintf("%-40s  %10s  %8s", "---", "---", "---"))
	anyBroken := false
	for _, r := range results {
		marker := ""
		if r.fprScan > 0.5 {
			marker = " <<< BROKEN (FPR > 0.5)"
			anyBroken = true
		} else if r.fprScan > 0.1 {
			marker = " *** HIGH FPR"
		}
		t.Logf("%-40s  %10.5f  %8.1f%s",
			r.distName, r.fprScan, r.bpkScan, marker)
	}
	if anyBroken {
		t.Log("BROKEN: at least one filter produced FPR > 0.5 — see rows marked above")
	} else {
		t.Log("No filter broken (FPR <= 0.5 on all strategies)")
	}
}
