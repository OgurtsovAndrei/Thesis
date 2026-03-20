package testutils

import "sort"

// MeasureFPR computes FPR over pre-generated queries against sorted keys.
// Queries that contain at least one key are skipped entirely.
// check(a, b) should return true if the filter says [a,b] is empty.
func MeasureFPR(sortedKeys []uint64, queries [][2]uint64, check func(a, b uint64) bool) float64 {
	fp, total := 0, 0
	for _, q := range queries {
		a, b := q[0], q[1]
		if b < a {
			continue
		}
		if !GroundTruth(sortedKeys, a, b) {
			continue
		}
		total++
		if !check(a, b) {
			fp++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(fp) / float64(total)
}

// MeasureFPRBatch computes FPR using a batch query function.
// It first filters to truly-empty queries, calls queryBatch once, then counts false positives.
func MeasureFPRBatch(sortedKeys []uint64, queries [][2]uint64, queryBatch func([][2]uint64) []bool) float64 {
	var emptyQueries [][2]uint64
	for _, q := range queries {
		a, b := q[0], q[1]
		if b < a {
			continue
		}
		if !GroundTruth(sortedKeys, a, b) {
			continue
		}
		emptyQueries = append(emptyQueries, q)
	}
	if len(emptyQueries) == 0 {
		return 0
	}
	results := queryBatch(emptyQueries)
	fp := 0
	for _, isEmpty := range results {
		if !isEmpty {
			fp++
		}
	}
	return float64(fp) / float64(len(emptyQueries))
}

// MeasureFPRShrink computes FPR with shrink mode: when a query [a,b] contains
// a key X, it is narrowed to [a, X-1] instead of being discarded. This avoids
// survivorship bias toward inter-cluster gaps at large range lengths.
// Queries where the first key equals a (no left gap) are still skipped.
func MeasureFPRShrink(sortedKeys []uint64, queries [][2]uint64, check func(a, b uint64) bool) float64 {
	fp, total := 0, 0
	for _, q := range queries {
		a, b := q[0], q[1]
		if b < a {
			continue
		}
		idx := sort.Search(len(sortedKeys), func(i int) bool { return sortedKeys[i] >= a })
		if idx < len(sortedKeys) && sortedKeys[idx] <= b {
			// Non-empty: shrink to the left gap [a, firstKey-1]
			firstKey := sortedKeys[idx]
			if firstKey == a {
				continue // key at the very start, no left gap
			}
			b = firstKey - 1
		}
		total++
		if !check(a, b) {
			fp++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(fp) / float64(total)
}
