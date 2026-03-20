package are_greedy_scan

import (
	"Thesis/bits"
	"Thesis/emptiness/are_adaptive"
	"Thesis/emptiness/are_trunc"
	"fmt"
	"math"
	mbits "math/bits"
	"sort"
)

type clusterFilter struct {
	filter *are_adaptive.AdaptiveApproximateRangeEmptiness
	minKey uint64
	maxKey uint64
}

type fallbackFilter struct {
	trunc *are_trunc.TruncARE
	n     int
}

func (f *fallbackFilter) IsEmpty(a, b bits.BitString) bool {
	if f.trunc != nil {
		return f.trunc.IsEmpty(a, b)
	}
	return true
}

func (f *fallbackFilter) SizeInBits() uint64 {
	if f.trunc != nil {
		return f.trunc.SizeInBits()
	}
	return 0
}

// GreedyScanARE segments sorted keys into consecutive clusters using greedy
// spread-threshold + hierarchical merge. Clusters with spread ≤ 2^K use exact
// mode (FPR=0). Clusters with spread > 2^K (SODA territory) are sent to a
// trunc fallback instead, which is L-independent.
type GreedyScanARE struct {
	clusters  []clusterFilter
	fallback  *fallbackFilter
	nClusters int
	nFallback int
	n         int
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

// NewGreedyScanAREFromKRaw builds without merge and without fallback — pure greedy split only.
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

	// Split segments: exact-mode clusters vs SODA-mode → trunc fallback.
	var exactSegs []segment
	var fallbackKeys []bits.BitString

	for _, seg := range segments {
		spread := seg.maxKey - seg.minKey
		spreadBits := uint32(0)
		if spread > 0 {
			spreadBits = uint32(64 - mbits.LeadingZeros64(spread))
		}
		if spreadBits <= K {
			exactSegs = append(exactSegs, seg)
		} else {
			fallbackKeys = append(fallbackKeys, seg.keys...)
		}
	}

	g := &GreedyScanARE{n: n}

	// Build exact-mode cluster filters.
	g.clusters = make([]clusterFilter, 0, len(exactSegs))
	for _, seg := range exactSegs {
		f, err := are_adaptive.NewAdaptiveAREFromK(seg.keys, rangeLen, K, 0)
		if err != nil {
			return nil, fmt.Errorf("cluster [%d, %d] build: %w", seg.minKey, seg.maxKey, err)
		}
		g.clusters = append(g.clusters, clusterFilter{
			filter: f,
			minKey: seg.minKey,
			maxKey: seg.maxKey,
		})
	}
	g.nClusters = len(g.clusters)

	// Build trunc fallback for SODA-mode segments.
	if len(fallbackKeys) > 0 {
		fb, err := are_trunc.NewTruncAREFromK(fallbackKeys, K)
		if err != nil {
			return nil, fmt.Errorf("fallback trunc build: %w", err)
		}
		g.fallback = &fallbackFilter{trunc: fb, n: len(fallbackKeys)}
		g.nFallback = len(fallbackKeys)
	}

	return g, nil
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

	if g.fallback != nil {
		if !g.fallback.IsEmpty(a, b) {
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
	if g.fallback != nil {
		total += g.fallback.SizeInBits()
	}
	total += uint64(len(g.clusters)) * 128
	return total
}

func (g *GreedyScanARE) Stats() (numClusters, fallbackKeys, totalKeys int) {
	return g.nClusters, g.nFallback, g.n
}
