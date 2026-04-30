package ere

import (
	"testing"
)

// TestZeroWidthSuffix exercises the w == 0 edge case: when the universe size
// equals k = floor(log2(n)), every block holds at most one key and the suffix
// width is zero. The linear-scan path used to index packedData unconditionally
// and panic with "index out of range [0] with length 0" because packUint64Local
// returns nil for bitWidth == 0. This test panicked before the fix and must
// pass after.
func TestZeroWidthSuffix(t *testing.T) {
	// n = 4, k = floor(log2(4)) = 2, keyBits = 2 → w = 0
	// 2-bit keys: 0, 1, 2, 3
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

// TestZeroWidthSuffix_Sparse covers the case where some blocks are empty so
// that boundary queries hit blockA != blockB code paths and a wrapped suffix
// search of [suffA, max] over the linear-scan fast path.
func TestZeroWidthSuffix_Sparse(t *testing.T) {
	// n = 2 → k = 1; keyBits = 1 → w = 0; numBlocks = 2.
	// 1-bit keys: 0, 1
	keys := []uint64{0, 1}
	e, err := NewExactRangeEmptiness(keys, 1)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if e.w != 0 {
		t.Fatalf("expected w=0, got w=%d", e.w)
	}
	if e.IsEmpty(0, 1) {
		t.Errorf("expected non-empty range over both keys")
	}
}
