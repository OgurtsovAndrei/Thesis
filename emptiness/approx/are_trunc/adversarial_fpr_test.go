package are_trunc

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
)

// kFromEpsTrunc maps the legacy Trunc target eps to K = ceil(log2(2n/eps)).
func kFromEpsTrunc(n int, eps float64) uint32 {
	k := uint32(math.Ceil(math.Log2(2.0 * float64(n) / eps)))
	if k == 0 {
		k = 1
	}
	if k > 64 {
		k = 64
	}
	return k
}

// TestARE_FPR_RandomEmptyRanges measures FPR on random point queries
// that are guaranteed to be empty. This validates the epsilon bound.
//
// Note: adversarial queries (e.g. gaps between consecutive keys) give FPR≈100%
// for prefix truncation — that's a known theoretical limitation, not a bug.
// ARE only guarantees epsilon-bounded FPR for random/oblivious queries.
func TestARE_FPR_RandomEmptyRanges(t *testing.T) {
	n := 10000
	epsilon := 0.01

	rng := rand.New(rand.NewSource(42))
	keySet := make(map[uint64]bool)
	keys := make([]uint64, 0, n)
	for len(keys) < n {
		val := rng.Uint64()
		if !keySet[val] {
			keySet[val] = true
			keys = append(keys, val)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	filter, err := NewTruncARE(keys, 64, Config{K: kFromEpsTrunc(len(keys), epsilon)})
	if err != nil {
		t.Fatal(err)
	}

	fpCount := 0
	trials := 0
	numQueries := 100_000

	for i := 0; i < numQueries; i++ {
		q := rng.Uint64()
		if keySet[q] {
			continue
		}
		trials++
		if !filter.IsEmpty(q, q) {
			fpCount++
		}
	}

	observedFPR := float64(fpCount) / float64(trials)
	fmt.Printf("\n--- ARE FPR (random empty point queries) ---\n")
	fmt.Printf("N: %d, Epsilon: %f, K-bits: %d\n", n, epsilon, filter.K)
	fmt.Printf("Trials: %d, False Positives: %d\n", trials, fpCount)
	fmt.Printf("Observed FPR: %f (Target: %f)\n", observedFPR, epsilon)

	// Allow 2x margin over target epsilon
	if observedFPR > 2*epsilon {
		t.Errorf("FPR too high: observed %f, target %f", observedFPR, epsilon)
	}
}
