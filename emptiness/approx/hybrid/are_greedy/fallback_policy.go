package are_greedy

import (
	"math"
	mbits "math/bits"
)

// FallbackPolicy decides whether to use TruncARE or AdaptiveARE (SODA) for
// fallback keys. The interface is sealed: only types defined in this package
// can implement it.
type FallbackPolicy interface {
	useTrunc(keys []uint64, K uint32, rangeLen uint64) bool
	String() string
}

// FallbackAuto uses the truncSafe heuristic (P5 gap vs phantom size).
type FallbackAuto struct{}

func (FallbackAuto) useTrunc(keys []uint64, K uint32, _ uint64) bool {
	if len(keys) < 2 {
		return true
	}
	return truncSafe(keys, K)
}
func (FallbackAuto) String() string { return "Auto" }

// FallbackAlwaysTrunc always uses TruncARE regardless of data distribution.
type FallbackAlwaysTrunc struct{}

func (FallbackAlwaysTrunc) useTrunc(_ []uint64, _ uint32, _ uint64) bool { return true }
func (FallbackAlwaysTrunc) String() string                               { return "Trunc" }

// FallbackAlwaysSODA always uses AdaptiveARE (SODA) regardless of data
// distribution.
type FallbackAlwaysSODA struct{}

func (FallbackAlwaysSODA) useTrunc(_ []uint64, _ uint32, _ uint64) bool { return false }
func (FallbackAlwaysSODA) String() string                               { return "SODA" }

// FallbackEstimateFPR uses trunc when estimated FPR (n/2^K) ≤ Epsilon, else SODA.
type FallbackEstimateFPR struct{ Epsilon float64 }

func (f FallbackEstimateFPR) useTrunc(keys []uint64, K uint32, _ uint64) bool {
	return float64(len(keys))/math.Pow(2, float64(K)) <= f.Epsilon
}
func (f FallbackEstimateFPR) String() string { return "EstFPR" }

// FallbackGapFraction uses trunc when the span-weighted fraction of gaps
// smaller than phantomSize (= spread / 2^K) is at most Epsilon.
type FallbackGapFraction struct{ Epsilon float64 }

func (f FallbackGapFraction) useTrunc(keys []uint64, K uint32, _ uint64) bool {
	n := len(keys)
	if n < 2 {
		return true
	}

	spread := keys[n-1] - keys[0]
	if spread == 0 {
		return true
	}

	spreadBits := uint32(64 - mbits.LeadingZeros64(spread))
	if spreadBits <= K {
		return true
	}

	phantomSize := spread >> K
	if phantomSize == 0 {
		phantomSize = 1
	}

	var smallSpan uint64
	for i := 0; i < n-1; i++ {
		g := keys[i+1] - keys[i]
		if g <= phantomSize {
			smallSpan += g
		}
	}
	return float64(smallSpan)/float64(spread) <= f.Epsilon
}
func (f FallbackGapFraction) String() string { return "GapFrac" }

// FallbackPhantom uses trunc when phantomSize (= spread / 2^K) < rangeLen.
type FallbackPhantom struct{}

func (FallbackPhantom) useTrunc(keys []uint64, K uint32, rangeLen uint64) bool {
	n := len(keys)
	if n < 2 {
		return true
	}

	spread := keys[n-1] - keys[0]
	if spread == 0 {
		return true
	}

	spreadBits := uint32(64 - mbits.LeadingZeros64(spread))
	if spreadBits <= K {
		return true
	}

	phantomSize := spread >> K
	return phantomSize < rangeLen
}
func (FallbackPhantom) String() string { return "Phantom" }

// truncSafe checks whether trunc fallback will work for the given keys.
// Trunc breaks when the smallest gaps (P5) are smaller than phantomSize = spread / 2^K.
func truncSafe(keys64 []uint64, K uint32) bool {
	n := len(keys64)
	if n < 2 {
		return true
	}

	spread := keys64[n-1] - keys64[0]
	if spread == 0 {
		return true
	}

	spreadBits := uint32(64 - mbits.LeadingZeros64(spread))
	if spreadBits <= K {
		return true
	}
	phantomSize := spread >> K
	if phantomSize == 0 {
		phantomSize = 1
	}

	gaps := make([]uint64, n-1)
	for i := 0; i < n-1; i++ {
		gaps[i] = keys64[i+1] - keys64[i]
	}
	idx := len(gaps) / 20
	if idx >= len(gaps) {
		idx = len(gaps) - 1
	}
	p5Gap := quickselectFP(gaps, idx)

	return p5Gap > phantomSize
}

// quickselectFP returns the k-th smallest element of arr (0-indexed) in-place.
// Renamed from quickselect to avoid colliding with any existing helper in
// this package.
func quickselectFP(arr []uint64, k int) uint64 {
	if len(arr) == 0 {
		return 0
	}
	lo, hi := 0, len(arr)-1
	for lo < hi {
		p := partition(arr, lo, hi)
		switch {
		case p == k:
			return arr[p]
		case p < k:
			lo = p + 1
		default:
			hi = p - 1
		}
	}
	return arr[lo]
}

func partition(arr []uint64, lo, hi int) int {
	pivot := arr[hi]
	i := lo
	for j := lo; j < hi; j++ {
		if arr[j] < pivot {
			arr[i], arr[j] = arr[j], arr[i]
			i++
		}
	}
	arr[i], arr[hi] = arr[hi], arr[i]
	return i
}
