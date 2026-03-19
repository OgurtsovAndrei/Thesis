package are_greedy_scan

import (
	"Thesis/bits"
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
