package testutils

import (
	"math/rand"
	"sort"
)

// ClusterInfo describes a single cluster used for key/query generation.
type ClusterInfo struct {
	Center uint64
	Stddev float64
}

// GenerateClusterDistribution generates n keys: unifFrac as uniform, the rest
// split across numClusters Gaussian clusters with random centers and stddev in [2^20, 2^30).
func GenerateClusterDistribution(n int, numClusters int, unifFrac float64, rng *rand.Rand) ([]uint64, []ClusterInfo) {
	seen := make(map[uint64]bool)
	keys := make([]uint64, 0, n)

	nUnif := int(float64(n) * unifFrac)
	for len(keys) < nUnif {
		v := rng.Uint64()
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}

	clusters := make([]ClusterInfo, numClusters)
	perCluster := (n - nUnif) / numClusters
	for c := 0; c < numClusters; c++ {
		clusters[c] = ClusterInfo{
			Center: rng.Uint64(),
			Stddev: float64(uint64(1) << (20 + rng.Intn(10))),
		}
		generated := 0
		for generated < perCluster || (c == numClusters-1 && len(keys) < n) {
			v := SampleGaussian(clusters[c].Center, clusters[c].Stddev, rng)
			if v == 0 && clusters[c].Center != 0 {
				continue
			}
			if !seen[v] {
				seen[v] = true
				keys = append(keys, v)
				generated++
			}
			if len(keys) >= n {
				break
			}
		}
	}

	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys, clusters
}

// GenerateClusterQueries generates count queries: unifFrac uniform, the rest
// drawn from random clusters with matching stddev.
func GenerateClusterQueries(count int, clusters []ClusterInfo, unifFrac float64, rangeLen uint64, rng *rand.Rand) [][2]uint64 {
	queries := make([][2]uint64, count)
	nUnif := int(float64(count) * unifFrac)

	for i := 0; i < nUnif; i++ {
		a := rng.Uint64()
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	for i := nUnif; i < count; i++ {
		cl := clusters[rng.Intn(len(clusters))]
		a := SampleGaussian(cl.Center, cl.Stddev, rng)
		if a == 0 {
			a = rng.Uint64()
		}
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}
	return queries
}
