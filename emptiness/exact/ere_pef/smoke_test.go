package ere_pef

import (
	"math/rand"
	"testing"

	ereOneD "Thesis/emptiness/exact/ere_one_d"
	"Thesis/testutils"
)

// TestSmoke_LargeBuild builds at n=100K on three distributions and checks
// (a) NewPEF completes, (b) IsEmpty matches ere_one_d on a sample, and
// (c) reports BPK comparison so we can eyeball PEF benefit on this data.
func TestSmoke_LargeBuild(t *testing.T) {
	const n = 100_000
	cases := []struct {
		name string
		gen  func(rng *rand.Rand) []uint64
	}{
		{"cluster_c16", func(rng *rand.Rand) []uint64 {
			keys, _ := testutils.GenerateClusterDistribution(n, 16, 0.05, rng)
			return keys
		}},
		{"uniform", func(rng *rand.Rand) []uint64 {
			keys := make([]uint64, n)
			for i := range keys {
				keys[i] = rng.Uint64()
			}
			return keys
		}},
		{"sequential", func(rng *rand.Rand) []uint64 {
			keys := make([]uint64, n)
			base := rng.Uint64() & ((uint64(1) << 50) - 1)
			for i := range keys {
				keys[i] = base + uint64(i)
			}
			return keys
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(7))
			keys := sortDedupMask(tc.gen(rng))
			pef, err := NewPEF(keys, parityKeyBits)
			if err != nil {
				t.Fatal(err)
			}
			ere, err := ereOneD.NewExactRangeEmptiness(keys, parityKeyBits)
			if err != nil {
				t.Fatal(err)
			}

			qrng := rand.New(rand.NewSource(13))
			mask := parityKeyMask()
			for i := 0; i < 1000; i++ {
				a := qrng.Uint64() & mask
				b := qrng.Uint64() & mask
				if a > b {
					a, b = b, a
				}
				if pef.IsEmpty(a, b) != ere.IsEmpty(a, b) {
					t.Fatalf("divergence on q#%d (%d,%d)", i, a, b)
				}
			}
			pefBPK := float64(pef.ByteSize()*8) / float64(len(keys))
			ereBPK := float64(ere.ByteSize()*8) / float64(len(keys))
			t.Logf("n=%d  pef_chunks=%d  pef_bpk=%.2f  ere_one_d_bpk=%.2f  Δ=%+.2f",
				len(keys), len(pef.chunks), pefBPK, ereBPK, pefBPK-ereBPK)
		})
	}
}
