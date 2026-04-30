package are_hybrid_scan

import (
	"Thesis/emptiness/approx/are_trunc"
	"fmt"
	"math"
	mbits "math/bits"
	"math/rand"
	"sort"
	"testing"

	"Thesis/testutils"
)

// TestS5Diagnostic investigates why S5 (equidistant keys at gap = DBSCAN eps) produces FPR ~73%.
//
// S5 parameters: rangeLen=1000, epsilon=0.01, N=100,000
//
//	gap = epsMultiplier * rangeLen / epsilon = 10 * 1000 / 0.01 = 1,000,000
//	keys[i] = base + i * gap
func TestS5Diagnostic(t *testing.T) {
	const (
		rangeLen   = uint64(1000)
		epsilon    = 0.01
		N          = 100_000
		queryCount = 200_000
		base       = uint64(1_000_000)
	)

	dbscanEpsF := float64(epsMultiplier) * float64(rangeLen) / epsilon
	gap := uint64(dbscanEpsF)

	keys := make([]uint64, N)
	for i := range keys {
		keys[i] = base + uint64(i)*gap
	}

	minKey := keys[0]
	maxKey := keys[N-1]
	spread := maxKey - minKey

	// Compute K exactly as NewHybridScanARE does.
	effectiveRangeLen := rangeLen + 1
	rTarget := float64(N) * float64(effectiveRangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		K = 64
	}

	spreadBits := uint32(64 - mbits.LeadingZeros64(spread))
	var phantomSize uint64
	if spreadBits > K {
		phantomSize = spread >> K
		if phantomSize == 0 {
			phantomSize = 1
		}
	}

	t.Log("=== S5 PARAMETERS ===")
	t.Logf("N=%d  rangeLen=%d  epsilon=%.4f", N, rangeLen, epsilon)
	t.Logf("gap=%d  base=%d", gap, base)
	t.Logf("minKey=%d  maxKey=%d  spread=%d (2^%.2f)", minKey, maxKey, spread, math.Log2(float64(spread)))
	t.Logf("rTarget = N*(L+1)/eps = %.4e  => K = ceil(log2(rTarget)) = %d", rTarget, K)
	t.Logf("spreadBits=%d  K=%d  t=spreadBits-K=%d", spreadBits, K, int(spreadBits)-int(K))
	t.Logf("phantom_size = spread >> K = %d", phantomSize)
	t.Logf("gap=%d  phantom_size=%d  gap>phantom: %v", gap, phantomSize, gap > phantomSize)

	// DBSCAN: check whether any keys form clusters.
	dbscanEps := uint64(float64(rangeLen) / epsilon * float64(epsMultiplier))
	t.Log("\n=== DBSCAN CLUSTER DETECTION ===")
	t.Logf("DBSCAN eps=%d  key gap=%d  equal: %v", dbscanEps, gap, gap == dbscanEps)
	t.Logf("With gap==eps each eps-window contains exactly 2 consecutive keys => only 1 neighbor => no core points (need %d)", dbscanMinPts)

	clusters, fallback := detectClustersDBSCAN(keys, dbscanEps, dbscanMinPts, minClusterSize)
	t.Logf("Result: %d clusters  %d fallback keys", len(clusters), len(fallback))

	// truncSafe verdict.
	isSafe := truncSafe(keys, K)
	t.Log("\n=== truncSafe CHECK ===")
	t.Logf("truncSafe(keys, K=%d) = %v  (expected true because gap=%d >> phantom_size=%d)", K, isSafe, gap, phantomSize)

	// Build the trunc filter using the same fallback keys.
	truncFilter, err := are_trunc.NewTruncAREFromK(fallback, 64, K)
	if err != nil {
		t.Fatalf("trunc build: %v", err)
	}

	// Normalization internals (uint64 arithmetic).
	// spreadStart = position of first set bit in spread (= bit length - 1, 0-indexed from LSB)
	spreadStart := uint32(mbits.Len64(spread)) - 1
	if spread == 0 {
		spreadStart = 0
	}

	t.Log("\n=== NORMALIZATION INTERNALS ===")
	t.Logf("spread=0x%x  spreadStart=%d  (first significant bit position)", spread, spreadStart)
	t.Logf("normalizeToK: extract K=%d bits from (key - minKey) >> spreadStart", K)

	// Show out-of-range key normalization.
	t.Log("\n=== OUT-OF-RANGE KEY NORMALIZATION ===")
	t.Logf("%-15s  %-22s  %-12s", "delta", "a = maxKey+delta", "offset")
	for _, delta := range []uint64{1, gap / 4, gap / 2, gap, 2 * gap, 10 * gap, 1 << 20, 1 << 27, 1 << 30} {
		a := maxKey + delta
		offset := a - minKey
		t.Logf("%-15d  %-22d  %-12d", delta, a, offset)
	}

	// Measure FPR with the same queries as strategy5.
	rng := rand.New(rand.NewSource(14142))
	queries := make([][2]uint64, queryCount)
	for i := range queries {
		a := rng.Uint64()
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	fprTrunc := testutils.MeasureFPR(keys, queries, truncFilter.IsEmpty)
	t.Log("\n=== FPR MEASUREMENT ===")
	t.Logf("Trunc FPR = %.5f  (target epsilon = %.4f, expected ~%.4f)", fprTrunc, epsilon, epsilon)

	// Categorize false positives: inside vs. outside key span.
	type fpClass struct{ inRange, outRange, total int }
	var fp, emptyQ fpClass
	for _, q := range queries {
		a, b := q[0], q[1]
		idx := sort.Search(len(keys), func(j int) bool { return keys[j] >= a })
		if idx < len(keys) && keys[idx] <= b {
			continue // not empty
		}
		emptyQ.total++
		outside := b < minKey || a > maxKey
		if outside {
			emptyQ.outRange++
		} else {
			emptyQ.inRange++
		}
		if !truncFilter.IsEmpty(a, b) {
			fp.total++
			if outside {
				fp.outRange++
			} else {
				fp.inRange++
			}
		}
	}

	t.Log("\n=== FALSE POSITIVE BREAKDOWN ===")
	t.Logf("Total empty queries: %d  (in-range: %d, out-of-range: %d)", emptyQ.total, emptyQ.inRange, emptyQ.outRange)
	t.Logf("Total FP:            %d  (in-range: %d, out-of-range: %d)", fp.total, fp.inRange, fp.outRange)
	if emptyQ.inRange > 0 {
		t.Logf("FPR in-range:     %.5f", float64(fp.inRange)/float64(emptyQ.inRange))
	}
	if emptyQ.outRange > 0 {
		t.Logf("FPR out-of-range: %.5f", float64(fp.outRange)/float64(emptyQ.outRange))
	}

	// Print a few out-of-range FP examples.
	t.Log("\n=== SAMPLE OUT-OF-RANGE FALSE POSITIVES ===")
	header := fmt.Sprintf("%-22s %-22s %-8s", "a", "b", "side")
	t.Log(header)
	printed := 0
	for _, q := range queries {
		if printed >= 12 {
			break
		}
		a, b := q[0], q[1]
		idx := sort.Search(len(keys), func(j int) bool { return keys[j] >= a })
		if idx < len(keys) && keys[idx] <= b {
			continue
		}
		if !truncFilter.IsEmpty(a, b) && (b < minKey || a > maxKey) {
			side := "a>max"
			if b < minKey {
				side = "b<min"
			}
			t.Logf("%-22d %-22d %-8s", a, b, side)
			printed++
		}
	}

	t.Log("\n=== ROOT CAUSE SUMMARY ===")
	switch {
	case fp.outRange > fp.inRange:
		t.Logf("PRIMARY CAUSE: out-of-range queries dominate FP (%d/%d)", fp.outRange, fp.total)
		t.Logf("When a > maxKey, trunc.IsEmpty does NOT return early.")
		t.Logf("  Offset (a - minKey) wraps within K=%d bits → phantom overlap beyond key span", K)
	case fp.inRange > fp.outRange:
		t.Logf("PRIMARY CAUSE: in-range queries dominate FP (%d/%d) — phantom overlap in the key span", fp.inRange, fp.total)
	default:
		t.Logf("FP split evenly: in-range=%d out-of-range=%d", fp.inRange, fp.outRange)
	}
}
