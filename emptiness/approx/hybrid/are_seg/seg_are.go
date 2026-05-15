package are_seg

import (
	"Thesis/emptiness/approx/are_adaptive"
	"Thesis/emptiness/approx/are_trunc"
	"Thesis/emptiness/approx/hybrid/hybridutil"
	"Thesis/emptiness/exact"
	"Thesis/utils/errutil"
	"fmt"
	"math"
	"sort"
)

// SegARE is a hybrid approximate range emptiness filter.
//
// Dense segments (≥ segMinPts=256 keys with average gap ≤ L/ε) go into
// AdaptiveARE sub-filters (exact mode when spread fits in 2^K_local bits,
// SODA hash otherwise). All other keys go to an AlwaysSODA fallback.
//
// Segmentation uses 1D DBSCAN without border expansion:
//   - δ = segMinPts · L/ε
//   - minimum segment size = segMinPts (implicit from core-point condition)
type SegARE struct {
	clusters  []hybridutil.ClusterFilter
	fallback  *hybridutil.FallbackFilter
	nClusters int
	nFallback int
	n         int
}

// NewSegARE builds SegARE for the given sorted keys.
//
// rangeLen is the query half-open length L (query covers [a, a+L-1]).
// epsilon is the target false positive rate ε ∈ (0, 1).
// keyBits is the number of significant key bits (≤ 64).
func NewSegARE(keys []uint64, keyBits uint32, rangeLen uint64, epsilon float64) (*SegARE, error) {
	errutil.BugOn(keyBits == 0 || keyBits > 64, "keyBits must be in [1,64], got %d", keyBits)
	errutil.BugOn(epsilon <= 0 || epsilon >= 1, "epsilon must be in (0,1), got %f", epsilon)

	n := len(keys)
	if n == 0 {
		return &SegARE{}, nil
	}

	K := kFromParams(n, rangeLen, epsilon)
	eps := segEps(rangeLen, epsilon)

	return newSegARE(keys, keyBits, K, rangeLen, eps, hybridutil.FallbackAlwaysSODA{}, exact.VariantAuto)
}

func newSegARE(keys []uint64, keyBits, K uint32, rangeLen, segDelta uint64, policy hybridutil.FallbackPolicy, backend exact.Variant) (*SegARE, error) {
	n := len(keys)
	s := &SegARE{n: n}

	if n < 2 {
		if n > 0 {
			fb, err := are_trunc.NewTruncAREFromKWithBackend(keys, keyBits, K, backend)
			if err != nil {
				return nil, fmt.Errorf("fallback build: %w", err)
			}
			s.fallback = &hybridutil.FallbackFilter{Trunc: fb, N: n}
			s.nFallback = n
		}
		return s, nil
	}

	segs, fallbackKeys := detectSegments(keys, segDelta)

	s.clusters = make([]hybridutil.ClusterFilter, 0, len(segs))
	for _, seg := range segs {
		Kc := hybridutil.LocalK(K, n, len(seg.keys))
		f, err := are_adaptive.NewAdaptiveAREFromKWithBackend(seg.keys, keyBits, Kc, 0, backend)
		if err != nil {
			return nil, fmt.Errorf("segment [%d,%d] build: %w", seg.minKey, seg.maxKey, err)
		}
		s.clusters = append(s.clusters, hybridutil.ClusterFilter{
			Filter: f,
			MinKey: seg.minKey,
			MaxKey: seg.maxKey,
		})
	}
	s.nClusters = len(s.clusters)

	if len(fallbackKeys) > 0 {
		Kfb := hybridutil.LocalK(K, n, len(fallbackKeys))
		fb, err := hybridutil.BuildFallback(fallbackKeys, keyBits, rangeLen, Kfb, policy, backend)
		if err != nil {
			return nil, err
		}
		s.fallback = fb
		s.nFallback = len(fallbackKeys)
	}

	return s, nil
}

