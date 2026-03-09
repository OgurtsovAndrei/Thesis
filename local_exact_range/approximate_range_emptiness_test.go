package local_exact_range

import (
	"Thesis/bits"
	"Thesis/testutils"
	"fmt"
	"sort"
	"testing"
)

func TestApproximateRangeEmptiness_Basic(t *testing.T) {
	strKeys := []string{
		"00000000",
		"00100000",
		"01000000",
		"10000000",
		"11111111",
	}

	keys := make([]bits.BitString, len(strKeys))
	for i, s := range strKeys {
		keys[i] = bits.NewFromBinary(s)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Compare(keys[j]) < 0
	})

	// Use a high epsilon to test truncation logic easily
	epsilon := 0.5 
	are, err := NewApproximateRangeEmptiness(keys, epsilon)
	if err != nil {
		t.Fatalf("Failed to create ApproximateRangeEmptiness: %v", err)
	}

	tests := []struct {
		a      string
		b      string
		expect bool // IsEmpty?
	}{
		{"00000000", "00000000", false}, // Exact match
		{"11111111", "11111111", false}, // Exact match
		{"01010000", "01110000", true},  // Empty interval
	}

	for _, tt := range tests {
		a := bits.NewFromBinary(tt.a)
		b := bits.NewFromBinary(tt.b)
		empty := are.IsEmpty(a, b)
		
		// If expect=true (should be empty), and are returns false (not empty),
		// it could be a False Positive!
		// However, for testing correctness of the wrapper, we check basic cases where FP shouldn't happen.
		if empty != tt.expect && tt.expect == false {
			t.Errorf("IsEmpty(%s, %s) = %v; expected %v (False Negative!)", tt.a, tt.b, empty, tt.expect)
		}
	}
}

func BenchmarkApproximateRangeEmptiness_Grid(b *testing.B) {
	counts := []int{1_000_000}
	bitLens := []int{64, 128, 256, 512}
	epsilons := []float64{0.01, 0.001} // 1% and 0.1% FP rate

	for _, epsilon := range epsilons {
		for _, bitLen := range bitLens {
			for _, count := range counts {
				name := fmt.Sprintf("Eps=%v/KeySize=%d/Keys=%d", epsilon, bitLen, count)
				keys := testutils.GetBenchKeys(bitLen, count)

				b.Run(name+"/Build", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						are, _ := NewApproximateRangeEmptiness(keys, epsilon)
						if i == 0 {
							size := are.ByteSize()
							b.ReportMetric(float64(size)*8/float64(count), "bits_per_key")
						}
					}
				})

				are, _ := NewApproximateRangeEmptiness(keys, epsilon)
				
				queryA := make([]bits.BitString, 100)
				queryB := make([]bits.BitString, 100)
				for i := 0; i < 100; i++ {
					idx1 := i * len(keys) / 100
					idx2 := (idx1 + 10) % len(keys)
					if idx1 > idx2 { idx1, idx2 = idx2, idx1 }
					queryA[i] = keys[idx1]
					queryB[i] = keys[idx2]
				}

				b.Run(name+"/Query", func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						idx := i % 100
						are.IsEmpty(queryA[idx], queryB[idx])
					}
				})
			}
		}
	}
}
