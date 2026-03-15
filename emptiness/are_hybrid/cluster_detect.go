package are_hybrid

import (
	"Thesis/bits"
	"sort"
)

type clusterSegment struct {
	keys   []bits.BitString
	minKey uint64
	maxKey uint64
}

// detectClusters splits pre-sorted keys into dense segments (clusters) and sparse leftovers (fallback).
//
// Algorithm: gap-based segmentation with percentile threshold.
//  1. Compute gaps between consecutive keys.
//  2. Find large gaps (>= P(gapPercentile)) that indicate segment boundaries.
//  3. Split key array at big gaps → contiguous segments.
//  4. Segments with >= minClusterFrac*n keys → clusters, rest → fallback.
//
// O(n log n) due to gap sorting; O(n) for everything else.
func detectClusters(keys []bits.BitString, gapPercentile float64, minClusterFrac float64) ([]clusterSegment, []bits.BitString) {
	n := len(keys)

	keys64 := make([]uint64, n)
	for i, k := range keys {
		keys64[i] = k.TrieUint64()
	}

	// Compute gaps
	gaps := make([]uint64, n-1)
	for i := 0; i < n-1; i++ {
		gaps[i] = keys64[i+1] - keys64[i]
	}

	// Percentile-based threshold: gaps at or above P(gapPercentile) are "big".
	gapsSorted := make([]uint64, len(gaps))
	copy(gapsSorted, gaps)
	sort.Slice(gapsSorted, func(i, j int) bool { return gapsSorted[i] < gapsSorted[j] })

	idx := int(gapPercentile * float64(len(gapsSorted)))
	if idx >= len(gapsSorted) {
		idx = len(gapsSorted) - 1
	}
	threshold := gapsSorted[idx]

	// Split at gaps >= threshold
	type segment struct {
		start, end int // inclusive indices into keys
	}
	var segments []segment
	segStart := 0
	for i := 0; i < len(gaps); i++ {
		if gaps[i] >= threshold {
			segments = append(segments, segment{segStart, i})
			segStart = i + 1
		}
	}
	segments = append(segments, segment{segStart, n - 1})

	// Classify: large segments → clusters, small → fallback
	sizeThreshold := int(minClusterFrac * float64(n))
	if sizeThreshold < 2 {
		sizeThreshold = 2
	}

	assigned := make([]bool, n)
	var clusters []clusterSegment
	for _, seg := range segments {
		size := seg.end - seg.start + 1
		if size < sizeThreshold {
			continue
		}
		clusters = append(clusters, clusterSegment{
			keys:   keys[seg.start : seg.end+1],
			minKey: keys64[seg.start],
			maxKey: keys64[seg.end],
		})
		for j := seg.start; j <= seg.end; j++ {
			assigned[j] = true
		}
	}

	var fallback []bits.BitString
	for i := 0; i < n; i++ {
		if !assigned[i] {
			fallback = append(fallback, keys[i])
		}
	}

	return clusters, fallback
}
