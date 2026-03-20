package are_greedy_scan

import (
	"Thesis/bits"
	"Thesis/emptiness/are_adaptive"
	"fmt"
	"math"
	"sort"
)

type clusterFilter struct {
	filter *are_adaptive.AdaptiveApproximateRangeEmptiness
	minKey uint64
	maxKey uint64
}

// GreedyScanARE segments sorted keys into consecutive clusters where each
// cluster's spread ≤ 2^K, guaranteeing exact mode (FPR=0) for every cluster.
// No fallback filter is needed — every key belongs to a cluster.
type GreedyScanARE struct {
	clusters []clusterFilter
	n        int
}

func NewGreedyScanARE(keys []bits.BitString, rangeLen uint64, epsilon float64) (*GreedyScanARE, error) {
	n := len(keys)
	if n == 0 {
		return &GreedyScanARE{}, nil
	}

	effectiveRangeLen := rangeLen + 1
	rTarget := float64(n) * float64(effectiveRangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		K = 64
	}

	return NewGreedyScanAREFromK(keys, rangeLen, K)
}

// NewGreedyScanAREFromKRaw builds without the merge pass — pure greedy split only.
func NewGreedyScanAREFromKRaw(keys []bits.BitString, rangeLen uint64, K uint32) (*GreedyScanARE, error) {
	return buildGreedy(keys, rangeLen, K, false)
}

func NewGreedyScanAREFromK(keys []bits.BitString, rangeLen uint64, K uint32) (*GreedyScanARE, error) {
	return buildGreedy(keys, rangeLen, K, true)
}

func buildGreedy(keys []bits.BitString, rangeLen uint64, K uint32, merge bool) (*GreedyScanARE, error) {
	n := len(keys)
	if n == 0 {
		return &GreedyScanARE{}, nil
	}

	refs := segmentBySpreadRefs(keys, K)
	if merge {
		refs = mergeSmallClustersRefs(refs, K)
	}
	segments := finalizeRefs(keys, refs)

	clusters := make([]clusterFilter, 0, len(segments))
	for _, seg := range segments {
		f, err := are_adaptive.NewAdaptiveAREFromK(seg.keys, rangeLen, K, 0)
		if err != nil {
			return nil, fmt.Errorf("cluster [%d, %d] build: %w", seg.minKey, seg.maxKey, err)
		}
		clusters = append(clusters, clusterFilter{
			filter: f,
			minKey: seg.minKey,
			maxKey: seg.maxKey,
		})
	}

	return &GreedyScanARE{clusters: clusters, n: n}, nil
}

func (g *GreedyScanARE) IsEmpty(a, b bits.BitString) bool {
	if g.n == 0 {
		return true
	}

	aVal := a.TrieUint64()
	bVal := b.TrieUint64()

	lo := sort.Search(len(g.clusters), func(i int) bool {
		return g.clusters[i].maxKey >= aVal
	})

	for i := lo; i < len(g.clusters) && g.clusters[i].minKey <= bVal; i++ {
		if !g.clusters[i].filter.IsEmpty(a, b) {
			return false
		}
	}

	return true
}

func (g *GreedyScanARE) SizeInBits() uint64 {
	total := uint64(0)
	for _, c := range g.clusters {
		total += c.filter.SizeInBits()
	}
	total += uint64(len(g.clusters)) * 128 // min/max bounds per cluster
	return total
}

func (g *GreedyScanARE) Stats() (numClusters, totalKeys int) {
	return len(g.clusters), g.n
}
