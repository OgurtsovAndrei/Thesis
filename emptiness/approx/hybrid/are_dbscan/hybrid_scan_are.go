package are_dbscan

import (
	"Thesis/emptiness/approx/are_adaptive"
	"Thesis/emptiness/approx/are_trunc"
	"Thesis/emptiness/exact"
	"Thesis/utils/errutil"
	"fmt"
	mbits "math/bits"
	"sort"
)

const (
	dbscanMinPts   = 10  // DBSCAN core threshold: neighbors in eps-window
	minClusterSize = 256 // post-filter: clusters smaller than this → fallback
	epsMultiplier  = 1   // density window scaling factor (see dbscanEpsFromK)
)

type clusterFilter struct {
	filter *are_adaptive.AdaptiveARE
	minKey uint64
	maxKey uint64
}

type fallbackFilter struct {
	trunc    *are_trunc.TruncARE
	adaptive *are_adaptive.AdaptiveARE
	n        int
}

func (f *fallbackFilter) IsEmpty(a, b uint64) bool {
	if f.trunc != nil {
		return f.trunc.IsEmpty(a, b)
	}
	if f.adaptive != nil {
		return f.adaptive.IsEmpty(a, b)
	}
	return true
}

func (f *fallbackFilter) SizeInBits() uint64 {
	if f.trunc != nil {
		return f.trunc.SizeInBits()
	}
	if f.adaptive != nil {
		return f.adaptive.SizeInBits()
	}
	return 0
}

// HybridScanARE uses 1D DBSCAN to segment keys into clusters (Adaptive ARE)
// and a fallback filter (Trunc or SODA) for non-cluster keys.
type HybridScanARE struct {
	clusters  []clusterFilter
	fallback  *fallbackFilter
	nClusters int
	nFallback int
	n         int
}

// Config holds construction parameters for NewHybridScanARE. K is the
// fingerprint width in bits; larger K → lower FPR, higher BPK.
//
// EREBackend selects the underlying exact range-emptiness implementation
// (see package exact). Zero value defaults to exact.VariantAuto. Use the
// WithEREBackend option to set it explicitly without relying on the
// zero-value alias.
type Config struct {
	K          uint32
	EREBackend exact.Variant
	backendSet bool
}

// WithEREBackend returns a copy of cfg with the chosen ERE backend.
func (cfg Config) WithEREBackend(v exact.Variant) Config {
	cfg.EREBackend = v
	cfg.backendSet = true
	return cfg
}

func (cfg Config) backend() exact.Variant {
	if cfg.backendSet {
		return cfg.EREBackend
	}
	return exact.VariantAuto
}

// ConfigWithPolicy holds construction parameters for NewHybridScanAREWithPolicy.
// RangeLen is forwarded to the FallbackPolicy (e.g. FallbackPhantom uses it to
// compare phantom region size against query length); it does not affect filter
// construction otherwise.
//
// EREBackend selects the ERE backend for cluster sub-filters. Zero value
// defaults to exact.VariantAuto. Use WithEREBackend to set explicitly.
//
// FallbackEREBackend selects the ERE backend for the fallback filter. If not
// set via WithFallbackEREBackend, it inherits EREBackend. This allows mixing
// backends: e.g. OneD for small dense clusters and PEF for the large sparse
// fallback.
type ConfigWithPolicy struct {
	K                  uint32
	RangeLen           uint64
	Policy             FallbackPolicy
	EREBackend         exact.Variant
	backendSet         bool
	FallbackEREBackend exact.Variant
	fallbackBackendSet bool
}

// WithEREBackend returns a copy of cfg with the chosen ERE backend for
// cluster sub-filters.
func (cfg ConfigWithPolicy) WithEREBackend(v exact.Variant) ConfigWithPolicy {
	cfg.EREBackend = v
	cfg.backendSet = true
	return cfg
}

// WithFallbackEREBackend returns a copy of cfg with the chosen ERE backend
// for the fallback filter only. Cluster sub-filters keep cfg.EREBackend.
func (cfg ConfigWithPolicy) WithFallbackEREBackend(v exact.Variant) ConfigWithPolicy {
	cfg.FallbackEREBackend = v
	cfg.fallbackBackendSet = true
	return cfg
}

