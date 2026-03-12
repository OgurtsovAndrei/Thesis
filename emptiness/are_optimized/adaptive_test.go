package are_optimized

import (
	"Thesis/bits"
	"fmt"
	"testing"
)

func TestOptimizedARE_AdaptiveMode(t *testing.T) {
	n := 100
	rangeLen := uint64(100)
	epsilon := 0.01

	// Сценарий 1: Компактные данные (Exact Mode)
	// Ключи от 1000 до 1000 + 100*10 = 2000.
	// Разброс M = log2(1000) ~ 10 бит.
	// Требуемое K для epsilon=0.01 при n=100 и L=100: log2(100*100/0.01) = log2(1,000,000) ~ 20 бит.
	// M (10) <= K (20) => Должен быть Exact Mode.
	keysCompact := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		keysCompact[i] = bits.NewFromUint64WithLength(uint64(1000+i*10), 64)
	}

	filterCompact, _ := NewOptimizedARE(keysCompact, rangeLen, epsilon, 0)
	fmt.Printf("\n--- Compact Data Test ---\n")
	fmt.Printf("IsExactMode: %v (Expected: true)\n", filterCompact.IsExactMode)
	fmt.Printf("K (bits): %d\n", filterCompact.K)

	if !filterCompact.IsExactMode {
		t.Errorf("Expected Exact Mode for compact data")
	}

	// Сценарий 2: Разреженные данные (SODA Mode)
	// Ключи: 0, 10^12, 2*10^12...
	// Разброс M = log2(10^14) ~ 46 бит.
	// K по-прежнему ~20 бит.
	// M (46) > K (20) => Должен быть SODA Mode.
	keysSpread := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		keysSpread[i] = bits.NewFromUint64WithLength(uint64(i)*1000000000000, 64)
	}

	filterSpread, _ := NewOptimizedARE(keysSpread, rangeLen, epsilon, 0)
	fmt.Printf("\n--- Spread Data Test ---\n")
	fmt.Printf("IsExactMode: %v (Expected: false)\n", filterSpread.IsExactMode)
	fmt.Printf("K (bits): %d\n", filterSpread.K)

	if filterSpread.IsExactMode {
		t.Errorf("Expected SODA Mode for spread data")
	}
}
