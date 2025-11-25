package succinct_bit_vector

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/hillbig/rsdic"
	trie "github.com/siongui/go-succinct-data-structure-trie"
)

// Benchmark для RSDic операций
func BenchmarkRSDic_PushBack(b *testing.B) {
	rs := rsdic.New()
	rand.Seed(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.PushBack(rand.Float32() < 0.5)
	}
}

func BenchmarkRSDic_Access(b *testing.B) {
	rs := rsdic.New()
	rand.Seed(42)

	// Подготовка данных
	size := 100000
	for i := 0; i < size; i++ {
		rs.PushBack(rand.Float32() < 0.3)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.Bit(uint64(i % size))
	}
}

func BenchmarkRSDic_Rank(b *testing.B) {
	rs := rsdic.New()
	rand.Seed(42)

	size := 100_000
	for i := 0; i < size; i++ {
		rs.PushBack(rand.Float32() < 0.3)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.Rank(uint64(i%size), true)
	}
}

func BenchmarkRSDic_Select(b *testing.B) {
	rs := rsdic.New()
	rand.Seed(42)

	// Подготовка данных
	size := 100000
	for i := 0; i < size; i++ {
		rs.PushBack(rand.Float32() < 0.3)
	}

	totalOnes := rs.Rank(rs.Num(), true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if totalOnes > 0 {
			rs.Select(uint64((i%int(totalOnes))+1), true)
		}
	}
}

// Benchmark различных размеров для RSDic
func BenchmarkRSDic_Rank_1K(b *testing.B)   { benchmarkRSdicRank(b, 1000) }
func BenchmarkRSDic_Rank_10K(b *testing.B)  { benchmarkRSdicRank(b, 10000) }
func BenchmarkRSDic_Rank_100K(b *testing.B) { benchmarkRSdicRank(b, 100000) }
func BenchmarkRSDic_Rank_1M(b *testing.B)   { benchmarkRSdicRank(b, 1000000) }

// Benchmark select операций для разных размеров
func BenchmarkRSDic_Select_1K(b *testing.B)   { benchmarkRSdicSelect(b, 1000) }
func BenchmarkRSDic_Select_10K(b *testing.B)  { benchmarkRSdicSelect(b, 10000) }
func BenchmarkRSDic_Select_100K(b *testing.B) { benchmarkRSdicSelect(b, 100000) }
func BenchmarkRSDic_Select_1M(b *testing.B)   { benchmarkRSdicSelect(b, 1000000) }

func benchmarkRSdicRank(b *testing.B, size int) {
	rs := rsdic.New()
	rand.Seed(42)

	// Подготовка данных
	for i := 0; i < size; i++ {
		rs.PushBack(rand.Float32() < 0.3)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.Rank(uint64(i%size), true)
	}
}

func benchmarkRSdicSelect(b *testing.B, size int) {
	rs := rsdic.New()
	rand.Seed(42)

	// Подготовка данных
	for i := 0; i < size; i++ {
		rs.PushBack(rand.Float32() < 0.3)
	}

	totalOnes := rs.Rank(rs.Num(), true)
	if totalOnes == 0 {
		b.Skip("No ones found in the data")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.Select(uint64((i%int(totalOnes))+1), true)
	}
}

func benchmarkTrieInsert(b *testing.B, wordCount int) {
	words := generateBenchmarkWords(wordCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := &trie.Trie{}
		t.Init()

		for j := 0; j < wordCount && j < len(words); j++ {
			t.Insert(words[j])
		}
	}
}

// Сравнительные benchmarks
func BenchmarkRSDic_vs_NaiveRank(b *testing.B) {
	size := 100_000
	density := 0.3

	// Подготовка RSDic
	rs := rsdic.New()
	naiveBits := make([]bool, size)
	rand.Seed(42)

	for i := 0; i < size; i++ {
		bit := rand.Float32() < float32(density)
		rs.PushBack(bit)
		naiveBits[i] = bit
	}

	b.Run("RSDic", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rs.Rank(uint64(i%size), true)
		}
	})

	b.Run("Naive", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pos := i % size
			count := 0
			for j := 0; j < pos; j++ {
				if naiveBits[j] {
					count++
				}
			}
		}
	})
}

// Benchmark памяти
func BenchmarkRSDic_Memory(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			rs := rsdic.New()
			rand.Seed(42)

			b.ResetTimer()
			for i := 0; i < size; i++ {
				rs.PushBack(rand.Float32() < 0.3)
			}

			// Принудительная сборка мусора для более точных измерений
			runtime.GC()

			// Выполняем несколько операций для активации структуры
			for i := 0; i < b.N; i++ {
				rs.Rank(uint64(i%size), true)
			}
		})
	}
}

// Утилитарные функции
func generateBenchmarkWords(count int) []string {
	rand.Seed(time.Now().UnixNano())

	prefixes := []string{
		"test", "bench", "word", "data", "algo", "struct", "node", "tree",
		"bit", "rank", "select", "trie", "hash", "map", "set", "list",
	}

	suffixes := []string{
		"ing", "ed", "er", "ly", "tion", "ness", "ment", "ful",
		"able", "ible", "al", "ary", "ic", "ous", "ive", "less",
	}

	words := make([]string, count)

	for i := 0; i < count; i++ {
		prefix := prefixes[rand.Intn(len(prefixes))]
		suffix := suffixes[rand.Intn(len(suffixes))]
		number := rand.Intn(1_000_000)

		word := fmt.Sprintf("%s%d%s", prefix, number, suffix)
		encodedString := base64.StdEncoding.EncodeToString([]byte(word))
		words[i] = encodedString
	}

	return words
}

func TestRSdicCorrectness(t *testing.T) {
	rs := rsdic.New()
	bits := []bool{true, false, true, true, false, false, true, false, true, false}

	for _, bit := range bits {
		rs.PushBack(bit)
	}

	for i, expectedBit := range bits {
		if rs.Bit(uint64(i)) != expectedBit {
			t.Errorf("Access(%d) = %t, ожидалось %t", i, rs.Bit(uint64(i)), expectedBit)
		}
	}

	// reminder: rank(i) = number of ones in range [0, i).
	expectedRanks := []int{0, 1, 1, 2, 3, 3, 3, 4, 4, 5, 5}
	for i, expectedRank := range expectedRanks {
		rank := rs.Rank(uint64(i), true)
		if int(rank) != expectedRank {
			t.Errorf("Rank1(%d) = %d, ожидалось %d", i, rank, expectedRank)
		}
	}

	// reminder: select(i) = position of i-th 1
	expectedSelects := []int{0, 2, 3, 6, 8 /* size of array instead of -1 */, 10, 10}
	for k, expectedPos := range expectedSelects {
		pos := rs.Select(uint64(k), true)
		if int(pos) != expectedPos {
			t.Errorf("Select1(%d) = %d, ожидалось %d", k+1, pos, expectedPos)
		}
	}
}
