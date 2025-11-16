package bits

import "math/bits"

// MostSignificantBit returns the index of the most significant bit.
func MostSignificantBit(x uint64) int {
	if x == 0 {
		return -1
	}
	// 63 - bits.LeadingZeros64(x) behaves identically to
	// 63 - __builtin_clzll(x)
	return 63 - bits.LeadingZeros64(x)
}

// TwoFattest returns the result of the "two fattest" bit operation on (a, b]
// Ported from Fast::twoFattest
func TwoFattest(a uint64, b uint64) uint64 {
	if a == b {
		return 0 // case (x, x]
	}
	//BugOn(a > b, "illegal arguments")
	msb := MostSignificantBit(a ^ b)
	if msb == -1 {
		return a
	}
	// C++: (LONG_ALL_ONE << mostSignificantBit(a ^ b)) & b
	// Go: (^uint64(0) << uint(msb)) & b
	res := ((^uint64(0)) << uint(msb)) & b
	return res
}

func trailingZeros(n uint64) uint64 {
	if n == 0 {
		return 0
	}
	return uint64(bits.TrailingZeros64(n))
}

func findTwoFattestMath(a uint64, b uint64) uint64 {
	maxFattest := uint64(0)
	maxFattestLength := uint64(0)

	for i := a; i <= b; i++ {
		currentFattest := trailingZeros(i)

		if currentFattest > maxFattest {
			maxFattest = currentFattest
			maxFattestLength = i
		}
	}
	return maxFattestLength
}
