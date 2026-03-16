package are_hybrid_scan

import (
	"fmt"
	"math"
	mbits "math/bits"
	"math/rand"
	"sort"
	"testing"

	"Thesis/bits"
	are_trunc "Thesis/emptiness/are_trunc"
	"Thesis/testutils"
)

// trieFirstSetBitDiag mimics the unexported trieFirstSetBit for diagnostic use.
func trieFirstSetBitDiag(bs bits.BitString) uint32 {
	W := bs.SizeBits()
	numWords := (W + 63) / 64
	for i := uint32(0); i < numWords; i++ {
		w := bs.Word(i)
		if w != 0 {
			return i*64 + uint32(mbits.TrailingZeros64(w))
		}
	}
	return W
}

// TestS5Diagnostic investigates why S5 (equidistant keys at gap = DBSCAN eps) produces FPR ~73%.
//
// S5 parameters: rangeLen=1000, epsilon=0.01, N=100,000
//
//	gap = epsMultiplier * rangeLen / epsilon = 10 * 1000 / 0.01 = 1,000,000
//	keys[i] = base + i * gap
func TestS5Diagnostic(t *testing.T) {
	const (
		rangeLen    = uint64(1000)
		epsilon     = 0.01
		N           = 100_000
		queryCount  = 200_000
		base        = uint64(1_000_000)
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

	bs := makeSortedBS(keys)
	clusters, fallback := detectClustersDBSCAN(bs, dbscanEps, dbscanMinPts, minClusterSize)
	t.Logf("Result: %d clusters  %d fallback keys", len(clusters), len(fallback))

	// truncSafe verdict.
	isSafe := truncSafe(keys, K)
	t.Log("\n=== truncSafe CHECK ===")
	t.Logf("truncSafe(keys, K=%d) = %v  (expected true because gap=%d >> phantom_size=%d)", K, isSafe, gap, phantomSize)

	// Build the trunc filter using the same fallback keys.
	truncFilter, err := are_trunc.NewApproximateRangeEmptinessFromK(fallback, K)
	if err != nil {
		t.Fatalf("trunc build: %v", err)
	}

	// Expose normalization internals.
	minBS := trieBS(minKey)
	maxBS := trieBS(maxKey)
	spreadBS := maxBS.Sub(minBS)
	spreadStart := trieFirstSetBitDiag(spreadBS)

	t.Log("\n=== NORMALIZATION INTERNALS ===")
	t.Logf("spreadStart (storage bit index of first set bit in spread) = %d", spreadStart)
	t.Logf("normalizeToK extracts bits [%d, %d) (K=%d bits) of (key - minKey)", spreadStart, spreadStart+K, K)

	truncMaxKey := spreadBS.BitRange(spreadStart, K).TrieUint64()
	t.Logf("truncated maxKey = %d  (should be ~2^K-1 = %d)", truncMaxKey, uint64(1)<<K-1)

	// Show what normalizeToK produces for out-of-range queries.
	t.Log("\n=== OUT-OF-RANGE KEY NORMALIZATION ===")
	t.Logf("%-15s  %-22s  %-12s  %-12s  %-20s", "delta", "a = maxKey+delta", "offset", "truncA", "truncA < truncMaxKey?")
	for _, delta := range []uint64{1, gap / 4, gap / 2, gap, 2 * gap, 10 * gap, 1 << 20, 1 << 27, 1 << 30} {
		a := maxKey + delta
		aBS := trieBS(a)
		off := aBS.Sub(minBS)
		tA := off.BitRange(spreadStart, K).TrieUint64()
		t.Logf("%-15d  %-22d  %-12d  %-12d  %v", delta, a, a-minKey, tA, tA < truncMaxKey)
	}

	// Show what trunc.IsEmpty does for an out-of-range query.
	// According to trunc.IsEmpty: if b < minKey -> return true (correct).
	// If a > maxKey: b is also > maxKey (since b = a + L - 1 >= a > maxKey),
	// so truncB is clamped to truncMaxKey. But truncA = normalizeToK(a) can wrap.
	t.Log("\n=== IsEmpty BEHAVIOR FOR a > maxKey ===")
	t.Log("When a > maxKey and b > maxKey:")
	t.Log("  b.Compare(minKey) >= 0 => does NOT return true early")
	t.Log("  truncB = normalizeToK(maxKey) = truncMaxKey (clamped)")
	t.Log("  truncA = normalizeToK(a) — may wrap within K bits")
	t.Log("  If truncA wraps to a small value: IsEmpty([small, truncMaxKey]) covers huge range => FP")

	// Demonstrate with a concrete value.
	aDemo := maxKey + gap // one gap past the end
	aDemoBS := trieBS(aDemo)
	tADemo := aDemoBS.Sub(minBS).BitRange(spreadStart, K).TrieUint64()
	bDemo := aDemo + rangeLen - 1
	bDemoBS := trieBS(bDemo)
	var tBDemo uint64
	if bDemoBS.Compare(maxBS) > 0 {
		tBDemo = truncMaxKey
	} else {
		tBDemo = bDemoBS.Sub(minBS).BitRange(spreadStart, K).TrieUint64()
	}
	isFP := !truncFilter.IsEmpty(testutils.TrieBS(aDemo), testutils.TrieBS(bDemo))
	t.Logf("\nConcrete demo: a = maxKey + gap = %d", aDemo)
	t.Logf("  truncA=%d  truncB=%d (clamped to truncMaxKey)", tADemo, tBDemo)
	t.Logf("  truncA < truncB? %v  =>  trunc.IsEmpty = %v  (FP: %v)", tADemo < tBDemo, !isFP, isFP)

	// Measure FPR with the same queries as strategy5.
	rng := rand.New(rand.NewSource(14142))
	queries := make([][2]uint64, queryCount)
	for i := range queries {
		a := rng.Uint64()
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	isEmptyTrunc := func(a, b uint64) bool {
		return truncFilter.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b))
	}
	fprTrunc := testutils.MeasureFPR(keys, queries, isEmptyTrunc)
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
		if !isEmptyTrunc(a, b) {
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

	// Print a few out-of-range FP examples with full normalization trace.
	t.Log("\n=== SAMPLE OUT-OF-RANGE FALSE POSITIVES ===")
	header := fmt.Sprintf("%-22s %-22s %-8s %-8s %-8s %-8s", "a", "b", "side", "truncA", "truncB", "t<tMax?")
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
		if !isEmptyTrunc(a, b) && (b < minKey || a > maxKey) {
			aBS2 := trieBS(a)
			bBS2 := trieBS(b)
			var tA2, tB2 uint64
			if aBS2.Compare(minBS) < 0 {
				tA2 = 0
			} else {
				tA2 = aBS2.Sub(minBS).BitRange(spreadStart, K).TrieUint64()
			}
			if bBS2.Compare(maxBS) > 0 {
				tB2 = truncMaxKey
			} else {
				tB2 = bBS2.Sub(minBS).BitRange(spreadStart, K).TrieUint64()
			}
			side := "a>max"
			if b < minKey {
				side = "b<min"
			}
			t.Logf("%-22d %-22d %-8s %-8d %-8d %v", a, b, side, tA2, tB2, tA2 < tB2)
			printed++
		}
	}

	t.Log("\n=== ROOT CAUSE SUMMARY ===")
	switch {
	case fp.outRange > fp.inRange:
		t.Logf("PRIMARY CAUSE: out-of-range queries dominate FP (%d/%d)", fp.outRange, fp.total)
		t.Logf("When a > maxKey, trunc.IsEmpty does NOT return early.")
		t.Logf("  truncB is clamped to truncMaxKey (~2^K - 1)")
		t.Logf("  truncA = (a - minKey).BitRange(%d, %d) wraps within K=%d bits", spreadStart, K, K)
		t.Logf("  Whenever truncA < truncMaxKey the ERE sees [truncA, truncMaxKey] = broad range => FP")
	case fp.inRange > fp.outRange:
		t.Logf("PRIMARY CAUSE: in-range queries dominate FP (%d/%d) — phantom overlap in the key span", fp.inRange, fp.total)
	default:
		t.Logf("FP split evenly: in-range=%d out-of-range=%d", fp.inRange, fp.outRange)
	}
}
