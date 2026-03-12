package are_optimized

import (
	"Thesis/bits"
	"fmt"
	"testing"
)

func TestOptimizedARE_NormalizationAndTruncation(t *testing.T) {
	// Создаем ключи с огромным смещением (базой)
	// База: 2^100
	base := bits.NewBitString(128)
	base.AppendBit(true) // Это не совсем 2^100, но для теста сойдет - установим какой-то бит высоко
	
	// Набор ключей: base + 1000, base + 2000, base + 3000...
	n := 100
	keys := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		offset := bits.NewFromUint64WithLength(uint64(i*1000), 128)
		keys[i] = base.Add(offset)
	}

	rangeLen := uint64(500)
	epsilon := 0.01
	truncateBits := uint32(5) // Отрезаем нижние 5 бит (~32 единицы)

	filter, err := NewOptimizedARE(keys, rangeLen, epsilon, truncateBits)
	if err != nil {
		t.Fatalf("Failed to create filter: %v", err)
	}

	fmt.Printf("\n--- Optimized ARE Test ---\n")
	fmt.Printf("Keys: %d, RangeLen: %d, Truncate: %d bits\n", n, rangeLen, truncateBits)
	fmt.Printf("SODA K: %d bits\n", filter.K)
	fmt.Printf("Total Size: %d bits (%.2f bits/key)\n", filter.SizeInBits(), float64(filter.SizeInBits())/float64(n))

	// Тест 1: Пустой диапазон между ключами
	// Ключи на 1000, 2000... 
	// Проверяем [base+1200, base+1500]
	a1 := base.Add(bits.NewFromUint64WithLength(1200, 128))
	b1 := base.Add(bits.NewFromUint64WithLength(1500, 128))
	res1 := filter.IsEmpty(a1, b1)
	fmt.Printf("Empty Range [1200, 1500]: IsEmpty = %v\n", res1)

	// Тест 2: Диапазон с ключом
	// Проверяем [base+900, base+1100] (содержит base+1000)
	a2 := base.Add(bits.NewFromUint64WithLength(900, 128))
	b2 := base.Add(bits.NewFromUint64WithLength(1100, 128))
	res2 := filter.IsEmpty(a2, b2)
	fmt.Printf("Range with Key [900, 1100]: IsEmpty = %v\n", res2)

	if res2 != false {
		t.Errorf("False Negative!")
	}
}
