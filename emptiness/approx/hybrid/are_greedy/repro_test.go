package are_greedy

import (
	"fmt"
	"sort"
	"testing"
)

func TestVerifyGreedyClusters(t *testing.T) {
	n := 1000000
	keys := make([]uint64, 0, n)
	for c := 0; c < 10; c++ {
		center := uint64(c+1) * (1 << 60)
		for i := 0; i < 10000; i++ {
			keys = append(keys, center + uint64(i)*10)
		}
	}
	for i := 0; i < 900000; i++ {
		keys = append(keys, uint64(i) * (1 << 40) + 123456789)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	
	K := uint32(16)
	f, _ := NewGreedyScanARE(keys, 64, Config{K: K})
	nc, nf, _ := f.Stats()
	fmt.Printf("Greedy-ARE (with Klocal bug): clusters=%d, fallback_keys=%d\n", nc, nf)
	
	// Check how many clusters segmentBySpreadRefs actually found before filtering
	refs := segmentBySpreadRefs(keys, K)
	fmt.Printf("segmentBySpreadRefs found %d raw segments\n", len(refs))
}
