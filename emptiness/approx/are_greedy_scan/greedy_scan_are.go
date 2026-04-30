package are_greedy_scan

import (
	"Thesis/emptiness/approx/are_adaptive"
	"Thesis/emptiness/approx/are_trunc"
	"Thesis/utils/errutil"
	"fmt"
	mbits "math/bits"
	"sort"
)

// Config holds construction parameters for GreedyScanARE. K is the fingerprint
// width in bits; larger K → lower FPR, higher BPK.
type Config struct {
	K uint32
}

// ConfigRaw holds construction parameters for GreedyScanARE with no merge pass —
// pure greedy split only, no hierarchical merge and no fallback.
type ConfigRaw struct {
	K uint32
}

type clusterFilter struct {
	filter *are_adaptive.AdaptiveARE
	minKey uint64
	maxKey uint64
}

type fallbackFilter struct {
	trunc *are_trunc.TruncARE
	n     int
}

func (f *fallbackFilter) isEmptyUint64(lo, hi uint64) bool {
	if f.trunc != nil {
		return f.trunc.IsEmpty(lo, hi)
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

// NewGreedyScanARE builds a GreedyScanARE from a copy of keys.
// keys must fit in keyBits bits (high bits above keyBits must be zero).
func NewGreedyScanARE(keys []uint64, keyBits uint32, cfg Config) (*GreedyScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	cp := append([]uint64(nil), keys...)
	return buildGreedy(cp, keyBits, cfg.K, true)
}

// NewGreedyScanARERaw builds without merge and without fallback — pure greedy split only.
// keys need not be sorted; a copy is sorted internally.
func NewGreedyScanARERaw(keys []uint64, keyBits uint32, cfg ConfigRaw) (*GreedyScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	cp := append([]uint64(nil), keys...)
	return buildGreedy(cp, keyBits, cfg.K, false)
}

func buildGreedy(keys []uint64, keyBits uint32, K uint32, merge bool) (*GreedyScanARE, error) {
	n := len(keys)
	if n == 0 {
		return &GreedyScanARE{}, nil
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	refs := segmentBySpreadRefs(keys, K)
	if merge {
		refs = mergeSmallClustersRefs(refs, K)
	}
	segments := finalizeRefs(keys, refs)

	// Split segments: exact-mode clusters vs SODA-mode → trunc fallback.
	var exactSegs []segment
	var fallbackKeys []uint64

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
		f, err := are_adaptive.NewAdaptiveAREFromK(seg.keys, keyBits, K, 0)
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
		fb, err := are_trunc.NewTruncAREFromK(fallbackKeys, keyBits, K)
		if err != nil {
			return nil, fmt.Errorf("fallback trunc build: %w", err)
		}
		g.fallback = &fallbackFilter{trunc: fb, n: len(fallbackKeys)}
		g.nFallback = len(fallbackKeys)
	}

	return g, nil
}

// IsEmpty reports whether [lo, hi] contains no stored key.
func (g *GreedyScanARE) IsEmpty(lo, hi uint64) bool {
	if g.n == 0 {
		return true
	}

	idx := sort.Search(len(g.clusters), func(i int) bool {
		return g.clusters[i].maxKey >= lo
	})

	for i := idx; i < len(g.clusters) && g.clusters[i].minKey <= hi; i++ {
		if !g.clusters[i].filter.IsEmpty(lo, hi) {
			return false
		}
	}

	if g.fallback != nil {
		if !g.fallback.isEmptyUint64(lo, hi) {
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
