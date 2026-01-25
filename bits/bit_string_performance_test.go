package bits

import (
	"testing"
)

// Test TrimTrailingZeros for all implementations
func TestTrimTrailingZeros(t *testing.T) {
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

	for _, impl := range []BitStringImpl{CharString, Uint64String, Uint64ArrayString} {
		oldImpl := SelectedImpl
		// We can't directly modify SelectedImpl, so we'll test each implementation directly

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var bs BitString

				switch impl {
				case CharString:
					bs = NewCharFromBinary(tc.input)
				case Uint64String:
					if len(tc.input) <= 64 {
						bs = NewUint64FromBinaryText(tc.input)
					} else {
						t.Skip("Uint64BitString limited to 64 bits")
					}
				case Uint64ArrayString:
					bs = NewUint64ArrayFromBinaryText(tc.input)
				}

				result := bs.TrimTrailingZeros()

				var expected BitString
				switch impl {
				case CharString:
					expected = NewCharFromBinary(tc.expected)
				case Uint64String:
					if len(tc.expected) <= 64 {
						expected = NewUint64FromBinaryText(tc.expected)
					} else {
						t.Skip("Uint64BitString limited to 64 bits")
					}
				case Uint64ArrayString:
					expected = NewUint64ArrayFromBinaryText(tc.expected)
				}

				if !result.Equal(expected) {
					t.Errorf("TrimTrailingZeros(%s) = %v, want %v", tc.input, result, expected)
				}
			})
		}

		_ = oldImpl // Restore if needed
	}
}

// Test AppendBit for all implementations
func TestAppendBit(t *testing.T) {
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

	for _, impl := range []BitStringImpl{CharString, Uint64String, Uint64ArrayString} {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var bs BitString

				switch impl {
				case CharString:
					bs = NewCharFromBinary(tc.input)
				case Uint64String:
					if len(tc.input) < 64 { // Must have room for one more bit
						bs = NewUint64FromBinaryText(tc.input)
					} else {
						t.Skip("Uint64BitString would exceed 64 bits")
					}
				case Uint64ArrayString:
					bs = NewUint64ArrayFromBinaryText(tc.input)
				}

				result := bs.AppendBit(tc.bit)

				var expected BitString
				switch impl {
				case CharString:
					expected = NewCharFromBinary(tc.expected)
				case Uint64String:
					expected = NewUint64FromBinaryText(tc.expected)
				case Uint64ArrayString:
					expected = NewUint64ArrayFromBinaryText(tc.expected)
				}

				if !result.Equal(expected) {
					t.Errorf("AppendBit(%s, %t) = %v, want %v", tc.input, tc.bit, result, expected)
				}
			})
		}
	}
}

// Test IsAllOnes for all implementations
func TestIsAllOnes(t *testing.T) {
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

	for _, impl := range []BitStringImpl{CharString, Uint64String, Uint64ArrayString} {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var bs BitString

				switch impl {
				case CharString:
					bs = NewCharFromBinary(tc.input)
				case Uint64String:
					if len(tc.input) <= 64 {
						bs = NewUint64FromBinaryText(tc.input)
					} else {
						t.Skip("Uint64BitString limited to 64 bits")
					}
				case Uint64ArrayString:
					bs = NewUint64ArrayFromBinaryText(tc.input)
				}

				result := bs.IsAllOnes()

				if result != tc.expected {
					t.Errorf("IsAllOnes(%s) = %t, want %t", tc.input, result, tc.expected)
				}
			})
		}
	}
}

// Test Successor for all implementations
func TestSuccessor(t *testing.T) {
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

	for _, impl := range []BitStringImpl{CharString, Uint64String, Uint64ArrayString} {
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var bs BitString

				switch impl {
				case CharString:
					bs = NewCharFromBinary(tc.input)
				case Uint64String:
					if len(tc.input) < 64 { // Must have room for potential overflow
						bs = NewUint64FromBinaryText(tc.input)
					} else {
						t.Skip("Uint64BitString would exceed 64 bits")
					}
				case Uint64ArrayString:
					bs = NewUint64ArrayFromBinaryText(tc.input)
				}

				result := bs.Successor()

				var expected BitString
				switch impl {
				case CharString:
					expected = NewCharFromBinary(tc.expected)
				case Uint64String:
					expected = NewUint64FromBinaryText(tc.expected)
				case Uint64ArrayString:
					expected = NewUint64ArrayFromBinaryText(tc.expected)
				}

				if !result.Equal(expected) {
					t.Errorf("Successor(%s) = %v, want %v", tc.input, result, expected)
				}
			})
		}
	}
}

// Benchmark the new methods to ensure they're faster than string-based alternatives
func BenchmarkTrimTrailingZeros(b *testing.B) {
	bs := NewUint64FromBinaryText("1010000000")

	b.Run("NewMethod", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = bs.TrimTrailingZeros()
		}
	})
}

func BenchmarkAppendBit(b *testing.B) {
	bs := NewUint64FromBinaryText("101010")

	b.Run("NewMethod", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = bs.AppendBit(true)
		}
	})
}

func BenchmarkIsAllOnes(b *testing.B) {
	bs := NewUint64FromBinaryText("111111111111111111111111111111")

	b.Run("NewMethod", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = bs.IsAllOnes()
		}
	})
}

func BenchmarkSuccessor(b *testing.B) {
	bs := NewUint64FromBinaryText("1010101010")

	b.Run("NewMethod", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = bs.Successor()
		}
	})
}
