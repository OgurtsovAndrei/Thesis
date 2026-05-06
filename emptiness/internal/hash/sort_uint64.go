package hash

import (
	mbits "math/bits"
	"slices"

	radix "github.com/dgryski/go-radixsort"
)

// This file provides several sort+dedup implementations for the SODA-style
// build path, where input keys are near-uniformly distributed in [0, 2^K).
// Variants:
//
//   SortAndDedupUint64Slices  — pdqsort via stdlib slices.Sort.
//   SortAndDedupUint64Radix   — 8-pass LSD radix sort (dgryski/go-radixsort).
//   SortAndDedupUint64RadixK  — K-aware LSD radix: only ⌈K/8⌉ passes.
//   SortAndDedupUint64Bitmap  — bitmap dedup + bit-extraction (no sort);
//                               requires K small enough that 2^K bits fit RAM.
//
// All variants return the sorted, deduplicated key set as a sub-slice; the
// backing array is either the input or an internally allocated buffer.

// SortAndDedupUint64Slices sorts keys with pdqsort (stdlib slices.Sort) and
// removes consecutive duplicates in place.
func SortAndDedupUint64Slices(keys []uint64) []uint64 {
	if len(keys) == 0 {
		return keys
	}
	slices.Sort(keys)
	return dedupSortedUint64(keys)
}

// SortAndDedupUint64Radix sorts keys with an 8-pass LSD radix sort and
// removes consecutive duplicates. Touches all 64 bits of every key, so it is
// independent of K. Allocates an n-element scratch buffer internally.
func SortAndDedupUint64Radix(keys []uint64) []uint64 {
	if len(keys) == 0 {
		return keys
	}
	radix.Uint64s(keys)
	return dedupSortedUint64(keys)
}

// SortAndDedupUint64RadixK is a K-aware LSD radix sort that performs only
// ⌈K/8⌉ passes. Caller guarantees keys ∈ [0, 2^K). For K=64 it falls back
// to 8 passes (equivalent to SortAndDedupUint64Radix).
//
// The returned slice may share the input backing array OR an internally
// allocated buffer, depending on the parity of the pass count. Callers must
// not assume identity with the input.
func SortAndDedupUint64RadixK(keys []uint64, K uint32) []uint64 {
	n := len(keys)
	if n == 0 {
		return keys
	}
	if K == 0 {
		// Degenerate: all keys must be 0; dedup yields ≤1 element.
		return dedupSortedUint64(keys)
	}
	if K > 64 {
		K = 64
	}

	passes := int((K + 7) / 8)
	buf := make([]uint64, n)
	src, dst := keys, buf

	for p := 0; p < passes; p++ {
		shift := uint(p * 8)

		var counts [256]int
		for _, x := range src {
			counts[byte(x>>shift)]++
		}

		var sum int
		for i := 0; i < 256; i++ {
			c := counts[i]
			counts[i] = sum
			sum += c
		}

		for _, x := range src {
			d := byte(x >> shift)
			dst[counts[d]] = x
			counts[d]++
		}

		src, dst = dst, src
	}

	return dedupSortedUint64(src[:n])
}

// SortAndDedupUint64Bitmap dedups by setting bits in a 2^K-bit bitmap, then
// extracts set bits in ascending order using TrailingZeros64. This avoids
// any comparison-based sort. Allocates 2^K / 8 bytes internally; therefore
// only feasible for moderate K. Above K=32 (512 MiB bitmap) the cost of
// allocation and cache misses dominates and the function falls back to
// SortAndDedupUint64Slices.
//
// Caller must guarantee keys ∈ [0, 2^K).
func SortAndDedupUint64Bitmap(keys []uint64, K uint32) []uint64 {
	if len(keys) == 0 {
		return keys
	}
	if K == 0 {
		return dedupSortedUint64(keys)
	}
	if K > 32 {
		return SortAndDedupUint64Slices(keys)
	}

	size := uint64(1) << K
	words := (size + 63) >> 6
	bm := make([]uint64, words)

	for _, x := range keys {
		bm[x>>6] |= uint64(1) << (x & 63)
	}

	out := keys[:0]
	for i, w := range bm {
		base := uint64(i) << 6
		for w != 0 {
			b := mbits.TrailingZeros64(w)
			out = append(out, base+uint64(b))
			w &= w - 1
		}
	}
	return out
}

// dedupSortedUint64 removes consecutive duplicates in place and returns
// the truncated sub-slice.
func dedupSortedUint64(keys []uint64) []uint64 {
	if len(keys) <= 1 {
		return keys
	}
	w := 1
	for i := 1; i < len(keys); i++ {
		if keys[i] != keys[i-1] {
			keys[w] = keys[i]
			w++
		}
	}
	return keys[:w]
}
