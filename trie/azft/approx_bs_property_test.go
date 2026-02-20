package azft

import (
	"Thesis/bits"
	"Thesis/errutil"
	"Thesis/trie/zft"
	"Thesis/utils"
	"context"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fpCount := 0
	bar := progressbar.Default(testRuns)
	for run := 0; run < testRuns; run++ {
		t.Run(time.Now().String(), func(t *testing.T) {
			select {
			case <-ctx.Done():
				t.Skip()
			default:
			}
			t.Parallel()

			seed := time.Now().UnixNano()
			r := rand.New(rand.NewSource(seed))

			keys := zft.GenerateRandomBitStrings(n, bitLen, r)

			azft, err := NewApproxZFastTrie[uint16, uint8, uint8](keys)
			if err != nil {
				cancel()
				t.Fatalf("failed to build Trie: %v", err)
			}

			// Build reference trie separately for verification
			referenceTrie, err := zft.BuildFromIterator(bits.NewSliceBitStringIterator(keys))
			if err != nil {
				cancel()
				t.Fatalf("failed to build reference trie: %v", err)
			}

			for _, key := range keys {
				node := azft.GetExistingPrefix(key)
				if node == nil {
					cancel()
					t.Fatalf("expected node for key %s, got nil (seed: %d)", key.PrettyString(), seed)
				}

				if uint32(node.extentLen) > key.Size() {
					cancel()
					t.Fatalf("found extent length %d is greater than key size %d", node.extentLen, key.Size())
				}

				prefix := key.Prefix(int(node.extentLen))
				if node.PSig != uint8(hashBitString(prefix, azft.seed)) {
					cancel()
					t.Fatalf("signature mismatch for key %s", key.PrettyString())
				}
			}

			// Count False Positives using random patterns
			for i := 0; i < iterations; i++ {
				randomPattern := zft.GenerateBitString(bitLen, r)
				node := azft.GetExistingPrefix(randomPattern)

				if node != nil {
					// A false positive occurs if the returned prefix doesn't exist in the reference Trie
					prefix := randomPattern.Prefix(int(node.extentLen))
					original_node := referenceTrie.GetExitNode(prefix)

					if original_node == nil {
						fpCount++
					} else if original_node.Extent.GetLCPLength(randomPattern) != prefix.Size() {
						fpCount++
					}
				}
			}
			_ = bar.Add(1)
		})
	}

	t.Logf("Tested %d random patterns. False Positives found: %d (Rate: %.7f)",
		iterations*testRuns, fpCount, float64(fpCount)/float64(iterations*testRuns))
}

