package are_trunc

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

func TestApproximateRangeEmptiness_Accuracy(t *testing.T) {
	n := 100000
	epsilons := []float64{0.01, 0.001}

	rng := rand.New(rand.NewSource(42))
	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)
	for len(keys) < n {
		v := rng.Uint64()
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	// NewTruncARE sorts internally
	for _, targetEps := range epsilons {
		t.Run(fmt.Sprintf("Eps=%v", targetEps), func(t *testing.T) {
			K := uint32(math.Ceil(math.Log2(2.0 * float64(n) / targetEps)))
			are, err := NewTruncARE(keys, 64, Config{K: K})
			if err != nil {
				t.Fatalf("Failed to build ARE: %v", err)
			}

			queryRng := rand.New(rand.NewSource(12345))

			numQueries := 100000
			falsePositives := 0
			queriesPerformed := 0

			// Build a set of normalized block indices for all stored keys,
			// so we can skip queries that fall in an occupied block (unavoidable FPs).
			prefixes := make(map[uint64]bool)
			for _, k := range keys {
				prefixes[normalizeToK(k, are.minKey, are.spreadLen, are.K)] = true
			}

			for queriesPerformed < numQueries {
				val := queryRng.Uint64()

				// Skip if this query key maps to an occupied block
				if prefixes[normalizeToK(val, are.minKey, are.spreadLen, are.K)] {
					continue
				}

				queriesPerformed++
				if !are.IsEmpty(val, val) {
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
