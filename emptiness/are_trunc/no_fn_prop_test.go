package are_trunc

import (
	"Thesis/bits"
	"Thesis/emptiness/internal/testutil"
	"math/rand"
	"testing"
)

const (
	testRuns      = 1_000
	minN          = 100
	maxExtraN     = 5000
	targetEpsilon = 0.001
)

func TestARE_NoFN_Properties(t *testing.T) {
	t.Parallel()
	testutil.RunBitStringNoFNProps(t, testRuns, minN, maxExtraN, func(keys []bits.BitString, _ *rand.Rand) (testutil.BitStringChecker, error) {
		return NewTruncARE(keys, targetEpsilon)
	})
}

func TestARE_NoFN_Properties_Clustered(t *testing.T) {
	t.Parallel()
	testutil.RunBitStringNoFNPropsClustered(t, 200, minN, maxExtraN, func(keys []bits.BitString, _ *rand.Rand) (testutil.BitStringChecker, error) {
		return NewTruncARE(keys, targetEpsilon)
	})
}
