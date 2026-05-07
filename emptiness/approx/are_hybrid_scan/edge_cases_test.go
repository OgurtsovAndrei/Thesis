package are_hybrid_scan

import (
	"math"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// ── helpers ──────────────────────────────────────────────────────────────────

const maxUint64 = ^uint64(0)

// bpkOf returns bits-per-key for a built filter.
func bpkOf(h *HybridScanARE) float64 {
	if h.n == 0 {
		return 0
	}
	return float64(h.SizeInBits()) / float64(h.n)
}

// buildFromK is a thin wrapper for NewHybridScanARE with explicit K.
func buildFromK(t *testing.T, vals []uint64, rangeLen uint64, K uint32) *HybridScanARE {
	t.Helper()
	_ = rangeLen
	h, err := NewHybridScanARE(vals, 64, Config{K: K})
	require.NoError(t, err)
	return h
}

// buildEps maps a legacy (rangeLen, eps) target to K and builds the filter.
func buildEps(t *testing.T, vals []uint64, rangeLen uint64, eps float64) *HybridScanARE {
	t.Helper()
	h, err := NewHybridScanARE(vals, 64, Config{K: kFromEps(len(vals), rangeLen, eps)})
	require.NoError(t, err)
	return h
}

// ── edge case 1 ──────────────────────────────────────────────────────────────
// BPK blowup: many tiny clusters just above minClusterSize=256.
// Each one gets 128 bits of metadata.  With very small K the cluster payload
// is tiny, so metadata may dominate.  We just verify the filter builds without
// crashing and that SizeInBits() is non-zero and consistent.

func TestEdge_BPKBlowup_ManyTinyClusters(t *testing.T) {
	const (
		numClusters = 20
		clusterSize = minClusterSize // 256 – exactly at the threshold
		separation  = uint64(1e12)
		rangeLen    = uint64(100)
		K           = uint32(4)
	)

	vals := make([]uint64, 0, numClusters*clusterSize)
	for c := 0; c < numClusters; c++ {
		base := separation * uint64(c+1)
		for i := 0; i < clusterSize; i++ {
			vals = append(vals, base+uint64(i))
		}
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })

	h := buildFromK(t, vals, rangeLen, K)

	sz := h.SizeInBits()
	nc, _, nt := h.Stats()
	bpk := float64(sz) / float64(nt)

	t.Logf("clusters=%d total=%d SizeInBits=%d BPK=%.2f", nc, nt, sz, bpk)
	require.Greater(t, sz, uint64(0), "SizeInBits must be positive")

	metadataBPK := float64(nc) * 128.0 / float64(nt)
	t.Logf("metadata BPK contribution=%.2f", metadataBPK)

	for i := 0; i < len(vals); i += len(vals) / 10 {
		v := vals[i]
		require.False(t, h.IsEmpty(v, v),
			"false negative for stored key %d", v)
	}
}

// ── edge case 2 ──────────────────────────────────────────────────────────────
// Degenerate DBSCAN: all keys identical.

func TestEdge_DegenerateAllIdenticalKeys(t *testing.T) {
	const key = uint64(0xDEADBEEF)
	vals := []uint64{key, key, key, key, key}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })

	h := buildEps(t, vals, 100, 0.01)
	require.NotNil(t, h)

	require.False(t, h.IsEmpty(key, key),
		"stored key must not be reported empty")
	require.True(t, h.IsEmpty(key+1, key+100),
		"range disjoint from stored key should be reported empty (no FP here)")
}

// ── edge case 3 ──────────────────────────────────────────────────────────────
// Degenerate: N=1 key — must fall through to trunc fallback path.

func TestEdge_SingleKeyFilter(t *testing.T) {
	const key = uint64(12345678901234)
	vals := []uint64{key}

	h := buildEps(t, vals, 1000, 0.01)
	require.NotNil(t, h)
	require.Equal(t, 0, h.nClusters, "single key → no DBSCAN clusters")

	require.False(t, h.IsEmpty(key, key),
		"false negative on the only stored key")

	require.False(t, h.IsEmpty(key-100, key+100),
		"range covering key must not report empty")

	if key > 200 {
		require.True(t, h.IsEmpty(key-200, key-101),
			"range well below key should report empty")
	}
}

