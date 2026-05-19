package are_seg

import (
	"fmt"
	"sort"
	"testing"

	"Thesis/emptiness/approx/hybrid/hybridutil"
)

func TestVerifySegARE(t *testing.T) {
	n := 1000000
	keys := make([]uint64, 0, n)
	for c := 0; c < 10; c++ {
		center := uint64(c+1) * (1 << 60)
		for i := 0; i < 10000; i++ {
			keys = append(keys, center+uint64(i)*10)
		}
	}
	for i := 0; i < 900000; i++ {
		keys = append(keys, uint64(i)*(1<<40)+123456789)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	K := uint32(16)
	thresholdFn := func(m int) uint64 {
		kLocal := hybridutil.LocalK(K, n, m)
		if kLocal >= 64 {
			return ^uint64(0)
		}
		return uint64(1) << kLocal
	}

	// Original formula
	pow := float64(uint64(1) << K)
	v := float64(segMinPts) * pow / float64(n)
	fmt.Printf("SegARE: K=%d, n=%d, pow=%.0f, v=%.2f, eps=%d\n", K, n, pow, v, uint64(v))

	segs, _ := detectSegments(keys, uint64(v), thresholdFn)
	fmt.Printf("detectSegments (eps=%d) found %d clusters\n", uint64(v), len(segs))

	// Scan-ARE formula
	vFixed := 10.0 * pow / 256.0
	fmt.Printf("Scan-ARE eps: v=%.2f, eps=%d\n", vFixed, uint64(vFixed))
	segsFixed, _ := detectSegments(keys, uint64(vFixed), thresholdFn)
	fmt.Printf("detectSegments (eps=%d) found %d clusters\n", uint64(vFixed), len(segsFixed))
}
