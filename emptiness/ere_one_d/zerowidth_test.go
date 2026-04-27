package ere_one_d

import (
	"Thesis/bits"
	"testing"
)

// TestZeroWidthSuffix exercises the w == 0 edge case where the universe size
// equals k = floor(log2(n)). The linear-scan path used to panic with
// "index out of range [0] with length 0" because packUint64Local returns nil
// for bitWidth == 0. This test panicked before the fix and must pass after.
func TestZeroWidthSuffix(t *testing.T) {
	universe := bits.NewBitString(2)
	keys := []bits.BitString{
		bits.NewFromTrieUint64(0, 2),
		bits.NewFromTrieUint64(1, 2),
		bits.NewFromTrieUint64(2, 2),
		bits.NewFromTrieUint64(3, 2),
	}
	e, err := NewExactRangeEmptiness(keys, universe)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if e.w != 0 {
		t.Fatalf("expected w=0, got w=%d", e.w)
	}

	tests := []struct {
		a, b   uint64
		expect bool // IsEmpty
	}{
		{0, 0, false},
		{1, 1, false},
		{2, 2, false},
		{3, 3, false},
		{0, 3, false},
	}
	for _, tt := range tests {
		a := bits.NewFromTrieUint64(tt.a, 2)
		b := bits.NewFromTrieUint64(tt.b, 2)
		got := e.IsEmpty(a, b)
		if got != tt.expect {
			t.Errorf("IsEmpty(%d,%d) = %v; want %v", tt.a, tt.b, got, tt.expect)
		}
	}
}
