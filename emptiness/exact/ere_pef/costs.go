// Package ere_pef implements 1D exact range emptiness using
// Partitioned Elias-Fano (Ottaviano & Venturini, SIGIR 2014). The
// implementation mirrors PISA (pisa-engine/pisa, Apache 2.0): cost
// functions and partition algorithm follow the paper formulas;
// per-chunk encoders are simplified to reuse Thesis primitives
// (succinct_bit_vector/rsdic, bits.UnpackBit) instead of PISA's
// bespoke bitvector layout.
package ere_pef

import "math/bits"

// efBitsizePaper returns the EF cost of encoding n keys into a chunk
// whose relative key range is [0, lastRel] (inclusive), in bits,
// without sampling overhead:
//
//	bits = n*ℓ + (n + ⌊(lastRel+1) / 2^ℓ⌋ + 2),  ℓ = ⌊log₂((lastRel+1) / n)⌋.
//
// lastRel = last - base (always ≥ 0 and < 2^64, no overflow).
// Mirrors compact_elias_fano::offsets minus pointers0/pointers1 (which
// are PISA's O(1)-select aux structure; rsdic supplies its own).
func efBitsizePaper(lastRel, n uint64) uint64 {
	if n == 0 {
		return 0
	}
	// universe = lastRel+1, but that may wrap to 0 at lastRel=MaxUint64.
	// For the EF formula the only use of universe is computing ell and
	// the high-part count.  We handle the boundary by computing
	// ell = bits.Len64(lastRel/n) - 1 (same result as Len64((lastRel+1)/n)-1
	// for all lastRel < MaxUint64; at lastRel == MaxUint64 the ±1 is
	// well within the (1+ε) approximation slack of the DP).
	var lo uint64
	if lastRel >= n {
		lo = uint64(bits.Len64(lastRel/n) - 1)
	}
	// numBuckets = ⌊lastRel >> lo⌋ + 1  (no overflow: lastRel < 2^64 and lo ≥ 0)
	higher := n + (lastRel>>lo) + 1 + 2
	return higher + n*lo
}

// bitmapBitsizePaper returns the cost of a flat bitmap chunk:
// universe = lastRel+1 bits. At lastRel == MaxUint64 the true cost is
// 2^64 bits, which is larger than any EF cost, so returning MaxUint64
// is safe for DP comparison purposes.
func bitmapBitsizePaper(lastRel, _ uint64) uint64 {
	if lastRel == ^uint64(0) {
		return ^uint64(0) // sentinel: bitmap impractical at full 2^64 range
	}
	return lastRel + 1
}

// allOnesBitsize is 0 when the chunk is a contiguous run of n integers
// (i.e. lastRel+1 == n, equivalently lastRel == n-1), else infinity.
// Mirrors all_ones_sequence::bitsize.
// The check `lastRel == n-1` is overflow-safe because n ≥ 1 always here.
func allOnesBitsize(lastRel, n uint64) uint64 {
	if lastRel == n-1 {
		return 0
	}
	return ^uint64(0)
}

// One discriminator bit selects between EF and bitmap; all-ones is
// implicit (chosen iff all_ones_bitsize returns 0).
const codecTypeBits = 1

// minCodecBitsize picks the cheapest of {all-ones, EF, bitmap} for a
// chunk of n keys whose inclusive relative range is [0, lastRel].
// Used as the cost function inside optimal_partition's DP.
func minCodecBitsize(lastRel, n uint64) uint64 {
	if lastRel == n-1 {
		return 0 // all-ones — guaranteed minimum, skip EF/bitmap evaluation
	}
	ef := efBitsizePaper(lastRel, n) + codecTypeBits
	bm := bitmapBitsizePaper(lastRel, n)
	// Guard against overflow: bm == MaxUint64 is the infinity sentinel.
	if bm < ^uint64(0) && bm+codecTypeBits < ef {
		return bm + codecTypeBits
	}
	return ef
}
