package are_optimized

import (
	"Thesis/bits"
	"Thesis/emptiness/are"
	"Thesis/emptiness/are_soda_hash"
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func TestTradeoff_FPR_vs_BPK(t *testing.T) {
	n := 10000
	rangeLen := uint64(100)
	queryCount := 100000 

	epsilons := []float64{0.1, 0.05, 0.02, 0.01, 0.005, 0.002, 0.001}

	f, _ := os.Create("are_optimized_tradeoff.csv")
	defer f.Close()
	fmt.Fprintln(f, "TargetEpsilon,BPK_Opt,FPR_Opt_Unif,FPR_Opt_Seq,BPK_Soda,FPR_Soda_Seq,BPK_Trunc,FPR_Trunc_Seq")

	fmt.Printf("%-8s | %-6s | %-10s | %-10s | %-10s | %-10s\n", "Eps", "BPK_Opt", "Opt_Unif", "Opt_Seq", "Soda_Seq", "Trunc_Seq")
	fmt.Println("----------------------------------------------------------------------------")

	for _, eps := range epsilons {
		r := rand.New(rand.NewSource(42))
		
		// Use a large but non-overflowing gap
		// 10^14 * 10^4 = 10^18 < 2^64 (~1.8*10^19)
		gap := uint64(100000000000000)
		
		uniformKeysBS := make([]bits.BitString, n)
		for i := 0; i < n; i++ {
			val := r.Uint64()
			uniformKeysBS[i] = bits.NewFromUint64WithLength(val, 64)
		}
		
		seqKeysBS := make([]bits.BitString, n)
		seqKeysU64 := make([]uint64, n)
		for i := 0; i < n; i++ {
			val := uint64(i) * gap
			seqKeysBS[i] = bits.NewFromUint64WithLength(val, 64)
			seqKeysU64[i] = val
		}

		// --- 1. Optimized Adaptive ARE ---
		filterOpt, err := NewOptimizedARE(seqKeysBS, rangeLen, eps, 0)
		bpkOpt := 0.0
		fprOptUnif := 0.0
		fprOptSeq := 0.0
		if err == nil && filterOpt != nil {
			bpkOpt = float64(filterOpt.SizeInBits()) / float64(n)
			
			filterOptUnif, _ := NewOptimizedARE(uniformKeysBS, rangeLen, eps, 0)
			fpOptUnif := 0
			for i := 0; i < queryCount; i++ {
				a, b := r.Uint64(), r.Uint64()
				if a > b { a, b = b, a }
				if b - a > rangeLen { b = a + uint64(r.Intn(int(rangeLen))) }
				if filterOptUnif.IsEmpty(bits.NewFromUint64WithLength(a, 64), bits.NewFromUint64WithLength(b, 64)) == false {
					fpOptUnif++
				}
			}
			fprOptUnif = float64(fpOptUnif) / float64(queryCount)

			fpOptSeq := 0
			for i := 0; i < queryCount; i++ {
				keyIdx := r.Intn(n-1)
				a := uint64(keyIdx)*gap + 1
				b := a + uint64(r.Intn(int(rangeLen)))
				if filterOpt.IsEmpty(bits.NewFromUint64WithLength(a, 64), bits.NewFromUint64WithLength(b, 64)) == false {
					fpOptSeq++
				}
			}
			fprOptSeq = float64(fpOptSeq) / float64(queryCount)
		}

		// --- 2. Original SODA ARE ---
		filterSoda, errSoda := are_soda_hash.NewApproximateRangeEmptinessSoda(seqKeysU64, rangeLen, eps)
		bpkSoda := 0.0
		fprSodaSeq := 0.0
		if errSoda == nil && filterSoda != nil {
			bpkSoda = float64(filterSoda.SizeInBits()) / float64(n)
			fpSodaSeq := 0
			for i := 0; i < queryCount; i++ {
				keyIdx := r.Intn(n-1)
				a := uint64(keyIdx)*gap + 1
				b := a + uint64(r.Intn(int(rangeLen)))
				if filterSoda.IsEmpty(a, b) == false {
					fpSodaSeq++
				}
			}
			fprSodaSeq = float64(fpSodaSeq) / float64(queryCount)
		}

		// --- 3. Truncation ARE ---
		filterTrunc, errTrunc := are.NewApproximateRangeEmptiness(seqKeysBS, eps)
		bpkTrunc := 0.0
		fprTruncSeq := 0.0
		if errTrunc == nil && filterTrunc != nil {
			bpkTrunc = float64(filterTrunc.SizeInBits()) / float64(n)
			fpTruncSeq := 0
			for i := 0; i < queryCount; i++ {
				keyIdx := r.Intn(n-1)
				a := uint64(keyIdx)*gap + 1
				b := a + uint64(r.Intn(int(rangeLen)))
				if filterTrunc.IsEmpty(bits.NewFromUint64WithLength(a, 64), bits.NewFromUint64WithLength(b, 64)) == false {
					fpTruncSeq++
				}
			}
			fprTruncSeq = float64(fpTruncSeq) / float64(queryCount)
		}

		fmt.Printf("%-8.3f | %-7.2f | %-10.6f | %-10.6f | %-10.6f | %-10.6f\n", 
			eps, bpkOpt, fprOptUnif, fprOptSeq, fprSodaSeq, fprTruncSeq)
		
		fmt.Fprintf(f, "%f,%f,%f,%f,%f,%f,%f,%f\n", 
			eps, bpkOpt, fprOptUnif, fprOptSeq, bpkSoda, fprSodaSeq, bpkTrunc, fprTruncSeq)
	}
}
