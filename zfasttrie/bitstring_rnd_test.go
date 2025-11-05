package zfasttrie

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestTrie_HeavyRandom_BitString(t *testing.T) {
	for test_id := range 100 {
		fmt.Println("TestTrie_HeavyRandom_BitString iteration:", test_id)
		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))
		tree := NewZFastTrie[bool](false)
		groundTruth := make(map[string]bool)
		insertedKeys := make([]string, 0)

		numOperations := 10_000

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Test panicked (Seed: %d): %v", seed, r)
			}
		}()

		for i := 0; i < numOperations; i++ {
			op := r.Intn(100)

			if op < 45 {
				k := r.Uint64()
				s := NewFromUint64(k).data

				if _, exists := groundTruth[s]; !exists {
					groundTruth[s] = true
					insertedKeys = append(insertedKeys, s)
					tree.Insert(s, true)

					if !tree.Contains(s) {
						t.Fatalf("Seed %d: Failed to find just-inserted key (uint64: %d)", seed, k)
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
					t.Fatalf("Seed %d: Found just-deleted key", seed)
				}
			} else {
				k := r.Uint64()
				s := NewFromUint64(k).data
				expected, _ := groundTruth[s]
				actual := tree.Contains(s)

				if actual != expected {
					t.Fatalf("Seed %d: Contains mismatch for key. Expected: %v, Got: %v", seed, expected, actual)
				}
			}
		}

		for key := range groundTruth {
			if !tree.Contains(key) {
				t.Fatalf("Seed %d: Final check failed: key %q in ground truth but not in trie", seed, key)
			}
		}

		if int(tree.size) != len(groundTruth) {
			t.Fatalf("Seed %d: Final size mismatch. Expected: %d, Got: %d", seed, len(groundTruth), tree.size)
		}
	}
}
