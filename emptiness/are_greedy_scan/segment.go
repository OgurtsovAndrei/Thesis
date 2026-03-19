package are_greedy_scan

import (
	"Thesis/bits"
	"math"
)

type segment struct {
	keys   []bits.BitString
	minKey uint64
	maxKey uint64
}

// segmentBySpread scans sorted keys left-to-right, creating a new cluster
// whenever the spread would exceed 2^K. O(n) time, every key assigned.
func segmentBySpread(keys []bits.BitString, K uint32) []segment {
	n := len(keys)
	if n == 0 {
		return nil
	}

	var maxSpread uint64
	if K >= 64 {
		maxSpread = ^uint64(0)
	} else {
		maxSpread = (uint64(1) << K) - 1
	}

	var segments []segment
	clusterStart := 0
	startVal := keys[0].TrieUint64()

	for i := 1; i < n; i++ {
		curVal := keys[i].TrieUint64()
		if curVal-startVal > maxSpread {
			segments = append(segments, buildSegment(keys[clusterStart:i]))
			clusterStart = i
			startVal = curVal
		}
	}
	segments = append(segments, buildSegment(keys[clusterStart:]))

	return segments
}

func buildSegment(keys []bits.BitString) segment {
	return segment{
		keys:   keys,
		minKey: keys[0].TrieUint64(),
		maxKey: keys[len(keys)-1].TrieUint64(),
	}
}

// estimateCost estimates the storage cost (bits) of a segment given parameter K.
// A cluster with spread S and n keys uses:
//   - exact (ERE) mode if S < 2^K: cost ≈ n * log2(2^K / n) bits
//   - SODA mode if S >= 2^K:       cost ≈ n * K bits
//
// Plus 128 bits for min/max boundary overhead per cluster.
func estimateCost(seg segment, K uint32) float64 {
	n := float64(len(seg.keys))
	if n == 0 {
		return 128
	}
	spread := seg.maxKey - seg.minKey + 1
	var universe float64
	if K >= 64 {
		universe = math.Exp2(64)
	} else {
		universe = math.Exp2(float64(K))
	}
	var dataCost float64
	if float64(spread) < universe {
		// ERE encoding: n * log2(universe / n)
		ratio := universe / n
		if ratio < 1 {
			ratio = 1
		}
		dataCost = n * math.Log2(ratio)
	} else {
		// SODA (hash-based): n * K bits
		dataCost = n * float64(K)
	}
	return dataCost + 128
}

// mergeSegments combines two adjacent segments into one.
// The keys slices must be contiguous in the original array (seg b follows seg a).
func mergeSegments(a, b segment) segment {
	merged := make([]bits.BitString, len(a.keys)+len(b.keys))
	copy(merged, a.keys)
	copy(merged[len(a.keys):], b.keys)
	return segment{
		keys:   merged,
		minKey: a.minKey,
		maxKey: b.maxKey,
	}
}

// mergeSmallClusters performs a greedy bottom-up merge pass over segments.
// Two adjacent segments A and B are merged whenever cost(A∪B) < cost(A)+cost(B).
// The pass repeats until no more beneficial merges are found.
func mergeSmallClusters(segments []segment, K uint32) []segment {
	for {
		merged := false
		for i := 0; i < len(segments)-1; i++ {
			costA := estimateCost(segments[i], K)
			costB := estimateCost(segments[i+1], K)
			combined := mergeSegments(segments[i], segments[i+1])
			costAB := estimateCost(combined, K)
			if costAB < costA+costB {
				segments[i] = combined
				segments = append(segments[:i+1], segments[i+2:]...)
				merged = true
			}
		}
		if !merged {
			break
		}
	}
	return segments
}
