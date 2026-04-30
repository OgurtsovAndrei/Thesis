package are_hybrid

import (
	"Thesis/bits"
	"Thesis/testutils"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

const mask60 = (uint64(1) << 60) - 1

func TestHybrid_ScalingDiagnostic(t *testing.T) {
	const (
		nClusters = 5
		unifFrac  = 0.15
		rangeLen  = uint64(1024)
		eps       = 0.01
	)

	for _, n := range []int{1 << 16, 1 << 18, 1 << 20} {
		t.Run(fmt.Sprintf("N=%d", n), func(t *testing.T) {
			rng := rand.New(rand.NewSource(99))
			rawKeys, _ := testutils.GenerateClusterDistribution(n, nClusters, unifFrac, rng)

			// Mask to 60 bits like the benchmark does
			seen := make(map[uint64]bool, len(rawKeys))
			keys := make([]uint64, 0, len(rawKeys))
			for _, k := range rawKeys {
				k &= mask60
				if !seen[k] {
					seen[k] = true
					keys = append(keys, k)
				}
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
			t.Logf("after mask60: %d keys (from %d raw)", len(keys), len(rawKeys))

			bs := make([]bits.BitString, len(keys))
			for i, k := range keys {
				bs[i] = bits.NewFromTrieUint64(k, 60)
			}

			// Check cluster detection directly
			segments, fallbackKeys := detectClusters(bs, 0.95, 0.01)
			t.Logf("detectClusters: %d segments, %d fallback keys (%.1f%%)",
				len(segments), len(fallbackKeys), 100*float64(len(fallbackKeys))/float64(len(keys)))
			for i, seg := range segments {
				t.Logf("  segment[%d]: %d keys, range [%x, %x], span=%d",
					i, len(seg.keys), seg.minKey, seg.maxKey, seg.maxKey-seg.minKey)
			}

			h, err := NewHybridARE(bs, rangeLen, eps)
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			nc, nf, nt := h.Stats()
			bpk := float64(h.SizeInBits()) / float64(nt)
			t.Logf("HybridARE: clusters=%d, fallback=%d, total=%d, BPK=%.2f, SizeInBits=%d",
				nc, nf, nt, bpk, h.SizeInBits())

			for i, c := range h.clusters {
				nKeys := 0
				for _, seg := range segments {
					if seg.minKey == c.minKey {
						nKeys = len(seg.keys)
						break
					}
				}
				cBPK := float64(c.filter.SizeInBits()) / float64(nKeys)
				t.Logf("  cluster[%d]: ~%d keys, range [%x, %x], SizeInBits=%d, BPK=%.2f",
					i, nKeys, c.minKey, c.maxKey, c.filter.SizeInBits(), cBPK)
			}
			if h.fallback != nil {
				fbBPK := float64(h.fallback.SizeInBits()) / float64(nf)
				t.Logf("  fallback: %d keys, SizeInBits=%d, BPK=%.2f",
					nf, h.fallback.SizeInBits(), fbBPK)
			}
		})
	}
}
