package hybridutil

import "testing"

// uniformKeys builds n keys with constant gap R, starting from 0.
func uniformKeys(n int, R uint64) []uint64 {
	out := make([]uint64, n)
	for i := range out {
		out[i] = uint64(i) * R
	}
	return out
}

func TestFallbackInGapFPR_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		keys []uint64
		K    uint32
		L    uint64
		eps  float64
		want bool
	}{
		{"empty", []uint64{}, 16, 8, 1e-3, true},
		{"one_key", []uint64{42}, 16, 8, 1e-3, true},
		{"zero_spread", []uint64{5, 5, 5}, 16, 8, 1e-3, true},
		// spread = 300, spreadBits = 9; K = 32 → spreadBits ≤ K → exact mode → true
		{"fits_K_bits", []uint64{0, 100, 200, 300}, 32, 8, 1e-3, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FallbackInGapFPR{Epsilon: tc.eps}.useTrunc(tc.keys, tc.K, tc.L)
			if got != tc.want {
				t.Errorf("useTrunc=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestFallbackInGapFPR_String(t *testing.T) {
	if got := (FallbackInGapFPR{Epsilon: 1e-3}).String(); got != "InGapFPR" {
		t.Errorf("String()=%q want %q", got, "InGapFPR")
	}
}

func TestFallbackInGapFPR_Uniform(t *testing.T) {
	// n = 1024, R = 1<<20 → spread = 1023<<20. With K = 12:
	//   P = spread >> 12  ≈ 1023<<8 ≈ 261888
	//   per-gap FPR ≈ P/(R-L) = 261888/(1048576-128) ≈ 0.25
	// → unsafe at ε=0.01
	t.Run("unsafe_K12_R1M_L128", func(t *testing.T) {
		keys := uniformKeys(1024, 1<<20)
		got := FallbackInGapFPR{Epsilon: 0.01}.useTrunc(keys, 12, 128)
		if got {
			t.Errorf("useTrunc=true want false (P/(R-L)≈0.25 ≫ 0.01)")
		}
	})

	// With K = 24:
	//   P = spread >> 24 ≈ 1023>>4 ≈ 63
	//   per-gap FPR ≈ 63/(1048576-128) ≈ 6.0e-5
	// → safe at ε=1e-3
	t.Run("safe_K24_R1M_L128", func(t *testing.T) {
		keys := uniformKeys(1024, 1<<20)
		got := FallbackInGapFPR{Epsilon: 1e-3}.useTrunc(keys, 24, 128)
		if !got {
			t.Errorf("useTrunc=false want true (P/(R-L)≈6e-5 ≪ 1e-3)")
		}
	})
}
