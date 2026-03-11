package are_soda_hash

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func TestSODA_AdversarialFPR_Collision(t *testing.T) {
	n := 10000
	epsilon := 0.01 // Target 1%
	maxQueryLen := uint64(10)
	
	// Generate sequential keys
	rng := rand.New(rand.NewSource(42))
	keys := make([]uint64, n)
	startVal := (rng.Uint64() >> 16) << 16
	for i := 0; i < n; i++ {
		keys[i] = startVal + uint64(i*10) // Spaced by 10 so we have gaps to query
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	filter, err := NewApproximateRangeEmptinessSoda(keys, maxQueryLen, epsilon)
	if err != nil {
		t.Fatalf("Failed to build SODA filter: %v", err)
	}
	
	fpCount := 0
	trials := n - 1

	for i := 0; i < trials; i++ {
		x := keys[i]
		
		// Query [x+1, x+2]. We know gap is 10, so it's empty.
		a := x + 1
		b := x + 2
		
		if !filter.IsEmpty(a, b) {
			fpCount++
		}
	}

	observedFPR := float64(fpCount) / float64(trials)
	fmt.Printf("\n--- SODA Hash Adversarial FPR Report ---\n")
	fmt.Printf("N: %d, Epsilon: %f, K-bits: %d\n", n, epsilon, filter.K)
	fmt.Printf("Trials: %d, False Positives: %d\n", trials, fpCount)
	fmt.Printf("Observed FPR: %f (Target: %f)\n", observedFPR, epsilon)
	
	// With the locality-preserving hash, the FPR should be well within epsilon
	if observedFPR > epsilon * 2 {
		t.Errorf("Adversarial FPR is too high: %f", observedFPR)
	}
}
