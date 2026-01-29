package bucket

import (
	"Thesis/bits"
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// TrieCompareSorter sorts BitStrings using TrieCompare for mixed-size string support
type TrieCompareSorter []bits.BitString

func (s TrieCompareSorter) Len() int           { return len(s) }
func (s TrieCompareSorter) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s TrieCompareSorter) Less(i, j int) bool { return s[i].TrieCompare(s[j]) < 0 }

func TestMonotoneHashWithMixedSizeStrings(t *testing.T) {
	t.Parallel()

	// Create a set of mixed-size strings with various scenarios
	testStrings := []string{
		"1",     // 1 bit
		"10",    // 2 bits
		"100",   // 3 bits (10 + trailing zero)
		"101",   // 3 bits
		"1010",  // 4 bits
		"10100", // 5 bits (1010 + trailing zero)
		"11",    // 2 bits
		"110",   // 3 bits (11 + trailing zero)
		"1100",  // 4 bits (11 + trailing zeros)
		"1101",  // 4 bits
	}

	// Convert to BitString
	keys := make([]bits.BitString, len(testStrings))
	for i, s := range testStrings {
		keys[i] = bits.NewFromBinary(s)
	}

	// Sort using TrieCompare (required for MMPH with mixed sizes)
	sort.Sort(TrieCompareSorter(keys))

	// Verify the order
	t.Logf("Sorted order:")
	for i, key := range keys {
		t.Logf("  %d: %s", i, key.PrettyString())
	}

	// Test MMPH construction with mixed-size strings
	mh, err := NewMonotoneHashWithTrie[uint16, uint8, uint8](keys)
	require.NoError(t, err, "failed to create MMPH with mixed-size strings")
	require.NotNil(t, mh, "MMPH should not be nil")

	t.Logf("MMPH built successfully with %d buckets, trie rebuild attempts: %d",
		len(mh.buckets), mh.TrieRebuildAttempts)

	// Verify that all keys have correct ranks
	for expectedRank, key := range keys {
		actualRank := mh.GetRank(key)
		require.Equal(t, expectedRank, actualRank,
			"rank mismatch for key %s: expected %d, got %d",
			key.PrettyString(), expectedRank, actualRank)
	}

	// Test with some keys not in the original set (should return -1)
	testNonExistentKeys := []string{"111", "1011", "001"}
	for _, s := range testNonExistentKeys {
		nonExistentKey := bits.NewFromBinary(s)
		rank := mh.GetRank(nonExistentKey)
		require.Equal(t, -1, rank, "non-existent key %s should return rank -1", s)
	}
}

func TestTrailingZerosBucketingConsistency(t *testing.T) {
	t.Parallel()

	// Test that strings with trailing zeros are handled consistently
	// This tests the core requirement that TrieCompare ordering is preserved
	testCases := []struct {
		name        string
		inputKeys   []string
		description string
	}{
		{
			name: "simple_trailing_zeros",
			inputKeys: []string{
				"10",   // 2 bits: 10
				"100",  // 3 bits: 100 (10 + trailing zero)
				"1000", // 4 bits: 1000 (10 + trailing zeros)
			},
			description: "simple case with one prefix and trailing zeros",
		},
		{
			name: "mixed_prefixes_trailing_zeros",
			inputKeys: []string{
				"1",    // 1 bit
				"10",   // 2 bits
				"100",  // 3 bits (10 + trailing zero)
				"101",  // 3 bits
				"11",   // 2 bits
				"110",  // 3 bits (11 + trailing zero)
				"1100", // 4 bits (11 + trailing zeros)
			},
			description: "multiple prefixes with trailing zeros",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Convert to BitString and sort using TrieCompare
			keys := make([]bits.BitString, len(tc.inputKeys))
			for i, s := range tc.inputKeys {
				keys[i] = bits.NewFromBinary(s)
			}

			// Sort using TrieCompare for consistency
			sort.Sort(TrieCompareSorter(keys))

			// Build MMPH
			mh, err := NewMonotoneHashWithTrie[uint16, uint8, uint8](keys)
			require.NoError(t, err, "failed to create MMPH for %s", tc.description)

			// Verify all keys work correctly
			for expectedRank, key := range keys {
				actualRank := mh.GetRank(key)
				require.Equal(t, expectedRank, actualRank,
					"rank mismatch in %s for key %s: expected %d, got %d",
					tc.description, key.PrettyString(), expectedRank, actualRank)
			}

			t.Logf("%s: successfully processed %d keys with %d trie rebuild attempts",
				tc.description, len(keys), mh.TrieRebuildAttempts)
		})
	}
}

