package hybridutil

import (
	"math"
	mbits "math/bits"
	"math/rand"
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
	return TruncSafe(keys, K)
}
func (FallbackAuto) String() string { return "Auto" }

// FallbackAlwaysTrunc always uses TruncARE regardless of data distribution.
type FallbackAlwaysTrunc struct{}

func (FallbackAlwaysTrunc) useTrunc(_ []uint64, _ uint32, _ uint64) bool { return true }
func (FallbackAlwaysTrunc) String() string                               { return "Trunc" }

// FallbackAlwaysSODA always uses AdaptiveARE (SODA) regardless of data distribution.
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

// TruncSafe checks whether TruncARE fallback is safe: P5 gap > phantomSize.
func TruncSafe(keys64 []uint64, K uint32) bool {
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
	return Quickselect(gaps, idx) > phantomSize
}

// Quickselect returns the k-th smallest element (0-indexed).
// Mutates the input slice. Average O(n), worst O(n²).
func Quickselect(a []uint64, k int) uint64 {
	rng := rand.New(rand.NewSource(42))
	lo, hi := 0, len(a)-1
	for lo < hi {
		pivot := a[lo+rng.Intn(hi-lo+1)]
		i, j := lo, hi
		for i <= j {
			for a[i] < pivot {
				i++
			}
			for a[j] > pivot {
				j--
			}
			if i <= j {
				a[i], a[j] = a[j], a[i]
				i++
				j--
			}
		}
		if k <= j {
			hi = j
		} else if k >= i {
			lo = i
		} else {
			break
		}
	}
	return a[k]
}