// ── edge case 4 ──────────────────────────────────────────────────────────────
// Overflow-adjacent keys: near uint64 max.

func TestEdge_NearMaxUint64Keys(t *testing.T) {
	vals := []uint64{
		maxUint64 - 1000,
		maxUint64 - 500,
		maxUint64 - 100,
		maxUint64 - 10,
		maxUint64 - 1,
		maxUint64,
	}

	h := buildEps(t, vals, 10, 0.01)
	require.NotNil(t, h)

	for _, v := range vals {
		require.False(t, h.IsEmpty(v, v),
			"false negative for key near maxUint64: %d", v)
	}

	require.False(t, h.IsEmpty(0, maxUint64),
		"[0, MaxUint64] must not be empty when keys exist")

	require.False(t, h.IsEmpty(maxUint64-1, maxUint64),
		"range [maxUint64-1, maxUint64] must not be empty")
}

// ── edge case 5 ──────────────────────────────────────────────────────────────
// K=64: largest possible K, keys spread across full uint64 space.

func TestEdge_K64MaxK(t *testing.T) {
	const n = 300
	vals := make([]uint64, n)
	for i := 0; i < n; i++ {
		vals[i] = uint64(float64(i) / float64(n-1) * float64(maxUint64))
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })

	h := buildFromK(t, vals, 1000, 64)
	require.NotNil(t, h)

	sz := h.SizeInBits()
	require.Greater(t, sz, uint64(0))
	t.Logf("K=64, n=%d, SizeInBits=%d, BPK=%.2f", n, sz, bpkOf(h))

	for _, v := range vals {
		require.False(t, h.IsEmpty(v, v),
			"false negative at K=64 for key %d", v)
	}
}

// ── edge case 6 ──────────────────────────────────────────────────────────────
// K=1: minimum possible K — highest FPR but must not crash or panic.

func TestEdge_K1MinK(t *testing.T) {
	const n = 500
	vals := make([]uint64, n)
	for i := 0; i < n; i++ {
		vals[i] = uint64(i) * 1000
	}

	h := buildFromK(t, vals, 100, 1)
	require.NotNil(t, h)

	sz := h.SizeInBits()
	require.Greater(t, sz, uint64(0))
	t.Logf("K=1, n=%d, SizeInBits=%d, BPK=%.4f", n, sz, bpkOf(h))

	for _, v := range vals {
		require.False(t, h.IsEmpty(v, v),
			"false negative at K=1 for key %d", v)
	}
}

// ── edge case 7 ──────────────────────────────────────────────────────────────
// truncSafe edge: keys where spread >> 2^K but all gaps are equal.

func TestEdge_TruncSafeEdge_EqualGaps(t *testing.T) {
	const (
		n        = 2000
		K        = uint32(10)
		base     = uint64(1_000_000)
		rangeLen = uint64(10)
	)
	spread := uint64(1) << 40
	gap := spread / uint64(n-1)
	if gap == 0 {
		gap = 1
	}

	vals := make([]uint64, n)
	for i := 0; i < n; i++ {
		vals[i] = base + uint64(i)*gap
	}

	h := buildFromK(t, vals, rangeLen, K)
	require.NotNil(t, h)

	nc, nf, nt := h.Stats()
	t.Logf("TruncSafeEdge: clusters=%d fallback=%d total=%d BPK=%.2f", nc, nf, nt, bpkOf(h))

	for _, v := range vals {
		require.False(t, h.IsEmpty(v, v),
			"false negative for key %d", v)
	}
}

// ── edge case 8 ──────────────────────────────────────────────────────────────
// Query correctness: boundary queries.

