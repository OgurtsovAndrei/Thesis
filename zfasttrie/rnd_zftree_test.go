package zfasttrie

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func randString(r *rand.Rand, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 1
	}

	var length int
	if maxLength == 1 {
		length = 1
	} else {
		// If maxLength > 1, then (maxLength-1) is >= 1, so Intn is safe.
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
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		tree := NewZFastTrie[bool](false)
		groundTruth := make(map[string]bool)
		insertedKeys := make([]string, 0)

		numOperations := 10000
		maxStrLength := 32
		prefixBias := 0.3

		for i := 0; i < numOperations; i++ {
			op := r.Intn(100)

			if op < 45 {
				s := generateString(r, insertedKeys, prefixBias, maxStrLength)
				if _, exists := groundTruth[s]; !exists {
					groundTruth[s] = true
					insertedKeys = append(insertedKeys, s)
					tree.Insert(s, true)

					if !tree.Contains(s) {
						fmt.Println(tree.String())
						fmt.Println(NewBitString(s))
						fmt.Println(!tree.Contains(s))
						t.Fatalf("Failed to find just-inserted key: %q", s)
					}
				}
			} else if op < 80 {
				if len(insertedKeys) == 0 {
					continue
				}
				idx := r.Intn(len(insertedKeys))
				s := insertedKeys[idx]

				delete(groundTruth, s)
				insertedKeys[idx] = insertedKeys[len(insertedKeys)-1]
				insertedKeys = insertedKeys[:len(insertedKeys)-1]

				tree.Erase(s)

				if tree.Contains(s) {
					t.Fatalf("Found just-deleted key: %q", s)
				}
			} else {
				s := generateString(r, insertedKeys, prefixBias, maxStrLength)
				expected, _ := groundTruth[s]
				actual := tree.Contains(s)

				if actual != expected {
					t.Fatalf("Contains mismatch for key %q. Expected: %v, Got: %v", s, expected, actual)
				}
			}
		}

		for key := range groundTruth {
			if !tree.Contains(key) {
				t.Fatalf("Final check failed: key %q in ground truth but not in trie", key)
			}
		}

		if int(tree.size) != len(groundTruth) {
			t.Fatalf("Final size mismatch. Expected: %d, Got: %d", len(groundTruth), tree.size)
		}
	}
}

func TestTrie_HeavyRandom_Prefixes(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	tree := NewZFastTrie[bool](false)
	groundTruth := make(map[string]bool)
	prefixGroundTruth := make(map[string]bool)
	insertedKeys := make([]string, 0)

	numOperations := 1000
	maxStrLength := 32
	prefixBias := 0.3

	for i := 0; i < numOperations; i++ {
		s := generateString(r, insertedKeys, prefixBias, maxStrLength)
		if _, exists := groundTruth[s]; exists {
			continue
		}

		groundTruth[s] = true
		insertedKeys = append(insertedKeys, s)
		tree.Insert(s, true)

		for j := 0; j <= len(s); j++ {
			prefixGroundTruth[s[:j]] = true
		}
	}

	for prefix := range prefixGroundTruth {
		if !tree.ContainsPrefix(prefix) {
			t.Fatalf("Prefix check failed: %q should be a prefix but was not found", prefix)
		}
	}

	for i := 0; i < numOperations/10; i++ {
		s := randString(r, maxStrLength)
		if _, isPrefix := prefixGroundTruth[s]; isPrefix {
			continue
		}

		if tree.ContainsPrefix(s) {
			t.Fatalf("Negative prefix check failed: %q should NOT be a prefix but was found", s)
		}
	}
}
