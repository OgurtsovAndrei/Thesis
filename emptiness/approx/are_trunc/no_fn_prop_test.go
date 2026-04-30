package are_trunc

import (
	"Thesis/emptiness/internal/testutil"
	"math"
	"math/rand"
	"testing"
)

const (
	testRuns      = 1_000
	minN          = 100
	maxExtraN     = 5000
	targetEpsilon = 0.001
)

// kForProps is the K used by the property tests. Scales with the largest n in
// the run (minN+maxExtraN), preserving the original eps=0.001 BPK target.
func kForProps(n int) uint32 {
	k := uint32(math.Ceil(math.Log2(2.0 * float64(n) / targetEpsilon)))
	if k == 0 {
		k = 1
	}
	if k > 64 {
		k = 64
	}
	return k
}

func TestARE_NoFN_Properties(t *testing.T) {
	t.Parallel()
	testutil.RunUint64NoFNProps(t, testRuns, minN, maxExtraN, ^uint64(0), func(keys []uint64, _ *rand.Rand) (testutil.Uint64Checker, error) {
		return NewTruncARE(keys, 64, Config{K: kForProps(len(keys))})
	})
}

func TestARE_NoFN_Properties_Clustered(t *testing.T) {
	t.Parallel()
	testutil.RunUint64NoFNPropsClustered(t, 200, minN, maxExtraN, ^uint64(0), func(keys []uint64, _ *rand.Rand) (testutil.Uint64Checker, error) {
		return NewTruncARE(keys, 64, Config{K: kForProps(len(keys))})
	})
}
