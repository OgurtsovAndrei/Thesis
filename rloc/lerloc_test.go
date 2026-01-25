package rloc

import (
	"Thesis/bits"
	"fmt"
	"sort"
	"testing"
	"time"
)

func TestLocalExactRangeLocator_EmptyKeys(t *testing.T) {
	lerl := NewLocalExactRangeLocator([]bits.BitString{})
	start, end, err := lerl.WeakPrefixSearch(bits.NewBitString("test"))
	if err != nil {
		t.Errorf("Expected no error for empty keys, got: %v", err)
	}
	if start != 0 || end != 0 {
		t.Errorf("Expected [0, 0) for empty keys, got: [%d, %d)", start, end)
	}
}

func TestLocalExactRangeLocator_EmptyPrefix(t *testing.T) {
	keys := []bits.BitString{
		bits.NewBitString("abc"),
		bits.NewBitString("def"),
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Compare(keys[j]) < 0
	})

	lerl := NewLocalExactRangeLocator(keys)

	start, end, err := lerl.WeakPrefixSearch(bits.NewBitString(""))
	if err != nil {
		t.Errorf("Expected no error for empty prefix, got: %v", err)
	}
	if start != 0 || end != 2 {
		t.Errorf("Expected [0, 2) for empty prefix, got: [%d, %d)", start, end)
	}
}

func TestLocalExactRangeLocator_AllPrefixes(t *testing.T) {
	for run := 0; run < testRuns; run++ {
		t.Run(fmt.Sprintf("run=%d", run), func(t *testing.T) {
			t.Parallel()
			seed := time.Now().UnixNano()
			keys := genUniqueBitStrings(seed)

			lerl := NewLocalExactRangeLocator(keys)

			for _, key := range keys {
				for prefixLen := uint32(0); prefixLen <= key.Size(); prefixLen++ {
					var prefix bits.BitString
					if prefixLen == 0 {
						prefix = bits.NewBitString("")
					} else {
						prefix = bits.NewBitStringPrefix(key, prefixLen)
					}

					start, end, err := lerl.WeakPrefixSearch(prefix)
					if err != nil {
						t.Fatalf("WeakPrefixSearch failed for prefix %s of key %s (seed: %d): %v",
							toBinary(prefix), toBinary(key), seed, err)
					}

					expectedStart, expectedEnd := findRange(keys, prefix)

					if start != expectedStart || end != expectedEnd {
						t.Errorf("Mismatch for prefix %s (seed: %d). Got: [%d, %d), Exp: [%d, %d)",
							toBinary(prefix), seed, start, end, expectedStart, expectedEnd)
						t.FailNow()
					}
				}
			}
		})
	}
}
