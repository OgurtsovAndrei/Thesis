package are_hybrid

import (
	"Thesis/bits"
	"Thesis/emptiness/are"
	"Thesis/emptiness/are_optimized"
	"fmt"
	"sort"
)

type clusterFilter struct {
	filter *are_optimized.OptimizedApproximateRangeEmptiness
	minKey uint64
	maxKey uint64
}

type HybridARE struct {
	clusters  []clusterFilter
	fallback  *are.ApproximateRangeEmptiness
	nClusters int
	nFallback int
	n         int
}

func NewHybridARE(keys []bits.BitString, rangeLen uint64, epsilon float64) (*HybridARE, error) {
	n := len(keys)
	h := &HybridARE{n: n}

	if n < 2 {
		// Too few keys for cluster detection — all to fallback
		if n > 0 {
			fb, err := are.NewApproximateRangeEmptiness(keys, epsilon)
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
		f, err := are_optimized.NewOptimizedARE(seg.keys, rangeLen, epsilon, 0)
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
		fb, err := are.NewApproximateRangeEmptiness(fallbackKeys, epsilon)
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
