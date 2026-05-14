package are_greedy

import (
	"Thesis/emptiness/approx/are_adaptive"
	"Thesis/emptiness/approx/hybrid/hybridutil"
	"Thesis/emptiness/exact"
	"Thesis/utils/errutil"
	"fmt"
	mbits "math/bits"
	"sort"
)

// Config holds construction parameters for GreedyScanARE. K is the fingerprint
// width in bits; larger K → lower FPR, higher BPK.
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

// ConfigRaw holds construction parameters for GreedyScanARE with no merge pass —
// pure greedy split only, no hierarchical merge and no fallback.
//
// EREBackend selects the underlying exact range-emptiness implementation
// (see package exact). Zero value defaults to exact.VariantAuto.
type ConfigRaw struct {
	K          uint32
	EREBackend exact.Variant
	backendSet bool
}

// WithEREBackend returns a copy of cfg with the chosen ERE backend.
func (cfg ConfigRaw) WithEREBackend(v exact.Variant) ConfigRaw {
	cfg.EREBackend = v
	cfg.backendSet = true
	return cfg
}

func (cfg ConfigRaw) backend() exact.Variant {
	if cfg.backendSet {
		return cfg.EREBackend
	}
	return exact.VariantAuto
}

// ConfigWithPolicy holds construction parameters for NewGreedyScanAREWithPolicy.
// RangeLen is forwarded to RangeLen-aware policies (e.g. FallbackPhantom);
// ignored otherwise.
//
// EREBackend selects the underlying exact range-emptiness implementation
// (see package exact). Zero value defaults to exact.VariantAuto.
type ConfigWithPolicy struct {
	K          uint32
	RangeLen   uint64
	Policy     hybridutil.FallbackPolicy
	EREBackend exact.Variant
	backendSet bool
}

// WithEREBackend returns a copy of cfg with the chosen ERE backend.
func (cfg ConfigWithPolicy) WithEREBackend(v exact.Variant) ConfigWithPolicy {
	cfg.EREBackend = v
	cfg.backendSet = true
	return cfg
}

func (cfg ConfigWithPolicy) backend() exact.Variant {
	if cfg.backendSet {
		return cfg.EREBackend
	}
	return exact.VariantAuto
}

// GreedyScanARE segments sorted keys into consecutive clusters using greedy
// spread-threshold + hierarchical merge. Clusters with spread ≤ 2^K use exact
// mode (FPR=0). Clusters with spread > 2^K (SODA territory) are sent to a
// trunc fallback instead, which is L-independent.
type GreedyScanARE struct {
	clusters  []hybridutil.ClusterFilter
	fallback  *hybridutil.FallbackFilter
	nClusters int
	nFallback int
	n         int
}

// NewGreedyScanARE builds a GreedyScanARE with the default AlwaysSODA fallback.
// keys must fit in keyBits bits (high bits above keyBits must be zero).
func NewGreedyScanARE(keys []uint64, keyBits uint32, cfg Config) (*GreedyScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	cp := append([]uint64(nil), keys...)
	return buildGreedy(cp, keyBits, cfg.K, 0, true, hybridutil.FallbackAlwaysSODA{}, cfg.backend())
}

// NewGreedyScanAREWithPolicy builds a GreedyScanARE with an explicit fallback
// policy. cfg.RangeLen is consulted only by RangeLen-aware policies.
func NewGreedyScanAREWithPolicy(keys []uint64, keyBits uint32, cfg ConfigWithPolicy) (*GreedyScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	cp := append([]uint64(nil), keys...)
	policy := cfg.Policy
	if policy == nil {
		policy = hybridutil.FallbackAlwaysTrunc{}
	}
	return buildGreedy(cp, keyBits, cfg.K, cfg.RangeLen, true, policy, cfg.backend())
}

// NewGreedyScanARERaw builds without merge and without fallback — pure greedy split only.
// keys need not be sorted; a copy is sorted internally.
func NewGreedyScanARERaw(keys []uint64, keyBits uint32, cfg ConfigRaw) (*GreedyScanARE, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)
	errutil.BugOn(cfg.K == 0 || cfg.K > 64, "K must be in (0, 64], got %d", cfg.K)
	cp := append([]uint64(nil), keys...)
	return buildGreedy(cp, keyBits, cfg.K, 0, false, hybridutil.FallbackAlwaysTrunc{}, cfg.backend())
}

func buildGreedy(keys []uint64, keyBits uint32, K uint32, rangeLen uint64, merge bool, policy hybridutil.FallbackPolicy, backend exact.Variant) (*GreedyScanARE, error) {
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
		Klocal := hybridutil.LocalK(K, n, len(seg.keys))
		spread := seg.maxKey - seg.minKey
		spreadBits := uint32(0)
		if spread > 0 {
			spreadBits = uint32(64 - mbits.LeadingZeros64(spread))
		}
		if spreadBits <= Klocal {
			exactSegs = append(exactSegs, seg)
		} else {
			fallbackKeys = append(fallbackKeys, seg.keys...)
		}
	}

	g := &GreedyScanARE{n: n}

	// Build exact-mode cluster filters with per-cluster K_local.
	g.clusters = make([]hybridutil.ClusterFilter, 0, len(exactSegs))
	for _, seg := range exactSegs {
		Kc := hybridutil.LocalK(K, n, len(seg.keys))
		f, err := are_adaptive.NewAdaptiveAREFromKWithBackend(seg.keys, keyBits, Kc, 0, backend)
		if err != nil {
			return nil, fmt.Errorf("cluster [%d, %d] build: %w", seg.minKey, seg.maxKey, err)
		}
		g.clusters = append(g.clusters, hybridutil.ClusterFilter{
			Filter: f,
			MinKey: seg.minKey,
			MaxKey: seg.maxKey,
		})
	}
	g.nClusters = len(g.clusters)

	// Build fallback filter (Trunc or SODA, per policy) with per-fallback K.
	if len(fallbackKeys) > 0 {
		Kfb := hybridutil.LocalK(K, n, len(fallbackKeys))
		fb, err := hybridutil.BuildFallback(fallbackKeys, keyBits, rangeLen, Kfb, policy, backend)
		if err != nil {
			return nil, err
		}
		g.fallback = fb
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
		return g.clusters[i].MaxKey >= lo
	})

	for i := idx; i < len(g.clusters) && g.clusters[i].MinKey <= hi; i++ {
		if !g.clusters[i].Filter.IsEmpty(lo, hi) {
			return false
		}
	}

	if g.fallback != nil {
		if !g.fallback.IsEmpty(lo, hi) {
			return false
		}
	}

	return true
}

func (g *GreedyScanARE) SizeInBits() uint64 {
	total := uint64(0)
	for _, c := range g.clusters {
		total += c.Filter.SizeInBits()
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
