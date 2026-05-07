package are_greedy_scan

import (
	"Thesis/emptiness/exact"
	"Thesis/testutils"
	"math"
	"math/rand"
	"sort"
	"testing"
)

// kFromEps maps the legacy (n, L, eps) target to K = ceil(log2(n*(L+1)/eps)).
func kFromEps(n int, rangeLen uint64, eps float64) uint32 {
	k := uint32(math.Ceil(math.Log2(float64(n) * (float64(rangeLen) + 1) / eps)))
	if k == 0 {
		k = 1
	}
	if k > 64 {
		k = 64
	}
	return k
}

func sortedUint64(raw []uint64) []uint64 {
	cp := append([]uint64(nil), raw...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	return cp
}

func TestGreedyScan_Empty(t *testing.T) {
	g, err := NewGreedyScanARE(nil, 60, Config{K: 20})
	if err != nil {
		t.Fatal(err)
	}
	if !g.IsEmpty(0, 100) {
		t.Error("expected empty result for nil keys")
	}
}

func TestGreedyScan_SingleKey(t *testing.T) {
	keys := []uint64{500}
	g, err := NewGreedyScanARE(keys, 60, Config{K: kFromEps(len(keys), 10, 0.01)})
	if err != nil {
		t.Fatal(err)
	}
	if g.IsEmpty(495, 505) {
		t.Error("query containing the single key should be non-empty")
	}
	if !g.IsEmpty(0, 10) {
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
			keys := sortedUint64(rawKeys)

			var rangeLen uint64 = 128
			g, err := NewGreedyScanARE(keys, 60, Config{K: kFromEps(len(keys), rangeLen, 0.01)})
			if err != nil {
				t.Fatal(err)
			}

			for _, k := range rawKeys {
				if g.IsEmpty(k, k+rangeLen) {
					t.Fatalf("false negative: key %d in [%d, %d]", k, k, k+rangeLen)
				}
			}
		})
	}
}

func TestGreedyScan_Stats(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	raw, _ := testutils.GenerateClusterDistribution(10000, 5, 0.15, rng)
	keys := sortedUint64(raw)

	g, err := NewGreedyScanARE(keys, 60, Config{K: 20})
	if err != nil {
		t.Fatal(err)
	}
	numClusters, fallbackKeys, totalKeys := g.Stats()
	t.Logf("clusters=%d, fallback=%d, totalKeys=%d, BPK=%.2f", numClusters, fallbackKeys, totalKeys, float64(g.SizeInBits())/float64(totalKeys))

	if numClusters == 0 && fallbackKeys == 0 {
		t.Error("expected at least one cluster or fallback keys")
	}
	if totalKeys != len(keys) {
		t.Errorf("totalKeys=%d, want %d", totalKeys, len(keys))
	}
}

func TestGreedyScan_EREBackendPEF(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	raw, _ := testutils.GenerateClusterDistribution(1024, 4, 0.2, rng)
	keys := sortedUint64(raw)

	cfg := Config{K: 20}.WithEREBackend(exact.VariantPEF)
	g, err := NewGreedyScanARE(keys, 60, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range keys {
		if g.IsEmpty(k, k) {
			t.Fatalf("false negative for stored key %d", k)
		}
	}
}

// Wide-spread fallback: with the legacy FallbackAlwaysTrunc default any
// near-key empty query collapses into the bucket of the nearest stored key,
// pinning FPR. FallbackAlwaysSODA (now default) avoids the collapse via
// SODA-hash fingerprints. Regression test for the OSM scan-ARE flatness
// investigated 2026-05-04.
func TestGreedyScan_NearKeyFallbackFPR_WideSpread(t *testing.T) {
	const (
		n        = 200
		rangeLen = uint64(128)
		K        = uint32(30) // phantomSize = 2^40 / 2^30 = 1024 ≫ rangeLen
		base     = uint64(1_000_000)
	)
	spread := uint64(1) << 40
	gap := spread / uint64(n-1)

	keys := make([]uint64, n)
	for i := 0; i < n; i++ {
		keys[i] = base + uint64(i)*gap
	}

	gDefault, err := NewGreedyScanARE(keys, 64, Config{K: K})
	if err != nil {
		t.Fatal(err)
	}
	fp := 0
	for _, v := range keys {
		lo := v + rangeLen + 1
		hi := lo + rangeLen
		if !gDefault.IsEmpty(lo, hi) {
			fp++
		}
	}
	fpr := float64(fp) / float64(n)
	t.Logf("default (AlwaysSODA) near-key empty FPR=%.4f (fp=%d / n=%d)", fpr, fp, n)
	if fpr >= 0.10 {
		t.Fatalf("near-key empty FPR pinned high (%.4f) — fallback policy regressed?", fpr)
	}

	gTrunc, err := NewGreedyScanAREWithPolicy(keys, 64, ConfigWithPolicy{
		K: K, RangeLen: rangeLen, Policy: FallbackAlwaysTrunc{},
	})
	if err != nil {
		t.Fatal(err)
	}
	fpTrunc := 0
	for _, v := range keys {
		lo := v + rangeLen + 1
		hi := lo + rangeLen
		if !gTrunc.IsEmpty(lo, hi) {
			fpTrunc++
		}
	}
	fprTrunc := float64(fpTrunc) / float64(n)
	t.Logf("AlwaysTrunc near-key empty FPR=%.4f (fp=%d / n=%d)", fprTrunc, fpTrunc, n)
	if fprTrunc <= 0.50 {
		t.Fatalf("AlwaysTrunc must reproduce the near-key collapse (got %.4f) — test design invalid", fprTrunc)
	}
}
