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
	queryCount := 50000 

	epsilons := []float64{0.1, 0.05, 0.02, 0.01, 0.005, 0.002, 0.001}

	f, _ := os.Create("are_optimized_tradeoff.csv")
	defer f.Close()
	fmt.Fprintln(f, "TargetEpsilon,"+
		"BPK_Opt,FPR_Opt_Unif,FPR_Opt_Seq,"+
		"BPK_Soda,FPR_Soda_Unif,FPR_Soda_Seq,"+
		"BPK_Trunc,FPR_Trunc_Unif,FPR_Trunc_Seq")

	fmt.Printf("%-6s | %-6s | %-10s | %-10s | %-10s | %-10s\n", "Eps", "BPK", "Opt_Seq", "Soda_Seq", "Trunc_Seq", "Trunc_Unif")
	fmt.Println("---------------------------------------------------------------------------------------")

	for _, eps := range epsilons {
		r := rand.New(rand.NewSource(42))
		
		// To BREAK normalization: first key at 0, others at 2^60
		base := uint64(1) << 60
		gap := uint64(200)
		
		uniformKeysBS := make([]bits.BitString, n)
		uniformKeysU64 := make([]uint64, n)
		for i := 0; i < n; i++ {
			val := r.Uint64()
			uniformKeysBS[i] = bits.NewFromUint64WithLength(val, 64)
			uniformKeysU64[i] = val
		}
		
		seqKeysBS := make([]bits.BitString, n)
		seqKeysU64 := make([]uint64, n)
		seqKeysBS[0] = bits.NewFromUint64WithLength(0, 64)
		seqKeysU64[0] = 0
		for i := 1; i < n; i++ {
			val := base + uint64(i)*gap
			seqKeysBS[i] = bits.NewFromUint64WithLength(val, 64)
			seqKeysU64[i] = val
		}

		measure := func(isUnif bool, isEmpty func(a, b uint64) bool) float64 {
			hits := 0
			for i := 0; i < queryCount; i++ {
				var a, b uint64
				if isUnif {
					a = r.Uint64()
					b = a + uint64(r.Intn(int(rangeLen)))
				} else {
					keyIdx := r.Intn(n - 2) + 1 // Pick from keys at 'base'
					a = base + uint64(keyIdx)*gap + 1 
					b = a + rangeLen
				}
				if !isEmpty(a, b) { hits++ }
			}
			return float64(hits) / float64(queryCount)
		}

		// 1. Adaptive
		var bpkOpt, fprOptUnif, fprOptSeq float64
		fOptU, _ := NewOptimizedARE(uniformKeysBS, rangeLen, eps, 0)
		fOptS, _ := NewOptimizedARE(seqKeysBS, rangeLen, eps, 0)
		if fOptS != nil {
			bpkOpt = float64(fOptS.SizeInBits()) / float64(n)
			fprOptUnif = measure(true, func(a, b uint64) bool {
				return fOptU.IsEmpty(bits.NewFromUint64WithLength(a, 64), bits.NewFromUint64WithLength(b, 64))
			})
			fprOptSeq = measure(false, func(a, b uint64) bool {
				return fOptS.IsEmpty(bits.NewFromUint64WithLength(a, 64), bits.NewFromUint64WithLength(b, 64))
			})
		}

		// 2. SODA
		var bpkSoda, fprSodaUnif, fprSodaSeq float64
		fSodaU, _ := are_soda_hash.NewApproximateRangeEmptinessSoda(uniformKeysU64, rangeLen, eps)
		fSodaS, _ := are_soda_hash.NewApproximateRangeEmptinessSoda(seqKeysU64, rangeLen, eps)
		if fSodaS != nil {
			bpkSoda = float64(fSodaS.SizeInBits()) / float64(n)
			fprSodaUnif = measure(true, func(a, b uint64) bool { return fSodaU.IsEmpty(a, b) })
			fprSodaSeq = measure(false, func(a, b uint64) bool { return fSodaS.IsEmpty(a, b) })
		}

		// 3. Truncation
		var bpkTrunc, fprTruncUnif, fprTruncSeq float64
		fTruncU, _ := are.NewApproximateRangeEmptiness(uniformKeysBS, eps)
		fTruncS, _ := are.NewApproximateRangeEmptiness(seqKeysBS, eps)
		if fTruncU != nil {
			bpkTrunc = float64(fTruncU.SizeInBits()) / float64(n)
			fprTruncUnif = measure(true, func(a, b uint64) bool {
				return fTruncU.IsEmpty(bits.NewFromUint64WithLength(a, 64), bits.NewFromUint64WithLength(b, 64))
			})
		}
		if fTruncS != nil {
			fprTruncSeq = measure(false, func(a, b uint64) bool {
				return fTruncS.IsEmpty(bits.NewFromUint64WithLength(a, 64), bits.NewFromUint64WithLength(b, 64))
			})
		}

		fmt.Printf("%-6.3f | %-6.2f | %-10.4f | %-10.4f | %-10.4f | %-10.4f\n", 
			eps, bpkOpt, fprOptSeq, fprSodaSeq, fprTruncSeq, fprTruncUnif)
		
		fmt.Fprintf(f, "%f,%f,%f,%f,%f,%f,%f,%f,%f,%f\n", 
			eps, bpkOpt, fprOptUnif, fprOptSeq, bpkSoda, fprSodaUnif, fprSodaSeq, bpkTrunc, fprTruncUnif, fprTruncSeq)
	}
}
