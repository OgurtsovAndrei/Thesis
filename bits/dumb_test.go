package bits

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBitStringHash(t *testing.T) {
	t.Parallel()
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

	t.Parallel()
	bs1 := NewFromText("test")
	bs2 := NewFromText("test")
	bs3 := NewFromText("diff")

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
	t.Parallel()
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

	// Traditional comparison: shorter < longer with same prefix (restored original logic)
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

func TestBitStringTrieCompare(t *testing.T) {
	t.Parallel()

	t.Run("trailing zeros vs trimmed", func(t *testing.T) {
		bs_short := NewFromBinary("11")
		bs_long := NewFromBinary("1100")

		if bs_long.TrieCompare(bs_short) >= 0 {
			t.Error("TrieCompare: 1100 should be < 11 (trailing zeros should come before trimmed)")
		}

		if bs_short.TrieCompare(bs_long) <= 0 {
			t.Error("TrieCompare: 11 should be > 1100 (trimmed should come after trailing zeros)")
		}
	})

	t.Run("compare vs trie compare behavior", func(t *testing.T) {
		bs1 := NewFromBinary("10")
		bs2 := NewFromBinary("100")

		if bs1.Compare(bs2) >= 0 {
			t.Error("Compare: 10 should be < 100 (standard lexicographic)")
		}

		if bs2.TrieCompare(bs1) >= 0 {
			t.Error("TrieCompare: 100 should be < 10 (trailing zeros before trimmed)")
		}
	})

	t.Run("equal strings", func(t *testing.T) {
		bs3 := NewFromBinary("101")
		bs4 := NewFromBinary("101")

		if bs3.TrieCompare(bs4) != 0 {
			t.Error("TrieCompare: equal strings should compare to 0")
		}
	})

	t.Run("different values same length", func(t *testing.T) {
		bs5 := NewFromBinary("101")
		bs6 := NewFromBinary("110")

		cmp := bs5.Compare(bs6)
		trieCmp := bs5.TrieCompare(bs6)
		if cmp != trieCmp {
			t.Error("Compare and TrieCompare should give same result for different values")
		}
	})

	t.Run("from mmph", func(t *testing.T) {
		bs1 := NewFromBinary("1111111100101")
		bs2 := NewFromBinary("11111111")

		trieCmp := bs1.TrieCompare(bs2)
		require.Less(t, trieCmp, 0)
	})

	t.Run("ones", func(t *testing.T) {
		bs1 := NewFromBinary("11")
		bs2 := NewFromBinary("1111")

		trieCmp := bs1.TrieCompare(bs2)
		require.Less(t, trieCmp, 0)
	})

	t.Run("ones after zero", func(t *testing.T) {
		bs1 := NewFromBinary("00")
		bs2 := NewFromBinary("0011")

		trieCmp := bs1.TrieCompare(bs2)
		require.Less(t, trieCmp, 0)
	})

	t.Run("zeros after ones", func(t *testing.T) {
		bs1 := NewFromBinary("1100")
		bs2 := NewFromBinary("11")

		trieCmp := bs1.TrieCompare(bs2)
		require.Less(t, trieCmp, 0)
	})

	t.Run("ones after zeros after ones", func(t *testing.T) {
		bs1 := NewFromBinary("11001")
		bs2 := NewFromBinary("110")

		trieCmp := bs1.TrieCompare(bs2)
		require.Less(t, trieCmp, 0)
	})
}

func TestBitStringLongCompare(t *testing.T) {
	if SelectedImpl == Uint64String {
		t.SkipNow()
	}
	t.Parallel()
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

func TestTrailingZerosVsTrimmed(t *testing.T) {
	t.Parallel()

	// Test specific trailing zero scenarios
	testCases := []struct {
		longer  string // String with trailing zeros
		shorter string // Trimmed version
		desc    string
	}{
		{"10", "1", "10 vs 1"},
		{"100", "10", "100 vs 10"},
		{"1000", "100", "1000 vs 100"},
		{"1100", "11", "1100 vs 11"},
		{"10100", "1010", "10100 vs 1010"},
		{"11000", "110", "11000 vs 110"},
	}

	for _, tc := range testCases {
		longer := NewFromBinary(tc.longer)
		shorter := NewFromBinary(tc.shorter)

		// With TrieCompare: longer (with trailing zeros) should be < shorter (trimmed)
		require.True(t, longer.TrieCompare(shorter) < 0,
			"TrieCompare: %s should be < %s (trailing zeros before trimmed)", tc.longer, tc.shorter)
		require.True(t, shorter.TrieCompare(longer) > 0,
			"TrieCompare: %s should be > %s (trimmed after trailing zeros)", tc.shorter, tc.longer)

		// With standard Compare: shorter should be < longer
		require.True(t, shorter.Compare(longer) < 0,
			"Compare: %s should be < %s (standard lexicographic)", tc.shorter, tc.longer)
		require.True(t, longer.Compare(shorter) > 0,
			"Compare: %s should be > %s (standard lexicographic)", tc.longer, tc.shorter)
	}
}
