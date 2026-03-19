package are_soda_hash

import (
	"Thesis/bits"
	"Thesis/emptiness/are_trunc"
	"Thesis/testutils"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func TestMaster_AccuracyRecalculation(t *testing.T) {
	n := 100000
	epsilon := 0.01
	rangeLens := []uint64{1, 10, 100, 1000}

	rng := rand.New(rand.NewSource(42))
	keys := make([]uint64, n)
	bsKeys := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		val := rng.Uint64()
		keys[i] = val
		bsKeys[i] = testutils.TrieBS(val)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	sort.Slice(bsKeys, func(i, j int) bool { return bsKeys[i].Compare(bsKeys[j]) < 0 })

	fmt.Printf("\n=== Accuracy Recalculation (Ground Truth Verified) ===\n")
	fmt.Printf("| RangeLen | Fast FPR | Robust FPR | Fast bits/k | Robust bits/k |\n")
	fmt.Printf("|----------|----------|------------|-------------|---------------|\n")

	for _, L := range rangeLens {
		filterTrunc, _ := are_trunc.NewTruncARE(bsKeys, epsilon)
		filterSoda, _ := NewSodaARE(keys, L, epsilon)

		fpT, fpS := 0, 0
		trials := 1000000
		queriesDone := 0
		
		for queriesDone < trials {
			a := rng.Uint64()
			b := a + L - 1
			if b < a { b = ^uint64(0) }

			// CRITICAL: Ground Truth check
			if testutils.GroundTruth(keys, a, b) {
				queriesDone++
				// Test Truncation (MSB-corrected)
				if !filterTrunc.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) {
					fpT++
				}
				// Test SODA
				if !filterSoda.IsEmpty(a, b) {
					fpS++
				}
			}
		}

		fprT := float64(fpT) / float64(trials)
		fprS := float64(fpS) / float64(trials)
		bitsT := float64(filterTrunc.SizeInBits()) / float64(n)
		bitsS := float64(filterSoda.SizeInBits()) / float64(n)

		fmt.Printf("| %8d | %8.6f | %10.6f | %11.2f | %13.2f |\n", L, fprT, fprS, bitsT, bitsS)
	}
}