func (cfg ConfigWithPolicy) backend() exact.Variant {
	if cfg.backendSet {
		return cfg.EREBackend
	}
	return exact.VariantAuto
}

func (cfg ConfigWithPolicy) fallbackBackend() exact.Variant {
	if cfg.fallbackBackendSet {
		return cfg.FallbackEREBackend
	}
	return cfg.backend()
}

// --- public constructors ---

// NewHybridScanARE builds Scan-ARE with the default AlwaysSODA fallback policy.
func NewHybridScanARE(keys []uint64, keyBits uint32, cfg Config) (*HybridScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	if len(keys) == 0 {
		return &HybridScanARE{n: 0}, nil
	}
	dbscanEps := dbscanEpsFromK(len(keys), cfg.K)
	b := cfg.backend()
	return newHybridScanARE(keys, keyBits, cfg.K, 0, dbscanEps, FallbackAlwaysSODA{}, b, b)
}

// NewHybridScanAREWithPolicy builds Scan-ARE with an explicit fallback policy.
// cfg.RangeLen is consulted only by RangeLen-aware policies (e.g. FallbackPhantom).
func NewHybridScanAREWithPolicy(keys []uint64, keyBits uint32, cfg ConfigWithPolicy) (*HybridScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	if len(keys) == 0 {
		return &HybridScanARE{n: 0}, nil
	}
	dbscanEps := dbscanEpsFromK(len(keys), cfg.K)
	return newHybridScanARE(keys, keyBits, cfg.K, cfg.RangeLen, dbscanEps, cfg.Policy, cfg.backend(), cfg.fallbackBackend())
}

// dbscanEpsFromK returns the DBSCAN density window in key-space units.
//
// We want DBSCAN to detect any cluster that is exact-mode-eligible: spread
// ≤ 2^K and size ≥ minClusterSize. Inside such a cluster the average gap is
// at most 2^K / minClusterSize. For DBSCAN core (dbscanMinPts neighbours in
// the eps-window), eps must be ≥ dbscanMinPts · avg_gap, i.e.
//
//	eps ≥ dbscanMinPts · 2^K / minClusterSize.
//
// The legacy formula (epsMultiplier · 2^K / n) used n in the denominator,
// which only matches when the cluster covers the whole dataset (size = n).
// On wide-spread real data (OSM, books_u64) it makes eps n/minClusterSize ×
// too small so DBSCAN detects 0 clusters and Hybrid-Scan collapses to plain
// fallback. The eps depends only on the smallest cluster we agree to track,
// not on n.
func dbscanEpsFromK(_ int, K uint32) uint64 {
	var pow float64
	if K >= 64 {
		pow = float64(^uint64(0)) + 1
	} else {
		pow = float64(uint64(1) << K)
	}
	v := float64(epsMultiplier) * float64(dbscanMinPts) * pow / float64(minClusterSize)
	if v < 1 {
		return 1
	}
	if v > float64(^uint64(0)) {
		return ^uint64(0)
	}
	return uint64(v)
}

// --- core build ---

// localK rescales the global K (sized for n_total keys) to the smaller K
// appropriate for a sub-filter holding only n_local keys. The construction
// pins log2(L/ε) — that part of K is independent of n — and replaces the
// global log2(n_total) with log2(n_local):
//
//	K_global = ceil(log2(n_total · L / ε))
//	K_local  = ceil(log2(n_local  · L / ε)) = K_global - log2(n_total) + log2(n_local)
//
// Each sub-filter (cluster ARE, fallback ARE) gets its own K matched to its
// own key count, instead of paying the full global K that's sized for the
// whole dataset. Per-key cost in succinct ERE/SODA is K - log2(n_local) + 2,
// so per-cluster cost collapses from `log2(L/ε) + log2(n_total/n_local)` to
// `log2(L/ε)` — the segmentation no longer leaks the n_total/n_local factor.
func localK(K uint32, nTotal, nLocal int) uint32 {
	if nLocal <= 0 || nTotal <= 0 {
		return K
	}
	if nLocal >= nTotal {
		return K
	}
	delta := mbits.Len64(uint64(nTotal-1)) - mbits.Len64(uint64(nLocal-1))
	if delta < 0 {
		delta = 0
	}
	if uint32(delta) >= K {
		return 1
	}
	return K - uint32(delta)
}

