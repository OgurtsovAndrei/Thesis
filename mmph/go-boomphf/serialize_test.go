package boomphf

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func generateRandomKeys(count int) []uint64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	keys := make([]uint64, 0, count)
	keyMap := make(map[uint64]struct{}, count)

	for len(keys) < count {
		newKey := r.Uint64()
		if _, exists := keyMap[newKey]; !exists && newKey != 0 {
			keyMap[newKey] = struct{}{}
			keys = append(keys, newKey)
		}
	}
	return keys
}

func checkDataEquality(t *testing.B, h1, h2 *H, name string) {
	h1B := h1.b
	h2B := h2.b

	require.Equal(t, len(h1B), len(h2B), "%s: h.b level count mismatch", name)

	for i := 0; i < len(h1B); i++ {
		bv1 := h1B[i]
		bv2 := h2B[i]
		require.Equal(t, len(bv1), len(bv2), "%s: h.b[%d] length mismatch", name, i)
		for j := 0; j < len(bv1); j++ {
			require.Equal(t, bv1[j], bv2[j], "%s: h.b[%d][%d] data mismatch", name, i, j)
		}
	}

	h1Ranks := h1.ranks
	h2Ranks := h2.ranks

	require.Equal(t, len(h1Ranks), len(h2Ranks), "%s: h.ranks level count mismatch", name)

	for i := 0; i < len(h1Ranks); i++ {
		r1 := h1Ranks[i]
		r2 := h2Ranks[i]
		require.Equal(t, len(r1), len(r2), "%s: h.ranks[%d] length mismatch", name, i)
		for j := 0; j < len(r1); j++ {
			require.Equal(t, r1[j], r2[j], "%s: h.ranks[%d][%d] data mismatch", name, i, j)
		}
	}
}

func BenchmarkSerializationIntegrity(b *testing.B) {
	for _, size := range benchKeyCounts {
		keys := generateRandomKeys(size)

		for _, gamma := range testGammas {
			name := fmt.Sprintf("Gamma_%.1f_Size_%d", gamma, size)
			b.Run(name, func(b *testing.B) {
				count := float64(size)

				hOriginal := New(gamma, keys)
				dataSizeInBytes := hOriginal.Size()
				require.NotNil(b, hOriginal, "Failed to create original PHF structure")

				{
					serializedData, err := hOriginal.Serialize()
					require.NoError(b, err, "Serialization failed")

					var hDeserialized H
					err = Deserialize(serializedData, &hDeserialized)
					require.NoError(b, err, "Deserialization failed")
					checkDataEquality(b, hOriginal, &hDeserialized, name)

					b.ReportMetric(float64(dataSizeInBytes)*8/count, "bits/key_in_mem")
					b.ReportMetric(float64(dataSizeInBytes), "bytes_in_mem")
					b.ReportMetric(float64(len(serializedData))*8/count, "bits/key_serialized")
					b.ReportMetric(float64(len(serializedData)), "bytes_serialized")
				}

				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					serializedData, err := hOriginal.Serialize()
					require.NoError(b, err, "Serialization failed")
					var hDeserialized H
					err = Deserialize(serializedData, &hDeserialized)
					require.NoError(b, err, "Deserialization failed")
				}
			})
		}
	}
}
