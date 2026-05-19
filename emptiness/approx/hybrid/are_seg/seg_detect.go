package are_seg

const segMinPts = 512

type segment struct {
	keys   []uint64
	minKey uint64
	maxKey uint64
}

// detectSegments partitions sorted keys into dense segments and fallback.
//
// Single forward sweep collects (start, end) index ranges where the sliding
// window of width eps contains >= segMinPts keys. Adjacent ranges within eps
// of each other are merged only when the combined span still fits the
// exact-mode budget reported by thresholdFn(m). All keys inside a merged
// range form a segment; everything else goes to fallback.
//
// thresholdFn(m) returns 2^K_local for a segment of m keys, i.e. the largest
// span that AdaptiveARE can encode in exact mode.
func detectSegments(keys []uint64, eps uint64, thresholdFn func(m int) uint64) ([]segment, []uint64) {
	n := len(keys)
	if n < 2 {
		return nil, append([]uint64(nil), keys...)
	}

	type run struct{ start, end int }
	var runs []run

	left := 0
	runStart := -1
	for right := 0; right < n; right++ {
		for keys[right]-keys[left] > eps {
			left++
		}
		if right-left+1 >= segMinPts {
			if runStart < 0 {
				runStart = left
			}
		} else if runStart >= 0 {
			runs = append(runs, run{runStart, right - 1})
			runStart = -1
		}
	}
	if runStart >= 0 {
		runs = append(runs, run{runStart, n - 1})
	}

	// Merge runs within eps of each other, but only if the resulting cluster
	// still satisfies the exact-mode condition. Otherwise keep them separate
	// to preserve the FPR ≤ ε guarantee.
	merged := make([]run, 0, len(runs))
	for _, r := range runs {
		if len(merged) > 0 {
			last := &merged[len(merged)-1]
			if keys[r.start]-keys[last.end] <= eps {
				combinedSpan := keys[r.end] - keys[last.start]
				combinedM := r.end - last.start + 1
				if combinedSpan < thresholdFn(combinedM) {
					last.end = r.end
					continue
				}
			}
		}
		merged = append(merged, r)
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
