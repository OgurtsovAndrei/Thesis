package testutils

// MeasureFPR computes FPR over pre-generated queries against sorted keys.
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
