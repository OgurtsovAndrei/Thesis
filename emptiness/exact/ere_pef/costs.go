// Package ere_pef implements 1D exact range emptiness using
// Partitioned Elias-Fano (Ottaviano & Venturini, SIGIR 2014). The
// implementation mirrors PISA (pisa-engine/pisa, Apache 2.0): cost
// functions and partition algorithm follow the paper formulas;
// per-chunk encoders are simplified to reuse Thesis primitives
// (succinct_bit_vector/rsdic, bits.UnpackBit) instead of PISA's
// bespoke bitvector layout.
package ere_pef

import "math/bits"

// efBitsizePaper returns the EF cost of encoding n keys into
// universe [0, universe), in bits, without sampling overhead:
//
//	bits = n*ℓ + (n + ⌊universe / 2^ℓ⌋ + 2),  ℓ = ⌊log₂(universe / n)⌋.
//
// Mirrors compact_elias_fano::offsets minus pointers0/pointers1 (which
// are PISA's O(1)-select aux structure; rsdic supplies its own).
func efBitsizePaper(universe, n uint64) uint64 {
	if n == 0 {
		return 0
	}
	var lo uint64
	if universe > n {
		lo = uint64(bits.Len64(universe/n) - 1)
	}
	higher := n + (universe >> lo) + 2
	return higher + n*lo
}

// bitmapBitsizePaper returns the cost of a flat bitmap chunk: just
// `universe` bits. Mirrors compact_ranked_bitvector minus rank/sample
// pointers; rsdic supplies them in the actual encoder.
func bitmapBitsizePaper(universe, _ uint64) uint64 {
	return universe
}

// allOnesBitsize is 0 when universe == n (a contiguous run of n
// integers), else infinity. Mirrors all_ones_sequence::bitsize.
func allOnesBitsize(universe, n uint64) uint64 {
	if universe == n {
		return 0
	}
	return ^uint64(0)
}

// One discriminator bit selects between EF and bitmap; all-ones is
// implicit (chosen iff all_ones_bitsize returns 0).
const codecTypeBits = 1

// minCodecBitsize picks the cheapest of {all-ones, EF, bitmap} for a
// chunk of n keys over universe of size `universe`. Used as the
// cost function inside optimal_partition's DP.
func minCodecBitsize(universe, n uint64) uint64 {
	if universe == n {
		return 0 // all-ones — guaranteed minimum, skip EF/bitmap evaluation
	}
	ef := efBitsizePaper(universe, n) + codecTypeBits
	if bm := bitmapBitsizePaper(universe, n) + codecTypeBits; bm < ef {
		return bm
	}
	return ef
}
