package are_greedy_scan

import (
	"Thesis/bits"
	"Thesis/testutils"
	"math/rand"
	"sort"
	"testing"
)

func sortedTrieBS(raw []uint64) []bits.BitString {
	sort.Slice(raw, func(i, j int) bool { return raw[i] < raw[j] })
	bs := make([]bits.BitString, len(raw))
	for i, v := range raw {
		bs[i] = testutils.TrieBS(v)
	}
	return bs
}

func TestGreedyScan_Empty(t *testing.T) {
	g, err := NewGreedyScanARE(nil, 100, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if !g.IsEmpty(testutils.TrieBS(0), testutils.TrieBS(100)) {
		t.Error("expected empty result for nil keys")
	}
}

func TestGreedyScan_SingleKey(t *testing.T) {
	bs := []bits.BitString{testutils.TrieBS(500)}
	g, err := NewGreedyScanARE(bs, 10, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if g.IsEmpty(testutils.TrieBS(495), testutils.TrieBS(505)) {
		t.Error("query containing the single key should be non-empty")
	}
	if !g.IsEmpty(testutils.TrieBS(0), testutils.TrieBS(10)) {
		t.Error("query far from key should be empty")
	}
}

func TestGreedyScan_NoFN(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for _, n := range []int{100, 1000, 10000} {
		t.Run("", func(t *testing.T) {
			rawKeys := make([]uint64, n)
			seen := make(map[uint64]bool, n)
			for i := range rawKeys {
				for {
					k := rng.Uint64() & ((1 << 60) - 1)
					if !seen[k] {
						seen[k] = true
						rawKeys[i] = k
						break
					}
				}
			}
			keys := sortedTrieBS(rawKeys)

			var rangeLen uint64 = 128
			g, err := NewGreedyScanARE(keys, rangeLen, 0.01)
			if err != nil {
				t.Fatal(err)
			}

			for _, k := range rawKeys {
				a := testutils.TrieBS(k)
				b := testutils.TrieBS(k + rangeLen)
				if g.IsEmpty(a, b) {
					t.Fatalf("false negative: key %d in [%d, %d]", k, k, k+rangeLen)
				}
			}
		})
	}
}

func TestGreedyScan_Stats(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	raw, _ := testutils.GenerateClusterDistribution(10000, 5, 0.15, rng)
	keys := sortedTrieBS(raw)

	g, err := NewGreedyScanAREFromK(keys, 128, 20)
	if err != nil {
		t.Fatal(err)
	}
	numClusters, totalKeys := g.Stats()
	t.Logf("clusters=%d, totalKeys=%d, BPK=%.2f", numClusters, totalKeys, float64(g.SizeInBits())/float64(totalKeys))

	if numClusters == 0 {
		t.Error("expected at least one cluster")
	}
	if totalKeys != len(keys) {
		t.Errorf("totalKeys=%d, want %d", totalKeys, len(keys))
	}
}