// clusterBackend is used for each dense cluster sub-filter;
// fallbackBackend is used for the sparse fallback filter.
// Pass the same value for both to apply one backend everywhere.
func newHybridScanARE(keys []uint64, keyBits uint32, K uint32, rangeLen uint64, dbscanEps uint64, policy FallbackPolicy, clusterBackend, fallbackBackend exact.Variant) (*HybridScanARE, error) {
	n := len(keys)
	h := &HybridScanARE{n: n}

	if n < 2 {
		if n > 0 {
			fb, err := are_trunc.NewTruncAREFromKWithBackend(keys, keyBits, K, fallbackBackend)
			if err != nil {
				return nil, fmt.Errorf("fallback build: %w", err)
			}
			h.fallback = &fallbackFilter{trunc: fb, n: n}
			h.nFallback = n
		}
		return h, nil
	}

	segments, fallbackKeys := detectClustersDBSCAN(keys, dbscanEps, dbscanMinPts, minClusterSize)

	h.clusters = make([]clusterFilter, 0, len(segments))
	for _, seg := range segments {
		Kc := localK(K, n, len(seg.keys))
		f, err := are_adaptive.NewAdaptiveAREFromKWithBackend(seg.keys, keyBits, Kc, 0, clusterBackend)
		if err != nil {
			return nil, fmt.Errorf("cluster [%d, %d] build: %w", seg.minKey, seg.maxKey, err)
		}
		h.clusters = append(h.clusters, clusterFilter{
			filter: f,
			minKey: seg.minKey,
			maxKey: seg.maxKey,
		})
	}
	h.nClusters = len(h.clusters)

	if len(fallbackKeys) > 0 {
		Kfb := localK(K, n, len(fallbackKeys))
		fb, err := buildFallback(fallbackKeys, keyBits, rangeLen, Kfb, policy, fallbackBackend)
		if err != nil {
			return nil, err
		}
		h.fallback = fb
		h.nFallback = len(fallbackKeys)
	}

	return h, nil
}

func buildFallback(keys []uint64, keyBits uint32, rangeLen uint64, K uint32, policy FallbackPolicy, backend exact.Variant) (*fallbackFilter, error) {
	if policy.useTrunc(keys, K, rangeLen) {
		fb, err := are_trunc.NewTruncAREFromKWithBackend(keys, keyBits, K, backend)
		if err != nil {
			return nil, fmt.Errorf("fallback trunc build: %w", err)
		}
		return &fallbackFilter{trunc: fb, n: len(keys)}, nil
	}

	fb, err := are_adaptive.NewAdaptiveAREFromKWithBackend(keys, keyBits, K, 0, backend)
	if err != nil {
		return nil, fmt.Errorf("fallback adaptive build: %w", err)
	}
	return &fallbackFilter{adaptive: fb, n: len(keys)}, nil
}

// --- query & metrics ---

func (h *HybridScanARE) IsEmpty(lo, hi uint64) bool {
	if h.n == 0 {
		return true
	}

	idx := sort.Search(len(h.clusters), func(i int) bool {
		return h.clusters[i].maxKey >= lo
	})

	for i := idx; i < len(h.clusters) && h.clusters[i].minKey <= hi; i++ {
		if !h.clusters[i].filter.IsEmpty(lo, hi) {
			return false
		}
	}

	if h.fallback != nil {
		if !h.fallback.IsEmpty(lo, hi) {
			return false
		}
	}

	return true
}

func (h *HybridScanARE) SizeInBits() uint64 {
	total := uint64(0)
	for _, c := range h.clusters {
		total += c.filter.SizeInBits()
	}
	if h.fallback != nil {
		total += h.fallback.SizeInBits()
	}
	total += uint64(len(h.clusters)) * 128
	return total
}

func (h *HybridScanARE) Stats() (numClusters, fallbackKeys, totalKeys int) {
	return h.nClusters, h.nFallback, h.n
}