func TestApproxZFastTrie_FalseNegatives(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fnCount := 0
	bar := progressbar.Default(testRuns)
	for run := 0; run < testRuns; run++ {
		t.Run(time.Now().String(), func(t *testing.T) {
			select {
			case <-ctx.Done():
				t.Skip()
			default:
			}
			t.Parallel()

			seed := time.Now().UnixNano()
			r := rand.New(rand.NewSource(seed))

			keys := zft.GenerateRandomBitStrings(n, bitLen, r)

			azft, err := NewApproxZFastTrie[uint16, uint8, uint8](keys)
			if err != nil {
				cancel()
				t.Fatalf("failed to build: %v", err)
			}

			// Build reference trie separately for verification
			referenceTrie, err := zft.BuildFromIterator(bits.NewSliceBitStringIterator(keys))
			if err != nil {
				cancel()
				t.Fatalf("failed to build reference: %v", err)
			}

			for i := 0; i < iterations; i++ {
				randomKey := keys[r.Intn(len(keys))]
				errutil.BugOn(!referenceTrie.ContainsPrefixBitString(randomKey), "")
				prefixLen := r.Intn(int(randomKey.Size())) + 1
				validPrefix := randomKey.Prefix(prefixLen)

				node := azft.GetExistingPrefix(validPrefix)
				if node == nil {
					cancel()
					t.Fatalf("False Negative: expected node for prefix of existing key (seed: %d), prefix: %s\n\ntree dump:\n%s\n", seed, validPrefix.PrettyString(), referenceTrie.String())
				}
				resultPrefix := validPrefix.Prefix(int(node.extentLen))

				expectedNode := referenceTrie.GetExistingPrefix(resultPrefix)
				if expectedNode == nil {
					cancel()
					t.Fatalf("Expected node nil")
				}

				if expectedNode.ExtentLength() != uint32(node.extentLen) {
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
		})
	}
	t.Logf("Tested %d random patterns. False Negatives found: %d (Rate: %.7f)",
		iterations*testRuns, fnCount, float64(fnCount)/float64(iterations*testRuns))
}

func TestApproxZFastTrie_PrefixRelationships(t *testing.T) {
	// Keys where some are prefixes of others
	keys := []bits.BitString{
		bits.NewFromText("10"),
		bits.NewFromText("100"),
		bits.NewFromText("101"),
		bits.NewFromText("11"),
	}

	azft, err := NewApproxZFastTrie[uint8, uint8, uint8](keys)
	require.NoError(t, err)

	// Query for "10"
	// GetExistingPrefix("10") should return node "10"
	// LowerBound("10") should return "10" (rank 0)
	c1, c2, c3, c6 := azft.LowerBound(bits.NewFromText("10"))
	candidates := []*NodeData[uint8, uint8, uint8]{c1, c2, c3, c6}

	found := -1
	for i, c := range candidates {
		if c != nil && c.Rank == 0 {
			found = i
			break
		}
	}
	require.True(t, found != -1, "Should find rank 0 for key '10'")
	t.Logf("Candidate %d matched rank 0 for '10'", found+1)
}

func TestApproxZFastTrie_LowerBound_FP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fpCount := 0
	totalMatchCounts := make([]int, 4)

	bar := progressbar.Default(testRuns)

	for run := 0; run < testRuns; run++ {
		t.Run(time.Now().String(), func(t *testing.T) {
			select {
			case <-ctx.Done():
				t.Skip()
			default:
			}
			t.Parallel()

			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			keys := zft.GenerateRandomBitStrings(n, bitLen, r)

			azft, err := NewApproxZFastTrie[uint16, uint8, uint8](keys)
			if err != nil {
				cancel()
				t.Fatalf("failed: %v", err)
			}

			for i := 0; i < iterations; i++ {
				pattern := zft.GenerateBitString(bitLen, r)

				c1, c2, c3, c6 := azft.LowerBound(pattern)

				// Check if any candidate has the expected rank
				expectedRank := -1
				for idx, k := range keys {
					if k.Compare(pattern) >= 0 {
						expectedRank = idx
						break
					}
				}

				maxIdx := uint8(^uint8(0))
				foundMatch := false
				candidates := []*NodeData[uint16, uint8, uint8]{c1, c2, c3, c6}
				matchCounts := make([]int, 4)
				for i, cand := range candidates {
					if cand != nil && cand.Rank != maxIdx && int(cand.Rank) == expectedRank {
						foundMatch = true
						matchCounts[i]++
						break
					}
				}

				if !foundMatch {
					fpCount++
				} else {
					for i := 0; i < 4; i++ {
						totalMatchCounts[i] += matchCounts[i]
					}
				}
			}
			_ = bar.Add(1)
		})
	}

	t.Logf("LowerBound False Positives found: %d (Rate: %.7f)",
		fpCount, float64(fpCount)/float64(iterations*testRuns))
	t.Logf("Candidate match statistics: Cand1: %d, Cand2: %d, Cand3: %d, Cand6: %d",
		totalMatchCounts[0], totalMatchCounts[1], totalMatchCounts[2], totalMatchCounts[3])
	utils.LogCandidateMatch("AZFT.LowerBound_FP", totalMatchCounts)
}

func TestApproxZFastTrie_LowerBound_FN(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCount := 0
	totalChecks := 0
	totalMatchCounts := make([]int, 4)

	runsWithErrors := 0

	bar := progressbar.Default(testRuns)
	for run := 0; run < testRuns; run++ {
		t.Run(time.Now().String(), func(t *testing.T) {
			select {
			case <-ctx.Done():
				t.Skip()
			default:
			}
			t.Parallel()

			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			keys := zft.GenerateRandomBitStrings(n, bitLen, r)

			azft, err := NewApproxZFastTrie[uint16, uint8, uint8](keys)
			if err != nil {
				cancel()
				t.Fatalf("failed: %v", err)
			}

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
					c1, c2, c3, c6 := azft.LowerBound(prefix)

					if !found {
						if c1 != nil || c2 != nil || c3 != nil || c6 != nil {
							errCount++
							errRun = 1
						}
						continue
					}

					// Check if any candidate has the expected rank
					expectedRank := -1
					for idx, k := range keys {
						if k.Equal(expectedKey) {
							expectedRank = idx
							break
						}
					}

					maxIdx := uint8(^uint8(0))
					foundMatch := false
					candidates := []*NodeData[uint16, uint8, uint8]{c1, c2, c3, c6}
					matchCounts := make([]int, 4)
					for i, cand := range candidates {
						if cand != nil && cand.Rank != maxIdx && int(cand.Rank) == expectedRank {
							foundMatch = true
							matchCounts[i]++
							break
						}
					}

					if !foundMatch {
						errCount++
						errRun = 1
					} else {
						for i := 0; i < 4; i++ {
							totalMatchCounts[i] += matchCounts[i]
						}
					}
				}
			}
			if errRun == 1 {
				runsWithErrors++
			}
			_ = bar.Add(1)
		})
	}
	t.Logf("LowerBound False Negatives/Errors on prefixes found: %d (Rate: %.7f)", errCount, float64(errCount)/float64(totalChecks))
	t.Logf("LowerBound False Negatives runs: %d (Rate: %.7f)", runsWithErrors, float64(runsWithErrors)/float64(testRuns))
	t.Logf("Candidate match statistics: Cand1: %d, Cand2: %d, Cand3: %d, Cand6: %d",
		totalMatchCounts[0], totalMatchCounts[1], totalMatchCounts[2], totalMatchCounts[3])
	utils.LogCandidateMatch("AZFT.LowerBound_FN", totalMatchCounts)
}
