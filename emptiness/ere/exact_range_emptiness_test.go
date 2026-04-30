package ere

import (
	"sort"
	"testing"
)

func TestExactRangeEmptiness_Basic(t *testing.T) {
	// 4-bit keys: 0b0000=0, 0b0010=2, 0b0100=4, 0b1000=8, 0b1010=10, 0b1100=12, 0b1110=14, 0b1111=15
	keys := []uint64{0, 2, 4, 8, 10, 12, 14, 15}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	ere, err := NewExactRangeEmptiness(keys, 4)
	if err != nil {
		t.Fatalf("Failed to create ExactRangeEmptiness: %v", err)
	}

	tests := []struct {
		a, b   uint64
		expect bool // IsEmpty?
	}{
		{0, 0, false},   // exact match 0b0000
		{1, 1, true},    // no match
		{1, 2, false},   // matches 0b0010=2
		{1, 3, false},   // matches 0b0010=2
		{3, 3, true},    // no match
		{5, 7, true},    // no match
		{7, 9, false},   // matches 0b1000=8
		{13, 13, true},  // no match
		{13, 14, false}, // matches 0b1110=14
		{11, 11, true},
		{0, 15, false}, // matches all
	}

	for _, tt := range tests {
		empty := ere.IsEmpty(tt.a, tt.b)
		if empty != tt.expect {
			t.Errorf("IsEmpty(%d, %d) = %v; expected %v", tt.a, tt.b, empty, tt.expect)
		}
	}
}

func TestExactRangeEmptiness_Empty(t *testing.T) {
	keys := []uint64{}
	ere, err := NewExactRangeEmptiness(keys, 1)
	if err != nil {
		t.Fatalf("Failed to create with empty keys: %v", err)
	}

	if !ere.IsEmpty(0, 1) {
		t.Errorf("Expected IsEmpty to return true for empty keys")
	}
}

func TestExactRangeEmptiness_UnsortedKeys(t *testing.T) {
	keys := []uint64{2, 1}
	_, err := NewExactRangeEmptiness(keys, 2)
	if err == nil {
		t.Fatalf("Expected error for unsorted keys")
	}
}

func TestExactRangeEmptiness_FixedWidth(t *testing.T) {
	// Fixed 3-bit keys: 0, 1, 2, 4 (binary: 000, 001, 010, 100)
	keys := []uint64{0, 1, 2, 4}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	ere, err := NewExactRangeEmptiness(keys, 3)
	if err != nil {
		t.Fatalf("Failed to create: %v", err)
	}

	// 3 (binary 011) is not in the set
	if !ere.IsEmpty(3, 3) {
		t.Errorf("Expected 3 to be absent")
	}
	// 2 is in the set
	if ere.IsEmpty(2, 2) {
		t.Errorf("Expected 2 to be present")
	}
}
