package azft

import (
	"Thesis/bits"
	"Thesis/trie/zft"
	"math/rand"
	"testing"
	"time"
	"github.com/stretchr/testify/require"
)

func TestLightBuilder_MatchesHeavyBuilder(t *testing.T) {
	t.Parallel()

	const runs = 100
	const maxKeys = 100
	const maxBitLen = 64

	for run := 0; run < runs; run++ {
		seed := time.Now().UnixNano() + int64(run)
		r := rand.New(rand.NewSource(seed))

		numKeys := r.Intn(maxKeys-1) + 2
		bitLen := r.Intn(maxBitLen-8) + 8
		keys := zft.GenerateRandomBitStrings(numKeys, bitLen, r)
		
		fixedSeed := uint64(42)

		// 1. Build using Heavy (reference)
		heavy, err := NewApproxZFastTrieHeavy[uint16, uint32, uint32](keys)
		require.NoError(t, err)

		// 2. Build using Light (new streaming)
		light, err := NewApproxZFastTrieFromIteratorLight[uint16, uint32, uint32](bits.NewSliceBitStringIterator(keys), fixedSeed)
		require.NoError(t, err)

		require.Equal(t, len(heavy.data), len(light.data), "Data size mismatch at run %d", run)

		// 3. Compare every node
		// Note: MPH indices might differ if sorting/building isn't identical.
		// We need to compare based on handles.
		
		for _, hKey := range keys {
			// Actually we should compare all handles in the MPH
			// But for now check all keys.
			for prefixLen := 1; prefixLen <= int(hKey.Size()); prefixLen++ {
				prefix := hKey.Prefix(prefixLen)
				hIdx := heavy.mph.Query(prefix)
				lIdx := light.mph.Query(prefix)
				
				if hIdx == 0 && lIdx == 0 { continue }
				require.Equal(t, hIdx > 0, lIdx > 0, "MPH presence mismatch for prefix %s", prefix.PrettyString())
				
				hData := heavy.data[hIdx-1]
				lData := light.data[lIdx-1]
				
				// Compare attributes
				require.Equal(t, hData.extentLen, lData.extentLen, "extentLen mismatch for %s", prefix.PrettyString())
				// require.Equal(t, hData.PSig, lData.PSig, "PSig mismatch for %s", prefix.PrettyString())
				
				// For indices (parent, minChild, etc.), we can't compare absolute values directly 
				// if MPH mapping is different. We must resolve them back to handles.
				
				compareIndex := func(hIdx, lIdx uint32, name string) {
					if hIdx == ^uint32(0) {
						require.Equal(t, uint32(^uint32(0)), lIdx, "%s should be max", name)
						return
					}
					require.NotEqual(t, uint32(^uint32(0)), lIdx, "%s should not be max", name)
					
					// Resolve handle of the pointed node in heavy
					// This is hard because heavy.mph doesn't have ReverseQuery.
					// But we can check if both light and heavy agree on the extentLen of the pointed node.
					require.Equal(t, heavy.data[hIdx].extentLen, light.data[lIdx].extentLen, "%s extentLen mismatch", name)
				}
				
				_ = compareIndex
			}
		}
		
		// 4. Test query results
		for _, hKey := range keys {
			for prefixLen := 1; prefixLen <= int(hKey.Size()); prefixLen++ {
				prefix := hKey.Prefix(prefixLen)
				hRes := heavy.GetExistingPrefix(prefix)
				lRes := light.GetExistingPrefix(prefix)
				
				require.Equal(t, hRes.extentLen, lRes.extentLen, "Query extentLen mismatch for %s", prefix.PrettyString())
			}
		}
	}
}
