package are_dp_scan

import (
	"Thesis/bits"
	"math"
)

type segment struct {
	keys   []bits.BitString
	minKey uint64
	maxKey uint64
}

func buildSegment(keys []bits.BitString) segment {
	return segment{
		keys:   keys,
		minKey: keys[0].TrieUint64(),
		maxKey: keys[len(keys)-1].TrieUint64(),
	}
}

// estimateCost returns the estimated storage cost in bits for a segment of
// nKeys keys with the given spread, using parameter K.
//
// Cost model:
//   - exact (ERE) mode when spread < 2^K: n * log2(2^K / n)
//   - SODA (hash) mode when spread >= 2^K: n * K
//
// Plus 128 bits per cluster for boundary overhead.
func estimateCost(nKeys int, spread uint64, K uint32) uint64 {
	if nKeys == 0 {
		return 128
	}
	n := float64(nKeys)
	var universe float64
	if K >= 64 {
		universe = math.Exp2(64)
	} else {
		universe = math.Exp2(float64(K))
	}

	var dataCost float64
	if float64(spread) < universe {
		ratio := universe / n
		if ratio < 1 {
			ratio = 1
		}
		dataCost = n * math.Log2(ratio)
	} else {
		dataCost = n * float64(K)
	}

	return uint64(math.Ceil(dataCost)) + 128
}

// segmentDP finds the optimal segmentation of sorted keys into consecutive
// clusters that minimizes total estimated bits, using dynamic programming.
//
// dp[i] = minimum total bits to encode keys[0..i-1]
// dp[i] = min over j in [0,i) of: dp[j] + estimateCost(i-j, spread(j,i-1), K)
//
// Time complexity: O(n²). For the key counts typical in benchmarks (≤100k),
// this is fast enough to serve as a gold-standard reference.
func segmentDP(keys []bits.BitString, K uint32) []segment {
	n := len(keys)
	if n == 0 {
		return nil
	}

	// dp[i] = min total cost for keys[0..i-1]; dp[0] = 0 (base case)
	dp := make([]uint64, n+1)
	// parent[i] = start index j of the last segment in the optimal solution
	parent := make([]int, n+1)
	// Initialise dp[i>0] to a safe upper bound: one singleton segment per key
	for i := 1; i <= n; i++ {
		dp[i] = dp[i-1] + estimateCost(1, 0, K)
		parent[i] = i - 1
	}

	vals := make([]uint64, n)
	for i, k := range keys {
		vals[i] = k.TrieUint64()
	}

	for i := 1; i <= n; i++ {
		for j := i - 1; j >= 0; j-- {
			spread := vals[i-1] - vals[j]
			cost := estimateCost(i-j, spread, K)
			if total := dp[j] + cost; total < dp[i] {
				dp[i] = total
				parent[i] = j
			}
		}
	}

	// Reconstruct segments by tracing parent pointers
	var boundaries []int
	pos := n
	for pos > 0 {
		boundaries = append(boundaries, pos)
		pos = parent[pos]
	}

	// boundaries is in reverse order; reverse it
	for lo, hi := 0, len(boundaries)-1; lo < hi; lo, hi = lo+1, hi-1 {
		boundaries[lo], boundaries[hi] = boundaries[hi], boundaries[lo]
	}

	segments := make([]segment, 0, len(boundaries))
	prev := 0
	for _, end := range boundaries {
		segments = append(segments, buildSegment(keys[prev:end]))
		prev = end
	}

	return segments
}
