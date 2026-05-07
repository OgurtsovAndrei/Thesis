package ere_pef

import (
	"math/rand"
	"sort"
	"testing"

	ereOneD "Thesis/emptiness/exact/ere_one_d"
	"Thesis/testutils"
)

const parityKeyBits = 60

func parityKeyMask() uint64 { return (uint64(1) << parityKeyBits) - 1 }

func sortDedupMask(keys []uint64) []uint64 {
	mask := parityKeyMask()
	for i, k := range keys {
		keys[i] = k & mask
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	j := 0
	for i, k := range keys {
		if i == 0 || k != keys[i-1] {
			keys[j] = k
			j++
		}
	}
	return keys[:j]
}

func TestParityWithEREOneD(t *testing.T) {
	cases := []struct {
		name string
		gen  func(rng *rand.Rand) []uint64
	}{
		{"cluster_n4096_c8", func(rng *rand.Rand) []uint64 {
			keys, _ := testutils.GenerateClusterDistribution(4096, 8, 0.1, rng)
			return keys
		}},
		{"uniform_n4096", func(rng *rand.Rand) []uint64 {
			keys := make([]uint64, 4096)
			for i := range keys {
				keys[i] = rng.Uint64()
			}
			return keys
		}},
		{"sequential_n4096", func(rng *rand.Rand) []uint64 {
			keys := make([]uint64, 4096)
			base := rng.Uint64() & ((uint64(1) << 50) - 1)
			for i := range keys {
				keys[i] = base + uint64(i)
			}
			return keys
		}},
		{"cluster_large_n16384_c16", func(rng *rand.Rand) []uint64 {
			keys, _ := testutils.GenerateClusterDistribution(16384, 16, 0.05, rng)
			return keys
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(0xDEADBEEF))
			keys := sortDedupMask(tc.gen(rng))
			if len(keys) == 0 {
				t.Skip("empty key set after dedup")
			}

			pef, err := NewPEF(keys, parityKeyBits)
			if err != nil {
				t.Fatal(err)
			}
			ere, err := ereOneD.NewExactRangeEmptiness(keys, parityKeyBits)
			if err != nil {
				t.Fatal(err)
			}

			// 1) Self-membership for every key
			for _, k := range keys {
				if pef.IsEmpty(k, k) {
					t.Fatalf("PEF reports self-key %d empty", k)
				}
				if ere.IsEmpty(k, k) {
					t.Fatalf("ERE reports self-key %d empty", k)
				}
			}

			// 2) 10⁴ random queries — each must agree
			qrng := rand.New(rand.NewSource(0xCAFEBABE))
			mask := parityKeyMask()
			for i := 0; i < 10000; i++ {
				a := qrng.Uint64() & mask
				b := qrng.Uint64() & mask
				if a > b {
					a, b = b, a
				}
				gotPEF := pef.IsEmpty(a, b)
				gotERE := ere.IsEmpty(a, b)
				if gotPEF != gotERE {
					t.Fatalf("divergence q#%d IsEmpty(%d,%d): pef=%v ere=%v",
						i, a, b, gotPEF, gotERE)
				}
			}
		})
	}
}

// TestPEF_MaxUint64Universe is the regression test for the 2^64-universe
// overflow bug: when keys span the full uint64 range, the old "+1" in
// "inputUniverse = lastKey + 1" wrapped to 0, causing the DP to pick
// kindAllOnes for all chunks and IsEmpty to return true for stored keys.
func TestPEF_MaxUint64Universe(t *testing.T) {
	const n = 300
	maxUint64 := ^uint64(0)

	// Generate n keys spread uniformly across [0, MaxUint64].
	raw := make([]uint64, n)
	for i := 0; i < n; i++ {
		raw[i] = uint64(float64(i) / float64(n-1) * float64(maxUint64))
	}
	// Dedup (float rounding may produce collisions near the boundary).
	sort.Slice(raw, func(i, j int) bool { return raw[i] < raw[j] })
	keys := raw[:0]
	for i, k := range raw {
		if i == 0 || k != raw[i-1] {
			keys = append(keys, k)
		}
	}

	p, err := NewPEF(keys, 64)
	if err != nil {
		t.Fatal(err)
	}

	// No false negatives: every stored key must be found.
	for _, v := range keys {
		if p.IsEmpty(v, v) {
			t.Fatalf("false negative for stored key %d", v)
		}
	}

	// At least one in-gap point between two consecutive keys must be
	// reported empty (PEF is an exact filter).
	foundGap := false
	for i := 1; i < len(keys); i++ {
		if keys[i] > keys[i-1]+1 {
			mid := keys[i-1] + (keys[i]-keys[i-1])/2
			if p.IsEmpty(mid, mid) {
				foundGap = true
				break
			}
		}
	}
	if !foundGap {
		t.Error("no in-gap point was reported empty — PEF over-reports non-empty")
	}
}

func TestParitySmallExhaustive(t *testing.T) {
	// On a small universe, sweep ALL (a,b) pairs and confirm both
	// implementations agree exactly. Catches off-by-ones in clamps,
	// gap detection, chunk boundary handling.
	keys := []uint64{3, 7, 8, 15, 30}
	const kb = 6 // universe = 64
	pef, err := NewPEF(keys, kb)
	if err != nil {
		t.Fatal(err)
	}
	ere, err := ereOneD.NewExactRangeEmptiness(keys, kb)
	if err != nil {
		t.Fatal(err)
	}
	for a := uint64(0); a < 64; a++ {
		for b := a; b < 64; b++ {
			gotPEF := pef.IsEmpty(a, b)
			gotERE := ere.IsEmpty(a, b)
			if gotPEF != gotERE {
				t.Fatalf("divergence (%d,%d): pef=%v ere=%v keys=%v",
					a, b, gotPEF, gotERE, keys)
			}
		}
	}
}
