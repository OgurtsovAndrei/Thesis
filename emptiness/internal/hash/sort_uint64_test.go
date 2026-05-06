package hash

import (
	"math/rand"
	"slices"
	"testing"
)

// genUniformKeys produces n pseudo-random uint64 keys uniformly distributed
// in [0, 2^K). This mimics the distribution of post-pairwise-hash values in
// the SODA build path.
func genUniformKeys(n int, K uint32, seed int64) []uint64 {
	rng := rand.New(rand.NewSource(seed))
	mask := ^uint64(0)
	if K < 64 {
		mask = (uint64(1) << K) - 1
	}
	out := make([]uint64, n)
	for i := range out {
		out[i] = rng.Uint64() & mask
	}
	return out
}

// referenceSortDedup is the trusted oracle: copy → slices.Sort → linear dedup.
func referenceSortDedup(keys []uint64) []uint64 {
	cp := slices.Clone(keys)
	slices.Sort(cp)
	w := 0
	for i, x := range cp {
		if i == 0 || x != cp[w-1] {
			cp[w] = x
			w++
		}
	}
	return cp[:w]
}

func TestSortAndDedupVariantsAgree(t *testing.T) {
	cases := []struct {
		name string
		n    int
		K    uint32
	}{
		{"empty", 0, 24},
		{"single", 1, 24},
		{"two_equal", 2, 24},
		{"all_zero", 100, 24},
		{"small_K20", 1024, 20},
		{"medium_K24", 16384, 24},
		{"large_K28", 262144, 28},
		{"K32", 65536, 32},
		{"K36", 65536, 36},
		{"K40_no_bitmap", 65536, 40},
		{"K64", 65536, 64},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			master := genUniformKeys(tc.n, tc.K, 0xC0FFEE)
			if tc.name == "all_zero" {
				for i := range master {
					master[i] = 0
				}
			}
			expected := referenceSortDedup(master)

			variants := []struct {
				name string
				fn   func([]uint64) []uint64
				skip bool
			}{
				{"slices", func(k []uint64) []uint64 { return SortAndDedupUint64Slices(k) }, false},
				{"radix", func(k []uint64) []uint64 { return SortAndDedupUint64Radix(k) }, false},
				{"radixK", func(k []uint64) []uint64 { return SortAndDedupUint64RadixK(k, tc.K) }, false},
				{"bitmap", func(k []uint64) []uint64 { return SortAndDedupUint64Bitmap(k, tc.K) }, false},
				{"current", func(k []uint64) []uint64 { return SortAndDedupUint64(k) }, false},
				{"americanFlag", func(k []uint64) []uint64 { return SortAndDedupUint64AmericanFlag(k, tc.K) }, false},
				{"msdBitmap", func(k []uint64) []uint64 { return SortAndDedupUint64MSDBitmap(k, tc.K) }, false},
				{"adaptive", func(k []uint64) []uint64 { return SortAndDedupUint64Adaptive(k, tc.K) }, false},
			}

			for _, v := range variants {
				if v.skip {
					continue
				}
				input := slices.Clone(master)
				got := v.fn(input)
				if !slices.Equal(got, expected) {
					t.Errorf("variant %q: got len=%d, want len=%d (first mismatch may follow)",
						v.name, len(got), len(expected))
					if len(got) == len(expected) {
						for i := range got {
							if got[i] != expected[i] {
								t.Errorf("  index %d: got %d, want %d", i, got[i], expected[i])
								break
							}
						}
					}
				}
			}
		})
	}
}

func TestSortAndDedupRadixK_RespectsK(t *testing.T) {
	// Verify that radixK with K=24 still works on keys whose top bits are zero.
	keys := []uint64{0xABCDEF, 0x123456, 0x000001, 0xABCDEF, 0xFFFFFF}
	expected := []uint64{0x000001, 0x123456, 0xABCDEF, 0xFFFFFF}
	got := SortAndDedupUint64RadixK(slices.Clone(keys), 24)
	if !slices.Equal(got, expected) {
		t.Errorf("got %v, want %v", got, expected)
	}
}

func TestSortAndDedupBitmap_LargeKFallback(t *testing.T) {
	// K=40 should trigger the slices.Sort fallback (no panic, correct result).
	keys := genUniformKeys(1000, 40, 7)
	expected := referenceSortDedup(keys)
	got := SortAndDedupUint64Bitmap(slices.Clone(keys), 40)
	if !slices.Equal(got, expected) {
		t.Errorf("bitmap K=40 fallback produced wrong result")
	}

	// K=33 also triggers fallback (just over the 512 MiB threshold).
	keys2 := genUniformKeys(1000, 33, 11)
	expected2 := referenceSortDedup(keys2)
	got2 := SortAndDedupUint64Bitmap(slices.Clone(keys2), 33)
	if !slices.Equal(got2, expected2) {
		t.Errorf("bitmap K=33 fallback produced wrong result")
	}
}
