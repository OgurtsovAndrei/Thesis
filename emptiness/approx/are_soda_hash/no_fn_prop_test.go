package are_soda_hash

import (
	"Thesis/emptiness/internal/testutil"
	"math/rand"
	"testing"
)

const (
	testRuns      = 1_000
	minN          = 100
	maxExtraN     = 1000
	targetEpsilon = 0.001
	maxQueryLen   = uint64(1000)
)

func TestSODA_NoFN_Properties(t *testing.T) {
	t.Parallel()
	testutil.RunUint64NoFNProps(t, testRuns, minN, maxExtraN, maxQueryLen, func(keys []uint64, _ *rand.Rand) (testutil.Uint64Checker, error) {
		return NewSodaARE(keys, maxQueryLen, targetEpsilon)
	})
}

func TestSODA_NoFN_Properties_Clustered(t *testing.T) {
	t.Parallel()
	testutil.RunUint64NoFNPropsClustered(t, 200, minN, maxExtraN, maxQueryLen, func(keys []uint64, _ *rand.Rand) (testutil.Uint64Checker, error) {
		return NewSodaARE(keys, maxQueryLen, targetEpsilon)
	})
}
