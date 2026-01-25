package bits

import "testing"

func TestBitStringLCP(t *testing.T) {
	// C++: assert(BitString::getLCPLength(BitString("1"), BitString("9")) == 3);
	bs1 := NewFromText("1") // "1" = 00110001
	bs9 := NewFromText("9") // "9" = 00111001

	// LCP = 3 (001)
	lcp := bs1.GetLCPLength(bs9)
	if lcp != 3 {
		t.Fatalf("GetLCPLength '1' vs '9' failed: expected 3, got %d", lcp)
	}
}
