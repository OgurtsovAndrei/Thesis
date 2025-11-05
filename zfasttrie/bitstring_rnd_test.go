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

func TestTrie_HeavyRandom_BitString_Get(t *testing.T) {
	for test_id := range 100 {
		fmt.Println("TestTrie_HeavyRandom_BitString_Get iteration:", test_id)
		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))

		emptyValue := -1
		tree := NewZFastTrie[int](emptyValue)
		groundTruth := make(map[string]int)

		numOperations := 100_000

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Test panicked (Seed: %d): %v", seed, r)
			}
		}()

		for i := 0; i < numOperations; i++ {
			op := r.Intn(100)

			if op < 50 {
				k := r.Uint64()
				s := NewFromUint64(k).data
				v := r.Intn(1000000)

				if _, exists := groundTruth[s]; !exists {
					groundTruth[s] = v
					tree.Insert(s, v)
				}
			} else {
				k := r.Uint64()
				s := NewFromUint64(k).data

				expected, ok := groundTruth[s]
				if !ok {
					expected = emptyValue
				}

				actual := tree.GetBitString(NewBitString(s))

				if actual != expected {
					t.Fatalf("Seed %d: GetBitString mismatch for key (uint64: %d). Expected: %v, Got: %v", seed, k, expected, actual)
				}
			}
		}

		for key, expectedValue := range groundTruth {
			actualValue := tree.GetBitString(NewBitString(key))
			if actualValue != expectedValue {
				t.Fatalf("Seed %d: Final check GetBitString mismatch for key %q. Expected: %v, Got: %v", seed, key, expectedValue, actualValue)
			}
		}
	}
}

func TestTrie_BitString_Exhaustive8Bit(t *testing.T) {
	numIterations := 1000
	for iter := 0; iter < numIterations; iter++ {
		fmt.Println("TestTrie_BitString_Exhaustive8Bit iteration:", iter)
		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))
		tree := NewZFastTrie[int](-1)

		allKeys := make([]string, 256)
		for i := 0; i < 256; i++ {
			allKeys[i] = string([]byte{byte(i)})
		}

		r.Shuffle(len(allKeys), func(i, j int) { allKeys[i], allKeys[j] = allKeys[j], allKeys[i] })

		keysToInsert := make(map[string]bool)
		for i := 0; i < 128; i++ {
			keysToInsert[allKeys[i]] = true
		}

		insertedSoFar := make(map[string]int)
		for _, key := range allKeys {
			insertedSoFar[key] = -1
		}

		for opNum, key := range allKeys {
			shouldInsert := keysToInsert[key]

			if shouldInsert {
				value := int(key[0])
				tree.Insert(key, value)
				insertedSoFar[key] = value
			}

			for _, checkKey := range allKeys {
				shouldBePresent := insertedSoFar[checkKey]
				actual := tree.GetBitString(NewBitString(checkKey))

				if actual != shouldBePresent {
					t.Fatalf("Seed %d, Iter %d, Op %d (Key %v): Mismatch for checkKey %v. Expected: %v, Got: %v",
						seed, iter, opNum, []byte(key), []byte(checkKey), shouldBePresent, actual)
				}
			}
		}

		if int(tree.size) != len(keysToInsert) {
			t.Fatalf("Seed %d, Iter %d: Final size mismatch. Expected: %d, Got: %d",
				seed, iter, len(keysToInsert), tree.size)
		}
	}
}
