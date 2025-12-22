package zfasttrie

import (
	"Thesis/errutil"
	"math/rand"
	"testing"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/stretchr/testify/require"
)

const (
	n          = 32
	bitLen     = 16
	iterations = 10_000
	testRuns   = 1000
)

func TestApproxZFastTrie_Properties(t *testing.T) {
	fpCount := 0
	bar := progressbar.Default(testRuns)
	for run := 0; run < testRuns; run++ {

		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))

		keys := generateRandomBitStrings(n, bitLen, r)

		azft, err := NewApproxZFastTrie[uint16, uint8](keys)
		require.NoError(t, err, "failed to build trie")

		referenceTrie := Build(keys)

		for _, key := range keys {
			node := azft.GetExistingPrefix(key)
			require.NotNil(t, node, "expected node for key %s, got nil (seed: %d)", key.String(), seed)

			require.LessOrEqual(t, uint32(node.extentLen), key.Size(), "found extent length %d is greater than key size %d", node.extentLen, key.Size())

			prefix := key.Prefix(int(node.extentLen))
			require.Equal(t, node.PSig, uint8(hashBitString(prefix, azft.seed)), "signature mismatch for key %s", key.String())
		}

		// Count False Positives using random patterns
		for i := 0; i < iterations; i++ {
			randomPattern := generateBitString(bitLen, r)
			node := azft.GetExistingPrefix(randomPattern)

			if node != nil {
				// A false positive occurs if the returned prefix doesn't exist in the reference trie
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
	fnCount := 0
	bar := progressbar.Default(testRuns)
	for run := 0; run < testRuns; run++ {
		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))

		keys := generateRandomBitStrings(n, bitLen, r)

		azft, err := NewApproxZFastTrie[uint16, uint8](keys)
		require.NoError(t, err)

		referenceTrie := azft.trie

		for i := 0; i < iterations; i++ {
			randomKey := keys[r.Intn(len(keys))]
			errutil.BugOn(!referenceTrie.containsPrefixBitString(randomKey), "")
			prefixLen := r.Intn(int(randomKey.Size())) + 1
			validPrefix := randomKey.Prefix(prefixLen)

			node := azft.GetExistingPrefix(validPrefix)
			require.NotNil(t, node, "False Negative: expected node for prefix of existing key (seed: %d)", seed)
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
