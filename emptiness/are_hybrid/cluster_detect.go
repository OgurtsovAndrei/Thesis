package are_hybrid

import (
	"Thesis/bits"
	"math/rand"
)

type clusterSegment struct {
	keys   []bits.BitString
	minKey uint64
	maxKey uint64
}

// detectClusters splits pre-sorted keys into dense segments (clusters) and sparse leftovers (fallback).
//
// Algorithm: 1D DBSCAN-inspired segmentation, O(n).
//
//  1. Compute gaps between consecutive sorted keys.
//  2. Derive eps = gap at the gapPercentile-th position (via quickselect).
//  3. Split at gaps strictly greater than eps — consecutive keys with gap <= eps
//     stay in the same segment. Using strict '>' ensures that data with uniform
//     spacing (all gaps equal) forms a single segment rather than n segments of 1.
//  4. Segments with >= minPts keys become clusters; the rest go to fallback.
//     minPts = max(2, minClusterFrac * n).
func detectClusters(keys []bits.BitString, gapPercentile float64, minClusterFrac float64) ([]clusterSegment, []bits.BitString) {
	n := len(keys)

	keys64 := make([]uint64, n)
	for i, k := range keys {
		keys64[i] = k.TrieUint64()
	}

	// Compute gaps between consecutive keys.
	gaps := make([]uint64, n-1)
	for i := 0; i < n-1; i++ {
		gaps[i] = keys64[i+1] - keys64[i]
	}

	// eps = gap at the given percentile, computed via quickselect O(n).
	k := int(gapPercentile * float64(len(gaps)))
	if k >= len(gaps) {
		k = len(gaps) - 1
	}
	gapsCopy := make([]uint64, len(gaps))
	copy(gapsCopy, gaps)
	eps := quickselect(gapsCopy, k)

	// minPts: minimum segment size to qualify as a cluster.
	minPts := int(minClusterFrac * float64(n))
	if minPts < 2 {
		minPts = 2
	}

	// Split at gaps strictly greater than eps.
	type segment struct {
		start, end int // inclusive indices into keys
	}
	var segments []segment
	segStart := 0
	for i := 0; i < len(gaps); i++ {
		if gaps[i] > eps {
			segments = append(segments, segment{segStart, i})
			segStart = i + 1
		}
	}
	segments = append(segments, segment{segStart, n - 1})

	// Classify: segments with enough points become clusters, rest go to fallback.
	assigned := make([]bool, n)
	var clusters []clusterSegment
	for _, seg := range segments {
		size := seg.end - seg.start + 1
		if size < minPts {
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

// quickselect returns the k-th smallest element (0-indexed).
// Mutates the input slice. Average O(n), worst O(n²).
func quickselect(a []uint64, k int) uint64 {
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
