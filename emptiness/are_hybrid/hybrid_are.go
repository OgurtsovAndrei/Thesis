package are_hybrid

import (
	"Thesis/bits"
	"Thesis/emptiness/are_trunc"
	"Thesis/emptiness/are_optimized"
	"fmt"
	"math"
	"sort"
)

type clusterFilter struct {
	filter *are_optimized.OptimizedApproximateRangeEmptiness
	minKey uint64
	maxKey uint64
}

type HybridARE struct {
	clusters  []clusterFilter
	fallback  *are_trunc.ApproximateRangeEmptiness
	nClusters int
	nFallback int
	n         int
}

func NewHybridARE(keys []bits.BitString, rangeLen uint64, epsilon float64) (*HybridARE, error) {
	n := len(keys)
	if n == 0 {
		return &HybridARE{n: 0}, nil
	}

	// Compute K for clusters (SODA formula) — use the larger of the two
	effectiveRangeLen := rangeLen + 1
	rTarget := float64(n) * float64(effectiveRangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		K = 64
	}

	return NewHybridAREFromK(keys, rangeLen, K)
}

func NewHybridAREFromK(keys []bits.BitString, rangeLen uint64, K uint32) (*HybridARE, error) {
	n := len(keys)
	h := &HybridARE{n: n}

	if n < 2 {
		if n > 0 {
			fb, err := are_trunc.NewApproximateRangeEmptinessFromK(keys, K)
			if err != nil {
				return nil, fmt.Errorf("fallback build: %w", err)
			}
			h.fallback = fb
			h.nFallback = n
		}
		return h, nil
	}

	segments, fallbackKeys := detectClusters(keys, 0.95, 0.01)

	// Build cluster filters
	h.clusters = make([]clusterFilter, 0, len(segments))
	for _, seg := range segments {
		f, err := are_optimized.NewOptimizedAREFromK(seg.keys, rangeLen, K, 0)
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

	// Build fallback filter
	if len(fallbackKeys) > 0 {
		fb, err := are_trunc.NewApproximateRangeEmptinessFromK(fallbackKeys, K)
		if err != nil {
			return nil, fmt.Errorf("fallback build: %w", err)
		}
		h.fallback = fb
		h.nFallback = len(fallbackKeys)
	}

	return h, nil
}

func (h *HybridARE) IsEmpty(a, b bits.BitString) bool {
	if h.n == 0 {
		return true
	}

	aVal := a.TrieUint64()
	bVal := b.TrieUint64()

	// Binary search: first cluster with maxKey >= aVal
	lo := sort.Search(len(h.clusters), func(i int) bool {
		return h.clusters[i].maxKey >= aVal
	})

	// Walk overlapping clusters
	for i := lo; i < len(h.clusters) && h.clusters[i].minKey <= bVal; i++ {
		if !h.clusters[i].filter.IsEmpty(a, b) {
			return false
		}
	}

	// Always check fallback
	if h.fallback != nil {
		if !h.fallback.IsEmpty(a, b) {
			return false
		}
	}

	return true
}

func (h *HybridARE) SizeInBits() uint64 {
	total := uint64(0)
	for _, c := range h.clusters {
		total += c.filter.SizeInBits()
	}
	if h.fallback != nil {
		total += h.fallback.SizeInBits()
	}
	// Metadata: 2 × uint64 per cluster boundary
	total += uint64(len(h.clusters)) * 128
	return total
}

func (h *HybridARE) Stats() (numClusters, fallbackKeys, totalKeys int) {
	return h.nClusters, h.nFallback, h.n
}

// NewHybridAREFromBPK builds a HybridARE targeting a given bits-per-key budget.
func NewHybridAREFromBPK(keys []bits.BitString, rangeLen uint64, targetBPK float64) (*HybridARE, error) {
	K := uint32(math.Ceil(targetBPK))
	if K == 0 {
		K = 1
	}
	if K > 64 {
		K = 64
	}
	return NewHybridAREFromK(keys, rangeLen, K)
}
