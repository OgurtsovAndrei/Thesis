package testutils

import "math/rand"

// SampleGaussian returns a uint64 drawn from N(center, stddev), clamped to [0, maxUint64].
func SampleGaussian(center uint64, stddev float64, rng *rand.Rand) uint64 {
	offset := rng.NormFloat64() * stddev
	if offset >= 0 {
		v := center + uint64(offset)
		if v < center {
			return 0
		}
		return v
	}
	neg := uint64(-offset)
	if neg > center {
		return 0
	}
	return center - neg
}
