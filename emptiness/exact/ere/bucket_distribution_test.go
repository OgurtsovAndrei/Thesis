package ere

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
)

// TestBucketSizeDistribution empirically measures the bucket-size distribution
// of ERE for n = 2^20 uniform random 64-bit keys with B = n blocks (lambda = 1).
// Used to validate analytical Poisson(1) predictions in the thesis.
func TestBucketSizeDistribution(t *testing.T) {
	const (
		logN = 20
		n    = 1 << logN
		seed = 42
	)

	r := rand.New(rand.NewSource(seed))
	keys := make([]uint64, n)
	seen := make(map[uint64]struct{}, n)
	for i := 0; i < n; i++ {
		for {
			k := r.Uint64()
			if _, dup := seen[k]; !dup {
				seen[k] = struct{}{}
				keys[i] = k
				break
			}
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	ere, err := NewExactRangeEmptiness(keys, 64)
	if err != nil {
		t.Fatalf("NewExactRangeEmptiness failed: %v", err)
	}

	stats := ere.GetStats()
	if stats.NumBlocks != n {
		t.Fatalf("expected B=n=%d blocks, got %d", n, stats.NumBlocks)
	}

	bucketSizes := ere.NonEmptyBlockSizes()
	nonEmpty := len(bucketSizes)
	emptyBlocks := stats.NumBlocks - nonEmpty

	emptyFrac := float64(emptyBlocks) / float64(stats.NumBlocks)

	var sumNonEmpty int
	for _, s := range bucketSizes {
		sumNonEmpty += s
	}
	meanNonEmpty := float64(sumNonEmpty) / float64(nonEmpty)

	var eq1, le2, le3 int
	for _, s := range bucketSizes {
		if s == 1 {
			eq1++
		}
		if s <= 2 {
			le2++
		}
		if s <= 3 {
			le3++
		}
	}
	pX1 := float64(eq1) / float64(nonEmpty)
	pXle2 := float64(le2) / float64(nonEmpty)
	pXle3 := float64(le3) / float64(nonEmpty)

	sortedSizes := make([]int, len(bucketSizes))
	copy(sortedSizes, bucketSizes)
	sort.Ints(sortedSizes)
	median := sortedSizes[len(sortedSizes)/2]
	p90Idx := int(math.Ceil(0.90*float64(len(sortedSizes)))) - 1
	if p90Idx < 0 {
		p90Idx = 0
	}
	p90 := sortedSizes[p90Idx]

	// Analytical predictions for Poisson(1):
	// P[X=0] = e^-1 ≈ 0.367879
	// E[X|X≥1] = 1/(1-e^-1) ≈ 1.581977
	// P[X=1|X≥1] = e^-1 / (1-e^-1) ≈ 0.581977
	// P[X≤2|X≥1] = (e^-1 + e^-1/2) / ... actually easier:
	//   P[1≤X≤k|X≥1] = (P[X≤k] - P[X=0]) / (1 - P[X=0])
	const (
		analyticalEmpty   = 0.36787944117144233
		analyticalMean    = 1.5819767068693264
		analyticalPX1     = 0.5819767068693264
		analyticalPXle2   = 0.8727150603102663
		analyticalPXle3   = 0.9696856459580773
		analyticalMedian  = 1.0
		analyticalP90     = 3.0
	)

	rows := []struct {
		name      string
		empirical float64
		analytic  float64
	}{
		{"P[X=0] (empty fraction)", emptyFrac, analyticalEmpty},
		{"E[X | X>=1] (mean non-empty)", meanNonEmpty, analyticalMean},
		{"P[X=1 | X>=1]", pX1, analyticalPX1},
		{"P[X<=2 | X>=1]", pXle2, analyticalPXle2},
		{"P[X<=3 | X>=1]", pXle3, analyticalPXle3},
		{"median(X | X>=1)", float64(median), analyticalMedian},
		{"P90(X | X>=1)", float64(p90), analyticalP90},
	}

	fmt.Println()
	fmt.Printf("=== ERE Bucket-Size Distribution (n=2^%d=%d, B=n, seed=%d) ===\n", logN, n, seed)
	fmt.Printf("NumBlocks=%d  NonEmpty=%d  Empty=%d\n", stats.NumBlocks, nonEmpty, emptyBlocks)
	fmt.Println()
	fmt.Printf("| %-32s | %12s | %12s | %12s |\n", "Quantity", "Empirical", "Analytical", "|Δ|")
	fmt.Printf("|%s|%s|%s|%s|\n",
		"----------------------------------",
		"--------------",
		"--------------",
		"--------------")
	for _, row := range rows {
		dev := math.Abs(row.empirical - row.analytic)
		fmt.Printf("| %-32s | %12.6f | %12.6f | %12.6f |\n",
			row.name, row.empirical, row.analytic, dev)
	}
	fmt.Println()
}
