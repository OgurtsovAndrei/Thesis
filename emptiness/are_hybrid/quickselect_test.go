package are_hybrid

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

func TestQuickselect_Property(t *testing.T) {
	t.Parallel()

	for iter := 0; iter < 10_000; iter++ {
		iter := iter
		t.Run(fmt.Sprintf("Iter%d", iter), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(iter)))

			n := 2 + rng.Intn(500)
			k := rng.Intn(n)

			a := make([]uint64, n)
			for i := range a {
				a[i] = rng.Uint64()
			}

			// Reference: sort a copy and pick k-th
			sorted := make([]uint64, n)
			copy(sorted, a)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
			want := sorted[k]

			// quickselect on original
			got := quickselect(a, k)

			if got != want {
				t.Errorf("n=%d k=%d: quickselect=%d, sort=%d", n, k, got, want)
			}
		})
	}
}

func TestQuickselect_Duplicates(t *testing.T) {
	t.Parallel()

	for iter := 0; iter < 5_000; iter++ {
		iter := iter
		t.Run(fmt.Sprintf("Iter%d", iter), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(iter + 100_000)))

			n := 2 + rng.Intn(500)
			k := rng.Intn(n)

			// Few distinct values to stress duplicate handling
			nDistinct := 1 + rng.Intn(5)
			vals := make([]uint64, nDistinct)
			for i := range vals {
				vals[i] = rng.Uint64()
			}

			a := make([]uint64, n)
			for i := range a {
				a[i] = vals[rng.Intn(nDistinct)]
			}

			sorted := make([]uint64, n)
			copy(sorted, a)
			sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
			want := sorted[k]

			got := quickselect(a, k)

			if got != want {
				t.Errorf("n=%d k=%d nDistinct=%d: quickselect=%d, sort=%d", n, k, nDistinct, got, want)
			}
		})
	}
}

func TestQuickselect_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("TwoElements", func(t *testing.T) {
		a := []uint64{42, 7}
		if got := quickselect(a, 0); got != 7 {
			t.Errorf("k=0: got %d, want 7", got)
		}
		a = []uint64{42, 7}
		if got := quickselect(a, 1); got != 42 {
			t.Errorf("k=1: got %d, want 42", got)
		}
	})

	t.Run("AllSame", func(t *testing.T) {
		a := make([]uint64, 100)
		for i := range a {
			a[i] = 999
		}
		if got := quickselect(a, 50); got != 999 {
			t.Errorf("got %d, want 999", got)
		}
	})

	t.Run("Sorted", func(t *testing.T) {
		a := make([]uint64, 100)
		for i := range a {
			a[i] = uint64(i)
		}
		if got := quickselect(a, 73); got != 73 {
			t.Errorf("got %d, want 73", got)
		}
	})

	t.Run("ReverseSorted", func(t *testing.T) {
		a := make([]uint64, 100)
		for i := range a {
			a[i] = uint64(99 - i)
		}
		if got := quickselect(a, 0); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})
}
