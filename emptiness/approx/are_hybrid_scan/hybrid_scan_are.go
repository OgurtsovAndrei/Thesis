package are_hybrid_scan

import (
	"Thesis/emptiness/approx/are_adaptive"
	"Thesis/emptiness/approx/are_trunc"
	"Thesis/utils/errutil"
	"fmt"
	"math"
	"sort"
)

const (
	dbscanMinPts   = 10  // DBSCAN core threshold: neighbors in eps-window
	minClusterSize = 256 // post-filter: clusters smaller than this → fallback
	epsMultiplier  = 10
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

// Config holds construction parameters for NewHybridScanARE.
type Config struct {
	RangeLen float64
	Eps      float64
}

// ConfigFromK holds construction parameters for NewHybridScanAREFromK.
type ConfigFromK struct {
	RangeLen float64
	K        uint32
}

// ConfigWithPolicy holds construction parameters for NewHybridScanAREWithPolicy.
type ConfigWithPolicy struct {
	RangeLen float64
	K        uint32
	Policy   FallbackPolicy
}

// ConfigFromBPK holds construction parameters for NewHybridScanAREFromBPK.
type ConfigFromBPK struct {
	RangeLen float64
	BPK      float64
}

// --- public constructors ---

func NewHybridScanARE(keys []uint64, keyBits uint32, cfg Config) (*HybridScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	n := len(keys)
	if n == 0 {
		return &HybridScanARE{n: 0}, nil
	}

	rangeLen := uint64(cfg.RangeLen)
	effectiveRangeLen := rangeLen + 1
	rTarget := float64(n) * float64(effectiveRangeLen) / cfg.Eps
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		K = 64
	}

	dbscanEps := uint64(cfg.RangeLen / cfg.Eps * epsMultiplier)
	return newHybridScanARE(keys, keyBits, rangeLen, K, dbscanEps, FallbackAuto{})
}

func NewHybridScanAREFromK(keys []uint64, keyBits uint32, cfg ConfigFromK) (*HybridScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	if len(keys) == 0 {
		return &HybridScanARE{n: 0}, nil
	}
	rangeLen := uint64(cfg.RangeLen)
	dbscanEps := dbscanEpsFromK(len(keys), rangeLen, cfg.K)
	return newHybridScanARE(keys, keyBits, rangeLen, cfg.K, dbscanEps, FallbackAuto{})
}

// NewHybridScanAREWithPolicy builds Scan-ARE with an explicit fallback policy.
func NewHybridScanAREWithPolicy(keys []uint64, keyBits uint32, cfg ConfigWithPolicy) (*HybridScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	if len(keys) == 0 {
		return &HybridScanARE{n: 0}, nil
	}
	rangeLen := uint64(cfg.RangeLen)
	dbscanEps := dbscanEpsFromK(len(keys), rangeLen, cfg.K)
	return newHybridScanARE(keys, keyBits, rangeLen, cfg.K, dbscanEps, cfg.Policy)
}

func NewHybridScanAREFromBPK(keys []uint64, keyBits uint32, cfg ConfigFromBPK) (*HybridScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	K := uint32(math.Ceil(cfg.BPK))
	if K == 0 {
		K = 1
	}
	if K > 64 {
		K = 64
	}
	return NewHybridScanAREFromK(keys, keyBits, ConfigFromK{RangeLen: cfg.RangeLen, K: K})
}

func dbscanEpsFromK(n int, rangeLen uint64, K uint32) uint64 {
	effectiveRangeLen := float64(rangeLen) + 1
	epsilon := float64(n) * effectiveRangeLen / math.Pow(2, float64(K))
	if epsilon <= 0 || epsilon > 1 {
		epsilon = 0.01
	}
	return uint64(float64(rangeLen) / epsilon * epsMultiplier)
}

// --- core build ---

func newHybridScanARE(keys []uint64, keyBits uint32, rangeLen uint64, K uint32, dbscanEps uint64, policy FallbackPolicy) (*HybridScanARE, error) {
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
		f, err := are_adaptive.NewAdaptiveAREFromK(seg.keys, keyBits, float64(rangeLen), K, 0)
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

	fb, err := are_adaptive.NewAdaptiveAREFromK(keys, keyBits, float64(rangeLen), K, 0)
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
