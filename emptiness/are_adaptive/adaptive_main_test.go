package are_adaptive

import (
	"Thesis/bits"
	"fmt"
	"testing"
)

func TestAdaptiveARE_NormalizationAndTruncation(t *testing.T) {
	// Keys with a large base offset to test normalization (minKey subtraction)
	base := bits.NewFromTrieUint64(1<<50, 64)

	n := 100
	keys := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		keys[i] = base.Add(bits.NewFromTrieUint64(uint64(i*1000), 64))
	}

	rangeLen := uint64(500)
	epsilon := 0.01
	truncateBits := uint32(5)

	filter, err := NewAdaptiveARE(keys, rangeLen, epsilon, truncateBits)
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}

	fmt.Printf("\n--- Adaptive ARE Test ---\n")
	fmt.Printf("Keys: %d, RangeLen: %d, Truncate: %d bits\n", n, rangeLen, truncateBits)
	fmt.Printf("SODA K: %d bits, ExactMode: %v\n", filter.K, filter.IsExactMode)
	fmt.Printf("Total Size: %d bits (%.2f bits/key)\n", filter.SizeInBits(), float64(filter.SizeInBits())/float64(n))

	// Test 1: Empty range between keys [base+1200, base+1500]
	a1 := base.Add(bits.NewFromTrieUint64(1200, 64))
	b1 := base.Add(bits.NewFromTrieUint64(1500, 64))
	res1 := filter.IsEmpty(a1, b1)
	fmt.Printf("Empty Range [base+1200, base+1500]: IsEmpty = %v\n", res1)

	// Test 2: Range containing key base+1000 → [base+900, base+1100]
	a2 := base.Add(bits.NewFromTrieUint64(900, 64))
	b2 := base.Add(bits.NewFromTrieUint64(1100, 64))
	res2 := filter.IsEmpty(a2, b2)
	fmt.Printf("Range with Key [base+900, base+1100]: IsEmpty = %v\n", res2)

	if res2 {
		t.Errorf("False Negative: range [base+900, base+1100] should contain key base+1000")
	}
}