func TestMMPHWithTrieCompareOrdering(t *testing.T) {
	t.Parallel()

	// Test that MMPH works correctly with TrieCompare ordered data
	// (comparison behavior itself is tested in bits package)

	key1 := bits.NewFromBinary("10")  // 2 bits: 10
	key2 := bits.NewFromBinary("100") // 3 bits: 100 (10 + trailing zero)

	// TrieCompare ordering: [100, 10] (trailing zeros before trimmed)
	keys := []bits.BitString{key2, key1}

	// Test MMPH with TrieCompare ordering
	mh, err := NewMonotoneHashWithTrie[uint16, uint8, uint8](keys)
	require.NoError(t, err, "MMPH should work with TrieCompare ordering")

	// Verify ranks match the sorted order
	require.Equal(t, 0, mh.GetRank(key2), "100 should have rank 0 in TrieCompare order")
	require.Equal(t, 1, mh.GetRank(key1), "10 should have rank 1 in TrieCompare order")

	t.Logf("MMPH correctly preserves TrieCompare ordering")
}

func TestLargerMixedSizeDataset(t *testing.T) {
	t.Parallel()

	// Test with a larger dataset of mixed-size strings
	var testStrings []string

	// Add strings of length 1-6 bits with various patterns
	patterns := []string{"1", "0"}
	for length := 1; length <= 4; length++ {
		newPatterns := []string{}
		for _, p := range patterns {
			if len(p) < 6 { // Limit total length
				newPatterns = append(newPatterns, p+"0", p+"1")
			}
		}
		patterns = append(patterns, newPatterns...)
	}

	// Use first 50 unique patterns to keep test reasonable
	seen := make(map[string]bool)
	for _, pattern := range patterns {
		if len(testStrings) >= 50 {
			break
		}
		if !seen[pattern] && len(pattern) <= 6 {
			testStrings = append(testStrings, pattern)
			seen[pattern] = true
		}
	}

	// Convert to BitString and sort using TrieCompare
	keys := make([]bits.BitString, len(testStrings))
	for i, s := range testStrings {
		keys[i] = bits.NewFromBinary(s)
	}
	sort.Sort(TrieCompareSorter(keys))

	// Build MMPH
	mh, err := NewMonotoneHashWithTrie[uint16, uint8, uint8](keys)
	require.NoError(t, err, "failed to create MMPH with larger mixed-size dataset")

	// Verify all keys
	for expectedRank, key := range keys {
		actualRank := mh.GetRank(key)
		require.Equal(t, expectedRank, actualRank,
			"rank mismatch for key %s in larger dataset", key.PrettyString())
	}

	t.Logf("Successfully processed %d mixed-size keys with %d trie rebuild attempts, %d buckets",
		len(keys), mh.TrieRebuildAttempts, len(mh.buckets))
}

