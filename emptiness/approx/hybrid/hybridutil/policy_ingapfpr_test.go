package hybridutil

import "testing"

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
