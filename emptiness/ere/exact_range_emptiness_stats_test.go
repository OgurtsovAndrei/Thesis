package ere

import (
	"Thesis/bits"
	"Thesis/testutils"
	"fmt"
	"math/rand/v2"
	"sort"
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

func TestExactRangeEmptiness_BucketStatsUint64(t *testing.T) {
	sizes := []int{1 << 20, 1 << 24, 1 << 27, 1 << 30}

	for _, n := range sizes {
		t.Run(fmt.Sprintf("N=%d", n), func(t *testing.T) {
			keys := generateSortedUint64(n)

			ere, err := NewExactRangeEmptinessUint64(keys, 64)
			if err != nil {
				t.Fatalf("build failed: %v", err)
			}
			keys = nil // free input array

			stats := ere.GetStats()
			fmt.Printf("\n--- ERE Uint64 Stats (N=%d) ---\n", n)
			fmt.Printf("Total Blocks:    %d\n", stats.NumBlocks)
			fmt.Printf("Non-Empty:       %d\n", stats.NonEmptyBlocks)
			fmt.Printf("Empty:           %d (%.2f%%)\n", stats.EmptyBlocks, stats.EmptyBlockPct)
			fmt.Printf("Avg Keys/Block:  %.2f\n", stats.AvgKeysPerBlock)
			fmt.Printf("Max Keys/Block:  %d\n", stats.MaxKeysInBlock)
			fmt.Printf("-------------------------------\n\n")
		})
	}
}

func generateSortedUint64(n int) []uint64 {
	keys := make([]uint64, n)
	for i := range keys {
		keys[i] = rand.Uint64()
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
