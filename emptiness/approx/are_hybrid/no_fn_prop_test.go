package are_hybrid

import (
	"Thesis/bits"
	"Thesis/emptiness/internal/testutil"
	"math/rand"
	"testing"
)

const (
	propTestRuns      = 1_000
	propMinN          = 100
	propMaxExtraN     = 5000
	propTargetEpsilon = 0.001
	propRangeLen      = uint64(100)
)

func TestHybridARE_NoFN_Properties(t *testing.T) {
	t.Parallel()
	testutil.RunBitStringNoFNProps(t, propTestRuns, propMinN, propMaxExtraN, func(keys []bits.BitString, _ *rand.Rand) (testutil.BitStringChecker, error) {
		return NewHybridARE(keys, propRangeLen, propTargetEpsilon)
	})
}

func TestHybridARE_NoFN_Properties_Clustered(t *testing.T) {
	t.Parallel()
	testutil.RunBitStringNoFNPropsClustered(t, 200, propMinN, propMaxExtraN, func(keys []bits.BitString, _ *rand.Rand) (testutil.BitStringChecker, error) {
		return NewHybridARE(keys, propRangeLen, propTargetEpsilon)
	})
}
