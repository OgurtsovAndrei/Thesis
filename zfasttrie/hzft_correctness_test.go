package zfasttrie

import (
	"Thesis/bits"
	"math/rand"
	"testing"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/stretchr/testify/require"
)

const (
	hzftTestRuns   = 1000
	hzftIterations = 1000
	hzftMaxKeys    = 64
	hzftMaxBitLen  = 64
)

func TestHZFastTrie_Correctness_AllPrefixes(t *testing.T) {
	errorCount := 0
	totalChecks := 0
	runsWithErrors := 0

	bar := progressbar.Default(hzftTestRuns)
	for run := 0; run < hzftTestRuns; run++ {
		seed := time.Now().UnixNano() + int64(run)
		r := rand.New(rand.NewSource(seed))

		// Generate random keys
		numKeys := r.Intn(hzftMaxKeys-1) + 2  // At least 2 keys
		bitLen := r.Intn(hzftMaxBitLen-8) + 8 // At least 8 bits
		keys := generateRandomBitStrings(numKeys, bitLen, r)

		// Build HZFT and reference trie
		hzft := NewHZFastTrie[uint32](keys)
		if hzft == nil {
			continue
		}
		referenceTrie := hzft.trie

		errorsInRun := 0

		// Test all prefixes of all keys - these should NEVER fail in a deterministic structure
		for _, key := range keys {
			for prefixLen := 1; prefixLen <= int(key.Size()); prefixLen++ {
				prefix := key.Prefix(prefixLen)
				totalChecks++

				// Get result from HZFT
				hzftResult := hzft.GetExistingPrefix(prefix)
				require.NotNil(t, hzftResult, "HZFT returned nil for prefix %s of key %s (seed: %d)",
					prefix.String(), key.String(), seed)

				// Get expected result from reference
				expectedNode := referenceTrie.getExitNode(prefix)
				require.NotNil(t, expectedNode, "Reference trie returned nil for prefix %s (seed: %d)",
					prefix.String(), seed)

				// Compare extent lengths
				if int32(hzftResult.extentLen+1) != expectedNode.nameLength {
					hzftResult = hzft.GetExistingPrefix(prefix)
					hzft = NewHZFastTrie[uint32](keys)
					t.Errorf("Extent length mismatch for prefix %s: HZFT=%d, Reference=%d (seed: %d)",
						prefix.String(), hzftResult.extentLen, expectedNode.extentLength(), seed)
					errorCount++
					errorsInRun++
				}

				// Verify the result makes sense
				//if uint32(hzftResult.extentLen) > prefix.Size() {
				//	t.Errorf("HZFT returned extent length %d > prefix length %d for %s (seed: %d)",
				//		hzftResult.extentLen, prefix.Size(), prefix.String(), seed)
				//	errorCount++
				//	errorsInRun++
				//}
			}
		}

		if errorsInRun > 0 {
			runsWithErrors++
		}
		_ = bar.Add(1)
	}

	t.Logf("Total checks: %d, Errors: %d (Rate: %.7f)", totalChecks, errorCount, float64(errorCount)/float64(totalChecks))
	t.Logf("Runs with errors: %d out of %d (Rate: %.7f)", runsWithErrors, hzftTestRuns, float64(runsWithErrors)/float64(hzftTestRuns))

	// HZFT should be deterministic - no errors allowed
	require.Equal(t, 0, errorCount, "HZFT should have zero errors as it's deterministic")
}

