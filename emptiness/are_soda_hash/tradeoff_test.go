package are_soda_hash

import (
	"Thesis/bits"
	"Thesis/emptiness/are_trunc"
	"Thesis/testutils"
	"fmt"
	"testing"
)

func TestSequentialSweep_Corrected(t *testing.T) {
	n := 10000
	epsilon := 0.01
	step := uint64(1000)
	// We sweep the range length L
	rangeLens := []uint64{1, 2, 4, 8, 16, 32, 64, 128, 256, 512}

	keys := make([]uint64, n)
	bsKeys := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		val := uint64(i) * step
		keys[i] = val
		bsKeys[i] = testutils.TrieBS(val)
	}

	fmt.Printf("\n--- Corrected Sequential Sweep (N=%d, Step=%d, eps=0.01) ---\n", n, step)
	fmt.Printf("| RangeLen | Trunc Bits | Soda Bits | Trunc FPR | Soda FPR |\n")
	fmt.Printf("|----------|------------|-----------|-----------|----------|\n")

	for _, L := range rangeLens {
		filterTrunc, _ := are_trunc.NewApproximateRangeEmptiness(bsKeys, epsilon)
		filterSoda, _ := NewApproximateRangeEmptinessSoda(keys, L, epsilon)

		fpT, fpS := 0, 0
		trials := n - 1
		for i := 0; i < trials; i++ {
			// Guaranteed empty range [key + 1, key + L]
			// as long as L < step (1000)
			a := keys[i] + 1
			b := a + L - 1
			
			if !filterTrunc.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) {
				fpT++
			}
			if !filterSoda.IsEmpty(a, b) {
				fpS++
			}
		}

		bitsT := float64(filterTrunc.SizeInBits()) / float64(n)
		bitsS := float64(filterSoda.SizeInBits()) / float64(n)
		fprT := float64(fpT) / float64(trials)
		fprS := float64(fpS) / float64(trials)

		fmt.Printf("| %8d | %10.2f | %9.2f | %9.4f | %8.4f |\n", L, bitsT, bitsS, fprT, fprS)
	}
}
