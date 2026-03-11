package ere

import (
	"Thesis/bits"
	"sort"
	"testing"
)

func TestExactRangeEmptiness_Basic(t *testing.T) {
	// Let's create some simple bitstrings
	strKeys := []string{
		"0000",
		"0010",
		"0100",
		"1000",
		"1010",
		"1100",
		"1110",
		"1111",
	}

	keys := make([]bits.BitString, len(strKeys))
	for i, s := range strKeys {
		keys[i] = bits.NewFromBinary(s)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Compare(keys[j]) < 0
	})

	universe := bits.NewFromBinary("111111")
	ere, err := NewExactRangeEmptiness(keys, universe)
	if err != nil {
		t.Fatalf("Failed to create ExactRangeEmptiness: %v", err)
	}

	tests := []struct {
		a      string
		b      string
		expect bool // IsEmpty?
	}{
		{"0000", "0000", false}, // exact match
		{"0001", "0001", true},  // no match
		{"0001", "0010", false}, // matches "0010"
		{"0001", "0011", false}, // matches "0010"
		{"0011", "0011", true},  // no match
		{"0101", "0111", true},  // no match
		{"0111", "1001", false}, // matches "1000"
		{"1101", "1101", true},  // no match
		{"1101", "1110", false}, // matches "1110"
		{"1011", "1011", true},
		{"0000", "1111", false}, // matches all
	}

	for _, tt := range tests {
		a := bits.NewFromBinary(tt.a)
		b := bits.NewFromBinary(tt.b)
		empty := ere.IsEmpty(a, b)
		if empty != tt.expect {
			t.Errorf("IsEmpty(%s, %s) = %v; expected %v", tt.a, tt.b, empty, tt.expect)
		}
	}
}

func TestExactRangeEmptiness_Empty(t *testing.T) {
	keys := []bits.BitString{}
	universe := bits.NewFromBinary("1")
	ere, err := NewExactRangeEmptiness(keys, universe)
	if err != nil {
		t.Fatalf("Failed to create with empty keys: %v", err)
	}

	if !ere.IsEmpty(bits.NewFromBinary("0"), bits.NewFromBinary("1")) {
		t.Errorf("Expected IsEmpty to return true for empty keys")
	}
}

func TestExactRangeEmptiness_UnsortedKeys(t *testing.T) {
	keys := []bits.BitString{
		bits.NewFromBinary("10"),
		bits.NewFromBinary("01"),
	}
	universe := bits.NewFromBinary("11")
	_, err := NewExactRangeEmptiness(keys, universe)
	if err == nil {
		t.Fatalf("Expected error for unsorted keys")
	}
}

func TestExactRangeEmptiness_VariableLength(t *testing.T) {
	strKeys := []string{
		"0",
		"00",
		"01",
		"1",
		"100",
	}

	keys := make([]bits.BitString, len(strKeys))
	for i, s := range strKeys {
		keys[i] = bits.NewFromBinary(s)
	}

	// Make sure they are sorted lexicographically
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Compare(keys[j]) < 0
	})

	universe := bits.NewFromBinary("111")
	ere, err := NewExactRangeEmptiness(keys, universe)
	if err != nil {
		t.Fatalf("Failed to create: %v", err)
	}

	// Verify "001" to "011" - wait, let's manually check which ones are there.
	// keys order:
	// "0", "00", "01", "1", "100" (Wait, Compare order. "0" < "00"? Actually let's just use the sorted keys)

	empty := ere.IsEmpty(bits.NewFromBinary("001"), bits.NewFromBinary("001"))
	if !empty {
		t.Errorf("Expected 001 to be empty")
	}
}
