package are_trunc

import (
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
	testutil.RunUint64NoFNProps(t, testRuns, minN, maxExtraN, ^uint64(0), func(keys []uint64, _ *rand.Rand) (testutil.Uint64Checker, error) {
		return NewTruncARE(keys, 64, Config{Eps: targetEpsilon})
	})
}

func TestARE_NoFN_Properties_Clustered(t *testing.T) {
	t.Parallel()
	testutil.RunUint64NoFNPropsClustered(t, 200, minN, maxExtraN, ^uint64(0), func(keys []uint64, _ *rand.Rand) (testutil.Uint64Checker, error) {
		return NewTruncARE(keys, 64, Config{Eps: targetEpsilon})
	})
}
