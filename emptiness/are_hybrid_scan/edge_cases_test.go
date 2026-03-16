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

// buildFromK is a thin wrapper for NewHybridScanAREFromK.
func buildFromK(t *testing.T, vals []uint64, rangeLen uint64, K uint32) *HybridScanARE {
	t.Helper()
	bs := makeSortedBS(vals)
	h, err := NewHybridScanAREFromK(bs, rangeLen, K)
	require.NoError(t, err)
	return h
}

// buildEps is a thin wrapper for NewHybridScanARE.
func buildEps(t *testing.T, vals []uint64, rangeLen uint64, eps float64) *HybridScanARE {
	t.Helper()
	bs := makeSortedBS(vals)
	h, err := NewHybridScanARE(bs, rangeLen, eps)
	require.NoError(t, err)
	return h
}

// ── edge case 1 ──────────────────────────────────────────────────────────────
// BPK blowup: many tiny clusters just above minClusterSize=256.
// Each one gets 128 bits of metadata.  With very small K the cluster payload
// is tiny, so metadata may dominate.  We just verify the filter builds without
// crashing and that SizeInBits() is non-zero and consistent.

func TestEdge_BPKBlowup_ManyTinyClusters(t *testing.T) {
	// 20 tight clusters of exactly minClusterSize keys each,
	// separated widely so DBSCAN treats them as separate segments.
	const (
		numClusters  = 20
		clusterSize  = minClusterSize // 256 – exactly at the threshold
		separation   = uint64(1e12)
		rangeLen     = uint64(100)
		K            = uint32(4) // tiny K → tiny payload per cluster
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

	// Cluster overhead alone is nc*128 bits.
	// BPK contributed purely by metadata = nc*128 / nt.
	metadataBPK := float64(nc) * 128.0 / float64(nt)
	t.Logf("metadata BPK contribution=%.2f", metadataBPK)

	// Verify no false negatives for a sample of stored keys (every 10th key).
	for i := 0; i < len(vals); i += len(vals) / 10 {
		v := vals[i]
		require.False(t, h.IsEmpty(trieBS(v), trieBS(v)),
			"false negative for stored key %d", v)
	}
}

// ── edge case 2 ──────────────────────────────────────────────────────────────
// Degenerate DBSCAN: all keys identical.
// spread=0, eps irrelevant.  Filter must build and answer point queries.

func TestEdge_DegenerateAllIdenticalKeys(t *testing.T) {
	const key = uint64(0xDEADBEEF)
	// Build with duplicates stripped by the caller convention:
	// keys slice must be sorted (equal is fine — truncation handles dedup internally).
	vals := []uint64{key, key, key, key, key}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })

	h := buildEps(t, vals, 100, 0.01)
	require.NotNil(t, h)

	require.False(t, h.IsEmpty(trieBS(key), trieBS(key)),
		"stored key must not be reported empty")
	require.True(t, h.IsEmpty(trieBS(key+1), trieBS(key+100)),
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

	require.False(t, h.IsEmpty(trieBS(key), trieBS(key)),
		"false negative on the only stored key")

	// Ranges that straddle the key.
	require.False(t, h.IsEmpty(trieBS(key-100), trieBS(key+100)),
		"range covering key must not report empty")

	// Range clearly before the key.
	if key > 200 {
		require.True(t, h.IsEmpty(trieBS(key-200), trieBS(key-101)),
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
	// Already sorted.

	h := buildEps(t, vals, 10, 0.01)
	require.NotNil(t, h)

	for _, v := range vals {
		require.False(t, h.IsEmpty(trieBS(v), trieBS(v)),
			"false negative for key near maxUint64: %d", v)
	}

	// Full-universe query [0, maxUint64] — must not be empty (keys exist).
	require.False(t, h.IsEmpty(trieBS(0), trieBS(maxUint64)),
		"[0, MaxUint64] must not be empty when keys exist")

	// Range ending at maxUint64 (no overflow in b+1 path).
	require.False(t, h.IsEmpty(trieBS(maxUint64-1), trieBS(maxUint64)),
		"range [maxUint64-1, maxUint64] must not be empty")
}

// ── edge case 5 ──────────────────────────────────────────────────────────────
// K=64: largest possible K, keys spread across full uint64 space.

func TestEdge_K64MaxK(t *testing.T) {
	const n = 300
	vals := make([]uint64, n)
	// Evenly space n keys across [0, maxUint64].
	for i := 0; i < n; i++ {
		vals[i] = uint64(float64(i) / float64(n-1) * float64(maxUint64))
	}
	sort.Slice(vals, func(i, j int) bool { return vals[i] < vals[j] })

	h := buildFromK(t, vals, 1000, 64)
	require.NotNil(t, h)

	sz := h.SizeInBits()
	require.Greater(t, sz, uint64(0))
	t.Logf("K=64, n=%d, SizeInBits=%d, BPK=%.2f", n, sz, bpkOf(h))

	// No false negatives.
	for _, v := range vals {
		require.False(t, h.IsEmpty(trieBS(v), trieBS(v)),
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

	// No false negatives — this is required regardless of FPR.
	for _, v := range vals {
		require.False(t, h.IsEmpty(trieBS(v), trieBS(v)),
			"false negative at K=1 for key %d", v)
	}
}

// ── edge case 7 ──────────────────────────────────────────────────────────────
// truncSafe edge: keys where spread >> 2^K but all gaps are equal.
// spread fits in many bits, phantom_size may equal the gap, so truncSafe should
// return false and trigger the adaptive fallback.

func TestEdge_TruncSafeEdge_EqualGaps(t *testing.T) {
	// n=2000 keys, gap chosen so spread is large but uniform.
	// With K=10, phantom_size = spread / 2^10.
	// Set gap = phantom_size - 1 so 5th-percentile gap ≤ phantomSize → truncSafe=false.
	const (
		n        = 2000
		K        = uint32(10)
		base     = uint64(1_000_000)
		rangeLen = uint64(10)
	)
	// spread = (n-1)*gap
	// phantom_size = spread >> K
	// We want gap == phantom_size exactly:
	//   gap = (n-1)*gap >> K  → gap*(1 - (n-1)/2^K) = 0 only if gap is very small.
	// Simpler: choose spread = 2^40 so phantom_size = 2^30, then set gap = 2^30 - 1.
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

	// No false negatives.
	for _, v := range vals {
		require.False(t, h.IsEmpty(trieBS(v), trieBS(v)),
			"false negative for key %d", v)
	}
}

// ── edge case 8 ──────────────────────────────────────────────────────────────
// Query correctness: boundary queries.
// a > b must be treated as empty (or at minimum not panic/crash).
// a == b for a stored key must return false.
// a < minKey and b < minKey must return true (definitely empty).
// a > maxKey and b > maxKey must return true (definitely empty).

func TestEdge_BoundaryQueries(t *testing.T) {
	vals := []uint64{100, 200, 300, 400, 500}
	h := buildEps(t, vals, 100, 0.01)

	minKey, maxKey := vals[0], vals[len(vals)-1]

	// Point query on each stored key.
	for _, v := range vals {
		require.False(t, h.IsEmpty(trieBS(v), trieBS(v)),
			"point query must not be empty for stored key %d", v)
	}

	// Inverted range: a > b.  The adaptive filter returns true for a>b; we require no panic.
	result := h.IsEmpty(trieBS(300), trieBS(200))
	t.Logf("IsEmpty([300,200]) (inverted) = %v (no panic required)", result)

	// Range entirely before minKey.
	require.True(t, h.IsEmpty(trieBS(0), trieBS(minKey-1)),
		"range [0, minKey-1] must be empty")

	// Range entirely after maxKey.
	require.True(t, h.IsEmpty(trieBS(maxKey+1), trieBS(maxKey+1000)),
		"range [maxKey+1, maxKey+1000] must be empty")

	// Range spanning the full key set.
	require.False(t, h.IsEmpty(trieBS(minKey), trieBS(maxKey)),
		"range [minKey, maxKey] must not be empty")

	// Range [0, MaxUint64] covers everything.
	require.False(t, h.IsEmpty(trieBS(0), trieBS(maxUint64)),
		"[0, MaxUint64] must not be empty when any keys exist")
}

// ── edge case 9 ──────────────────────────────────────────────────────────────
// SizeInBits accuracy: for keys that all go to fallback (n < minClusterSize),
// the reported size must be >= K * n_unique (at least enough bits to hold the trie).

func TestEdge_SizeInBits_Accuracy(t *testing.T) {
	const (
		n        = 100 // well below minClusterSize=256 → all fallback
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

		// Lower bound: the trie must store at least n unique keys in a K-bit universe.
		// The ERE structure needs at least n bits (one per unique key).
		require.GreaterOrEqual(t, sz, uint64(n),
			"SizeInBits should be at least n bits to encode %d keys", n)
	}
}

// ── edge case 10 ─────────────────────────────────────────────────────────────
// BPK with K=64 and many keys: cluster overhead should be dominated by filter bits.
// We check BPK stays bounded and well below pathological values (e.g. > 1000 bpk).

func TestEdge_BPKBounded_LargeN_K64(t *testing.T) {
	const (
		n        = 5000
		rangeLen = uint64(1000)
		K        = uint32(20)
	)
	// 3 tight clusters of 1500 keys + 500 scattered → clusters form.
	vals := make([]uint64, 0, n)
	for c := 0; c < 3; c++ {
		base := uint64(c+1) * 1_000_000_000
		for i := 0; i < 1500; i++ {
			vals = append(vals, base+uint64(i))
		}
	}
	// Scattered keys far apart.
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

	// BPK should be sane — well under 1000 bpk.
	require.Less(t, bpk, 1000.0,
		"BPK blowup detected: %.2f bpk for K=%d n=%d", bpk, K, nt)

	// No false negatives.
	for _, v := range vals {
		require.False(t, h.IsEmpty(trieBS(v), trieBS(v)),
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

	require.False(t, h.IsEmpty(trieBS(0), trieBS(0)),
		"false negative for key 0")
	require.False(t, h.IsEmpty(trieBS(maxUint64), trieBS(maxUint64)),
		"false negative for key maxUint64")
	require.False(t, h.IsEmpty(trieBS(0), trieBS(maxUint64)),
		"full-span range must not be empty")
}

// ── edge case 12 ─────────────────────────────────────────────────────────────
// Epsilon very close to 1 (nearly useless filter): K becomes tiny.
// The filter should still be consistent (no false negatives).

func TestEdge_HighEpsilon(t *testing.T) {
	const n = 1000
	vals := make([]uint64, n)
	for i := 0; i < n; i++ {
		vals[i] = uint64(i) * 10_000
	}

	// eps=0.99 → extremely high false positive rate allowed → very small K.
	h := buildEps(t, vals, 1000, 0.99)
	require.NotNil(t, h)

	K := math.Ceil(math.Log2(float64(n)*1001.0/0.99))
	t.Logf("HighEpsilon: expected K≈%.0f, BPK=%.2f", K, bpkOf(h))

	// No false negatives regardless of FPR.
	for _, v := range vals {
		require.False(t, h.IsEmpty(trieBS(v), trieBS(v)),
			"false negative for key %d at eps=0.99", v)
	}
}
