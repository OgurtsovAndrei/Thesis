package are_hybrid_scan

import (
	"Thesis/bits"
	"math"
	mbits "math/bits"
)

// FallbackPolicy decides whether to use TruncARE or Adaptive/SODA for fallback keys.
// The interface is sealed: only types defined in this package can implement it.
type FallbackPolicy interface {
	useTrunc(keys []bits.BitString, K uint32, rangeLen uint64) bool
	String() string
}

// FallbackAuto uses the truncSafe heuristic (P5 gap vs phantom size).
type FallbackAuto struct{}

func (FallbackAuto) useTrunc(keys []bits.BitString, K uint32, _ uint64) bool {
	if len(keys) < 2 {
		return true
	}
	keys64 := make([]uint64, len(keys))
	for i, k := range keys {
		keys64[i] = k.TrieUint64()
	}
	return truncSafe(keys64, K)
}
func (FallbackAuto) String() string { return "Auto" }

// FallbackAlwaysTrunc always uses TruncARE regardless of data distribution.
type FallbackAlwaysTrunc struct{}

func (FallbackAlwaysTrunc) useTrunc(_ []bits.BitString, _ uint32, _ uint64) bool { return true }
func (FallbackAlwaysTrunc) String() string                                        { return "Trunc" }

// FallbackAlwaysSODA always uses Adaptive/SODA regardless of data distribution.
type FallbackAlwaysSODA struct{}

func (FallbackAlwaysSODA) useTrunc(_ []bits.BitString, _ uint32, _ uint64) bool { return false }
func (FallbackAlwaysSODA) String() string                                        { return "SODA" }

// FallbackEstimateFPR uses trunc when estimated FPR (n/2^K) ≤ Epsilon, else SODA.
// Assumes keys are uniformly distributed in truncated space — works well on uniform data
// but underestimates FPR on clustered distributions like OSM.
type FallbackEstimateFPR struct{ Epsilon float64 }

func (f FallbackEstimateFPR) useTrunc(keys []bits.BitString, K uint32, _ uint64) bool {
	return float64(len(keys))/math.Pow(2, float64(K)) <= f.Epsilon
}
func (f FallbackEstimateFPR) String() string { return "EstFPR" }

// FallbackGapFraction uses trunc when the span-weighted fraction of gaps
// smaller than phantomSize (= spread / 2^K) is at most Epsilon.
// A random empty query lands in gap_i with probability g_i/span, so weighting
// by gap size gives the true expected FPR of trunc: Σ g_i/span for g_i < phantomSize.
// Works well for random queries but does not account for near-key query bias.
type FallbackGapFraction struct{ Epsilon float64 }

func (f FallbackGapFraction) useTrunc(keys []bits.BitString, K uint32, _ uint64) bool {
	n := len(keys)
	if n < 2 {
		return true
	}

	keys64 := make([]uint64, n)
	for i, k := range keys {
		keys64[i] = k.TrieUint64()
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

	var smallSpan uint64
	for i := 0; i < n-1; i++ {
		g := keys64[i+1] - keys64[i]
		if g <= phantomSize {
			smallSpan += g
		}
	}
	return float64(smallSpan)/float64(spread) <= f.Epsilon
}
func (f FallbackGapFraction) String() string { return "GapFrac" }

// FallbackPhantom uses trunc when phantomSize (= spread / 2^K) < rangeLen.
// Near-key queries sit at distance ~O(L) from a key. If phantomSize ≥ L, those
// queries land inside the phantom region and always produce false positives.
// This correctly rejects trunc on wide-spread data (OSM, uniform with large span)
// at low K, and approves trunc once K is large enough that phantoms are smaller
// than the query range.
type FallbackPhantom struct{}

func (FallbackPhantom) useTrunc(keys []bits.BitString, K uint32, rangeLen uint64) bool {
	n := len(keys)
	if n < 2 {
		return true
	}

	spread := keys[n-1].TrieUint64() - keys[0].TrieUint64()
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
// Trunc breaks when the smallest gaps (P5) are smaller than phantom_size = spread / 2^K.
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
	p5Gap := quickselect(gaps, idx)

	return p5Gap > phantomSize
}
