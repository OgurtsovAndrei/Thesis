package are_soda_hash_test

import (
	"Thesis/emptiness/are_soda_hash"
	"Thesis/testutils"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"
)

const mask60 = (uint64(1) << 60) - 1

func TestTradeoff_SodaPlots(t *testing.T) {
	t.Skip("skip: overwrites SVGs with manual annotations")
	const (
		n          = 1 << 18
		queryCount = 1 << 18
		nRuns      = 3
		rangeLen   = uint64(128)
	)

	kGrid := []uint32{
		4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		21, 22, 23, 24, 25, 26, 28, 30, 32, 34, 36,
	}
	seeds := []int64{12345, 54321, 99999}

	type distSpec struct {
		name      string
		keys      []uint64
		queryFunc func(seed int64) [][2]uint64
	}

	// Uniform keys + uniform queries
	uniformKeys := func() []uint64 {
		rng := rand.New(rand.NewSource(42))
		seen := make(map[uint64]bool, n)
		keys := make([]uint64, 0, n)
		for len(keys) < n {
			k := rng.Uint64() & mask60
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		return keys
	}()

	// Clustered keys + cluster-aware queries
	clusterRng := rand.New(rand.NewSource(99))
	clusterKeys, clusters := testutils.GenerateClusterDistribution(n, 5, 0.15, clusterRng)
	// mask to 60 bits and dedup
	{
		seen := make(map[uint64]bool, len(clusterKeys))
		masked := make([]uint64, 0, len(clusterKeys))
		for _, k := range clusterKeys {
			k &= mask60
			if !seen[k] {
				seen[k] = true
				masked = append(masked, k)
			}
		}
		sort.Slice(masked, func(i, j int) bool { return masked[i] < masked[j] })
		clusterKeys = masked
	}

	dists := []distSpec{
		{
			name: "uniform",
			keys: uniformKeys,
			queryFunc: func(seed int64) [][2]uint64 {
				qrng := rand.New(rand.NewSource(seed))
				qs := make([][2]uint64, queryCount)
				for i := range qs {
					a := qrng.Uint64() & mask60
					qs[i] = [2]uint64{a, a + rangeLen - 1}
				}
				return qs
			},
		},
		{
			name: "clustered",
			keys: clusterKeys,
			queryFunc: func(seed int64) [][2]uint64 {
				qrng := rand.New(rand.NewSource(seed))
				qs := testutils.GenerateClusterQueries(queryCount, clusters, 0.15, rangeLen, qrng)
				// mask queries
				out := make([][2]uint64, len(qs))
				for i, q := range qs {
					a := q[0] & mask60
					b := q[1] & mask60
					if b < a {
						b = a
					}
					out[i] = [2]uint64{a, b}
				}
				return out
			},
		},
	}

	os.MkdirAll(".", 0755)

	for _, dist := range dists {
		t.Run(dist.name, func(t *testing.T) {
			querySets := make([][][2]uint64, nRuns)
			for r := 0; r < nRuns; r++ {
				querySets[r] = dist.queryFunc(seeds[r])
			}

			theoretical := &testutils.SeriesData{Name: "Theoretical", Color: "#ef4444", Dashed: true, Marker: "none"}
			sodaSeries := &testutils.SeriesData{Name: "SODA", Color: "#4dd88a", Marker: "diamond"}

			for _, K := range kGrid {
				thEps := float64(rangeLen) / math.Exp2(float64(K))
				if thEps >= 1e-6 && thEps <= 1 {
					theoretical.Points = append(theoretical.Points,
						testutils.Point{X: float64(K), Y: thEps})
				}
			}

			type task struct {
				K         uint32
				bpk       float64
				isEmptyFn func(a, b uint64) bool
			}
			var tasks []task

			for _, K := range kGrid {
				if f, err := are_soda_hash.NewSodaAREFromK(dist.keys, rangeLen, K); err == nil {
					bpk := float64(f.SizeInBits()) / float64(len(dist.keys))
					f := f
					tasks = append(tasks, task{K, bpk,
						func(a, b uint64) bool { return f.IsEmpty(a, b) }})
				}
			}

			results := make([]testutils.Point, len(tasks))
			var wg sync.WaitGroup
			for i, tk := range tasks {
				i, tk := i, tk
				wg.Add(1)
				go func() {
					defer wg.Done()
					sum := 0.0
					for _, qs := range querySets {
						sum += testutils.MeasureFPR(dist.keys, qs, tk.isEmptyFn)
					}
					results[i] = testutils.Point{X: tk.bpk, Y: sum / float64(nRuns)}
				}()
			}
			wg.Wait()

			for _, pt := range results {
				sodaSeries.Points = append(sodaSeries.Points, pt)
			}

			svgPath := fmt.Sprintf("tradeoff_%s_L128.svg", dist.name)
			err := testutils.GenerateTradeoffSVG(
				fmt.Sprintf("FPR vs BPK — %s (n=%d, L=%d)", dist.name, len(dist.keys), rangeLen),
				"Bits per Key (BPK)",
				"False Positive Rate (FPR)",
				[]testutils.SeriesData{*theoretical, *sodaSeries},
				svgPath,
			)
			if err != nil {
				t.Errorf("SVG generation failed: %v", err)
			} else {
				fmt.Printf("SVG written to %s\n", svgPath)
			}
		})
	}
}
