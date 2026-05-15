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

// mixedKeys builds nLarge keys with gap = largeGap, then appends nSmall keys
// packed tightly at the end (gap = 1). Useful for "X% saturated gaps" cases.
func mixedKeys(nLarge int, largeGap uint64, nSmall int) []uint64 {
	out := make([]uint64, 0, nLarge+nSmall)
	for i := 0; i < nLarge; i++ {
		out = append(out, uint64(i)*largeGap)
	}
	base := uint64(nLarge) * largeGap
	for i := 0; i < nSmall; i++ {
		out = append(out, base+uint64(i+1))
	}
	return out
}

func TestFallbackInGapFPR_Saturated(t *testing.T) {
	// 1000 large gaps of 2^30, then 50 saturated gaps of 1. L=0, K=20.
	//   P = spread >> K ≈ 2^20.
	//   Large gap R = 2^30 ⇒ P/R ≈ 10^-3 (negligible).
	//   Small gap R = 1 ≤ P ⇒ saturated (sum += 1).
	// 49 saturated out of 1049 gaps; mean ≈ 49/1049 ≈ 0.0467.
	// (The 1000th gap is 2^30 + 1, still well above P, also negligible.)
	keys := mixedKeys(1000, 1<<30, 50)
	t.Run("eps_001_reject", func(t *testing.T) {
		got := FallbackInGapFPR{Epsilon: 0.01}.useTrunc(keys, 20, 0)
		if got {
			t.Errorf("useTrunc=true want false (mean FPR ≈ 0.047 > 0.01)")
		}
	})
	t.Run("eps_010_accept", func(t *testing.T) {
		got := FallbackInGapFPR{Epsilon: 0.10}.useTrunc(keys, 20, 0)
		if !got {
			t.Errorf("useTrunc=false want true (mean FPR ≈ 0.047 < 0.10)")
		}
	})
}

func TestFallbackInGapFPR_DenseGapsLE_L(t *testing.T) {
	// All gaps = 4 ≤ L = 8 → every gap contributes FPR_i = 0 → safe.
	keys := uniformKeys(1024, 4)
	got := FallbackInGapFPR{Epsilon: 1e-9}.useTrunc(keys, 12, 8)
	if !got {
		t.Errorf("useTrunc=false want true (all gaps ≤ L → sum=0)")
	}
}

func TestFallbackInGapFPR_HugeSparseSpread(t *testing.T) {
	// OSM-like: spread = 2^59, n = 2^16 → typical gap ≈ 2^43.
	// K = 37 → P ≈ 2^22. Per-gap FPR ≈ 2^22 / (2^43 - L) ≈ 2^-21 ≈ 5e-7.
	keys := uniformKeys(1<<16, 1<<43)
	got := FallbackInGapFPR{Epsilon: 1e-3}.useTrunc(keys, 37, 128)
	if !got {
		t.Errorf("useTrunc=false want true (huge sparse gaps → FPR ≪ ε)")
	}
}
