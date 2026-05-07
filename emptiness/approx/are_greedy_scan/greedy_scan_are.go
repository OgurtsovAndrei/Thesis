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

// ConfigWithPolicy holds construction parameters for NewGreedyScanAREWithPolicy.
// RangeLen is forwarded to RangeLen-aware policies (e.g. FallbackPhantom);
// ignored otherwise.
type ConfigWithPolicy struct {
	K        uint32
	RangeLen uint64
	Policy   FallbackPolicy
}

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

func (f *fallbackFilter) isEmptyUint64(lo, hi uint64) bool {
	if f.trunc != nil {
		return f.trunc.IsEmpty(lo, hi)
	}
	if f.adaptive != nil {
		return f.adaptive.IsEmpty(lo, hi)
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

// NewGreedyScanARE builds a GreedyScanARE with the default Trunc fallback.
// keys must fit in keyBits bits (high bits above keyBits must be zero).
func NewGreedyScanARE(keys []uint64, keyBits uint32, cfg Config) (*GreedyScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	cp := append([]uint64(nil), keys...)
	return buildGreedy(cp, keyBits, cfg.K, 0, true, FallbackAlwaysTrunc{})
}

// NewGreedyScanAREWithPolicy builds a GreedyScanARE with an explicit fallback
// policy. cfg.RangeLen is consulted only by RangeLen-aware policies.
func NewGreedyScanAREWithPolicy(keys []uint64, keyBits uint32, cfg ConfigWithPolicy) (*GreedyScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	cp := append([]uint64(nil), keys...)
	policy := cfg.Policy
	if policy == nil {
		policy = FallbackAlwaysTrunc{}
	}
	return buildGreedy(cp, keyBits, cfg.K, cfg.RangeLen, true, policy)
}

// NewGreedyScanARERaw builds without merge and without fallback — pure greedy split only.
// keys need not be sorted; a copy is sorted internally.
func NewGreedyScanARERaw(keys []uint64, keyBits uint32, cfg ConfigRaw) (*GreedyScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	cp := append([]uint64(nil), keys...)
	return buildGreedy(cp, keyBits, cfg.K, 0, false, FallbackAlwaysTrunc{})
}

func buildGreedy(keys []uint64, keyBits uint32, K uint32, rangeLen uint64, merge bool, policy FallbackPolicy) (*GreedyScanARE, error) {
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

	// Split segments: exact-mode clusters vs SODA-mode → fallback.
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

	// Build exact-mode cluster filters with per-cluster K_local
	// (K rescaled to the cluster's own size — see localK doc).
	g.clusters = make([]clusterFilter, 0, len(exactSegs))
	for _, seg := range exactSegs {
		Kc := localK(K, n, len(seg.keys))
		f, err := are_adaptive.NewAdaptiveAREFromK(seg.keys, keyBits, Kc, 0)
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

	// Build fallback filter (Trunc or SODA, per policy) with per-fallback K.
	if len(fallbackKeys) > 0 {
		Kfb := localK(K, n, len(fallbackKeys))
		fb, err := buildFallback(fallbackKeys, keyBits, rangeLen, Kfb, policy)
		if err != nil {
			return nil, err
		}
		g.fallback = fb
		g.nFallback = len(fallbackKeys)
	}

	return g, nil
}

// localK rescales the global K (sized for n_total keys) to the smaller K
// appropriate for a sub-filter holding only n_local keys. See doc on the
// identical helper in are_hybrid_scan/hybrid_scan_are.go.
func localK(K uint32, nTotal, nLocal int) uint32 {
	if nLocal <= 0 || nTotal <= 0 || nLocal >= nTotal {
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