func TestInputSortingValidation(t *testing.T) {
	t.Parallel()

	// Test case 1: Correctly sorted data (should succeed)
	t.Run("correctly_sorted", func(t *testing.T) {
		keys := []bits.BitString{
			bits.NewFromBinary("100"), // 3 bits: 100 (10 + trailing zero)
			bits.NewFromBinary("10"),  // 2 bits: 10
			bits.NewFromBinary("1"),   // 1 bit: 1
		}
		// This is already in TrieCompare order

		mh, err := NewMonotoneHashWithTrie[uint16, uint8, uint8](keys)
		require.NoError(t, err, "correctly sorted data should not produce an error")
		require.NotNil(t, mh, "MMPH should be created successfully")

		// Verify it works correctly
		for i, key := range keys {
			rank := mh.GetRank(key)
			require.Equal(t, i, rank, "rank should match expected order")
		}
	})

	// Test case 2: Incorrectly sorted data (should fail)
	t.Run("incorrectly_sorted", func(t *testing.T) {
		keys := []bits.BitString{
			bits.NewFromBinary("10"),  // 2 bits: 10
			bits.NewFromBinary("100"), // 3 bits: 100 (10 + trailing zero)
			bits.NewFromBinary("1"),   // 1 bit: 1
		}
		// This is in standard Compare order, NOT TrieCompare order

		mh, err := NewMonotoneHashWithTrie[uint16, uint8, uint8](keys)
		require.Error(t, err, "incorrectly sorted data should produce an error")
		require.Nil(t, mh, "MMPH should not be created")
		require.Contains(t, err.Error(), "input data must be sorted in TrieCompare order",
			"error message should mention TrieCompare ordering")

		t.Logf("Correctly detected unsorted input: %s", err.Error())
	})

	// Test case 3: Empty data (should succeed)
	t.Run("empty_data", func(t *testing.T) {
		var keys []bits.BitString

		mh, err := NewMonotoneHashWithTrie[uint16, uint8, uint8](keys)
		require.NoError(t, err, "empty data should not produce an error")
		require.NotNil(t, mh, "MMPH should be created for empty data")
	})

	// Test case 4: Single element (should succeed)
	t.Run("single_element", func(t *testing.T) {
		keys := []bits.BitString{
			bits.NewFromBinary("101"),
		}

		mh, err := NewMonotoneHashWithTrie[uint16, uint8, uint8](keys)
		require.NoError(t, err, "single element should not produce an error")
		require.NotNil(t, mh, "MMPH should be created for single element")

		rank := mh.GetRank(keys[0])
		require.Equal(t, 0, rank, "single element should have rank 0")
	})

	// Test case 5: Demonstrate correct usage pattern
	t.Run("correct_usage_pattern", func(t *testing.T) {
		// Start with unsorted mixed-size strings
		unsortedStrings := []string{"10", "100", "1", "110", "11"}

		// Convert to BitString
		keys := make([]bits.BitString, len(unsortedStrings))
		for i, s := range unsortedStrings {
			keys[i] = bits.NewFromBinary(s)
		}

		// Sort using TrieCompareSorter (this is the correct way)
		sort.Sort(TrieCompareSorter(keys))

		// Now MMPH construction should succeed
		mh, err := NewMonotoneHashWithTrie[uint16, uint8, uint8](keys)
		require.NoError(t, err, "properly sorted data should succeed")
		require.NotNil(t, mh, "MMPH should be created")

		// Verify all keys work
		for i, key := range keys {
			rank := mh.GetRank(key)
			require.Equal(t, i, rank, "rank should match position in sorted array")
		}

		t.Logf("Correct usage pattern: sort first, then create MMPH")
	})
}

func TestLargeRandomMixedSizeStrings(t *testing.T) {
	t.Parallel()

	sizes := []int{1, 10, 100, 1_000, 10_000, 100_000}
	runs := 100 // Multiple runs for each size

	for _, size := range sizes {
		for run := 0; run < runs; run++ {
			testName := fmt.Sprintf("MixedSize_%d_Run_%d", size, run+1)
			t.Run(testName, func(t *testing.T) {
				t.Parallel()

				// Generate mixed-size binary strings with various bit lengths
				keys := generateRandomMixedSizeKeys(size)

				// Sort using TrieCompare for mixed-size support
				sort.Sort(TrieCompareSorter(keys))

				// Build MMPH
				mh, err := NewMonotoneHashWithTrie[uint8, uint16, uint16](keys)
				if err != nil {
					t.Fatalf("Failed to create MMPH with %d mixed-size keys: %v", size, err)
				}

				t.Logf("Size %d, Run %d: Built MMPH with %d buckets, %d trie rebuild attempts",
					size, run+1, len(mh.buckets), mh.TrieRebuildAttempts)

				// Verify all keys have correct ranks
				for expectedRank, key := range keys {
					actualRank := mh.GetRank(key)
					if actualRank != expectedRank {
						t.Errorf("Rank mismatch for key %s: expected %d, got %d",
							key.PrettyString(), expectedRank, actualRank)
					}
				}

				// Test some random non-existent keys
				for i := 0; i < 10; i++ {
					// Generate a random non-existent key with random size
					size := 1 + rand.Intn(8)
					b := make([]byte, size)
					_, _ = rand.Read(b)
					nonExistentKey := bits.NewFromText(string(b))
					rank := mh.GetRank(nonExistentKey)
					// Should return -1 for non-existent keys (or false positive due to hash collision)
					if rank != -1 {
						t.Logf("False positive for non-existent key %s: rank %d",
							nonExistentKey.PrettyString(), rank)
					}
				}
			})
		}
	}
}

// generateRandomMixedSizeKeys creates a set of random strings using efficient byte generation
func generateRandomMixedSizeKeys(count int) []bits.BitString {
	keys := make([]bits.BitString, count)
	unique := make(map[string]bool, count)

	for i := 0; i < count; i++ {
		for {
			size := 1 + rand.Intn(15)
			b := make([]byte, size)
			_, _ = rand.Read(b)
			s := string(b)
			if !unique[s] {
				keys[i] = bits.NewFromText(s)
				unique[s] = true
				break
			}
		}
	}

	return keys
}
