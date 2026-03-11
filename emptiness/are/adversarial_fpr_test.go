package are

import (
	"Thesis/bits"
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func TestARE_AdversarialFPR_Collision(t *testing.T) {
	n := 10000
	epsilon := 0.01 // Целевая точность 1%
	
	// 1. Генерируем ключи так, чтобы оставить место для "плохих" запросов
	rng := rand.New(rand.NewSource(42))
	keys := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		// Генерируем ключи с 0 на конце, чтобы x+1 был валидным ключом
		val := (rng.Uint64() >> 8) << 8 
		keys[i] = bits.NewFromUint64(val)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Compare(keys[j]) < 0 })

	filter, _ := NewApproximateRangeEmptiness(keys, epsilon)
	K := filter.K // Узнаем, сколько бит префикса используется
	
	fpCount := 0
	trials := n

	// 2. Атака: создаем запросы, которые "чуть-чуть" не дотягивают до реальных ключей
	for i := 0; i < trials; i++ {
		x := keys[i]
		
		// Создаем запрос [x+1, x+2]
		// x+1 гарантированно нет в S (так как мы обнулили последние 8 бит)
		// Но с высокой вероятностью f(x+1) == f(x)
		a := x.Successor()
		b := a.Successor()
		
		if filter.IsEmpty(a, b) == false {
			// Проверяем, что это действительно False Positive
			// (в нашем случае мы знаем, что диапазон пуст)
			fpCount++
		}
	}

	observedFPR := float64(fpCount) / float64(trials)
	fmt.Printf("\n--- Adversarial FPR Report ---\n")
	fmt.Printf("N: %d, Epsilon: %f, K-bits: %d\n", n, epsilon, K)
	fmt.Printf("Trials: %d, False Positives: %d\n", trials, fpCount)
	fmt.Printf("Observed FPR: %f (Target: %f)\n", observedFPR, epsilon)
	
	// В состязательном тесте FPR может быть выше epsilon, 
	// так как epsilon гарантируется для СЛУЧАЙНЫХ запросов.
	// Но если он равен 1.0 — значит у нас дыра в логике.
	if observedFPR > 0.5 {
		t.Errorf("Adversarial FPR is too high: %f", observedFPR)
	}
}