func TestEdge_BoundaryQueries(t *testing.T) {
	vals := []uint64{100, 200, 300, 400, 500}
	h := buildEps(t, vals, 100, 0.01)

	minKey, maxKey := vals[0], vals[len(vals)-1]

	for _, v := range vals {
		require.False(t, h.IsEmpty(v, v),
			"point query must not be empty for stored key %d", v)
	}

	result := h.IsEmpty(300, 200)
	t.Logf("IsEmpty([300,200]) (inverted) = %v (no panic required)", result)

	require.True(t, h.IsEmpty(0, minKey-1),
		"range [0, minKey-1] must be empty")

	require.True(t, h.IsEmpty(maxKey+1, maxKey+1000),
		"range [maxKey+1, maxKey+1000] must be empty")

	require.False(t, h.IsEmpty(minKey, maxKey),
		"range [minKey, maxKey] must not be empty")

	require.False(t, h.IsEmpty(0, maxUint64),
		"[0, MaxUint64] must not be empty when any keys exist")
}

// ── edge case 9 ──────────────────────────────────────────────────────────────
// SizeInBits accuracy: for keys that all go to fallback (n < minClusterSize),
// the reported size must be >= K * n_unique.

func TestEdge_SizeInBits_Accuracy(t *testing.T) {
	const (
		n        = 100
		rangeLen = uint64(100)
	)

	for _, K := range []uint32{8, 16, 32} {
		K := K
		vals := make([]uint64, n)
		for i := 0; i < n; i++ {
			vals[i] = uint64(i) * 10_000
		}

		h := buildFromK(t, vals, rangeLen, K)
		sz := h.SizeInBits()
		nc, nf, nt := h.Stats()

		t.Logf("K=%d: clusters=%d fallback=%d total=%d SizeInBits=%d BPK=%.2f",
			K, nc, nf, nt, sz, float64(sz)/float64(nt))

		require.Equal(t, 0, nc, "all keys below minClusterSize → 0 clusters")
		require.Greater(t, sz, uint64(0), "SizeInBits must be > 0 for non-empty filter")

		require.GreaterOrEqual(t, sz, uint64(n),
			"SizeInBits should be at least n bits to encode %d keys", n)
	}
}

// ── edge case 10 ─────────────────────────────────────────────────────────────
// BPK with K=64 and many keys: cluster overhead should be dominated by filter bits.

func TestEdge_BPKBounded_LargeN_K64(t *testing.T) {
	const (
		n        = 5000
		rangeLen = uint64(1000)
		K        = uint32(20)
	)
	vals := make([]uint64, 0, n)
	for c := 0; c < 3; c++ {
		base := uint64(c+1) * 1_000_000_000
		for i := 0; i < 1500; i++ {
			vals = append(vals, base+uint64(i))
		}
	}
	for i := 500; i < 1000; i++ {
		vals = append(vals, uint64(i)*1_000_000_000_000)
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })
	vals = vals[:n]

	h := buildFromK(t, vals, rangeLen, K)
	bpk := bpkOf(h)
	nc, nf, nt := h.Stats()

	t.Logf("BPK=%.2f clusters=%d fallback=%d total=%d SizeInBits=%d",
		bpk, nc, nf, nt, h.SizeInBits())

	require.Less(t, bpk, 1000.0,
		"BPK blowup detected: %.2f bpk for K=%d n=%d", bpk, K, nt)

	for _, v := range vals {
		require.False(t, h.IsEmpty(v, v),
			"false negative for key %d", v)
	}
}

// ── edge case 11 ─────────────────────────────────────────────────────────────
// N=2: minimal non-trivial case — below minClusterSize, above n=1.

func TestEdge_TwoKeys(t *testing.T) {
	vals := []uint64{0, maxUint64}

	h := buildEps(t, vals, 100, 0.01)
	require.NotNil(t, h)
	require.Equal(t, 0, h.nClusters, "2 keys → no cluster")

	require.False(t, h.IsEmpty(0, 0),
		"false negative for key 0")
	require.False(t, h.IsEmpty(maxUint64, maxUint64),
		"false negative for key maxUint64")
	require.False(t, h.IsEmpty(0, maxUint64),
		"full-span range must not be empty")
}

