package are_hybrid

import (
	"Thesis/bits"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

// generateZipfianKeys mirrors the industry benchmark generator.
// 100 prefixes (40-bit), top 10% get 80% of keys, suffix is random 20-bit.
func generateZipfianKeys(n, nPrefixes int, rng *rand.Rand) []uint64 {
	prefixes := make([]uint64, nPrefixes)
	for i := range prefixes {
		prefixes[i] = rng.Uint64() & ((1 << 40) - 1)
	}
	sort.Slice(prefixes, func(i, j int) bool { return prefixes[i] < prefixes[j] })

	nTop := nPrefixes / 10
	nHot := n * 80 / 100

	seen := make(map[uint64]bool, n)
	keys := make([]uint64, 0, n)
	for len(keys) < nHot {
		pref := prefixes[rng.Intn(nTop)]
		k := (pref << 20) | (rng.Uint64() & ((1 << 20) - 1))
		k &= mask60
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for len(keys) < n {
		pref := prefixes[nTop+rng.Intn(nPrefixes-nTop)]
		k := (pref << 20) | (rng.Uint64() & ((1 << 20) - 1))
		k &= mask60
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func TestHybrid_ZipfianDiagnostic(t *testing.T) {
	const (
		nPrefixes = 100
		rangeLen  = uint64(1024)
		eps       = 0.01
	)

	for _, n := range []int{1 << 16, 1 << 18, 1 << 20} {
		t.Run(fmt.Sprintf("N=%d", n), func(t *testing.T) {
			rng := rand.New(rand.NewSource(77))
			keys := generateZipfianKeys(n, nPrefixes, rng)
			t.Logf("generated %d zipfian keys (%d prefixes, top 10%% get 80%% of keys)", len(keys), nPrefixes)

			bs := make([]bits.BitString, len(keys))
			for i, k := range keys {
				bs[i] = bits.NewFromTrieUint64(k, 60)
			}

			// Cluster detection
			segments, fallbackKeys := detectClusters(bs, 0.95, 0.01)
			t.Logf("detectClusters: %d segments, %d fallback keys (%.1f%%)",
				len(segments), len(fallbackKeys), 100*float64(len(fallbackKeys))/float64(len(keys)))

			totalClusterKeys := 0
			for i, seg := range segments {
				totalClusterKeys += len(seg.keys)
				t.Logf("  segment[%d]: %d keys (%.1f%%), range [%x, %x], span=%d",
					i, len(seg.keys), 100*float64(len(seg.keys))/float64(len(keys)),
					seg.minKey, seg.maxKey, seg.maxKey-seg.minKey)
			}
			t.Logf("total in clusters: %d (%.1f%%)", totalClusterKeys, 100*float64(totalClusterKeys)/float64(len(keys)))

			// Build hybrid
			h, err := NewHybridARE(bs, rangeLen, eps)
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			nc, nf, nt := h.Stats()
			bpk := float64(h.SizeInBits()) / float64(nt)
			t.Logf("HybridARE: clusters=%d, fallback=%d (%.1f%%), total=%d, BPK=%.2f",
				nc, nf, 100*float64(nf)/float64(nt), nt, bpk)

			for i, c := range h.clusters {
				nKeys := 0
				for _, seg := range segments {
					if seg.minKey == c.minKey {
						nKeys = len(seg.keys)
						break
					}
				}
				cBPK := float64(c.filter.SizeInBits()) / float64(nKeys)
				t.Logf("  cluster[%d]: %d keys, span=%d, BPK=%.2f",
					i, nKeys, c.maxKey-c.minKey, cBPK)
			}
			if h.fallback != nil {
				fbBPK := float64(h.fallback.SizeInBits()) / float64(nf)
				t.Logf("  fallback: %d keys, BPK=%.2f", nf, fbBPK)
			}

			// Measure FPR
			qrng := rand.New(rand.NewSource(12345))
			fp, totalEmpty := 0, 0
			for q := 0; q < 200_000; q++ {
				a := qrng.Uint64() & mask60
				b := a + rangeLen - 1
				if b < a {
					continue
				}
				idx := sort.Search(len(keys), func(j int) bool { return keys[j] >= a })
				if idx < len(keys) && keys[idx] <= b {
					continue
				}
				totalEmpty++
				if !h.IsEmpty(bits.NewFromTrieUint64(a, 60), bits.NewFromTrieUint64(b, 60)) {
					fp++
				}
			}
			fpr := float64(fp) / float64(totalEmpty)
			t.Logf("FPR: %.6f (%d/%d empty queries)", fpr, fp, totalEmpty)
		})
	}
}
