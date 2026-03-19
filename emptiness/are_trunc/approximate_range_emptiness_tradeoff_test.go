package are_trunc

import (
	"Thesis/bits"
	"Thesis/emptiness/ere"
	"Thesis/testutils"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
)

func TestTruncARE_FinalSmooth(t *testing.T) {
	queryRng := rand.New(rand.NewSource(1337))
	numQueries := 1000000
	type qry struct {
		a, b bits.BitString
	}
	queries := make([]qry, numQueries)
	for i := 0; i < numQueries; i++ {
		v1 := queryRng.Uint64()
		v2 := v1 + uint64(queryRng.Intn(200))
		queries[i] = qry{bits.NewFromUint64(v1), bits.NewFromUint64(v2)}
	}

	fmt.Println("N,K,BitsPerKey,ActualFPR")

	nValues := []int{135000, 150000, 170000, 195000, 220000, 250000}
	
	var wg sync.WaitGroup
	resultsChan := make(chan string, 200)
	semaphore := make(chan struct{}, runtime.NumCPU())

	for _, n := range nValues {
		keys := testutils.GetBenchKeys(64, n)
		
		for K := uint32(18); K <= 30; K++ {
			wg.Add(1)
			go func(nVal int, kVal uint32, kset []bits.BitString) {
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
					if validQueries >= 1000000 { break }
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

func isTrulyEmptyFinal(keys []bits.BitString, a, b bits.BitString) bool {
	l, r := 0, len(keys)
	for l < r {
		mid := l + (r-l)/2
		if keys[mid].Compare(a) < 0 {
			l = mid + 1
		} else {
			r = mid
		}
	}
	if l < len(keys) && keys[l].Compare(b) <= 0 {
		return false
	}
	return true
}

func buildAREWithKFinal(keys []bits.BitString, K uint32) (*TruncARE, error) {
	n := len(keys)
	if n == 0 {
		return &TruncARE{K: K}, nil
	}
	minKey := keys[0]
	maxKey := keys[n-1]
	spread := maxKey.Sub(minKey)
	spreadStart := trieFirstSetBit(spread)
	truncatedKeys := make([]bits.BitString, 0, n)
	var lastKey bits.BitString
	for i, k := range keys {
		trunc := normalizeToK(k, minKey, spreadStart, K)
		if i == 0 || trunc.Compare(lastKey) > 0 {
			truncatedKeys = append(truncatedKeys, trunc)
			lastKey = trunc
		}
	}
	universe := bits.NewBitString(K)
	exact, _ := ere.NewExactRangeEmptiness(truncatedKeys, universe)
	return &TruncARE{exact: exact, K: K, minKey: minKey, maxKey: maxKey, spreadStart: spreadStart}, nil
}