func (s *SegARE) IsEmpty(lo, hi uint64) bool {
	if s.n == 0 {
		return true
	}

	idx := sort.Search(len(s.clusters), func(i int) bool {
		return s.clusters[i].MaxKey >= lo
	})
	for i := idx; i < len(s.clusters) && s.clusters[i].MinKey <= hi; i++ {
		if !s.clusters[i].Filter.IsEmpty(lo, hi) {
			return false
		}
	}

	if s.fallback != nil && !s.fallback.IsEmpty(lo, hi) {
		return false
	}

	return true
}

func (s *SegARE) SizeInBits() uint64 {
	var total uint64
	for _, c := range s.clusters {
		total += c.Filter.SizeInBits()
	}
	if s.fallback != nil {
		total += s.fallback.SizeInBits()
	}
	total += uint64(len(s.clusters)) * 128
	return total
}

func (s *SegARE) Stats() (numSegments, fallbackKeys, totalKeys int) {
	return s.nClusters, s.nFallback, s.n
}

// NewSegAREFromK builds SegARE with an explicit fingerprint width K.
// rangeLen is used only to compute δ; it does not affect K.
func NewSegAREFromK(keys []uint64, keyBits, K uint32, rangeLen uint64) (*SegARE, error) {
	errutil.BugOn(keyBits == 0 || keyBits > 64, "keyBits must be in [1,64], got %d", keyBits)
	errutil.BugOn(K == 0 || K > 64, "K must be in (0,64], got %d", K)

	n := len(keys)
	if n == 0 {
		return &SegARE{}, nil
	}

	// δ = segMinPts · L/ε ≈ segMinPts · 2^K / n   (since 2^K ≈ n·L/ε)
	var pow float64
	if K >= 64 {
		pow = float64(^uint64(0)) + 1
	} else {
		pow = float64(uint64(1) << K)
	}
	v := float64(segMinPts) * pow / float64(n)
	var eps uint64
	switch {
	case v < 1:
		eps = 1
	case v >= float64(math.MaxUint64):
		eps = math.MaxUint64
	default:
		eps = uint64(v)
	}

	return newSegARE(keys, keyBits, K, rangeLen, eps, hybridutil.FallbackAlwaysSODA{}, exact.VariantAuto)
}

// NewSegAREFromKWithBackend is NewSegAREFromK with an explicit ERE backend.
func NewSegAREFromKWithBackend(keys []uint64, keyBits, K uint32, rangeLen uint64, backend exact.Variant) (*SegARE, error) {
	errutil.BugOn(keyBits == 0 || keyBits > 64, "keyBits must be in [1,64], got %d", keyBits)
	errutil.BugOn(K == 0 || K > 64, "K must be in (0,64], got %d", K)

	n := len(keys)
	if n == 0 {
		return &SegARE{}, nil
	}

	var pow float64
	if K >= 64 {
		pow = float64(^uint64(0)) + 1
	} else {
		pow = float64(uint64(1) << K)
	}
	v := float64(segMinPts) * pow / float64(n)
	var eps uint64
	switch {
	case v < 1:
		eps = 1
	case v >= float64(math.MaxUint64):
		eps = math.MaxUint64
	default:
		eps = uint64(v)
	}

	return newSegARE(keys, keyBits, K, rangeLen, eps, hybridutil.FallbackAlwaysSODA{}, backend)
}

// segEps computes the DBSCAN neighbourhood radius δ = segMinPts · L/ε.
func segEps(rangeLen uint64, epsilon float64) uint64 {
	v := float64(segMinPts) * float64(rangeLen) / epsilon
	if v < 1 {
		return 1
	}
	if v >= float64(math.MaxUint64) {
		return math.MaxUint64
	}
	return uint64(v)
}

// kFromParams computes K = ceil(log2(n · (L+1) / ε)).
func kFromParams(n int, rangeLen uint64, epsilon float64) uint32 {
	k := uint32(math.Ceil(math.Log2(float64(n) * (float64(rangeLen) + 1) / epsilon)))
	if k == 0 {
		k = 1
	}
	if k > 64 {
		k = 64
	}
	return k
}
