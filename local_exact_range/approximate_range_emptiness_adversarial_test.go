package local_exact_range

import (
	"Thesis/bits"
	"Thesis/testutils"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
)

// TestARE_AdversarialTradeoff measures the FPR vs BPK tradeoff under multiple
// data distributions and query strategies. The output is a CSV consumed by
// scripts/plot_are_adversarial.py.
//
// Scenarios:
//   - uniform_random:   uniform keys + random point queries (baseline)
//   - uniform_wide:     uniform keys + random range queries (width ~10000)
//   - spread_random:    spread keys (all prefixes distinct) + random point queries
//   - spread_gap:       spread keys + adversarial gap queries
//   - clustered_random: clustered keys + random point queries
func TestARE_AdversarialTradeoff(t *testing.T) {
	const (
		N          = 200000
		numQueries = 500000
	)

	// Pre-generate key sets
	uniformKeys := testutils.GetBenchKeys(64, N)

	kValues := []uint32{18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28}

	type scenario struct {
		name    string
		keys    []bits.BitString
		queryFn func(keys []bits.BitString, K uint32) [][2]bits.BitString
	}

	// Random point queries
	randomPointQueries := func(_ []bits.BitString, _ uint32) [][2]bits.BitString {
		rng := rand.New(rand.NewSource(7777))
		qs := make([][2]bits.BitString, numQueries)
		for i := range qs {
			v := rng.Uint64()
			bs := bits.NewFromUint64(v)
			qs[i] = [2]bits.BitString{bs, bs}
		}
		return qs
	}

	// Wide range queries (width up to 10000)
	wideRangeQueries := func(_ []bits.BitString, _ uint32) [][2]bits.BitString {
		rng := rand.New(rand.NewSource(7777))
		qs := make([][2]bits.BitString, numQueries)
		for i := range qs {
			v := rng.Uint64()
			w := uint64(rng.Intn(10000))
			qs[i] = [2]bits.BitString{
				bits.NewFromUint64(v),
				bits.NewFromUint64(v + w),
			}
		}
		return qs
	}

	// Adversarial gap queries
	gapQueries := func(keys []bits.BitString, _ uint32) [][2]bits.BitString {
		return testutils.GenGapQueries(keys, numQueries, 5000, 9999)
	}

	scenarios := []scenario{
		{"uniform_point", uniformKeys, randomPointQueries},
		{"uniform_wide", uniformKeys, wideRangeQueries},
		{"spread_point", nil, randomPointQueries},     // keys generated per-K
		{"spread_gap", nil, gapQueries},                // keys generated per-K
		{"clustered_point", nil, randomPointQueries},   // fixed cluster keys
	}

	// Pre-generate clustered keys (100 clusters)
	clusteredKeys := testutils.GenClusteredKeys(N, 100, 4242)

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

				// Select keys for this scenario
				var keys []bits.BitString
				switch s.name {
				case "spread_point", "spread_gap":
					keys = testutils.GenSpreadKeys(N, kVal, 1234)
				case "clustered_point":
					keys = clusteredKeys
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
					if isTrulyEmptyFinal(keys, q[0], q[1]) {
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
