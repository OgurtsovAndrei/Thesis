package succinct_bit_vector

import (
	"math/rand"
	"testing"

	bits "github.com/siongui/go-succinct-data-structure-trie/reference"
)

// Benchmark для Succinct BitString операций
func BenchmarkBitString_Get(b *testing.B) {
	// Создаем тестовые данные
	bs := &bits.BitString{}
	bs.Init("YWJhY2FiYWJhY2FiYWJhY2FiYQ==") // base64 encoded data

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs.Get(uint(i%100), 1) // Get single bit at position
	}
}

func BenchmarkBitString_Count(b *testing.B) {
	// Создаем тестовые данные
	bs := &bits.BitString{}
	bs.Init("YWJhY2FiYWJhY2FiYWJhY2FiYQ==")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs.Count(uint(i%50), 10) // Count bits in 10-bit window
	}
}

func BenchmarkBitString_Rank(b *testing.B) {
	// Создаем тестовые данные
	bs := &bits.BitString{}
	bs.Init("YWJhY2FiYWJhY2FiYWJhY2FiYQ==")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs.Rank(uint(i % 100))
	}
}

// Benchmark различных размеров для BitString
func BenchmarkBitString_Get_1K(b *testing.B)   { benchmarkBitStringGet(b, 1000) }
func BenchmarkBitString_Get_10K(b *testing.B)  { benchmarkBitStringGet(b, 10_000) }
func BenchmarkBitString_Get_100K(b *testing.B) { benchmarkBitStringGet(b, 100_000) }
func BenchmarkBitString_Get_1M(b *testing.B)   { benchmarkBitStringGet(b, 1_000_000) }

func benchmarkBitStringGet(b *testing.B, size int) {
	// Генерируем случайную BASE64 строку
	data := generateRandomBase64Data(size)
	bs := &bits.BitString{}
	bs.Init(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs.Get(uint(i%size), 1)
	}
}

func generateRandomBase64Data(approxBits int) string {
	// Каждый символ base64 представляет 6 бит
	charsNeeded := (approxBits + 5) / 6
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	result := make([]byte, charsNeeded)
	for i := 0; i < charsNeeded; i++ {
		result[i] = base64Chars[rand.Intn(len(base64Chars))]
	}
	return string(result)
}

func TestBitStringCorrectness(t *testing.T) {
	bs := &bits.BitString{}
	bs.Init("YWJhY2FiYQ==") // "abacaba" in base64

	// Проверяем что данные были инициализированы
	if bs.GetData() == "" {
		t.Error("BitString data should not be empty")
	}

	// Тестируем операцию Get
	bit := bs.Get(0, 1)
	if bit > 1 {
		t.Errorf("Bit should be 0 or 1, got %d", bit)
	}

	// Тестируем операцию Count
	count := bs.Count(0, 8)
	if count > 8 {
		t.Errorf("Count of 8 bits should be <= 8, got %d", count)
	}
}

// Размеры для RankDirectory
func BenchmarkRankDirectory_Rank_1K(b *testing.B)   { benchmarkRankDirectoryRank(b, 1000) }
func BenchmarkRankDirectory_Rank_10K(b *testing.B)  { benchmarkRankDirectoryRank(b, 10_000) }
func BenchmarkRankDirectory_Rank_100K(b *testing.B) { benchmarkRankDirectoryRank(b, 100_000) }
func BenchmarkRankDirectory_Rank_1M(b *testing.B)   { benchmarkRankDirectoryRank(b, 1_000_000) }

func BenchmarkRankDirectory_Select_1K(b *testing.B)   { benchmarkRankDirectorySelect(b, 1000) }
func BenchmarkRankDirectory_Select_10K(b *testing.B)  { benchmarkRankDirectorySelect(b, 10_000) }
func BenchmarkRankDirectory_Select_100K(b *testing.B) { benchmarkRankDirectorySelect(b, 100_000) }
func BenchmarkRankDirectory_Select_1M(b *testing.B)   { benchmarkRankDirectorySelect(b, 1_000_000) }

func benchmarkRankDirectoryRank(b *testing.B, approxBits int) {
	data := generateRandomBase64Data(approxBits)
	numBits := uint(len(data) * 6)

	rd := bits.CreateRankDirectory(data, numBits, 32*32, 32)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rd.Rank(1, uint(i%int(numBits)))
	}
}

func benchmarkRankDirectorySelect(b *testing.B, approxBits int) {
	data := generateRandomBase64Data(approxBits)
	numBits := uint(len(data) * 6)

	rd := bits.CreateRankDirectory(data, numBits, 32*32, 32)
	totalOnes := rd.Rank(1, numBits-1)

	if totalOnes == 0 {
		b.Skip("No ones found in the data")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rd.Select(1, uint(i%int(totalOnes))+1)
	}
}
