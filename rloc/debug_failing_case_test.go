package rloc

import (
	"Thesis/zfasttrie"
	"testing"
)

// TestDebugFailingCase is a helper test to manually debug a saved failing case
// To use:
// 1. Run the main tests until a failure is saved
// 2. Update the filename below to match the saved file
// 3. Run: go test -v -run TestDebugFailingCase
func TestDebugFailingCase(t *testing.T) {
	t.Skip("Skipping debug test - enable manually when needed")

	// Update this to the actual saved file
	filename := "failing_case_seed_1769679785375971000.json"

	keys, err := LoadFailingCase(filename)
	if err != nil {
		t.Fatalf("Failed to load failing case: %v", err)
	}

	// Verify keys are sorted and unique
	for i := 1; i < len(keys); i++ {
		cmp := keys[i].TrieCompare(keys[i-1])
		if cmp < 0 {
			t.Fatalf("Keys are not sorted at index %d", i)
		}
		if cmp == 0 {
			t.Fatalf("Duplicate key found at index %d", i)
		}
	}

	t.Logf("Testing with %d keys", len(keys))

	// Try to build the RangeLocator
	zt := zfasttrie.Build(keys)
	rl, err := NewRangeLocator(zt)
	if err != nil {
		t.Logf("RangeLocator construction failed as expected: %v", err)

		// Here you can add additional debugging code:
		// - Print the P set
		// - Print delimiter indices
		// - Test ApproxZFastTrie directly
		// - etc.

		t.FailNow()
	}

	t.Logf("RangeLocator construction succeeded with %d bytes", rl.ByteSize())
}
