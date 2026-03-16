package are_trunc_test

import (
	"Thesis/bits"
	"Thesis/emptiness/are_soda_hash"
	"Thesis/emptiness/are_trunc"
	"Thesis/testutils"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"
)

func TestTradeoff_TheoreticalVsSodaVsTrunc(t *testing.T) {
	const (
		n          = 1 << 18
		queryCount = 1 << 18
		nRuns      = 3
		rangeLen   = uint64(128)
	)

	rng := rand.New(rand.NewSource(42))
	keys := make([]uint64, 0, n)
	seen := make(map[uint64]bool, n)
	for len(keys) < n {
		k := rng.Uint64() & ((1 << 60) - 1)
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	keysBS := make([]bits.BitString, len(keys))
	for i, v := range keys {
		keysBS[i] = testutils.TrieBS(v)
	}

	seeds := []int64{12345, 54321, 99999}
	querySets := make([][][2]uint64, nRuns)
	for r := 0; r < nRuns; r++ {
		qrng := rand.New(rand.NewSource(seeds[r]))
		qs := make([][2]uint64, queryCount)
		for i := range qs {
			a := qrng.Uint64() & ((1 << 60) - 1)
			qs[i] = [2]uint64{a, a + rangeLen - 1}
		}
		querySets[r] = qs
	}

	kGrid := []uint32{
		4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
		21, 22, 23, 24, 25, 26, 28, 30, 32, 34, 36,
	}

	theoretical := &testutils.SeriesData{Name: "Theoretical", Color: "#ef4444", Dashed: true, Marker: "none"}
	truncSeries := &testutils.SeriesData{Name: "Truncation", Color: "#9b59b6", Marker: "triangle"}
	sodaSeries := &testutils.SeriesData{Name: "SODA", Color: "#4dd88a", Marker: "diamond"}

	for _, K := range kGrid {
		thEps := float64(rangeLen) / math.Exp2(float64(K))
		if thEps >= 1e-6 && thEps <= 1 {
			theoretical.Points = append(theoretical.Points,
				testutils.Point{X: float64(K), Y: thEps})
		}
	}

	type task struct {
		series    string
		K         uint32
		bpk       float64
		isEmptyFn func(a, b uint64) bool
	}
	var tasks []task

	for _, K := range kGrid {
		if f, err := are_trunc.NewApproximateRangeEmptinessFromK(keysBS, K); err == nil {
			bpk := float64(f.SizeInBits()) / float64(n)
			f := f
			tasks = append(tasks, task{"Truncation", K, bpk,
				func(a, b uint64) bool { return f.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }})
		}
		if f, err := are_soda_hash.NewApproximateRangeEmptinessSodaFromK(keys, rangeLen, K); err == nil {
			bpk := float64(f.SizeInBits()) / float64(n)
			f := f
			tasks = append(tasks, task{"SODA", K, bpk,
				func(a, b uint64) bool { return f.IsEmpty(a, b) }})
		}
	}

	results := make([]testutils.Point, len(tasks))
	seriesNames := make([]string, len(tasks))
	var wg sync.WaitGroup
	for i, tk := range tasks {
		i, tk := i, tk
		wg.Add(1)
		go func() {
			defer wg.Done()
			sum := 0.0
			for _, qs := range querySets {
				sum += testutils.MeasureFPR(keys, qs, tk.isEmptyFn)
			}
			fpr := sum / float64(nRuns)
			results[i] = testutils.Point{X: tk.bpk, Y: fpr}
			seriesNames[i] = tk.series
		}()
	}
	wg.Wait()

	for i, pt := range results {
		switch seriesNames[i] {
		case "Truncation":
			truncSeries.Points = append(truncSeries.Points, pt)
		case "SODA":
			sodaSeries.Points = append(sodaSeries.Points, pt)
		}
	}

	svgPath := "tradeoff_uniform_L128.svg"

	err := testutils.GenerateTradeoffSVG(
		fmt.Sprintf("FPR vs BPK — Uniform (n=%d, L=%d)", n, rangeLen),
		"Bits per Key (BPK)",
		"False Positive Rate (FPR)",
		[]testutils.SeriesData{*theoretical, *truncSeries, *sodaSeries},
		svgPath,
	)
	if err != nil {
		t.Errorf("SVG generation failed: %v", err)
	} else {
		fmt.Printf("SVG written to %s\n", svgPath)
	}

	// Clean up old L16 artifact if present
	os.Remove("tradeoff_uniform_L16.svg")
}
