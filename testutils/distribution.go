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
// split across numClusters Gaussian clusters with random centers. Each
// cluster's stddev is drawn from [2^20, 2^30) and then floored at clusterSize
// so the cluster's effective support (≈6σ) always exceeds the requested
// count — preventing rejection-loop pathologies at very large n where
// clusterSize can rival the small end of the σ range.
//
// A maxAttempts guard also protects against pathological centers (e.g. near
// numerical boundaries) collapsing draws to 0; if a cluster cannot be filled
// the function returns whatever uniques it gathered rather than spinning.
func GenerateClusterDistribution(n int, numClusters int, unifFrac float64, rng *rand.Rand) ([]uint64, []ClusterInfo) {
	seen := make(map[uint64]struct{}, n+n/8)
	keys := make([]uint64, 0, n)

	nUnif := int(float64(n) * unifFrac)
	for len(keys) < nUnif {
		v := rng.Uint64()
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			keys = append(keys, v)
		}
	}

	nClust := n - nUnif
	weights := make([]float64, numClusters)
	var wSum float64
	for i := range weights {
		weights[i] = rng.ExpFloat64()
		wSum += weights[i]
	}
	clusterSizes := make([]int, numClusters)
	assigned := 0
	for i := range clusterSizes {
		clusterSizes[i] = int(weights[i] / wSum * float64(nClust))
		assigned += clusterSizes[i]
	}
	clusterSizes[numClusters-1] += nClust - assigned

	clusters := make([]ClusterInfo, numClusters)
	for c := 0; c < numClusters; c++ {
		baseStddev := float64(uint64(1) << (20 + rng.Intn(10)))
		// Floor at clusterSize: ensures ±3σ effective support ≥ 6×count, so
		// the rejection loop below cannot exhaust unique values for any
		// cluster size.
		stddev := baseStddev
		if minS := float64(clusterSizes[c]); minS > stddev {
			stddev = minS
		}
		clusters[c] = ClusterInfo{
			Center: rng.Uint64(),
			Stddev: stddev,
		}
		generated := 0
		maxAttempts := clusterSizes[c]*8 + 1024
		attempts := 0
		for generated < clusterSizes[c] && attempts < maxAttempts {
			attempts++
			v := SampleGaussian(clusters[c].Center, clusters[c].Stddev, rng)
			if v == 0 && clusters[c].Center != 0 {
				continue
			}
			if _, ok := seen[v]; !ok {
				seen[v] = struct{}{}
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
