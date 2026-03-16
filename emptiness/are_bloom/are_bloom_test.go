package are_bloom

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"Thesis/testutils"
)

func TestNoFalseNegatives(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	keys := make([]uint64, 1000)
	for i := range keys {
		keys[i] = rng.Uint64() >> 16 // leave room for ranges
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	f, err := NewBloomARE(keys, 10, 0.01)
	require.NoError(t, err)

	for _, k := range keys {
		require.False(t, f.IsEmpty(k, k), "key %d must not be reported empty", k)
		if k >= 5 {
			require.False(t, f.IsEmpty(k-5, k+5), "range around key %d must not be empty", k)
		}
	}
}

func TestFPRSanity(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	n := 10000
	keys := make([]uint64, n)
	seen := make(map[uint64]bool)
	for i := 0; i < n; i++ {
		for {
			v := rng.Uint64()
			if !seen[v] {
				seen[v] = true
				keys[i] = v
				break
			}
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	eps := 0.01
	rangeLen := uint64(100)
	f, err := NewBloomARE(keys, rangeLen, eps)
	require.NoError(t, err)

	fp, totalEmpty := 0, 0
	qrng := rand.New(rand.NewSource(123))
	for i := 0; i < 100_000; i++ {
		a := qrng.Uint64()
		b := a + rangeLen - 1
		if b < a {
			continue
		}
		// Check ground truth: range is truly empty if no key falls in [a,b]
		idx := sort.Search(len(keys), func(j int) bool { return keys[j] >= a })
		if idx < len(keys) && keys[idx] <= b {
			continue // range is not actually empty
		}
		totalEmpty++
		if !f.IsEmpty(a, b) {
			fp++
		}
	}

	if totalEmpty == 0 {
		t.Skip("no empty ranges found")
	}
	fpr := float64(fp) / float64(totalEmpty)
	t.Logf("FPR: %.4f (target ε=%.4f, %d empty ranges)", fpr, eps, totalEmpty)
	require.Less(t, fpr, eps*3, "FPR should be within ~3x of target ε")
}

func TestPointQuery(t *testing.T) {
	keys := []uint64{100, 200, 300}
	f, err := NewBloomARE(keys, 1, 0.01)
	require.NoError(t, err)

	// L=1: point FPR = ε, so IsEmpty checks a single point
	require.False(t, f.IsEmpty(100, 100))
	require.False(t, f.IsEmpty(200, 200))
	require.False(t, f.IsEmpty(300, 300))
}

func TestSizeInBits(t *testing.T) {
	keys := make([]uint64, 1000)
	for i := range keys {
		keys[i] = uint64(i)
	}
	f, err := NewBloomARE(keys, 100, 0.01)
	require.NoError(t, err)
	require.Greater(t, f.SizeInBits(), uint64(0))
}

func TestBloomARE_NoFN_MultiSeed_Uniform(t *testing.T) {
	t.Parallel()
	for i := 0; i < 100; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 3000)))
			n := 500 + rng.Intn(1501)
			keys := make([]uint64, n)
			seen := make(map[uint64]bool, n)
			for j := 0; j < n; {
				v := rng.Uint64()
				if !seen[v] {
					seen[v] = true
					keys[j] = v
					j++
				}
			}
			sort.Slice(keys, func(a, b int) bool { return keys[a] < keys[b] })

			f, err := NewBloomARE(keys, 10, 0.01)
			require.NoError(t, err)

			for _, k := range keys {
				require.False(t, f.IsEmpty(k, k), "Iter%d: key %d reported empty", i, k)
				if k >= 5 {
					require.False(t, f.IsEmpty(k-5, k+5), "Iter%d: range around key %d reported empty", i, k)
				}
			}
		})
	}
}

func TestBloomARE_NoFN_MultiSeed_Clustered(t *testing.T) {
	t.Parallel()
	for i := 0; i < 100; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 4000)))
			n := 500 + rng.Intn(1501)
			keys, _ := testutils.GenerateClusterDistribution(n, 5, 0.15, rng)

			f, err := NewBloomARE(keys, 10, 0.01)
			require.NoError(t, err)

			for _, k := range keys {
				require.False(t, f.IsEmpty(k, k), "Iter%d: key %d reported empty", i, k)
				if k >= 5 {
					require.False(t, f.IsEmpty(k-5, k+5), "Iter%d: range around key %d reported empty", i, k)
				}
			}
		})
	}
}
