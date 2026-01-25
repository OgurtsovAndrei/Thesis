package bits

import "testing"

func TestBitStringHash(t *testing.T) {
	bs1 := NewFromText("A")
	bs2 := NewFromText("A")
	bs3 := NewFromText("B")

	hash1 := bs1.Hash()
	hash2 := bs2.Hash()
	hash3 := bs3.Hash()

	if hash1 != hash2 {
		t.Errorf("Equal bitstrings should have same hash: %d != %d", hash1, hash2)
	}

	if hash1 == hash3 {
		t.Errorf("Different bitstrings should likely have different hashes: %d == %d", hash1, hash3)
	}

	// Test with binary strings
	bsBin1 := NewFromBinary("1010101010101010")
	bsBin2 := NewFromBinary("1010101010101010")
	bsBin3 := NewFromBinary("0101010101010101")

	hashBin1 := bsBin1.Hash()
	hashBin2 := bsBin2.Hash()
	hashBin3 := bsBin3.Hash()

	if hashBin1 != hashBin2 {
		t.Errorf("Equal binary bitstrings should have same hash: %d != %d", hashBin1, hashBin2)
	}

	if hashBin1 == hashBin3 {
		t.Errorf("Different binary bitstrings should likely have different hashes: %d == %d", hashBin1, hashBin3)
	}
}

func TestBitStringEq(t *testing.T) {
	bs1 := NewFromText("test")
	bs2 := NewFromText("test")
	bs3 := NewFromText("different")

	if !bs1.Eq(bs2) {
		t.Error("Equal bitstrings should return true for Eq")
	}

	if bs1.Eq(bs3) {
		t.Error("Different bitstrings should return false for Eq")
	}

	// Test with binary strings
	bsBin1 := NewFromBinary("11001100")
	bsBin2 := NewFromBinary("11001100")
	bsBin3 := NewFromBinary("11001101")

	if !bsBin1.Eq(bsBin2) {
		t.Error("Equal binary bitstrings should return true for Eq")
	}

	if bsBin1.Eq(bsBin3) {
		t.Error("Different binary bitstrings should return false for Eq")
	}

	// Eq should be same as Equal
	if bs1.Eq(bs2) != bs1.Equal(bs2) {
		t.Error("Eq should return same result as Equal")
	}
}

func TestBitStringCompare(t *testing.T) {
	// Test with binary strings for precise control
	bs1 := NewFromBinary("1010")
	bs2 := NewFromBinary("1010")
	bs3 := NewFromBinary("1011")
	bs4 := NewFromBinary("1001")
	bs5 := NewFromBinary("101")

	// Equal comparison
	if bs1.Compare(bs2) != 0 {
		t.Error("Equal bitstrings should compare to 0")
	}

	// First < Second
	if bs4.Compare(bs1) >= 0 {
		t.Error("1001 should be < 1010")
	}

	// First > Second
	if bs3.Compare(bs1) <= 0 {
		t.Error("1011 should be > 1010")
	}

	// Shorter < Longer with same prefix
	if bs5.Compare(bs1) >= 0 {
		t.Error("101 should be < 1010")
	}

	// Longer > Shorter with same prefix
	if bs1.Compare(bs5) <= 0 {
		t.Error("1010 should be > 101")
	}

	// Test with text strings
	// Note: ASCII 'A'=65=01000001 becomes 10000010 in bit order
	//       ASCII 'B'=66=01000010 becomes 01000010 in bit order
	//       So in bit-wise lexicographic order: 'A' > 'B' (first bit: 1 > 0)
	bsA := NewFromText("A")
	bsB := NewFromText("B")
	bsAA := NewFromText("AA")

	if bsA.Compare(bsB) <= 0 {
		t.Error("'A' should be > 'B' in bit-wise lexicographic order")
	}

	if bsB.Compare(bsA) >= 0 {
		t.Error("'B' should be < 'A' in bit-wise lexicographic order")
	}

	if bsA.Compare(bsAA) >= 0 {
		t.Error("'A' should be < 'AA' (shorter < longer with same prefix)")
	}

	// Test empty strings
	bsEmpty1 := NewFromBinary("")
	bsEmpty2 := NewFromBinary("")
	bsNonEmpty := NewFromBinary("1")

	if bsEmpty1.Compare(bsEmpty2) != 0 {
		t.Error("Empty bitstrings should compare equal")
	}

	if bsEmpty1.Compare(bsNonEmpty) >= 0 {
		t.Error("Empty bitstring should be < non-empty")
	}

	if bsNonEmpty.Compare(bsEmpty1) <= 0 {
		t.Error("Non-empty bitstring should be > empty")
	}
}

func TestBitStringLongCompare(t *testing.T) {
	// Test with longer bitstrings that span multiple uint64 words
	long1 := NewFromBinary("1010101010101010101010101010101010101010101010101010101010101010" +
		"1100110011001100110011001100110011001100110011001100110011001100")
	long2 := NewFromBinary("1010101010101010101010101010101010101010101010101010101010101010" +
		"1100110011001100110011001100110011001100110011001100110011001100")
	long3 := NewFromBinary("1010101010101010101010101010101010101010101010101010101010101010" +
		"1100110011001100110011001100110011001100110011001100110011001101")

	// Test Hash
	hash1 := long1.Hash()
	hash2 := long2.Hash()
	hash3 := long3.Hash()

	if hash1 != hash2 {
		t.Error("Equal long bitstrings should have same hash")
	}

	if hash1 == hash3 {
		t.Error("Different long bitstrings should likely have different hashes")
	}

	// Test Eq
	if !long1.Eq(long2) {
		t.Error("Equal long bitstrings should return true for Eq")
	}

	if long1.Eq(long3) {
		t.Error("Different long bitstrings should return false for Eq")
	}

	// Test Compare
	if long1.Compare(long2) != 0 {
		t.Error("Equal long bitstrings should compare to 0")
	}

	if long1.Compare(long3) >= 0 {
		t.Error("First long bitstring should be < second")
	}

	if long3.Compare(long1) <= 0 {
		t.Error("Second long bitstring should be > first")
	}
}
