package are_seg

const segMinPts = 256

type segment struct {
	keys   []uint64
	minKey uint64
	maxKey uint64
}

// detectSegments partitions sorted keys into dense segments and fallback.
//
// Algorithm: 1D DBSCAN without border expansion and without a minClusterSize
// post-filter. Only core-point runs (and their merges) form segments; all
// other keys go to fallback.
//
//  1. Forward + reverse sweep identifies core points: those with ≥ segMinPts
//     neighbours within eps.
//  2. Contiguous core-point runs form segment cores.
//  3. Adjacent cores within eps are merged.
//  4. No border expansion: non-core points always go to fallback.
func detectSegments(keys []uint64, eps uint64) ([]segment, []uint64) {
	n := len(keys)
	if n < 2 {
		return nil, append([]uint64(nil), keys...)
	}

	isCore := make([]bool, n)

	left := 0
	for right := 0; right < n; right++ {
		for keys[right]-keys[left] > eps {
			left++
		}
		if right-left+1 >= segMinPts {
			isCore[right] = true
		}
	}
	right := n - 1
	for left := n - 1; left >= 0; left-- {
		for keys[right]-keys[left] > eps {
			right--
		}
		if right-left+1 >= segMinPts {
			isCore[left] = true
		}
	}

	type run struct{ start, end int }
	var runs []run
	for i := 0; i < n; {
		if !isCore[i] {
			i++
			continue
		}
		start := i
		for i+1 < n && isCore[i+1] && keys[i+1]-keys[i] <= eps {
			i++
		}
		runs = append(runs, run{start, i})
		i++
	}

	merged := make([]run, 0, len(runs))
	for _, r := range runs {
		if len(merged) > 0 && keys[r.start]-keys[merged[len(merged)-1].end] <= eps {
			merged[len(merged)-1].end = r.end
		} else {
			merged = append(merged, r)
		}
	}

	assigned := make([]bool, n)
	segs := make([]segment, 0, len(merged))
	for _, r := range merged {
		segs = append(segs, segment{
			keys:   keys[r.start : r.end+1],
			minKey: keys[r.start],
			maxKey: keys[r.end],
		})
		for j := r.start; j <= r.end; j++ {
			assigned[j] = true
		}
	}

	var fallback []uint64
	for i, k := range keys {
		if !assigned[i] {
			fallback = append(fallback, k)
		}
	}

	return segs, fallback
}
