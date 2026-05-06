package are_hybrid_scan

import (
	"Thesis/emptiness/approx/are_adaptive"
	"Thesis/emptiness/approx/are_trunc"
	"Thesis/utils/errutil"
	"fmt"
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
type Config struct {
	K uint32
}

// ConfigWithPolicy holds construction parameters for NewHybridScanAREWithPolicy.
// RangeLen is forwarded to the FallbackPolicy (e.g. FallbackPhantom uses it to
// compare phantom region size against query length); it does not affect filter
// construction otherwise.
type ConfigWithPolicy struct {
	K        uint32
	RangeLen uint64
	Policy   FallbackPolicy
}

// --- public constructors ---

// NewHybridScanARE builds Scan-ARE with the default Auto fallback policy.
func NewHybridScanARE(keys []uint64, keyBits uint32, cfg Config) (*HybridScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	if len(keys) == 0 {
		return &HybridScanARE{n: 0}, nil
	}
	dbscanEps := dbscanEpsFromK(len(keys), cfg.K)
	return newHybridScanARE(keys, keyBits, cfg.K, 0, dbscanEps, FallbackAuto{})
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
	return newHybridScanARE(keys, keyBits, cfg.K, cfg.RangeLen, dbscanEps, cfg.Policy)
}

// dbscanEpsFromK returns the DBSCAN density window in key-space units.
//
// We want DBSCAN to detect any cluster that is exact-mode-eligible: spread
// ≤ 2^K and size ≥ minClusterSize. Inside such a cluster the average gap is
// at most 2^K / minClusterSize. For DBSCAN core (dbscanMinPts neighbours in
// the eps-window), eps must be ≥ dbscanMinPts · avg_gap, i.e.
//
//   eps ≥ dbscanMinPts · 2^K / minClusterSize.
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

func newHybridScanARE(keys []uint64, keyBits uint32, K uint32, rangeLen uint64, dbscanEps uint64, policy FallbackPolicy) (*HybridScanARE, error) {
	n := len(keys)
	h := &HybridScanARE{n: n}

	if n < 2 {
		if n > 0 {
			fb, err := are_trunc.NewTruncAREFromK(keys, keyBits, K)
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
		f, err := are_adaptive.NewAdaptiveAREFromK(seg.keys, keyBits, K, 0)
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
		fb, err := buildFallback(fallbackKeys, keyBits, rangeLen, K, policy)
		if err != nil {
			return nil, err
		}
		h.fallback = fb
		h.nFallback = len(fallbackKeys)
	}

	return h, nil
}

func buildFallback(keys []uint64, keyBits uint32, rangeLen uint64, K uint32, policy FallbackPolicy) (*fallbackFilter, error) {
	if policy.useTrunc(keys, K, rangeLen) {
		fb, err := are_trunc.NewTruncAREFromK(keys, keyBits, K)
		if err != nil {
			return nil, fmt.Errorf("fallback trunc build: %w", err)
		}
		return &fallbackFilter{trunc: fb, n: len(keys)}, nil
	}

	fb, err := are_adaptive.NewAdaptiveAREFromK(keys, keyBits, K, 0)
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
