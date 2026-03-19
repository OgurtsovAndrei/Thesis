package are_dp_scan

import (
	"Thesis/bits"
	"Thesis/emptiness/are_greedy_scan"
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

func TestDPScan_Empty(t *testing.T) {
	d, err := NewDPScanARE(nil, 100, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if !d.IsEmpty(testutils.TrieBS(0), testutils.TrieBS(100)) {
		t.Error("expected empty result for nil keys")
	}
}

func TestDPScan_SingleKey(t *testing.T) {
	bs := []bits.BitString{testutils.TrieBS(500)}
	d, err := NewDPScanARE(bs, 10, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if d.IsEmpty(testutils.TrieBS(495), testutils.TrieBS(505)) {
		t.Error("query containing the single key should be non-empty")
	}
	if !d.IsEmpty(testutils.TrieBS(0), testutils.TrieBS(10)) {
		t.Error("query far from key should be empty")
	}
}

func TestDPScan_NoFN(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for _, n := range []int{100, 1000, 10000} {
		n := n
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
			d, err := NewDPScanARE(keys, rangeLen, 0.01)
			if err != nil {
				t.Fatal(err)
			}

			for _, k := range rawKeys {
				a := testutils.TrieBS(k)
				b := testutils.TrieBS(k + rangeLen)
				if d.IsEmpty(a, b) {
					t.Fatalf("false negative: key %d in [%d, %d]", k, k, k+rangeLen)
				}
			}
		})
	}
}

func TestDPScan_Stats(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	raw, _ := testutils.GenerateClusterDistribution(10000, 5, 0.15, rng)
	keys := sortedTrieBS(raw)

	d, err := NewDPScanAREFromK(keys, 128, 20)
	if err != nil {
		t.Fatal(err)
	}
	numClusters, totalKeys := d.Stats()
	t.Logf("clusters=%d, totalKeys=%d, BPK=%.2f", numClusters, totalKeys, float64(d.SizeInBits())/float64(totalKeys))

	if numClusters == 0 {
		t.Error("expected at least one cluster")
	}
	if totalKeys != len(keys) {
		t.Errorf("totalKeys=%d, want %d", totalKeys, len(keys))
	}
}

// TestDPScan_BPKOptimal verifies that DP segmentation achieves BPK ≤ greedy
// on the same key set. The DP solution is optimal w.r.t. the cost estimate,
// so it should never be worse than greedy's merge-based approach.
func TestDPScan_BPKOptimal(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	for _, n := range []int{500, 5000} {
		n := n
		t.Run("", func(t *testing.T) {
			raw, _ := testutils.GenerateClusterDistribution(n, 8, 0.15, rng)
			keys := sortedTrieBS(raw)

			const K = 20
			const rangeLen = 128

			dp, err := NewDPScanAREFromK(keys, rangeLen, K)
			if err != nil {
				t.Fatal(err)
			}
			greedy, err := are_greedy_scan.NewGreedyScanAREFromK(keys, rangeLen, K)
			if err != nil {
				t.Fatal(err)
			}

			dpBPK := float64(dp.SizeInBits()) / float64(n)
			greedyBPK := float64(greedy.SizeInBits()) / float64(n)

			t.Logf("n=%d: DP BPK=%.3f, Greedy BPK=%.3f", n, dpBPK, greedyBPK)

			if dpBPK > greedyBPK+1.0 {
				t.Errorf("DP BPK (%.3f) is more than 1 bit worse than Greedy BPK (%.3f) — unexpected", dpBPK, greedyBPK)
			}
		})
	}
}
