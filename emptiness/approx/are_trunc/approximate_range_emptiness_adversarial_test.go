package are_trunc

import (
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"testing"
)

// genSpreadKeysUint64 generates N uint64 keys where the low minK bits are
// spread sequentially — maximises distinct K-bit prefixes (worst case for ARE space).
func genSpreadKeysUint64(n int, minK uint32, seed int64) []uint64 {
	rng := rand.New(rand.NewSource(seed))
	result := make([]uint64, n)
	maxDistinct := uint64(1) << minK
	for i := 0; i < n; i++ {
		upper := rng.Uint64() & ^(maxDistinct - 1)
		result[i] = upper | (uint64(i) % maxDistinct)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// genClusteredKeysUint64 generates N uint64 keys grouped into numClusters clusters.
func genClusteredKeysUint64(n, numClusters int, seed int64) []uint64 {
	rng := rand.New(rand.NewSource(seed))
	keysPerCluster := n / numClusters
	extra := n - keysPerCluster*numClusters

	unique := make(map[uint64]bool, n)
	result := make([]uint64, 0, n)

	for c := 0; c < numClusters; c++ {
		clusterLow := uint64(rng.Intn(1 << 20))
		count := keysPerCluster
		if c < extra {
			count++
		}
		for i := 0; i < count; i++ {
			for {
				upper := rng.Uint64() & ^uint64((1<<20)-1)
				val := upper | clusterLow
				if !unique[val] {
					unique[val] = true
					result = append(result, val)
					break
				}
			}
		}
	}

	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// genGapQueriesUint64 generates queries that fall in gaps between adjacent keys.
func genGapQueriesUint64(keys []uint64, numQueries int, maxWidth uint64, seed int64) [][2]uint64 {
	rng := rand.New(rand.NewSource(seed))
	queries := make([][2]uint64, 0, numQueries)
	n := len(keys)
	if n < 2 {
		return queries
	}

	for attempts := 0; len(queries) < numQueries && attempts < numQueries*10; attempts++ {
		idx := rng.Intn(n - 1)
		aVal := keys[idx]
		bVal := keys[idx+1]

		if bVal <= aVal+2 {
			continue
		}
		gap := bVal - aVal

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
		queries = append(queries, [2]uint64{lo, hi})
	}
	return queries
}

// isTrulyEmptyUint64 returns true when no key from keys falls in [a, b].
func isTrulyEmptyUint64(keys []uint64, a, b uint64) bool {
	l, r := 0, len(keys)
	for l < r {
		mid := l + (r-l)/2
		if keys[mid] < a {
			l = mid + 1
		} else {
			r = mid
		}
	}
	return l >= len(keys) || keys[l] > b
}

// TestARE_AdversarialTradeoff measures the FPR vs BPK tradeoff under multiple
// data distributions and query strategies.
func TestARE_AdversarialTradeoff(t *testing.T) {
	const (
		N          = 200000
		numQueries = 500000
	)

	rngU := rand.New(rand.NewSource(42))
	seen := make(map[uint64]bool, N)
	uniformKeys := make([]uint64, 0, N)
	for len(uniformKeys) < N {
		v := rngU.Uint64()
		if !seen[v] {
			seen[v] = true
			uniformKeys = append(uniformKeys, v)
		}
	}
	sort.Slice(uniformKeys, func(i, j int) bool { return uniformKeys[i] < uniformKeys[j] })

	kValues := []uint32{18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28}

	type scenario struct {
		name    string
		keys    []uint64
		queryFn func(keys []uint64, K uint32) [][2]uint64
	}

	randomPointQueries := func(_ []uint64, _ uint32) [][2]uint64 {
		rng := rand.New(rand.NewSource(7777))
		qs := make([][2]uint64, numQueries)
		for i := range qs {
			v := rng.Uint64()
			qs[i] = [2]uint64{v, v}
		}
		return qs
	}

	wideRangeQueries := func(_ []uint64, _ uint32) [][2]uint64 {
		rng := rand.New(rand.NewSource(7777))
		qs := make([][2]uint64, numQueries)
		for i := range qs {
			v := rng.Uint64()
			w := uint64(rng.Intn(10000))
			qs[i] = [2]uint64{v, v + w}
		}
		return qs
	}

	gapQueries := func(keys []uint64, _ uint32) [][2]uint64 {
		return genGapQueriesUint64(keys, numQueries, 5000, 9999)
	}

	clusteredKeys := genClusteredKeysUint64(N, 100, 4242)

	scenarios := []scenario{
		{"uniform_point", uniformKeys, randomPointQueries},
		{"uniform_wide", uniformKeys, wideRangeQueries},
		{"spread_point", nil, randomPointQueries},
		{"spread_gap", nil, gapQueries},
		{"clustered_point", clusteredKeys, randomPointQueries},
	}

	fmt.Println("Scenario,N,K,BitsPerKey,ActualFPR")

	var wg sync.WaitGroup
	resultsChan := make(chan string, 500)
	semaphore := make(chan struct{}, runtime.NumCPU())

	for _, sc := range scenarios {
		for _, K := range kValues {
			wg.Add(1)
			go func(s scenario, kVal uint32) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				var keys []uint64
				switch s.name {
				case "spread_point", "spread_gap":
					keys = genSpreadKeysUint64(N, kVal, 1234)
				default:
					keys = s.keys
				}

				are, err := buildAREWithKFinal(keys, kVal)
				if err != nil {
					return
				}

				queries := s.queryFn(keys, kVal)

				falsePositives := 0
				validQueries := 0
				for _, q := range queries {
					if isTrulyEmptyUint64(keys, q[0], q[1]) {
						validQueries++
						if !are.IsEmpty(q[0], q[1]) {
							falsePositives++
						}
					}
					if validQueries >= numQueries {
						break
					}
				}

				if validQueries == 0 {
					return
				}

				fpr := float64(falsePositives) / float64(validQueries)
				bpk := float64(are.ByteSize()) * 8 / float64(N)
				resultsChan <- fmt.Sprintf("%s,%d,%d,%.4f,%.10f",
					s.name, N, kVal, bpk, fpr)
			}(sc, K)
		}
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for res := range resultsChan {
		fmt.Println(res)
	}
}