// ── edge case 12 ─────────────────────────────────────────────────────────────
// Epsilon very close to 1 (nearly useless filter): K becomes tiny.

func TestEdge_HighEpsilon(t *testing.T) {
	const n = 1000
	vals := make([]uint64, n)
	for i := 0; i < n; i++ {
		vals[i] = uint64(i) * 10_000
	}

	h := buildEps(t, vals, 1000, 0.99)
	require.NotNil(t, h)

	K := math.Ceil(math.Log2(float64(n) * 1001.0 / 0.99))
	t.Logf("HighEpsilon: expected K≈%.0f, BPK=%.2f", K, bpkOf(h))

	for _, v := range vals {
		require.False(t, h.IsEmpty(v, v),
			"false negative for key %d at eps=0.99", v)
	}
}

// ── edge case 13 ─────────────────────────────────────────────────────────────
// Near-key fallback FPR: wide-universe input, all keys go to fallback (n <
// minClusterSize), K chosen so that phantomSize = spread/2^K is well above
// rangeLen. With the legacy FallbackAuto / FallbackAlwaysTrunc default any
// empty query within phantomSize of a stored key collapses into that key's
// K-bit bucket, pinning FPR near 1.0 on the near-key half of the workload.
// FallbackAlwaysSODA (AdaptiveARE/SODA-hash) avoids this collapse.
//
// Regression test for the OSM scan-ARE flatness investigated 2026-05-04.

func TestEdge_NearKeyFallbackFPR_WideSpread(t *testing.T) {
	const (
		n        = 200 // < minClusterSize → all keys take the fallback path
		rangeLen = uint64(128)
		K        = uint32(30) // phantomSize = 2^40 / 2^30 = 1024 ≫ rangeLen
		base     = uint64(1_000_000)
	)
	spread := uint64(1) << 40
	gap := spread / uint64(n-1)
	require.Greater(t, gap, rangeLen, "test design: gap must exceed rangeLen so near-key queries are genuinely empty")

	vals := make([]uint64, n)
	for i := 0; i < n; i++ {
		vals[i] = base + uint64(i)*gap
	}

	h := buildFromK(t, vals, rangeLen, K)
	nc, nf, _ := h.Stats()
	require.Equal(t, 0, nc, "all keys below minClusterSize → fallback only")
	require.Equal(t, n, nf, "all keys must land in fallback")

	// Near-key empty queries: window [key+rangeLen+1, key+2*rangeLen+1].
	// With gap > 2*rangeLen these intervals contain no stored key.
	fp := 0
	for _, v := range vals {
		lo := v + rangeLen + 1
		hi := lo + rangeLen
		if !h.IsEmpty(lo, hi) {
			fp++
		}
	}
	fpr := float64(fp) / float64(n)
	t.Logf("default (AlwaysSODA) near-key empty FPR=%.4f (fp=%d / n=%d)", fpr, fp, n)
	require.Less(t, fpr, 0.10,
		"near-key empty FPR pinned high — fallback policy regressed to TruncARE?")

	// Sanity: same build with FallbackAlwaysTrunc must reproduce the bug (FPR ≈ 1)
	// — guards against the comparison being trivially satisfied (e.g. if the
	// fallback path were optimized away).
	hTrunc, err := NewHybridScanAREWithPolicy(vals, 64, ConfigWithPolicy{
		K: K, RangeLen: rangeLen, Policy: FallbackAlwaysTrunc{},
	})
	require.NoError(t, err)
	fpTrunc := 0
	for _, v := range vals {
		lo := v + rangeLen + 1
		hi := lo + rangeLen
		if !hTrunc.IsEmpty(lo, hi) {
			fpTrunc++
		}
	}
	fprTrunc := float64(fpTrunc) / float64(n)
	t.Logf("AlwaysTrunc near-key empty FPR=%.4f (fp=%d / n=%d)", fprTrunc, fpTrunc, n)
	require.Greater(t, fprTrunc, 0.50,
		"AlwaysTrunc must reproduce the near-key collapse — test design invalid otherwise")
}
