package zfasttrie

import "math/bits"

// MostSignificantBit returns the index of the most significant bit.
// Ported from Fast::mostSignificantBit
func MostSignificantBit(x uint64) int {
	if x == 0 {
		return -1
	}
	// 63 - bits.LeadingZeros64(x) behaves identically to
	// 63 - __builtin_clzll(x)
	return 63 - bits.LeadingZeros64(x)
}

// TwoFattest returns the result of the "two fattest" bit operation on [a, b]
// Ported from Fast::twoFattest
func TwoFattest(a uint64, b uint64) uint64 {
	if a == b {
		return 0
	}
	msb := MostSignificantBit(a ^ b)
	if msb == -1 {
		return 0
	}
	// C++: (LONG_ALL_ONE << mostSignificantBit(a ^ b)) & b
	// Go: (^uint64(0) << uint(msb)) & b
	return ((^uint64(0)) << uint(msb)) & b
}