func TestHZFastTrie_Correctness_RandomPatterns(t *testing.T) {
	falsePositives := 0
	falseNegatives := 0
	totalChecks := 0

	bar := progressbar.Default(hzftTestRuns)
	for run := 0; run < hzftTestRuns; run++ {
		seed := time.Now().UnixNano() + int64(run)
		r := rand.New(rand.NewSource(seed))

		numKeys := r.Intn(hzftMaxKeys-1) + 2
		bitLen := r.Intn(hzftMaxBitLen-8) + 8
		keys := generateRandomBitStrings(numKeys, bitLen, r)

		hzft := NewHZFastTrie[uint32](keys)
		if hzft == nil {
			continue
		}
		referenceTrie := hzft.trie

		// Test with random patterns
		for i := 0; i < hzftIterations; i++ {
			pattern := generateBitString(bitLen, r)
			totalChecks++

			hzftResult := hzft.GetExistingPrefix(pattern)
			expectedNode := referenceTrie.getExistingPrefix(pattern)

			// Both should return non-nil (root at minimum)
			require.NotNil(t, hzftResult, "HZFT returned nil for pattern %s", pattern.String())
			require.NotNil(t, expectedNode, "Reference returned nil for pattern %s", pattern.String())

			// Check for false positives/negatives
			if uint32(hzftResult.extentLen) != expectedNode.extentLength() {
				if uint32(hzftResult.extentLen) > expectedNode.extentLength() {
					falsePositives++
					t.Logf("False positive: HZFT extent=%d, Reference=%d for pattern %s (seed: %d)",
						hzftResult.extentLen, expectedNode.extentLength(), pattern.String(), seed)
				} else {
					falseNegatives++
					t.Logf("False negative: HZFT extent=%d, Reference=%d for pattern %s (seed: %d)",
						hzftResult.extentLen, expectedNode.extentLength(), pattern.String(), seed)
				}
			}
		}
		_ = bar.Add(1)
	}

	t.Logf("Total random pattern checks: %d", totalChecks)
	t.Logf("False positives: %d (Rate: %.7f)", falsePositives, float64(falsePositives)/float64(totalChecks))
	t.Logf("False negatives: %d (Rate: %.7f)", falseNegatives, float64(falseNegatives)/float64(totalChecks))

	// HZFT should have zero false positives and negatives
	require.Equal(t, 0, falsePositives, "HZFT should have zero false positives")
	require.Equal(t, 0, falseNegatives, "HZFT should have zero false negatives")
}

func TestHZFastTrie_EdgeCases(t *testing.T) {
	t.Run("Empty trie", func(t *testing.T) {
		var keys []bits.BitString
		hzft := NewHZFastTrie[uint32](keys)
		require.Nil(t, hzft, "Empty trie should return nil")
	})

	t.Run("Single key", func(t *testing.T) {
		keys := []bits.BitString{bits.NewBitString("test")}
		hzft := NewHZFastTrie[uint32](keys)
		require.NotNil(t, hzft)

		// Test all prefixes of the single key
		key := keys[0]
		for i := 1; i <= int(key.Size()); i++ {
			prefix := key.Prefix(i)
			result := hzft.GetExistingPrefix(prefix)
			require.NotNil(t, result, "Should find prefix %s", prefix.String())
			require.LessOrEqual(t, uint32(result.extentLen), prefix.Size(),
				"Extent length should not exceed prefix length")
		}
	})

	t.Run("Two keys with common prefix", func(t *testing.T) {
		keys := []bits.BitString{
			bits.NewBitString("abc"),
			bits.NewBitString("abd"),
		}
		hzft := NewHZFastTrie[uint32](keys)
		require.NotNil(t, hzft)

		// Test common prefix "ab"
		commonPrefix := bits.NewBitString("ab")
		result := hzft.GetExistingPrefix(commonPrefix)
		require.NotNil(t, result, "Should find common prefix")

		// Compare with reference
		expected := hzft.trie.getExistingPrefix(commonPrefix)
		require.Equal(t, expected.extentLength(), uint32(result.extentLen),
			"Should match reference result")
	})
}

func TestHZFastTrie_CompareConstruction(t *testing.T) {
	// Test that HZFT construction produces consistent results
	for run := 0; run < 10; run++ {
		r := rand.New(rand.NewSource(int64(run)))
		keys := generateRandomBitStrings(20, 16, r)

		hzft := NewHZFastTrie[uint32](keys)
		if hzft == nil {
			continue
		}

		// Verify all keys can be found
		for _, key := range keys {
			result := hzft.GetExistingPrefix(key)
			require.NotNil(t, result, "Should find key %s", key.String())

			// For full keys, the extent length should be >= key length
			require.LessOrEqual(t, key.Size(), uint32(result.extentLen),
				"Key %s should have extent length >= key length, got %d", key.String(), result.extentLen)
		}

		// Verify handle-to-node mapping is consistent
		require.Equal(t, len(hzft.data), len(hzft.trie.handle2NodeMap),
			"HZFT data length should match reference handle map length")
	}
}
