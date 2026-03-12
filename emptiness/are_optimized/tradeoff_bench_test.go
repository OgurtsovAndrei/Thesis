package are_optimized

import (
	"Thesis/bits"
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func TestTradeoff_FPR_vs_BPK(t *testing.T) {
	n := 10000
	rangeLen := uint64(100)
	queryCount := 100000

	// We'll test across different target epsilons
	epsilons := []float64{0.1, 0.05, 0.02, 0.01, 0.005, 0.002, 0.001, 0.0005, 0.0002, 0.0001}

	// Output CSV
	cwd, _ := os.Getwd()
	fmt.Println("CWD of test:", cwd)
	f, _ := os.Create("are_optimized_tradeoff.csv")
	defer f.Close()
	fmt.Fprintln(f, "TargetEpsilon,BitsPerKey,ActualFPR_Uniform,ActualFPR_Sequential,K,Mode")

	fmt.Printf("%-10s | %-8s | %-12s | %-12s | %-4s | %-10s\n", "Epsilon", "BPK", "FPR_Unif", "FPR_Seq", "K", "Mode")
	fmt.Println("----------------------------------------------------------------------------")

	for _, eps := range epsilons {
		// 1. Uniform Keys
		uniformKeys := make([]bits.BitString, n)
		r := rand.New(rand.NewSource(42))
		for i := 0; i < n; i++ {
			uniformKeys[i] = bits.NewFromUint64WithLength(r.Uint64(), 64)
		}
		
		// 2. Sequential Keys (Adversarial for Truncation, now Robust in SODA)
		seqKeys := make([]bits.BitString, n)
		for i := 0; i < n; i++ {
			seqKeys[i] = bits.NewFromUint64WithLength(uint64(i*1000), 64)
		}

		// Create Filter for Uniform
		filterUnif, _ := NewOptimizedARE(uniformKeys, rangeLen, eps, 0)
		bpk := float64(filterUnif.SizeInBits()) / float64(n)

		// Measure FPR for Uniform
		falsePositivesUnif := 0
		for i := 0; i < queryCount; i++ {
			a := r.Uint64()
			b := a + uint64(r.Intn(int(rangeLen)))
			
			// Simple ground truth: for uniform, it's very likely empty
			// For precise measure, we'd need a map, but for 10k keys in 2^64 it's fine.
			if filterUnif.IsEmpty(bits.NewFromUint64WithLength(a, 64), bits.NewFromUint64WithLength(b, 64)) == false {
				falsePositivesUnif++
			}
		}
		actualFPRUnif := float64(falsePositivesUnif) / float64(queryCount)

		// Measure FPR for Sequential
		filterSeq, _ := NewOptimizedARE(seqKeys, rangeLen, eps, 0)
		falsePositivesSeq := 0
		for i := 0; i < queryCount; i++ {
			// Query in gaps: e.g. [500, 600], [1500, 1600]
			keyIdx := r.Intn(n - 1)
			a := uint64(keyIdx*1000 + 500)
			b := a + uint64(r.Intn(int(rangeLen)))
			
			if filterSeq.IsEmpty(bits.NewFromUint64WithLength(a, 64), bits.NewFromUint64WithLength(b, 64)) == false {
				falsePositivesSeq++
			}
		}
		actualFPRSeq := float64(falsePositivesSeq) / float64(queryCount)

		mode := "SODA"
		if filterUnif.IsExactMode {
			mode = "Exact"
		}

		fmt.Printf("%-10.4f | %-8.2f | %-12.6f | %-12.6f | %-4d | %-10s\n", 
			eps, bpk, actualFPRUnif, actualFPRSeq, filterUnif.K, mode)
		
		fmt.Fprintf(f, "%f,%f,%f,%f,%d,%s\n", 
			eps, bpk, actualFPRUnif, actualFPRSeq, filterUnif.K, mode)
	}
}
