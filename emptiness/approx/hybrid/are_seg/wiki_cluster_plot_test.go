package are_seg

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
	"testing"
)

// Quick analysis: how does SegARE see Wikipedia at the exact plot params
// (n = 2^20, L = 128, ε = 0.01). Also report spread and exact-mode budget.
func TestClusterStatsWikiPlot(t *testing.T) {
	const wikiPath = "/Users/andrei.ogurtsov/Thesis-Bench-industry/bench/sosd_data/wiki_ts_200M_uint64"
	f, err := os.Open(wikiPath)
	if err != nil {
		t.Skipf("no SOSD wiki: %v", err)
	}
	defer f.Close()
	var count uint64
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		t.Fatal(err)
	}
	readN := 1 << 20
	if int(count) < readN {
		readN = int(count)
	}
	raw := make([]uint64, readN)
	if err := binary.Read(f, binary.LittleEndian, raw); err != nil {
		t.Fatal(err)
	}
	sort.Slice(raw, func(i, j int) bool { return raw[i] < raw[j] })
	j := 0
	for i := 1; i < len(raw); i++ {
		if raw[i] != raw[j] {
			j++
			raw[j] = raw[i]
		}
	}
	keys := raw[:j+1]
	n := len(keys)
	spread := keys[n-1] - keys[0]
	spreadBits := uint32(64 - math.Floor(math.Log2(1)) - 0)
	_ = spreadBits

	fmt.Printf("\nN=%d  unique=%d  spread=%d (≈ 2^%.2f)  min=%d  max=%d\n",
		readN, n, spread, math.Log2(float64(spread)), keys[0], keys[n-1])

	for _, L := range []uint64{1, 16, 128, 1024, 4096, 16384, 65536} {
		eps := segEps(L, 0.01)
		segs, fallback := detectSegmentsN(keys, eps, segMinPts)
		nInCluster := n - len(fallback)
		fmt.Printf("  L=%6d  δ=%14d  clusters=%3d  in_cluster=%7d (%5.1f%%)  fallback=%7d",
			L, eps, len(segs), nInCluster,
			100*float64(nInCluster)/float64(n), len(fallback))
		if len(segs) > 0 {
			fmt.Printf("  first_seg_size=%d", len(segs[0].keys))
		}
		fmt.Println()
	}

	// For the actual plot params: L=128, K sweep. Show what K's hit exact mode.
	L := uint64(128)
	eps := segEps(L, 0.01)
	segs, _ := detectSegmentsN(keys, eps, segMinPts)
	fmt.Printf("\nAt L=128, ε=0.01: clusters=%d\n", len(segs))
	for i, s := range segs {
		span := s.maxKey - s.minKey
		spanBits := math.Ceil(math.Log2(float64(span + 1)))
		fmt.Printf("  cluster[%d]: size=%d  span=%d (≈ 2^%.2f)  bits_needed=%d (exact-mode requires K_local ≥ %.0f)\n",
			i, len(s.keys), span, math.Log2(float64(span+1)), int(spanBits), spanBits)
		if i >= 5 {
			fmt.Printf("  ...and %d more\n", len(segs)-i-1)
			break
		}
	}
}
