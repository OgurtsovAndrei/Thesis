package ere_one_d

import (
	"testing"
)

// TestZeroWidthSuffix exercises the w == 0 edge case where the universe size
// equals k = floor(log2(n)). The linear-scan path used to panic with
// "index out of range [0] with length 0" because packUint64Local returns nil
// for bitWidth == 0. This test panicked before the fix and must pass after.
//
// With n=4 keys and keyBits=2: k=floor(log2(4))=2, w=keyBits-k=0.
func TestZeroWidthSuffix(t *testing.T) {
	// 4 keys, keyBits=2 → k=2, w=0
	keys := []uint64{0, 1, 2, 3}

	e, err := NewExactRangeEmptiness(keys, 2)
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
		got := e.IsEmpty(tt.a, tt.b)
		if got != tt.expect {
			t.Errorf("IsEmpty(%d,%d) = %v; want %v", tt.a, tt.b, got, tt.expect)
		}
	}
}
