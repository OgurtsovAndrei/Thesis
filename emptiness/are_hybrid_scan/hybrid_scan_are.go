package are_hybrid_scan

import (
	"Thesis/bits"
	"Thesis/emptiness/are_adaptive"
	"Thesis/emptiness/are_trunc"
	"fmt"
	"math"
	mbits "math/bits"
	"sort"
)

const (
	dbscanMinPts   = 10  // DBSCAN core threshold: neighbors in eps-window
	minClusterSize = 256 // post-filter: clusters smaller than this → fallback
	epsMultiplier  = 10
)

type clusterFilter struct {
	filter *are_adaptive.AdaptiveApproximateRangeEmptiness
	minKey uint64
	maxKey uint64
}

// fallback is either trunc (when gaps are large enough) or adaptive/SODA
// (when trunc would suffer from phantom overlap).
type fallbackFilter struct {
	trunc    *are_trunc.ApproximateRangeEmptiness
	adaptive *are_adaptive.AdaptiveApproximateRangeEmptiness
	n        int
}

func (f *fallbackFilter) IsEmpty(a, b bits.BitString) bool {
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

type HybridScanARE struct {
	clusters  []clusterFilter
	fallback  *fallbackFilter
	nClusters int
	nFallback int
	n         int
}

func NewHybridScanARE(keys []bits.BitString, rangeLen uint64, epsilon float64) (*HybridScanARE, error) {
	n := len(keys)
	if n == 0 {
		return &HybridScanARE{n: 0}, nil
	}

	effectiveRangeLen := rangeLen + 1
	rTarget := float64(n) * float64(effectiveRangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		K = 64
	}

	eps := uint64(float64(rangeLen) / epsilon * epsMultiplier)

	return newHybridScanARE(keys, rangeLen, K, eps)
}

func NewHybridScanAREFromK(keys []bits.BitString, rangeLen uint64, K uint32) (*HybridScanARE, error) {
	n := len(keys)
	if n == 0 {
		return &HybridScanARE{n: 0}, nil
	}

	effectiveRangeLen := float64(rangeLen) + 1
	epsilon := float64(n) * effectiveRangeLen / math.Pow(2, float64(K))
	if epsilon <= 0 || epsilon > 1 {
		epsilon = 0.01
	}

	eps := uint64(float64(rangeLen) / epsilon * epsMultiplier)

	return newHybridScanARE(keys, rangeLen, K, eps)
}

// truncSafe checks whether trunc fallback will work for the given keys.
// Trunc breaks when min gap < phantom_size = spread * epsilon / (2*n).
// We use the 5th-percentile gap as a robust proxy for min gap.
func truncSafe(keys64 []uint64, K uint32) bool {
	n := len(keys64)
	if n < 2 {
		return true
	}

	spread := keys64[n-1] - keys64[0]
	if spread == 0 {
		return true
	}

	// phantom_size = spread / 2^K
	spreadBits := uint32(64 - mbits.LeadingZeros64(spread))
	if spreadBits <= K {
		return true // spread fits in K bits → exact mode would trigger in adaptive anyway
	}
	phantomSize := spread >> K
	if phantomSize == 0 {
		phantomSize = 1
	}

	// Find 5th-percentile gap as robust min gap estimate.
	gaps := make([]uint64, n-1)
	for i := 0; i < n-1; i++ {
		gaps[i] = keys64[i+1] - keys64[i]
	}
	idx := len(gaps) / 20 // 5th percentile
	if idx >= len(gaps) {
		idx = len(gaps) - 1
	}
	// Partial sort to find the idx-th smallest gap.
	p5Gap := quickselect(gaps, idx)

	return p5Gap > phantomSize
}

func newHybridScanARE(keys []bits.BitString, rangeLen uint64, K uint32, eps uint64) (*HybridScanARE, error) {
	n := len(keys)
	h := &HybridScanARE{n: n}

	if n < 2 {
		if n > 0 {
			fb, err := are_trunc.NewApproximateRangeEmptinessFromK(keys, K)
			if err != nil {
				return nil, fmt.Errorf("fallback build: %w", err)
			}
			h.fallback = &fallbackFilter{trunc: fb, n: n}
			h.nFallback = n
		}
		return h, nil
	}

	segments, fallbackKeys := detectClustersDBSCAN(keys, eps, dbscanMinPts, minClusterSize)

	h.clusters = make([]clusterFilter, 0, len(segments))
	for _, seg := range segments {
		f, err := are_adaptive.NewAdaptiveAREFromK(seg.keys, rangeLen, K, 0)
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
		// Decide: trunc or adaptive/SODA for fallback.
		fbKeys64 := make([]uint64, len(fallbackKeys))
		for i, k := range fallbackKeys {
			fbKeys64[i] = k.TrieUint64()
		}

		if truncSafe(fbKeys64, K) {
			fb, err := are_trunc.NewApproximateRangeEmptinessFromK(fallbackKeys, K)
			if err != nil {
				return nil, fmt.Errorf("fallback trunc build: %w", err)
			}
			h.fallback = &fallbackFilter{trunc: fb, n: len(fallbackKeys)}
		} else {
			fb, err := are_adaptive.NewAdaptiveAREFromK(fallbackKeys, rangeLen, K, 0)
			if err != nil {
				return nil, fmt.Errorf("fallback adaptive build: %w", err)
			}
			h.fallback = &fallbackFilter{adaptive: fb, n: len(fallbackKeys)}
		}
		h.nFallback = len(fallbackKeys)
	}

	return h, nil
}

func (h *HybridScanARE) IsEmpty(a, b bits.BitString) bool {
	if h.n == 0 {
		return true
	}

	aVal := a.TrieUint64()
	bVal := b.TrieUint64()

	lo := sort.Search(len(h.clusters), func(i int) bool {
		return h.clusters[i].maxKey >= aVal
	})

	for i := lo; i < len(h.clusters) && h.clusters[i].minKey <= bVal; i++ {
		if !h.clusters[i].filter.IsEmpty(a, b) {
			return false
		}
	}

	if h.fallback != nil {
		if !h.fallback.IsEmpty(a, b) {
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

func NewHybridScanAREFromBPK(keys []bits.BitString, rangeLen uint64, targetBPK float64) (*HybridScanARE, error) {
	K := uint32(math.Ceil(targetBPK))
	if K == 0 {
		K = 1
	}
	if K > 64 {
		K = 64
	}
	return NewHybridScanAREFromK(keys, rangeLen, K)
}
