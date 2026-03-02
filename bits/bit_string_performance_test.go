package bits

import (
	"testing"
)

// Test TrimTrailingZeros
func TestTrimTrailingZeros(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"all zeros", "00000", ""},
		{"no trailing zeros", "10101", "10101"},
		{"some trailing zeros", "10100", "101"},
		{"single one", "10000", "1"},
		{"single zero", "0", ""},
		{"alternating pattern", "1010100", "10101"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bs := NewFromBinary(tc.input)
			result := bs.TrimTrailingZeros()
			expected := NewFromBinary(tc.expected)

			if !result.Equal(expected) {
				t.Errorf("TrimTrailingZeros(%s) = %v, want %v", tc.input, result, expected)
			}
		})
	}
}

// Test AppendBit
func TestAppendBit(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    string
		bit      bool
		expected string
	}{
		{"append 0 to empty", "", false, "0"},
		{"append 1 to empty", "", true, "1"},
		{"append 0", "101", false, "1010"},
		{"append 1", "101", true, "1011"},
		{"append to single bit", "1", false, "10"},
		{"append to single bit", "0", true, "01"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bs := NewFromBinary(tc.input)
			result := bs.AppendBit(tc.bit)
			expected := NewFromBinary(tc.expected)

			if !result.Equal(expected) {
				t.Errorf("AppendBit(%s, %t) = %v, want %v", tc.input, tc.bit, result, expected)
			}
		})
	}
}

// Test IsAllOnes
func TestIsAllOnes(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"single zero", "0", false},
		{"single one", "1", true},
		{"all ones", "1111", true},
		{"mixed bits", "1101", false},
		{"all zeros", "0000", false},
		{"long all ones", "11111111111111111", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bs := NewFromBinary(tc.input)
			result := bs.IsAllOnes()

			if result != tc.expected {
				t.Errorf("IsAllOnes(%s) = %t, want %t", tc.input, result, tc.expected)
			}
		})
	}
}

// Test Successor
func TestSuccessor(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", "1"},
		{"zero", "0", "1"},
		{"one", "1", "10"},
		{"simple increment", "10", "11"},
		{"carry propagation", "11", "100"},
		{"longer carry", "111", "1000"},
		{"mixed bits", "1010", "1011"},
		{"increment with carry", "1001", "1010"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bs := NewFromBinary(tc.input)
			result := bs.Successor()
			expected := NewFromBinary(tc.expected)

			if !result.Equal(expected) {
				t.Errorf("Successor(%s) = %v, want %v", tc.input, result, expected)
			}
		})
	}
}

func BenchmarkTrimTrailingZeros(b *testing.B) {
	bs := NewFromBinary("1010000000")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.TrimTrailingZeros()
	}
}

func BenchmarkAppendBit(b *testing.B) {
	bs := NewFromBinary("101010")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.AppendBit(true)
	}
}

func BenchmarkIsAllOnes(b *testing.B) {
	bs := NewFromBinary("111111111111111111111111111111")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.IsAllOnes()
	}
}

func BenchmarkSuccessor(b *testing.B) {
	bs := NewFromBinary("1010101010")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bs.Successor()
	}
}
