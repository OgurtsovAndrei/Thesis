package testutils

import (
	"Thesis/bits"
	"math/rand"
	"sort"
)

// GenSpreadKeys generates N 64-bit keys where the low K bits are maximally
// spread (sequential 0..min(N,2^K)-1). After Prefix(K) truncation,
// the maximum possible number of distinct values is stored — worst case for ARE space.
func GenSpreadKeys(n int, minK uint32, seed int64) []bits.BitString {
	rng := rand.New(rand.NewSource(seed))
	result := make([]bits.BitString, n)

	maxDistinct := uint64(1) << minK
	for i := 0; i < n; i++ {
		// Low minK bits: sequential (mod 2^K) → maximize distinct prefixes
		// Upper bits: random noise to avoid trivial patterns
		upper := rng.Uint64() & ^(maxDistinct - 1)
		val := upper | (uint64(i) % maxDistinct)
		result[i] = bits.NewFromUint64(val)
	}

	sort.Sort(bitStringSorter(result))
	return result
}

// GenClusteredKeys generates N keys grouped into numClusters clusters.
// Keys within a cluster share the same low-bit prefix, causing many
// collisions after truncation — the best case for ARE (fewer stored values).
func GenClusteredKeys(n, numClusters int, seed int64) []bits.BitString {
	rng := rand.New(rand.NewSource(seed))
	keysPerCluster := n / numClusters
	extra := n - keysPerCluster*numClusters

	unique := make(map[uint64]bool, n)
	result := make([]bits.BitString, 0, n)

	for c := 0; c < numClusters; c++ {
		// Cluster prefix in the low 20 bits (shared after truncation)
		clusterLow := uint64(rng.Intn(1 << 20))
		count := keysPerCluster
		if c < extra {
			count++
		}
		for i := 0; i < count; i++ {
			for {
				// Vary upper bits, keep low 20 bits as cluster prefix
				upper := rng.Uint64() & ^uint64((1<<20)-1)
				val := upper | clusterLow
				if !unique[val] {
					unique[val] = true
					result = append(result, bits.NewFromUint64(val))
					break
				}
			}
		}
	}

	sort.Sort(bitStringSorter(result))
	return result
}

// GenGapQueries generates queries that fall in gaps between adjacent keys.
// These are truly-empty intervals that stress the filter.
func GenGapQueries(keys []bits.BitString, numQueries int, maxWidth uint64, seed int64) [][2]bits.BitString {
	rng := rand.New(rand.NewSource(seed))
	queries := make([][2]bits.BitString, 0, numQueries)
	n := len(keys)
	if n < 2 {
		return queries
	}

	for attempts := 0; len(queries) < numQueries && attempts < numQueries*10; attempts++ {
		idx := rng.Intn(n - 1)
		aData := keys[idx].Data()
		bData := keys[idx+1].Data()
		aVal := leUint64(aData)
		bVal := leUint64(bData)

		if bVal <= aVal+2 {
			continue
		}
		gap := bVal - aVal

		// Pick a point inside the gap (handle large gaps safely)
		var offset uint64
		if gap-1 <= 1<<62 {
			offset = uint64(rng.Int63n(int64(gap - 1)))
		} else {
			offset = rng.Uint64() % (gap - 1)
		}
		lo := aVal + 1 + offset

		w := maxWidth
		if w > gap-2 {
			w = gap - 2
		}
		hi := lo
		if w > 1 {
			if w <= 1<<62 {
				hi = lo + uint64(rng.Int63n(int64(w)))
			} else {
				hi = lo + rng.Uint64()%w
			}
		}
		if hi >= bVal {
			hi = bVal - 1
		}
		if lo > hi {
			continue
		}

		queries = append(queries, [2]bits.BitString{
			bits.NewFromUint64(lo),
			bits.NewFromUint64(hi),
		})
	}
	return queries
}

func leUint64(data []byte) uint64 {
	var v uint64
	for i := 0; i < 8 && i < len(data); i++ {
		v |= uint64(data[i]) << (i * 8)
	}
	return v
}
