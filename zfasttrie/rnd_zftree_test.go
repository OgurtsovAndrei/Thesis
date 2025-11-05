package zfasttrie

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

type testOperation struct {
	Op       string `json:"op"`
	Key      string `json:"key"`
	Value    bool   `json:"value,omitempty"`
	Expected bool   `json:"expected,omitempty"`
}

type testHistory struct {
	Seed       int64           `json:"seed"`
	Operations []testOperation `json:"operations"`
}

func saveHistoryAndFail(t *testing.T, seed int64, history []testOperation, format string, args ...interface{}) {
	t.Helper()

	fileName := fmt.Sprintf("fail_history_%d.json", seed)
	historyData := testHistory{
		Seed:       seed,
		Operations: history,
	}

	jsonData, err := json.MarshalIndent(historyData, "", "  ")
	if err != nil {
		t.Logf("!!! Failed to marshal failure history: %v", err)
	} else {
		if err := os.WriteFile(fileName, jsonData, 0644); err != nil {
			t.Logf("!!! Failed to write failure history to %s: %v", fileName, err)
		} else {
			t.Logf("--- FAILURE HISTORY SAVED TO: %s ---", fileName)
		}
	}

	t.Fatalf(format, args...)
}

func randString(r *rand.Rand, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 1
	}

	var length int
	if maxLength == 1 {
		length = 1
	} else {
		length = r.Intn(maxLength-1) + 1
	}

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

func generateString(r *rand.Rand, existingKeys []string, prefixBias float64, maxLength int) string {
	if len(existingKeys) > 0 && r.Float64() < prefixBias {
		baseKey := existingKeys[r.Intn(len(existingKeys))]
		if len(baseKey) <= 1 {
			return randString(r, maxLength)
		}

		prefixLen := r.Intn(len(baseKey)-1) + 1
		prefix := baseKey[:prefixLen]

		suffixLen := r.Intn(maxLength/2) + 1
		suffix := randString(r, suffixLen)

		return prefix + suffix
	}
	return randString(r, maxLength)
}

func TestTrie_HeavyRandom_InsertDeleteContains(t *testing.T) {
	for range 100 {
		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))
		tree := NewZFastTrie[bool](false)
		groundTruth := make(map[string]bool)
		insertedKeys := make([]string, 0)

		numOperations := 10000
		maxStrLength := 3
		prefixBias := 0.3

		history := make([]testOperation, 0, numOperations)

		defer func() {
			if r := recover(); r != nil {
				saveHistoryAndFail(t, seed, history, "Test panicked: %v", r)
			}
		}()

		for i := 0; i < numOperations; i++ {
			op := r.Intn(100)

			if op < 45 {
				s := generateString(r, insertedKeys, prefixBias, maxStrLength)
				history = append(history, testOperation{Op: "Insert", Key: s, Value: true})

				if _, exists := groundTruth[s]; !exists {
					groundTruth[s] = true
					insertedKeys = append(insertedKeys, s)
					tree.Insert(s, true)

					if !tree.Contains(s) {
						fmt.Println(tree.String())
						fmt.Println(NewBitString(s))
						fmt.Println(!tree.Contains(s))
						saveHistoryAndFail(t, seed, history, "Failed to find just-inserted key: %q", s)
					}
				}
			} else if op < 80 {
				if len(insertedKeys) == 0 {
					history = append(history, testOperation{Op: "Erase", Key: "SKIPPED"})
					continue
				}
				idx := r.Intn(len(insertedKeys))
				s := insertedKeys[idx]
				history = append(history, testOperation{Op: "Erase", Key: s})

				delete(groundTruth, s)
				insertedKeys[idx] = insertedKeys[len(insertedKeys)-1]
				insertedKeys = insertedKeys[:len(insertedKeys)-1]

				tree.Erase(s)

				if tree.Contains(s) {
					saveHistoryAndFail(t, seed, history, "Found just-deleted key: %q", s)
				}
			} else {
				s := generateString(r, insertedKeys, prefixBias, maxStrLength)
				expected, _ := groundTruth[s]
				history = append(history, testOperation{Op: "Contains", Key: s, Expected: expected})

				actual := tree.Contains(s)

				if actual != expected {
					saveHistoryAndFail(t, seed, history, "Contains mismatch for key %q. Expected: %v, Got: %v", s, expected, actual)
				}
			}
		}

		for key := range groundTruth {
			if !tree.Contains(key) {
				saveHistoryAndFail(t, seed, history, "Final check failed: key %q in ground truth but not in trie", key)
			}
		}

		if int(tree.size) != len(groundTruth) {
			saveHistoryAndFail(t, seed, history, "Final size mismatch. Expected: %d, Got: %d", len(groundTruth), tree.size)
		}
	}
}

func TestTrie_HeavyRandom_Prefixes(t *testing.T) {
	seed := time.Now().UnixNano()
	r := rand.New(rand.NewSource(seed))
	tree := NewZFastTrie[bool](false)
	groundTruth := make(map[string]bool)
	prefixGroundTruth := make(map[string]bool)
	insertedKeys := make([]string, 0)

	numOperations := 1000
	maxStrLength := 3
	prefixBias := 0.3

	history := make([]testOperation, 0, numOperations)

	defer func() {
		if r := recover(); r != nil {
			saveHistoryAndFail(t, seed, history, "Test panicked: %v", r)
		}
	}()

	for i := 0; i < numOperations; i++ {
		s := generateString(r, insertedKeys, prefixBias, maxStrLength)
		if _, exists := groundTruth[s]; exists {
			history = append(history, testOperation{Op: "Insert", Key: s, Value: false})
			continue
		}

		history = append(history, testOperation{Op: "Insert", Key: s, Value: true})
		groundTruth[s] = true
		insertedKeys = append(insertedKeys, s)
		tree.Insert(s, true)

		for j := 0; j <= len(s); j++ {
			prefixGroundTruth[s[:j]] = true
		}
	}

	for prefix := range prefixGroundTruth {
		history = append(history, testOperation{Op: "ContainsPrefix", Key: prefix, Expected: true})
		if !tree.ContainsPrefix(prefix) {
			saveHistoryAndFail(t, seed, history, "Prefix check failed: %q should be a prefix but was not found", prefix)
		}
	}

	for i := 0; i < numOperations/10; i++ {
		s := randString(r, maxStrLength)
		if _, isPrefix := prefixGroundTruth[s]; isPrefix {
			history = append(history, testOperation{Op: "ContainsPrefix", Key: s, Expected: true})
			continue
		}

		history = append(history, testOperation{Op: "ContainsPrefix", Key: s, Expected: false})
		if tree.ContainsPrefix(s) {
			saveHistoryAndFail(t, seed, history, "Negative prefix check failed: %q should NOT be a prefix but was found", s)
		}
	}
}
