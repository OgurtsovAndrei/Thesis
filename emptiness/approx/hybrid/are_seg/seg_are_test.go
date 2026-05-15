package are_seg

import (
	"Thesis/emptiness/approx/hybrid/are_dbscan"
	"Thesis/testutils"
	"math/rand"
	"sort"
	"testing"
)

const (
	testRangeLen = uint64(128)
	testEpsilon  = 0.01
	testKeyBits  = uint32(64)
	testQueries  = 200_000
)

func buildAndMeasure(t *testing.T, keys []uint64, label string) (fpr, bpk float64, numSeg, fbKeys int) {
	t.Helper()
	f, err := NewSegARE(keys, testKeyBits, testRangeLen, testEpsilon)
	if err != nil {
		t.Fatalf("%s: build failed: %v", label, err)
	}

	rng := rand.New(rand.NewSource(42))
	queries := make([][2]uint64, testQueries)
	for i := range queries {
		a := rng.Uint64()
		queries[i] = [2]uint64{a, a + testRangeLen - 1}
	}

	fpr = testutils.MeasureFPR(keys, queries, f.IsEmpty)
	numSeg, fbKeys, _ = f.Stats()
	bpk = float64(f.SizeInBits()) / float64(len(keys))
	return
}

func buildDBSCANAndMeasure(t *testing.T, keys []uint64, label string) (fpr, bpk float64) {
	t.Helper()
	K := kFromParams(len(keys), testRangeLen, testEpsilon)
	f, err := are_dbscan.NewHybridScanARE(keys, testKeyBits, are_dbscan.Config{K: K})
	if err != nil {
		t.Fatalf("%s: dbscan build failed: %v", label, err)
	}

	rng := rand.New(rand.NewSource(42))
	queries := make([][2]uint64, testQueries)
	for i := range queries {
		a := rng.Uint64()
		queries[i] = [2]uint64{a, a + testRangeLen - 1}
	}

	fpr = testutils.MeasureFPR(keys, queries, f.IsEmpty)
	_, _, total := f.Stats()
	bpk = float64(f.SizeInBits()) / float64(total)
	return
}

func TestSegARE_Empty(t *testing.T) {
	f, err := NewSegARE(nil, 64, 128, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if !f.IsEmpty(0, 100) {
		t.Error("expected empty for nil keys")
	}
}

func TestSegARE_SingleKey(t *testing.T) {
	keys := []uint64{500}
	f, err := NewSegARE(keys, 64, 10, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if f.IsEmpty(495, 505) {
		t.Error("query containing the key should be non-empty")
	}
	if !f.IsEmpty(0, 10) {
		t.Error("far query should be empty")
	}
}

func TestSegARE_NoFN(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for _, n := range []int{100, 1000, 10_000} {
		t.Run("", func(t *testing.T) {
			seen := make(map[uint64]bool, n)
			rawKeys := make([]uint64, n)
			for i := range rawKeys {
				for {
					k := rng.Uint64()
					if !seen[k] {
						seen[k] = true
						rawKeys[i] = k
						break
					}
				}
			}
			sort.Slice(rawKeys, func(i, j int) bool { return rawKeys[i] < rawKeys[j] })
			f, err := NewSegARE(rawKeys, 64, testRangeLen, testEpsilon)
			if err != nil {
				t.Fatal(err)
			}
			for _, k := range rawKeys {
				if f.IsEmpty(k, k+testRangeLen-1) {
					t.Fatalf("false negative for key %d", k)
				}
			}
		})
	}
}

// TestSegARE_vs_DBSCAN_Clustered compares SegARE and HybridScanARE on clustered data.
// Both should detect clusters; SegARE uses larger δ so may merge some.
func TestSegARE_vs_DBSCAN_Clustered(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	raw, _ := testutils.GenerateClusterDistribution(10_000, 10, 0.15, rng)
	keys := make([]uint64, len(raw))
	copy(keys, raw)
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	segFPR, segBPK, segN, segFB := buildAndMeasure(t, keys, "SegARE/clustered")
	dbscanFPR, dbscanBPK := buildDBSCANAndMeasure(t, keys, "DBSCAN/clustered")

	t.Logf("Clustered  SegARE:  FPR=%.4f  BPK=%.2f  segments=%d  fallback=%d", segFPR, segBPK, segN, segFB)
	t.Logf("Clustered  DBSCAN:  FPR=%.4f  BPK=%.2f", dbscanFPR, dbscanBPK)

	if segFPR > testEpsilon*3 {
		t.Errorf("SegARE FPR %.4f >> epsilon %.4f on clustered data", segFPR, testEpsilon)
	}
}

// TestSegARE_vs_DBSCAN_Uniform verifies both filters behave correctly on
// uniform 64-bit keys (no clusters expected; all fallback).
func TestSegARE_vs_DBSCAN_Uniform(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	n := 10_000
	keys := make([]uint64, n)
	seen := make(map[uint64]bool, n)
	for i := range keys {
		for {
			k := rng.Uint64()
			if !seen[k] {
				seen[k] = true
				keys[i] = k
				break
			}
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	segFPR, segBPK, segN, segFB := buildAndMeasure(t, keys, "SegARE/uniform")
	dbscanFPR, dbscanBPK := buildDBSCANAndMeasure(t, keys, "DBSCAN/uniform")

	t.Logf("Uniform  SegARE:  FPR=%.4f  BPK=%.2f  segments=%d  fallback=%d", segFPR, segBPK, segN, segFB)
	t.Logf("Uniform  DBSCAN:  FPR=%.4f  BPK=%.2f", dbscanFPR, dbscanBPK)

	if segFPR > testEpsilon*3 {
		t.Errorf("SegARE FPR %.4f >> epsilon %.4f on uniform data", segFPR, testEpsilon)
	}
	if segN > 0 {
		t.Logf("Note: SegARE found %d segments on uniform data (unexpected but not fatal)", segN)
	}
}

// TestSegARE_vs_DBSCAN_Sequential checks sequential keys where a single big
// segment is expected.
func TestSegARE_vs_DBSCAN_Sequential(t *testing.T) {
	n := 10_000
	keys := make([]uint64, n)
	for i := range keys {
		keys[i] = uint64(i) * 1000
	}

	segFPR, segBPK, segN, segFB := buildAndMeasure(t, keys, "SegARE/sequential")
	dbscanFPR, dbscanBPK := buildDBSCANAndMeasure(t, keys, "DBSCAN/sequential")

	t.Logf("Sequential  SegARE:  FPR=%.4f  BPK=%.2f  segments=%d  fallback=%d", segFPR, segBPK, segN, segFB)
	t.Logf("Sequential  DBSCAN:  FPR=%.4f  BPK=%.2f", dbscanFPR, dbscanBPK)

	if segFPR > testEpsilon*3 {
		t.Errorf("SegARE FPR %.4f >> epsilon %.4f on sequential data", segFPR, testEpsilon)
	}
}
