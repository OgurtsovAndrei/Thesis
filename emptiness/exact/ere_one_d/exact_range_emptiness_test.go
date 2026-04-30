package ere_one_d

import (
	"sort"
	"testing"
)

func TestExactRangeEmptiness_Basic(t *testing.T) {
	// 4-bit keys (MSB-first trie order matches numeric order for fixed-width).
	// Binary strings from the original test, interpreted as 4-bit uint64 values.
	rawKeys := []uint64{
		0b0000, // 0
		0b0010, // 2
		0b0100, // 4
		0b1000, // 8
		0b1010, // 10
		0b1100, // 12
		0b1110, // 14
		0b1111, // 15
	}

	sort.Slice(rawKeys, func(i, j int) bool { return rawKeys[i] < rawKeys[j] })

	ere, err := NewExactRangeEmptiness(rawKeys, 4)
	if err != nil {
		t.Fatalf("Failed to create ExactRangeEmptiness: %v", err)
	}

	tests := []struct {
		a, b   uint64
		expect bool // IsEmpty?
	}{
		{0b0000, 0b0000, false}, // exact match
		{0b0001, 0b0001, true},  // no match
		{0b0001, 0b0010, false}, // matches 0010
		{0b0001, 0b0011, false}, // matches 0010
		{0b0011, 0b0011, true},  // no match
		{0b0101, 0b0111, true},  // no match
		{0b0111, 0b1001, false}, // matches 1000
		{0b1101, 0b1101, true},  // no match
		{0b1101, 0b1110, false}, // matches 1110
		{0b1011, 0b1011, true},
		{0b0000, 0b1111, false}, // matches all
	}

	for _, tt := range tests {
		empty := ere.IsEmpty(tt.a, tt.b)
		if empty != tt.expect {
			t.Errorf("IsEmpty(%04b, %04b) = %v; expected %v", tt.a, tt.b, empty, tt.expect)
		}
	}
}

func TestExactRangeEmptiness_Empty(t *testing.T) {
	ere, err := NewExactRangeEmptiness([]uint64{}, 1)
	if err != nil {
		t.Fatalf("Failed to create with empty keys: %v", err)
	}

	if !ere.IsEmpty(0, 1) {
		t.Errorf("Expected IsEmpty to return true for empty keys")
	}
}

func TestExactRangeEmptiness_UnsortedKeys(t *testing.T) {
	keys := []uint64{0b10, 0b01}
	_, err := NewExactRangeEmptiness(keys, 2)
	if err == nil {
		t.Fatalf("Expected error for unsorted keys")
	}
}

// TestExactRangeEmptiness_FixedWidth replaces the old VariableLength test.
// The original test used BitStrings of unequal widths (1, 2, 1, 1, 3 bits), which
// has no direct equivalent under a fixed-keyBits API. Instead we verify that keys
// laid out left-justified at keyBits=4 preserve trie ordering (MSB-first numeric
// order is identical for fixed-width keys).
//
// Keys chosen to mirror the spirit of the original: values spread across the
// low half of the 4-bit universe.
func TestExactRangeEmptiness_FixedWidth(t *testing.T) {
	// keyBits=4; keys in numeric (= trie) order:
	// 0b0000=0, 0b0100=4, 0b0101=5, 0b1000=8, 0b1001=9
	keys := []uint64{0b0000, 0b0100, 0b0101, 0b1000, 0b1001}

	ere, err := NewExactRangeEmptiness(keys, 4)
	if err != nil {
		t.Fatalf("Failed to create: %v", err)
	}

	// A point not in the set should be empty.
	if !ere.IsEmpty(0b0001, 0b0001) {
		t.Errorf("Expected 0001 to be empty")
	}
	// A range spanning two present keys should be non-empty.
	if ere.IsEmpty(0b0100, 0b1000) {
		t.Errorf("Expected range [0100, 1000] to be non-empty")
	}
}
