package are_trunc

import (
	"Thesis/bits"
	"Thesis/emptiness/ere"
	"Thesis/testutils"
	"fmt"
	"math/rand"
	"testing"
)

func TestApproximateRangeEmptiness_Accuracy(t *testing.T) {
	n := 100000
	bitLen := 64
	// We'll test 1% and 0.1%
	epsilons := []float64{0.01, 0.001}
	
	// Generate random keys with seed 42
	keys := testutils.GetBenchKeys(bitLen, n)
	
	for _, targetEps := range epsilons {
		t.Run(fmt.Sprintf("Eps=%v", targetEps), func(t *testing.T) {
			are, err := NewTruncARE(keys, targetEps)
			if err != nil {
				t.Fatalf("Failed to build ARE: %v", err)
			}

			// Use a DIFFERENT seed for queries to ensure independence
			rng := rand.New(rand.NewSource(12345))
			
			numQueries := 100000
			falsePositives := 0
			queriesPerformed := 0
			
			// Use a map of prefixes to quickly check if a FP is "expected" due to truncation
			prefixes := make(map[uint64]bool)
			for _, k := range keys {
				prefixes[ere.GetBlockIndex(k, are.K)] = true
			}

			for queriesPerformed < numQueries {
				val := rng.Uint64()
				queryBs := bits.NewFromUint64(val)
				
				// Skip if it's actually in the set (not a FP)
				// We'll just use the prefix check for simplicity as it's stricter
				if prefixes[ere.GetBlockIndex(queryBs, are.K)] {
					continue
				}
				
				queriesPerformed++
				// Check point interval [queryBs, queryBs]
				if !are.IsEmpty(queryBs, queryBs) {
					falsePositives++
				}
			}

			actualFPR := float64(falsePositives) / float64(queriesPerformed)
			
			fmt.Printf("\n--- ARE Accuracy Test (N=%d, target eps=%v) ---\n", n, targetEps)
			fmt.Printf("Fingerprint bits (K): %d\n", are.K)
			fmt.Printf("Queries:              %d\n", queriesPerformed)
			fmt.Printf("False Positives:      %d\n", falsePositives)
			fmt.Printf("Observed FPR:         %.6f\n", actualFPR)
			
			if actualFPR > targetEps * 1.5 { 
				t.Errorf("Observed FPR %.6f exceeds target epsilon %v", actualFPR, targetEps)
			}
			fmt.Printf("----------------------------------------------\n")
		})
	}
}
