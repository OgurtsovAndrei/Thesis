package are_trunc

import (
	"Thesis/bits"
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

			// Build a set of normalized block indices for all stored keys,
			// so we can skip queries that fall in an occupied block (unavoidable FPs).
			prefixes := make(map[uint64]bool)
			for _, k := range keys {
				prefixes[normalizeToK(k, are.minKey, are.spreadStart, are.K).TrieUint64()] = true
			}

			for queriesPerformed < numQueries {
				val := rng.Uint64()
				queryBs := bits.NewFromUint64(val)

				// Skip if this query key maps to an occupied block
				if prefixes[normalizeToK(queryBs, are.minKey, are.spreadStart, are.K).TrieUint64()] {
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

			if actualFPR > targetEps*1.5 {
				t.Errorf("Observed FPR %.6f exceeds target epsilon %v", actualFPR, targetEps)
			}
			fmt.Printf("----------------------------------------------\n")
		})
	}
}
