package hybridutil

import (
	"Thesis/emptiness/approx/are_adaptive"
	"Thesis/emptiness/approx/are_trunc"
	"Thesis/emptiness/exact"
	"fmt"
	mbits "math/bits"
)

// ClusterFilter holds a dense sub-filter and its key range.
type ClusterFilter struct {
	Filter *are_adaptive.AdaptiveARE
	MinKey uint64
	MaxKey uint64
}

// FallbackFilter holds either a TruncARE or an AdaptiveARE for sparse keys.
type FallbackFilter struct {
	Trunc    *are_trunc.TruncARE
	Adaptive *are_adaptive.AdaptiveARE
	N        int
}

func (f *FallbackFilter) IsEmpty(a, b uint64) bool {
	if f.Trunc != nil {
		return f.Trunc.IsEmpty(a, b)
	}
	if f.Adaptive != nil {
		return f.Adaptive.IsEmpty(a, b)
	}
	return true
}

func (f *FallbackFilter) SizeInBits() uint64 {
	if f.Trunc != nil {
		return f.Trunc.SizeInBits()
	}
	if f.Adaptive != nil {
		return f.Adaptive.SizeInBits()
	}
	return 0
}

// LocalK rescales the global K (sized for nTotal keys) to the smaller K
// appropriate for a sub-filter holding only nLocal keys:
//
//	K_local = K_global - floor(log2(nTotal/nLocal))
func LocalK(K uint32, nTotal, nLocal int) uint32 {
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

// BuildFallback builds the fallback filter (TruncARE or AdaptiveARE/SODA)
// for sparse keys not assigned to any cluster.
func BuildFallback(keys []uint64, keyBits uint32, rangeLen uint64, K uint32, policy FallbackPolicy, backend exact.Variant) (*FallbackFilter, error) {
	if policy.useTrunc(keys, K, rangeLen) {
		fb, err := are_trunc.NewTruncAREFromKWithBackend(keys, keyBits, K, backend)
		if err != nil {
			return nil, fmt.Errorf("fallback trunc build: %w", err)
		}
		return &FallbackFilter{Trunc: fb, N: len(keys)}, nil
	}
	fb, err := are_adaptive.NewAdaptiveAREFromKWithBackend(keys, keyBits, K, 0, backend)
	if err != nil {
		return nil, fmt.Errorf("fallback adaptive build: %w", err)
	}
	return &FallbackFilter{Adaptive: fb, N: len(keys)}, nil
}
