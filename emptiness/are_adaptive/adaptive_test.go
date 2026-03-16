package are_adaptive

import (
	"Thesis/bits"
	"fmt"
	"testing"
)

func TestAdaptiveARE_AdaptiveMode(t *testing.T) {
	n := 100
	rangeLen := uint64(100)
	epsilon := 0.01

	// Scenario 1: Compact data (Exact Mode)
	// Keys: 1000, 1010, ..., 1990 — spread M = log2(990) ~ 10 bits
	// Required K for eps=0.01, n=100, L=100: log2(100*101/0.01) ~ 20 bits
	// M (10) <= K (20) => Exact Mode
	keysCompact := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		keysCompact[i] = bits.NewFromTrieUint64(uint64(1000+i*10), 64)
	}

	filterCompact, err := NewAdaptiveARE(keysCompact, rangeLen, epsilon, 0)
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	fmt.Printf("\n--- Compact Data Test ---\n")
	fmt.Printf("IsExactMode: %v (Expected: true)\n", filterCompact.IsExactMode)
	fmt.Printf("K (bits): %d\n", filterCompact.K)

	if !filterCompact.IsExactMode {
		t.Errorf("Expected Exact Mode for compact data")
	}

	// Scenario 2: Spread data (SODA Mode)
	// Keys: 0, 10^12, 2*10^12, ... — spread M ~ 46 bits
	// K ~ 20 bits => M > K => SODA Mode
	keysSpread := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		keysSpread[i] = bits.NewFromTrieUint64(uint64(i)*1_000_000_000_000, 64)
	}

	filterSpread, err := NewAdaptiveARE(keysSpread, rangeLen, epsilon, 0)
	if err != nil {
		t.Fatalf("Spread: %v", err)
	}
	fmt.Printf("\n--- Spread Data Test ---\n")
	fmt.Printf("IsExactMode: %v (Expected: false)\n", filterSpread.IsExactMode)
	fmt.Printf("K (bits): %d\n", filterSpread.K)

	if filterSpread.IsExactMode {
		t.Errorf("Expected SODA Mode for spread data")
	}
}
