package are_soda_hash

import (
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	fastPathN         = 100_000
	fastPathRangeLen  = uint64(1024)
	fastPathEpsilon   = 0.01
	fastPathNumProbes = 500
)

func generateUniqueUint64Keys(n int, seed int64) []uint64 {
	rng := rand.New(rand.NewSource(seed))
	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)
	for len(keys) < n {
		v := rng.Uint64()
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func generateProbes(count int, rangeLen uint64, seed int64) [][2]uint64 {
	rng := rand.New(rand.NewSource(seed))
	probes := make([][2]uint64, count)
	for i := range probes {
		a := rng.Uint64()
		probes[i] = [2]uint64{a, a + rangeLen - 1}
	}
	return probes
}

func TestSodaARE_Uint64FastPath_Equivalence(t *testing.T) {
	keys := generateUniqueUint64Keys(fastPathN, 42)

	ref, err := NewSodaARE(keys, fastPathRangeLen, fastPathEpsilon)
	require.NoError(t, err)

	fast, err := NewSodaAREUint64(keys, fastPathRangeLen, fastPathEpsilon)
	require.NoError(t, err)

	require.Equal(t, ref.K, fast.K, "K should match between paths")
	require.Equal(t, ref.hashA, fast.hashA, "hashA seed should match between paths")
	require.Equal(t, ref.hashB, fast.hashB, "hashB seed should match between paths")

	probes := generateProbes(fastPathNumProbes, fastPathRangeLen, 1234)
	for _, q := range probes {
		require.Equal(t,
			ref.IsEmpty(q[0], q[1]),
			fast.IsEmpty(q[0], q[1]),
			"answers must agree on probe [%d, %d]", q[0], q[1],
		)
	}
}

func TestSodaARE_Uint64FastPath_InPlace_DoesNotChangeAnswers(t *testing.T) {
	keys := generateUniqueUint64Keys(fastPathN, 42)

	ref, err := NewSodaARE(keys, fastPathRangeLen, fastPathEpsilon)
	require.NoError(t, err)

	keysCopy := make([]uint64, len(keys))
	copy(keysCopy, keys)

	fast, err := NewSodaAREUint64InPlace(keysCopy, fastPathRangeLen, ref.K)
	require.NoError(t, err)

	require.Equal(t, ref.K, fast.K, "K should match between paths")
	require.Equal(t, ref.hashA, fast.hashA, "hashA seed should match between paths")
	require.Equal(t, ref.hashB, fast.hashB, "hashB seed should match between paths")

	probes := generateProbes(fastPathNumProbes, fastPathRangeLen, 1234)
	for _, q := range probes {
		require.Equal(t,
			ref.IsEmpty(q[0], q[1]),
			fast.IsEmpty(q[0], q[1]),
			"answers must agree on probe [%d, %d]", q[0], q[1],
		)
	}
}
