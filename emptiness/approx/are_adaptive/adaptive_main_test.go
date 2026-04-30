package are_adaptive

import (
	"fmt"
	"testing"
)

func TestAdaptiveARE_NormalizationAndTruncation(t *testing.T) {
	base := uint64(1 << 50)

	n := 100
	keys := make([]uint64, n)
	for i := 0; i < n; i++ {
		keys[i] = base + uint64(i*1000)
	}

	// Equivalent to (rangeLen=500, eps=0.01, t=5): K = ceil(log2(n*((L>>t)+1)/eps)) = 14.
	const truncateBits = 5
	const K uint32 = 14

	filter, err := NewAdaptiveARE(keys, 64, Config{K: K, Threshold: truncateBits})
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}

	fmt.Printf("\n--- Adaptive ARE Test ---\n")
	fmt.Printf("Keys: %d, K: %d, Truncate: %d bits\n", n, K, truncateBits)
	fmt.Printf("SODA K: %d bits, ExactMode: %v\n", filter.K, filter.IsExactMode)
	fmt.Printf("Total Size: %d bits (%.2f bits/key)\n", filter.SizeInBits(), float64(filter.SizeInBits())/float64(n))

	// Test 1: Empty range between keys [base+1200, base+1500]
	a1 := base + 1200
	b1 := base + 1500
	res1 := filter.IsEmpty(a1, b1)
	fmt.Printf("Empty Range [base+1200, base+1500]: IsEmpty = %v\n", res1)

	// Test 2: Range containing key base+1000 → [base+900, base+1100]
	a2 := base + 900
	b2 := base + 1100
	res2 := filter.IsEmpty(a2, b2)
	fmt.Printf("Range with Key [base+900, base+1100]: IsEmpty = %v\n", res2)

	if res2 {
		t.Errorf("False Negative: range [base+900, base+1100] should contain key base+1000")
	}
}
