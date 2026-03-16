package are_hybrid_scan

import (
	"Thesis/bits"
	"math/rand"
	"sort"
)

type clusterSegment struct {
	keys   []bits.BitString
	minKey uint64
	maxKey uint64
}

// detectClustersDBSCAN performs 1D DBSCAN on pre-sorted keys.
//
// On sorted data, density reachability reduces to a two-pointer sweep: O(n).
//
//  1. Sweep with two pointers to identify core points (those with >= minPts
//     neighbors within eps distance).
//  2. Contiguous runs of core points form cluster cores.
//  3. Non-core points adjacent to (within eps of) a core point are border points
//     and join the nearest cluster.
//  4. Clusters smaller than minClusterSize are dissolved into fallback.
//  5. Everything else is noise and goes to fallback.
//
// minPts is the DBSCAN core threshold (typically 5-10).
// minClusterSize is the post-filter: clusters below this size go to fallback.
func detectClustersDBSCAN(keys []bits.BitString, eps uint64, minPts int, minClusterSize int) ([]clusterSegment, []bits.BitString) {
	n := len(keys)
	if n < 2 {
		return nil, append([]bits.BitString{}, keys...)
	}

	keys64 := make([]uint64, n)
	for i, k := range keys {
		keys64[i] = k.TrieUint64()
	}

	// Phase 1: identify core points via two-pointer sweep.
	isCore := make([]bool, n)
	left := 0
	for right := 0; right < n; right++ {
		for keys64[right]-keys64[left] > eps {
			left++
		}
		if right-left+1 >= minPts {
			isCore[right] = true
		}
	}
	// Reverse sweep: the forward pass only marks the rightmost point of each
	// dense window. We need a backward pass to mark leftward core points too.
	right := n - 1
	for left := n - 1; left >= 0; left-- {
		for keys64[right]-keys64[left] > eps {
			right--
		}
		if right-left+1 >= minPts {
			isCore[left] = true
		}
	}

	// Phase 2: form clusters from density-connected core-point runs.
	// Two adjacent core points (by index) belong to the same cluster only if
	// their key-space distance is <= eps (density-reachable).
	type coreRun struct {
		start, end int // inclusive
	}
	var runs []coreRun
	i := 0
	for i < n {
		if !isCore[i] {
			i++
			continue
		}
		start := i
		for i+1 < n && isCore[i+1] && keys64[i+1]-keys64[i] <= eps {
			i++
		}
		runs = append(runs, coreRun{start, i})
		i++
	}

	// Merge runs that are within eps of each other (connected via border points).
	merged := make([]coreRun, 0, len(runs))
	for _, r := range runs {
		if len(merged) > 0 && keys64[r.start]-keys64[merged[len(merged)-1].end] <= eps {
			merged[len(merged)-1].end = r.end
		} else {
			merged = append(merged, r)
		}
	}

	// Phase 3: expand each cluster to include border points.
	assigned := make([]bool, n)
	var clusters []clusterSegment
	for _, r := range merged {
		lo := r.start
		for lo > 0 && keys64[r.start]-keys64[lo-1] <= eps {
			lo--
		}
		hi := r.end
		for hi < n-1 && keys64[hi+1]-keys64[r.end] <= eps {
			hi++
		}

		clusters = append(clusters, clusterSegment{
			keys:   keys[lo : hi+1],
			minKey: keys64[lo],
			maxKey: keys64[hi],
		})
		for j := lo; j <= hi; j++ {
			assigned[j] = true
		}
	}

	// Phase 4: post-filter — dissolve clusters smaller than minClusterSize.
	// Rebuild assigned from scratch using only kept clusters.
	var filtered []clusterSegment
	for i := range assigned {
		assigned[i] = false
	}
	for _, c := range clusters {
		if len(c.keys) < minClusterSize {
			continue
		}
		filtered = append(filtered, c)
		// Find index range via binary search on minKey.
		lo := sort.Search(n, func(j int) bool { return keys64[j] >= c.minKey })
		for j := lo; j < n && keys64[j] <= c.maxKey; j++ {
			assigned[j] = true
		}
	}

	// Phase 5: collect noise + dissolved small clusters (fallback).
	var fallback []bits.BitString
	for i := 0; i < n; i++ {
		if !assigned[i] {
			fallback = append(fallback, keys[i])
		}
	}

	return filtered, fallback
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
