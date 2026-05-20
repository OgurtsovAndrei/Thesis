package are_seg

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
	"testing"
)

// Build SegARE on real Wikipedia at the same params as the n=2^20 L=128 plot
// and report per-cluster details across the K sweep.
func TestSegAREWikiPlotBreakdown(t *testing.T) {
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
	fmt.Printf("\nN=%d  unique=%d  spread=2^%.2f\n\n", readN, len(keys),
		math.Log2(float64(keys[len(keys)-1]-keys[0])))

	for _, K := range []uint32{22, 24, 26, 27, 28, 30, 32} {
		f, err := NewSegAREFromK(keys, 32, K, 128)
		if err != nil {
			t.Fatalf("K=%d: %v", K, err)
		}
		nc, nfb, ntot := f.Stats()
		bpk := float64(f.SizeInBits()) / float64(ntot)
		fmt.Printf("  K=%d  clusters=%d  fallback=%d  total=%d  bpk=%.3f\n", K, nc, nfb, ntot, bpk)
		if nc > 0 && nc <= 12 {
			for i, c := range f.clusters {
				span := c.MaxKey - c.MinKey
				spanBits := uint32(math.Ceil(math.Log2(float64(span + 1))))
				fmt.Printf("      seg[%d]: min=%d max=%d span=%d (≈2^%d)  spanBits=%d\n",
					i, c.MinKey, c.MaxKey, span, int(math.Log2(float64(span+1))), spanBits)
			}
		}
	}
}
