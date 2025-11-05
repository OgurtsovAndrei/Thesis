package zfasttrie

import (
	"math/rand"
	"testing"
)

func generateBitStringKeys(n int) []string {
	r := rand.New(rand.NewSource(42))
	keys := make([]string, n)
	set := make(map[string]struct{}, n)

	for i := 0; i < n; {
		s := NewFromUint64(r.Uint64()).data
		if _, ok := set[s]; !ok {
			set[s] = struct{}{}
			keys[i] = s
			i++
		}
	}
	return keys
}

func setupBitStringTrie(b *testing.B, n int) (*ZFastTrie[bool], []string) {
	b.Helper()
	b.StopTimer()
	keys := generateBitStringKeys(n)
	tree := NewZFastTrie[bool](false)
	for _, s := range keys {
		tree.Insert(s, true)
	}
	b.StartTimer()
	return tree, keys
}

func BenchmarkTrie_BitString_Insert(b *testing.B) {
	b.StopTimer()
	keys := generateBitStringKeys(b.N)
	tree := NewZFastTrie[bool](false)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		tree.Insert(keys[i], true)
	}
}

func BenchmarkTrie_BitString_Contains_Hit_100k(b *testing.B) {
	tree, keys := setupBitStringTrie(b, 100_000)
	mask := len(keys) - 1

	for i := 0; i < b.N; i++ {
		tree.Contains(keys[i&mask])
	}
}

func BenchmarkTrie_BitString_Contains_Miss_100k(b *testing.B) {
	tree, _ := setupBitStringTrie(b, 100_000)
	b.StopTimer()
	missKeys := generateBitStringKeys(b.N)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		tree.Contains(missKeys[i])
	}
}

func BenchmarkTrie_BitString_Erase_Hit_100k(b *testing.B) {
	b.StopTimer()

	keys := generateBitStringKeys(100_000 + b.N)
	tree := NewZFastTrie[bool](false)

	for i := 0; i < 100_000; i++ {
		tree.Insert(keys[i], true)
	}

	eraseKeys := keys[100_000:]
	for i := 0; i < b.N; i++ {
		tree.Insert(eraseKeys[i], true)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tree.Erase(eraseKeys[i])
	}
}

func BenchmarkTrie_BitString_Insert_Erase_100k(b *testing.B) {
	tree, _ := setupBitStringTrie(b, 100_000)
	b.StopTimer()
	keys := generateBitStringKeys(b.N)
	mask := len(keys) - 1
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		s := keys[i&mask]
		tree.Insert(s, true)
		tree.Erase(s)
	}
}
