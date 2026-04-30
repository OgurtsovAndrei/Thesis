package are_trunc

import (
	"Thesis/emptiness/exact/ere_one_d"
	"fmt"
	"math/bits"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"testing"
)

func TestTruncARE_FinalSmooth(t *testing.T) {
	queryRng := rand.New(rand.NewSource(1337))
	numQueries := 1000000
	type qry struct {
		a, b uint64
	}
	queries := make([]qry, numQueries)
	for i := 0; i < numQueries; i++ {
		v1 := queryRng.Uint64()
		v2 := v1 + uint64(queryRng.Intn(200))
		queries[i] = qry{v1, v2}
	}

	fmt.Println("N,K,BitsPerKey,ActualFPR")

	nValues := []int{135000, 150000, 170000, 195000, 220000, 250000}

	var wg sync.WaitGroup
	resultsChan := make(chan string, 200)
	semaphore := make(chan struct{}, runtime.NumCPU())

	for _, n := range nValues {
		rng := rand.New(rand.NewSource(42))
		seen := make(map[uint64]bool, n)
		keys := make([]uint64, 0, n)
		for len(keys) < n {
			v := rng.Uint64()
			if !seen[v] {
				seen[v] = true
				keys = append(keys, v)
			}
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

		for K := uint32(18); K <= 30; K++ {
			wg.Add(1)
			go func(nVal int, kVal uint32, kset []uint64) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				are, _ := buildAREWithKFinal(kset, kVal)

				falsePositives := 0
				validQueries := 0
				for _, q := range queries {
					if isTrulyEmptyFinal(kset, q.a, q.b) {
						validQueries++
						if !are.IsEmpty(q.a, q.b) {
							falsePositives++
						}
					}
					if validQueries >= 1000000 {
						break
					}
				}

				fpr := float64(falsePositives) / float64(validQueries)
				bitsPerKey := float64(are.ByteSize()) * 8 / float64(nVal)
				resultsChan <- fmt.Sprintf("%d,%d,%.2f,%.10f", nVal, kVal, bitsPerKey, fpr)
			}(n, K, keys)
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

func isTrulyEmptyFinal(keys []uint64, a, b uint64) bool {
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

func buildAREWithKFinal(keys []uint64, K uint32) (*TruncARE, error) {
	n := len(keys)
	if n == 0 {
		return &TruncARE{K: K}, nil
	}
	minKey := keys[0]
	maxKey := keys[n-1]
	spread := maxKey - minKey

	spreadLen := uint32(bits.Len64(spread))

	truncatedKeys := make([]uint64, 0, n)
	var lastTrunc uint64
	for i, k := range keys {
		trunc := normalizeToK(k, minKey, spreadLen, K)
		if i == 0 || trunc > lastTrunc {
			truncatedKeys = append(truncatedKeys, trunc)
			lastTrunc = trunc
		}
	}
	exact, _ := ere_one_d.NewExactRangeEmptiness(truncatedKeys, K)
	return &TruncARE{
		exact:     exact,
		K:         K,
		keyBits:   64,
		minKey:    minKey,
		maxKey:    maxKey,
		spreadLen: spreadLen,
	}, nil
}
