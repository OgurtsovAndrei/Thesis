package ere_pef

import "testing"

// All cost-function tests use lastRel = universe - 1 (inclusive upper
// bound), matching the Strategy-B refactor that avoids +1 overflow at
// the 2^64 boundary.

func TestEFBitsizePaper(t *testing.T) {
	cases := []struct {
		name          string
		lastRel, n    uint64
		want          uint64
	}{
		// lastRel = old_universe - 1; want recomputed with new formula:
		// numBuckets = (lastRel >> lo) + 1,  higher = n + numBuckets + 2.
		{"lastRel99_n10", 99, 10, 55},  // lo=3, numBuckets=13, higher=25, +30 lows
		{"lastRel10_n10", 10, 10, 23},  // lo=0 (10>=10, Len64(1)-1=0), higher=23, +0 lows
		{"lastRel7_n8", 7, 8, 18},      // 7<8 so lo=0, higher=18, +0 lows
		{"lastRel15_n4", 15, 4, 18},    // lo=1, numBuckets=8, higher=14, +4 lows
		{"lastRel0_n1", 0, 1, 4},       // 0<1 so lo=0, higher=4, +0 lows
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := efBitsizePaper(c.lastRel, c.n)
			if got != c.want {
				t.Fatalf("efBitsizePaper(%d, %d) = %d, want %d", c.lastRel, c.n, got, c.want)
			}
		})
	}
}

func TestBitmapBitsizePaper(t *testing.T) {
	// lastRel=99 → universe=100 bits
	if got := bitmapBitsizePaper(99, 10); got != 100 {
		t.Fatalf("bitmapBitsizePaper(99, 10) = %d, want 100", got)
	}
}

func TestAllOnesBitsize(t *testing.T) {
	// lastRel=7, n=8: 7 == n-1=7 → all-ones
	if got := allOnesBitsize(7, 8); got != 0 {
		t.Fatalf("allOnesBitsize(7, 8) = %d, want 0", got)
	}
	// lastRel=7, n=7: 7 != n-1=6 → not all-ones
	if got := allOnesBitsize(7, 7); got != ^uint64(0) {
		t.Fatalf("allOnesBitsize(7, 7) = %d, want max uint64", got)
	}
	// boundary: lastRel=MaxUint64, n=1: MaxUint64 == 0? No → not all-ones
	if got := allOnesBitsize(^uint64(0), 1); got != ^uint64(0) {
		t.Fatalf("allOnesBitsize(MaxUint64, 1) should be infinity")
	}
}

func TestMinCodecBitsize(t *testing.T) {
	// lastRel=7, n=8: contiguous run → all-ones
	if got := minCodecBitsize(7, 8); got != 0 {
		t.Fatalf("contiguous run picks all-ones: got %d, want 0", got)
	}
	// lastRel=10, n=10: dense → bitmap cheapest: 11+1=12
	if got := minCodecBitsize(10, 10); got != 12 {
		t.Fatalf("dense small chunk picks bitmap: got %d, want 11+1=12", got)
	}
	// lastRel=99, n=10: sparse → EF cheapest: 55+1=56
	if got := minCodecBitsize(99, 10); got != 56 {
		t.Fatalf("sparse chunk picks EF: got %d, want 55+1=56", got)
	}
}
