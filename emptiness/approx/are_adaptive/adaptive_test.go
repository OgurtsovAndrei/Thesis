package are_adaptive

import (
	"fmt"
	"testing"
)

func TestAdaptiveARE_AdaptiveMode(t *testing.T) {
	n := 100

	// Equivalent to (rangeLen=100, eps=0.01): K = ceil(log2(n*(L+1)/eps)) ≈ 20 bits.
	const K uint32 = 20

	// Scenario 1: Compact data (Exact Mode)
	// Keys: 1000, 1010, ..., 1990 — spread M = log2(990) ~ 10 bits
	// M (10) <= K (20) => Exact Mode
	keysCompact := make([]uint64, n)
	for i := 0; i < n; i++ {
		keysCompact[i] = uint64(1000 + i*10)
	}

	filterCompact, err := NewAdaptiveARE(keysCompact, 64, Config{K: K, Threshold: 0})
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
	// K = 20 bits => M > K => SODA Mode
	keysSpread := make([]uint64, n)
	for i := 0; i < n; i++ {
		keysSpread[i] = uint64(i) * 1_000_000_000_000
	}

	filterSpread, err := NewAdaptiveARE(keysSpread, 64, Config{K: K, Threshold: 0})
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
