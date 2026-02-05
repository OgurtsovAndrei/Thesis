package zfasttrie

import (
	"Thesis/bits"
	"Thesis/errutil"
	"math/rand"
	"testing"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/stretchr/testify/require"
)

const (
	n          = 32
	bitLen     = 64
	iterations = 1_000
	testRuns   = 1_000
)

func TestApproxZFastTrie_Properties(t *testing.T) {
	t.Parallel()
	fpCount := 0
	bar := progressbar.Default(testRuns)
	for run := 0; run < testRuns; run++ {

		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))

		keys := generateRandomBitStrings(n, bitLen, r)

		azft, err := NewApproxZFastTrie[uint16, uint8, uint8](keys, true)
		require.NoError(t, err, "failed to build Trie")

		referenceTrie := azft.Trie

		for _, key := range keys {
			node := azft.GetExistingPrefix(key)
			require.NotNil(t, node, "expected node for key %s, got nil (seed: %d)", key.PrettyString(), seed)

			require.LessOrEqual(t, uint32(node.extentLen), key.Size(), "found extent length %d is greater than key size %d", node.extentLen, key.Size())

			prefix := key.Prefix(int(node.extentLen))
			require.Equal(t, node.PSig, uint8(hashBitString(prefix, azft.seed)), "signature mismatch for key %s", key.PrettyString())
		}

		// Count False Positives using random patterns
		for i := 0; i < iterations; i++ {
			randomPattern := generateBitString(bitLen, r)
			node := azft.GetExistingPrefix(randomPattern)

			if node != nil {
				// A false positive occurs if the returned prefix doesn't exist in the reference Trie
				prefix := randomPattern.Prefix(int(node.extentLen))
				original_node := referenceTrie.getExitNode(prefix)

				if original_node == nil {
					fpCount++
				} else if original_node.extent.GetLCPLength(randomPattern) != prefix.Size() {
					fpCount++
				}
			}
		}
		_ = bar.Add(1)
	}

	t.Logf("Tested %d random patterns. False Positives found: %d (Rate: %.7f)",
		iterations*testRuns, fpCount, float64(fpCount)/float64(iterations*testRuns))
}

func TestApproxZFastTrie_FalseNegatives(t *testing.T) {
	t.Parallel()
	fnCount := 0
	bar := progressbar.Default(testRuns)
	for run := 0; run < testRuns; run++ {
		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))

		keys := generateRandomBitStrings(n, bitLen, r)

		azft, err := NewApproxZFastTrie[uint16, uint8, uint8](keys, true)
		require.NoError(t, err)

		referenceTrie := azft.Trie

		for i := 0; i < iterations; i++ {
			randomKey := keys[r.Intn(len(keys))]
			errutil.BugOn(!referenceTrie.containsPrefixBitString(randomKey), "")
			prefixLen := r.Intn(int(randomKey.Size())) + 1
			validPrefix := randomKey.Prefix(prefixLen)

			node := azft.GetExistingPrefix(validPrefix)
			if node == nil {
				node := azft.GetExistingPrefix(validPrefix)
				require.NotNil(t, node, "False Negative: expected node for prefix of existing key (seed: %d), prefix: %s\n\ntree dump:\n%s\n", seed, validPrefix.PrettyString(), referenceTrie.String())
			}
			resultPrefix := validPrefix.Prefix(int(node.extentLen))

			expectedNode := referenceTrie.getExistingPrefix(resultPrefix)
			require.NotNil(t, expectedNode)

			if expectedNode.extentLength() != uint32(node.extentLen) {
				fnCount++
				continue
			}

			sig := uint8(hashBitString(validPrefix.Prefix(int(node.extentLen)), azft.seed))
			if node.PSig != sig {
				fnCount++
				continue
			}
		}
		_ = bar.Add(1)
	}
	t.Logf("Tested %d random patterns. False Negatives found: %d (Rate: %.7f)",
		iterations*testRuns, fnCount, float64(fnCount)/float64(iterations*testRuns))
}

func TestApproxZFastTrie_LowerBound_FP(t *testing.T) {
	t.Parallel()
	fpCount := 0
	bar := progressbar.Default(testRuns)
	for run := 0; run < testRuns; run++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		keys := generateRandomBitStrings(n, bitLen, r)

		azft, err := NewApproxZFastTrie[uint16, uint8, uint8](keys, true)
		require.NoError(t, err)

		for i := 0; i < iterations; i++ {
			pattern := generateBitString(bitLen, r)

			var expectedKey bits.BitString
			found := false
			for _, k := range keys {
				if k.Compare(pattern) >= 0 {
					expectedKey = k
					found = true
					break
				}
			}
			if !found {
				// struct always predict keys, even if lower bound does not exist
				continue
			}

			c1, c2, c3, c4, c5, c6 := azft.LowerBound(pattern)

			if c1 == nil || c2 == nil || c3 == nil || c4 == nil || c5 == nil || c6 == nil ||
				(!c1.originalNode.extent.Equal(expectedKey) && !c2.originalNode.extent.Equal(expectedKey) && !c3.originalNode.extent.Equal(expectedKey) && !c4.originalNode.extent.Equal(expectedKey) && !c5.originalNode.extent.Equal(expectedKey) && !c6.originalNode.extent.Equal(expectedKey)) {
				c1, c2, c3, c4, c5, c6 = azft.LowerBound(pattern)
				fpCount++
			}
		}
		_ = bar.Add(1)
	}
	t.Logf("LowerBound False Positives found: %d (Rate: %.7f)",
		fpCount, float64(fpCount)/float64(iterations*testRuns))
}

func TestApproxZFastTrie_LowerBound_FN(t *testing.T) {
	t.Parallel()
	errCount := 0
	totalChecks := 0

	runsWithErrors := 0

	bar := progressbar.Default(testRuns)
	for run := 0; run < testRuns; run++ {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		keys := generateRandomBitStrings(n, bitLen, r)

		azft, err := NewApproxZFastTrie[uint16, uint8, uint8](keys, true)
		require.NoError(t, err)

		errRun := 0
		for _, key := range keys {
			for length := 1; length <= int(key.Size()); length++ {
				prefix := key.Prefix(length)

				var expectedKey bits.BitString
				found := false
				for _, k := range keys {
					if k.Compare(prefix) >= 0 {
						expectedKey = k
						found = true
						break
					}
				}

				totalChecks++
				c1, c2, c3, c4, c5, c6 := azft.LowerBound(prefix)

				if !found {
					if c1 != nil || c2 != nil || c3 != nil || c4 != nil || c5 != nil || c6 != nil {
						errCount++
						errRun = 1
					}
					continue
				}

				if (c1 == nil || !c1.originalNode.extent.Equal(expectedKey)) &&
					(c2 == nil || !c2.originalNode.extent.Equal(expectedKey)) &&
					(c3 == nil || !c3.originalNode.extent.Equal(expectedKey)) &&
					(c4 == nil || !c4.originalNode.extent.Equal(expectedKey)) &&
					(c5 == nil || !c5.originalNode.extent.Equal(expectedKey)) &&
					(c6 == nil || !c6.originalNode.extent.Equal(expectedKey)) {
					errCount++
					errRun = 1
				}
			}
		}
		runsWithErrors += errRun
		_ = bar.Add(1)
	}
	t.Logf("LowerBound False Negatives/Errors on prefixes found: %d (Rate: %.7f)", errCount, float64(errCount)/float64(totalChecks))
	t.Logf("LowerBound False Negatives runs: %d (Rate: %.7f)", runsWithErrors, float64(runsWithErrors)/float64(testRuns))
}
