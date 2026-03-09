package local_exact_range

import (
	"Thesis/bits"
	"Thesis/testutils"
	"fmt"
	"testing"
)

func TestExactRangeEmptiness_RealStats(t *testing.T) {
	n := 1_000_000
	bitLen := 64
	keys := testutils.GetBenchKeys(bitLen, n)
	universe := bits.NewBitString(uint32(bitLen))
	ere, _ := NewExactRangeEmptiness(keys, universe)
	
	stats := ere.GetStats()
	fmt.Printf("\n--- ExactRangeEmptiness Stats (N=%d) ---\n", n)
	fmt.Printf("Total Blocks:    %d\n", stats.NumBlocks)
	fmt.Printf("Non-Empty:       %d\n", stats.NonEmptyBlocks)
	fmt.Printf("Empty:           %d (%.2f%%)\n", stats.EmptyBlocks, stats.EmptyBlockPct)
	fmt.Printf("Avg Keys/Block:  %.2f\n", stats.AvgKeysPerBlock)
	fmt.Printf("Max Keys/Block:  %d\n", stats.MaxKeysInBlock)
	fmt.Printf("--------------------------------------\n\n")
}
