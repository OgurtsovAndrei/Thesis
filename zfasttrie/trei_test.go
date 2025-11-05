package zfasttrie

import (
	"fmt"
	"testing"
)

func TestFast(t *testing.T) {
	if MostSignificantBit(0) != -1 {
		t.Fatal("MostSignificantBit(0) failed")
	}

	if TwoFattest(0, 0) != 0 {
		t.Fatal("TwoFattest(0, 0] failed")
	}
	if TwoFattest(0, 6) != 4 {
		t.Fatal("TwoFattest(0, 6] failed")
	}
	if TwoFattest(0, 8) != 8 {
		t.Fatal("TwoFattest(0, 8] failed")
	}
	if TwoFattest(1, 8) != 8 {
		t.Fatal("TwoFattest(1, 8] failed")
	}
	if TwoFattest(0, 9) != 8 {
		t.Fatal("TwoFattest(0, 9] failed")
	}
	if TwoFattest(0, 4) != 4 {
		t.Fatal("TwoFattest(0, 4] failed")
	}
	if TwoFattest(0, 7) != 4 {
		t.Fatal("TwoFattest(0, 7] failed")
	}
	if TwoFattest(5, 7) != 6 {
		t.Fatal("TwoFattest(5, 7] failed")
	}
	if TwoFattest(4, 7) != 6 {
		t.Fatal("TwoFattest(4, 7] failed")
	}
	if TwoFattest(3, 7) != 4 {
		t.Fatal("TwoFattest(3, 7] failed")
	}
	if TwoFattest(10, 11) != 11 {
		t.Fatal("TwoFattest(10, 11] failed")
	}
	if TwoFattest(9, 11) != 10 {
		t.Fatal("TwoFattest(9, 11] failed")
	}
	//if TwoFattest(^uint64(0), 8) != 0 {
	//	t.Fatal("TwoFattest(-1, 8) failed")
	//}
	if TwoFattest(7, 8) != 8 {
		t.Fatal("TwoFattest(7, 8] failed")
	}
	if TwoFattest(8, 8) != 0 {
		t.Fatal("TwoFattest(8, 8] failed")
	}
}

func TestBitStringLCP(t *testing.T) {
	// C++: assert(BitString::getLCPLength(BitString("1"), BitString("9")) == 3);
	bs1 := NewBitString("1") // "1" = 00110001
	bs9 := NewBitString("9") // "9" = 00111001

	// LCP = 3 (001)
	lcp := GetLCPLength(bs1, bs9)
	if lcp != 3 {
		t.Fatalf("GetLCPLength '1' vs '9' failed: expected 3, got %d", lcp)
	}
}

func testTrie(t *testing.T, texts []string, isInverse bool) {
	tree := NewZFastTrie[bool](false)

	for i := 0; i < len(texts); i++ {
		tree.Insert(texts[i], true)
	}

	for i := 0; i < len(texts); i++ {
		if !tree.Contains(texts[i]) {
			t.Fatalf("Failed to find inserted text: %s\n\n%s\n\n%q", texts[i], tree.String(), texts)
		}
	}

	//fmt.Println(strings.Join(texts, ", "))
	//fmt.Println(tree.String())

	for i := 0; i < len(texts); i++ {
		for j := 0; j <= len(texts[i]); j++ { // Include empty prefix (j=0) and full string (j=len)
			prefix := texts[i][:j]
			if !tree.ContainsPrefix(prefix) {
				t.Fatalf("Failed to find prefix: %q (from %s)", prefix, texts[i])
			}
		}
	}

	for i := 0; i < len(texts); i++ {
		for j := 0; j < len(texts[i]); j++ {
			prefix := texts[i][:j] + "X"
			if tree.Contains(prefix) {
				t.Fatalf("Found non-existent text: %s", prefix)
			}
			if tree.ContainsPrefix(prefix) {
				t.Fatalf("Found non-existent prefix: %s", prefix)
			}
		}
	}

	fmt.Println(tree.String())

	for i := 0; i < len(texts); i++ {
		index := i
		if isInverse {
			index = len(texts) - 1 - i
		}
		tree.Erase(texts[index])
		if tree.Contains(texts[index]) {
			fmt.Println(tree.String())
			fmt.Println(tree.Contains(texts[index]))
			t.Fatalf("Found erased text: %s", texts[index])
		}
	}
}

func TestTrie(t *testing.T) {
	texts1 := []string{"A", "AA", "AAA", "AAAA", "AAAAA", "AAAAAA", "AAAAAAA", "AAAAAAAA", "B"}
	t.Run("A_B", func(t *testing.T) {
		testTrie(t, texts1, false)
		testTrie(t, texts1, true)
	})

	texts2 := []string{"0000", "0000000100000011", "00000002", "0000000100000012", "000000010000001100000021"}
	t.Run("Complex_0", func(t *testing.T) {
		testTrie(t, texts2, false)
		testTrie(t, texts2, true)
	})

	texts3 := []string{"0000000100000011", "00000001", "000000010000", "00000"}
	t.Run("Prefixes", func(t *testing.T) {
		testTrie(t, texts3, false)
		testTrie(t, texts3, true)
	})

	texts4 := []string{"0a", "0b", "0c", "0d", "0e", "0f", "0"}
	t.Run("Siblings", func(t *testing.T) {
		testTrie(t, texts4, false)
		testTrie(t, texts4, true)
	})

	texts5 := []string{"aa", "a", "b", "ba"}
	t.Run("Simple", func(t *testing.T) {
		testTrie(t, texts5, false)
		testTrie(t, texts5, true)
	})
}
