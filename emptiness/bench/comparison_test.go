package bench_test

import (
	"Thesis/bits"
	"Thesis/emptiness/are"
	"Thesis/emptiness/are_hybrid"
	"Thesis/emptiness/are_optimized"
	"Thesis/emptiness/are_pgm"
	"Thesis/emptiness/are_soda_hash"
	"Thesis/testutils"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestTradeoff_Cluster(t *testing.T) {
	const (
		n          = 1 << 16
		rangeLen   = uint64(100)
		queryCount = 200_000
		nClusters  = 5
	)

	epsilons := []float64{0.1, 0.05, 0.02, 0.01, 0.005, 0.002, 0.001}

	const unifFrac = 0.15

	// Same seeds as are_optimized/tradeoff_bench_test.go for identical data
	rng := rand.New(rand.NewSource(99))
	clusterU64, clusters := testutils.GenerateClusterDistribution(n, nClusters, unifFrac, rng)
	clusterBS := make([]bits.BitString, len(clusterU64))
	for i, v := range clusterU64 {
		clusterBS[i] = testutils.TrieBS(v)
	}

	qrng := rand.New(rand.NewSource(12345))
	queries := testutils.GenerateClusterQueries(queryCount, clusters, unifFrac, rangeLen, qrng)

	tValues := []uint32{1, 2, 3, 4}
	adaptiveColors := []string{"#6495ED", "#4169E1", "#1E3A8A", "#0F1D45"}

	allSeries := map[string]*testutils.SeriesData{
		"Theoretical":    {Name: "Theoretical", Color: "#ef4444", Dashed: true, Marker: "circle"},
		"Adaptive (t=0)": {Name: "Adaptive (t=0)", Color: "#2a7fff", Marker: "square"},
		"SODA":           {Name: "SODA", Color: "#22a06b", Marker: "diamond"},
		"Truncation":     {Name: "Truncation", Color: "#e6a800", Marker: "triangle"},
		"Hybrid":         {Name: "Hybrid", Color: "#9b59b6", Marker: "star"},
		"CDF-ARE":        {Name: "CDF-ARE", Color: "#e05d10", Marker: "circle"},
	}
	for i, tv := range tValues {
		name := fmt.Sprintf("Adaptive (t=%d)", tv)
		allSeries[name] = &testutils.SeriesData{Name: name, Color: adaptiveColors[i], Dashed: true, Marker: "square"}
	}

	os.MkdirAll("../../bench_results/plots", 0755)
	csvF, _ := os.Create("../../bench_results/plots/are_cluster_data.csv")
	defer csvF.Close()
	fmt.Fprintln(csvF, "Epsilon,Series,BPK,FPR")

	fmt.Printf("\n=== Cluster Distribution (%d keys, %d clusters) ===\n", n, nClusters)
	fmt.Printf("%-6s | %-20s | %8s | %12s\n", "Eps", "Series", "BPK", "FPR")
	fmt.Println(strings.Repeat("-", 55))

	for _, eps := range epsilons {
		thBPK := math.Log2(float64(rangeLen) / eps)
		allSeries["Theoretical"].Points = append(allSeries["Theoretical"].Points, testutils.Point{X: thBPK, Y: eps})
		fmt.Fprintf(csvF, "%f,Theoretical,%f,%f\n", eps, thBPK, eps)
		fmt.Printf("%-6.3f | %-20s | %8.2f | %12.6f\n", eps, "Theoretical", thBPK, eps)

		fSoda, errSoda := are_soda_hash.NewApproximateRangeEmptinessSoda(clusterU64, rangeLen, eps)
		fTrunc, errTrunc := are.NewApproximateRangeEmptiness(clusterBS, eps)
		fHybrid, errHybrid := are_hybrid.NewHybridARE(clusterBS, rangeLen, eps)

		type m struct {
			name  string
			err   error
			bpk   float64
			check func(a, b uint64) bool
		}

		// Adaptive t=0..4
		var ms []m
		for _, tv := range append([]uint32{0}, tValues...) {
			name := fmt.Sprintf("Adaptive (t=%d)", tv)
			f, err := are_optimized.NewOptimizedARE(clusterBS, rangeLen, eps, tv)
			var bpk float64
			if err == nil {
				bpk = float64(f.SizeInBits()) / float64(n)
			}
			fCopy := f
			ms = append(ms, m{name, err, bpk, func(a, b uint64) bool { return fCopy.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }})
		}

		// SODA
		if errSoda == nil {
			fCopy := fSoda
			ms = append(ms, m{"SODA", nil, float64(fSoda.SizeInBits()) / float64(n), func(a, b uint64) bool { return fCopy.IsEmpty(a, b) }})
		} else {
			ms = append(ms, m{"SODA", errSoda, 0, nil})
		}
		// Truncation
		if errTrunc == nil {
			fCopy := fTrunc
			ms = append(ms, m{"Truncation", nil, float64(fTrunc.SizeInBits()) / float64(n), func(a, b uint64) bool { return fCopy.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }})
		} else {
			ms = append(ms, m{"Truncation", errTrunc, 0, nil})
		}
		// Hybrid
		if errHybrid == nil {
			fCopy := fHybrid
			ms = append(ms, m{"Hybrid", nil, float64(fHybrid.SizeInBits()) / float64(n), func(a, b uint64) bool { return fCopy.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }})
		} else {
			ms = append(ms, m{"Hybrid", errHybrid, 0, nil})
		}
		// CDF-ARE
		fCdf, errCdf := are_pgm.NewPGMApproximateRangeEmptiness(clusterU64, rangeLen, eps, 64)
		if errCdf == nil {
			fCopy := fCdf
			ms = append(ms, m{"CDF-ARE", nil, float64(fCdf.TotalSizeInBits()) / float64(n), func(a, b uint64) bool { return fCopy.IsEmpty(a, b) }})
		} else {
			ms = append(ms, m{"CDF-ARE", errCdf, 0, nil})
		}

		for _, me := range ms {
			if me.err != nil {
				fmt.Printf("%-6.3f | %-20s | %8s | %12s (err: %v)\n", eps, me.name, "N/A", "N/A", me.err)
				continue
			}
			fpr := testutils.MeasureFPR(clusterU64, queries, me.check)
			allSeries[me.name].Points = append(allSeries[me.name].Points, testutils.Point{X: me.bpk, Y: fpr})
			fmt.Fprintf(csvF, "%f,%s,%f,%f\n", eps, me.name, me.bpk, fpr)
			fmt.Printf("%-6.3f | %-20s | %8.2f | %12.6f\n", eps, me.name, me.bpk, fpr)
		}
	}

	orderedSeries := []testutils.SeriesData{
		*allSeries["Theoretical"],
		*allSeries["Adaptive (t=0)"],
	}
	for _, tv := range tValues {
		orderedSeries = append(orderedSeries, *allSeries[fmt.Sprintf("Adaptive (t=%d)", tv)])
	}
	orderedSeries = append(orderedSeries, *allSeries["SODA"], *allSeries["Truncation"], *allSeries["Hybrid"], *allSeries["CDF-ARE"])

	err := testutils.GenerateTradeoffSVG(
		"Range Emptiness: FPR vs BPK (Cluster Distribution)",
		"Bits per Key (BPK)",
		"False Positive Rate (FPR)",
		orderedSeries,
		"../../bench_results/plots/are_cluster_comparison.svg",
	)
	if err != nil {
		t.Errorf("SVG generation failed: %v", err)
	} else {
		fmt.Println("\nSVG written to bench_results/plots/are_cluster_comparison.svg")
	}
}

func TestScalability(t *testing.T) {
	sizes := []int{1 << 16, 1 << 20}
	const (
		rangeLen   = uint64(100)
		queryCount = 500_000
		nClusters  = 5
		unifFrac   = 0.15
		eps        = 0.01
	)

	type filterEntry struct {
		name  string
		build func(keysBS []bits.BitString, keysU64 []uint64) (func(a, b uint64) bool, uint64, string, error)
	}

	filters := []filterEntry{
		{"Adaptive(t=0)", func(bs []bits.BitString, u64 []uint64) (func(a, b uint64) bool, uint64, string, error) {
			f, err := are_optimized.NewOptimizedARE(bs, rangeLen, eps, 0)
			if err != nil {
				return nil, 0, "", err
			}
			return func(a, b uint64) bool { return f.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }, f.SizeInBits(), "-", nil
		}},
		{"SODA", func(bs []bits.BitString, u64 []uint64) (func(a, b uint64) bool, uint64, string, error) {
			f, err := are_soda_hash.NewApproximateRangeEmptinessSoda(u64, rangeLen, eps)
			if err != nil {
				return nil, 0, "", err
			}
			return func(a, b uint64) bool { return f.IsEmpty(a, b) }, f.SizeInBits(), "-", nil
		}},
		{"Truncation", func(bs []bits.BitString, u64 []uint64) (func(a, b uint64) bool, uint64, string, error) {
			f, err := are.NewApproximateRangeEmptiness(bs, eps)
			if err != nil {
				return nil, 0, "", err
			}
			return func(a, b uint64) bool { return f.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }, f.SizeInBits(), "-", nil
		}},
		{"Hybrid", func(bs []bits.BitString, u64 []uint64) (func(a, b uint64) bool, uint64, string, error) {
			f, err := are_hybrid.NewHybridARE(bs, rangeLen, eps)
			if err != nil {
				return nil, 0, "", err
			}
			nc, nf, nt := f.Stats()
			info := fmt.Sprintf("%dc/%d%%fb", nc, 100*nf/nt)
			return func(a, b uint64) bool { return f.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }, f.SizeInBits(), info, nil
		}},
		{"CDF-ARE", func(bs []bits.BitString, u64 []uint64) (func(a, b uint64) bool, uint64, string, error) {
			f, err := are_pgm.NewPGMApproximateRangeEmptiness(u64, rangeLen, eps, 64)
			if err != nil {
				return nil, 0, "", err
			}
			return func(a, b uint64) bool { return f.IsEmpty(a, b) }, f.TotalSizeInBits(), "-", nil
		}},
	}

	for _, n := range sizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			rng := rand.New(rand.NewSource(99))
			keysU64, clusters := testutils.GenerateClusterDistribution(n, nClusters, unifFrac, rng)
			keysBS := make([]bits.BitString, len(keysU64))
			for i, v := range keysU64 {
				keysBS[i] = testutils.TrieBS(v)
			}

			qrng := rand.New(rand.NewSource(12345))
			queries := testutils.GenerateClusterQueries(queryCount, clusters, unifFrac, rangeLen, qrng)

			fmt.Printf("\n=== n=%d (ε=%.3f, rangeLen=%d, %d queries) ===\n", n, eps, rangeLen, queryCount)
			fmt.Printf("%-16s | %8s | %12s | %12s | %12s | %s\n",
				"Filter", "BPK", "FPR", "Build(ms)", "Query(ns/op)", "Info")
			fmt.Println(strings.Repeat("-", 90))

			for _, fe := range filters {
				// Build
				buildStart := time.Now()
				check, sizeBits, info, err := fe.build(keysBS, keysU64)
				buildDur := time.Since(buildStart)
				if err != nil {
					fmt.Printf("%-16s | %8s | %12s | %12s | %12s | err: %v\n",
						fe.name, "N/A", "N/A", "N/A", "N/A", err)
					continue
				}

				bpk := float64(sizeBits) / float64(n)

				// Measure FPR
				fp, totalEmpty := 0, 0
				for _, q := range queries {
					a, b := q[0], q[1]
					if b < a {
						continue
					}
					if !testutils.GroundTruth(keysU64, a, b) {
						continue
					}
					totalEmpty++
					if !check(a, b) {
						fp++
					}
				}
				var fpr float64
				if totalEmpty > 0 {
					fpr = float64(fp) / float64(totalEmpty)
				}

				// Measure query latency (warm, on all queries — not just empty)
				queryStart := time.Now()
				for _, q := range queries {
					check(q[0], q[1])
				}
				queryDur := time.Since(queryStart)
				queryNs := float64(queryDur.Nanoseconds()) / float64(queryCount)

				fmt.Printf("%-16s | %8.2f | %12.6f | %12.1f | %12.1f | %s\n",
					fe.name, bpk, fpr, float64(buildDur.Milliseconds()), queryNs, info)
			}
		})
	}
}

func TestTradeoff_Full(t *testing.T) {
	const (
		n          = 10000
		rangeLen   = uint64(100)
		queryCount = 100_000
	)

	epsilons := []float64{0.1, 0.05, 0.02, 0.01, 0.005, 0.002, 0.001}

	rng := rand.New(rand.NewSource(42))
	seen := make(map[uint64]bool)
	unifU64 := make([]uint64, 0, n)
	for len(unifU64) < n {
		v := rng.Uint64()
		if !seen[v] {
			seen[v] = true
			unifU64 = append(unifU64, v)
		}
	}
	sort.Slice(unifU64, func(i, j int) bool { return unifU64[i] < unifU64[j] })
	unifBS := make([]bits.BitString, n)
	for i, v := range unifU64 {
		unifBS[i] = testutils.TrieBS(v)
	}

	qrng := rand.New(rand.NewSource(12345))
	queries := make([][2]uint64, queryCount)
	for i := range queries {
		a := qrng.Uint64()
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	// Sequential: evenly spaced keys in a narrow range
	seqGap := uint64(1000)
	seqBase := uint64(1) << 40
	seqU64 := make([]uint64, n)
	seqBS := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		v := seqBase + uint64(i)*seqGap
		seqU64[i] = v
		seqBS[i] = testutils.TrieBS(v)
	}

	qrngS := rand.New(rand.NewSource(12345))
	seqQueries := make([][2]uint64, queryCount)
	for i := range seqQueries {
		a := qrngS.Uint64()
		seqQueries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	allSeries := map[string]*testutils.SeriesData{
		"Theoretical":       {Name: "Theoretical", Color: "#ef4444", Dashed: true, Marker: "circle"},
		"Adaptive (Unif)":   {Name: "Adaptive (Unif)", Color: "#2a7fff", Marker: "square"},
		"Adaptive (Seq)":    {Name: "Adaptive (Seq)", Color: "#2a7fff", Dashed: true, Marker: "square"},
		"SODA (Unif)":       {Name: "SODA (Unif)", Color: "#22a06b", Marker: "diamond"},
		"SODA (Seq)":        {Name: "SODA (Seq)", Color: "#22a06b", Dashed: true, Marker: "diamond"},
		"Truncation (Unif)": {Name: "Truncation (Unif)", Color: "#e6a800", Marker: "triangle"},
		"Truncation (Seq)":  {Name: "Truncation (Seq)", Color: "#e6a800", Dashed: true, Marker: "triangle"},
		"Hybrid (Unif)":     {Name: "Hybrid (Unif)", Color: "#9b59b6", Marker: "star"},
		"Hybrid (Seq)":      {Name: "Hybrid (Seq)", Color: "#9b59b6", Dashed: true, Marker: "star"},
		"CDF-ARE (Unif)":    {Name: "CDF-ARE (Unif)", Color: "#e05d10", Marker: "circle"},
		"CDF-ARE (Seq)":     {Name: "CDF-ARE (Seq)", Color: "#e05d10", Dashed: true, Marker: "circle"},
	}

	os.MkdirAll("../../bench_results/plots", 0755)
	csvF, _ := os.Create("../../bench_results/plots/are_tradeoff_data.csv")
	defer csvF.Close()
	fmt.Fprintln(csvF, "Epsilon,Series,BPK,FPR")

	fmt.Printf("\n=== Full Comparison (Uniform + Sequential, %d keys) ===\n", n)
	fmt.Printf("%-6s | %-20s | %8s | %12s\n", "Eps", "Series", "BPK", "FPR")
	fmt.Println(strings.Repeat("-", 55))

	for _, eps := range epsilons {
		thBPK := math.Log2(float64(rangeLen) / eps)
		allSeries["Theoretical"].Points = append(allSeries["Theoretical"].Points, testutils.Point{X: thBPK, Y: eps})
		fmt.Fprintf(csvF, "%f,Theoretical,%f,%f\n", eps, thBPK, eps)
		fmt.Printf("%-6.3f | %-20s | %8.2f | %12.6f\n", eps, "Theoretical", thBPK, eps)

		fOptU, errOptU := are_optimized.NewOptimizedARE(unifBS, rangeLen, eps, 0)
		fOptS, errOptS := are_optimized.NewOptimizedARE(seqBS, rangeLen, eps, 0)
		fSodaU, errSodaU := are_soda_hash.NewApproximateRangeEmptinessSoda(unifU64, rangeLen, eps)
		fSodaS, errSodaS := are_soda_hash.NewApproximateRangeEmptinessSoda(seqU64, rangeLen, eps)
		fTruncU, errTruncU := are.NewApproximateRangeEmptiness(unifBS, eps)
		fTruncS, errTruncS := are.NewApproximateRangeEmptiness(seqBS, eps)
		fHybridU, errHybridU := are_hybrid.NewHybridARE(unifBS, rangeLen, eps)
		fHybridS, errHybridS := are_hybrid.NewHybridARE(seqBS, rangeLen, eps)
		fCdfU, errCdfU := are_pgm.NewPGMApproximateRangeEmptiness(unifU64, rangeLen, eps, 64)
		fCdfS, errCdfS := are_pgm.NewPGMApproximateRangeEmptiness(seqU64, rangeLen, eps, 64)

		type mm struct {
			name    string
			err     error
			bpk     float64
			keys    []uint64
			queries [][2]uint64
			check   func(a, b uint64) bool
		}

		var ms []mm
		add := func(name string, err error, sizeBits uint64, keys []uint64, qs [][2]uint64, fn func(a, b uint64) bool) {
			var bpk float64
			if err == nil {
				bpk = float64(sizeBits) / float64(n)
			}
			ms = append(ms, mm{name, err, bpk, keys, qs, fn})
		}

		add("Adaptive (Unif)", errOptU, safeSize(fOptU), unifU64, queries, func(a, b uint64) bool { return fOptU.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) })
		add("Adaptive (Seq)", errOptS, safeSize(fOptS), seqU64, seqQueries, func(a, b uint64) bool { return fOptS.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) })
		add("SODA (Unif)", errSodaU, safeSizeSoda(fSodaU), unifU64, queries, func(a, b uint64) bool { return fSodaU.IsEmpty(a, b) })
		add("SODA (Seq)", errSodaS, safeSizeSoda(fSodaS), seqU64, seqQueries, func(a, b uint64) bool { return fSodaS.IsEmpty(a, b) })
		add("Truncation (Unif)", errTruncU, safeSizeTrunc(fTruncU), unifU64, queries, func(a, b uint64) bool { return fTruncU.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) })
		add("Truncation (Seq)", errTruncS, safeSizeTrunc(fTruncS), seqU64, seqQueries, func(a, b uint64) bool { return fTruncS.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) })
		add("Hybrid (Unif)", errHybridU, safeSizeHybrid(fHybridU), unifU64, queries, func(a, b uint64) bool { return fHybridU.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) })
		add("Hybrid (Seq)", errHybridS, safeSizeHybrid(fHybridS), seqU64, seqQueries, func(a, b uint64) bool { return fHybridS.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) })
		add("CDF-ARE (Unif)", errCdfU, safeSizeCdf(fCdfU), unifU64, queries, func(a, b uint64) bool { return fCdfU.IsEmpty(a, b) })
		add("CDF-ARE (Seq)", errCdfS, safeSizeCdf(fCdfS), seqU64, seqQueries, func(a, b uint64) bool { return fCdfS.IsEmpty(a, b) })

		for _, me := range ms {
			if me.err != nil {
				fmt.Printf("%-6.3f | %-20s | %8s | %12s (err: %v)\n", eps, me.name, "N/A", "N/A", me.err)
				continue
			}
			fpr := testutils.MeasureFPR(me.keys, me.queries, me.check)
			allSeries[me.name].Points = append(allSeries[me.name].Points, testutils.Point{X: me.bpk, Y: fpr})
			fmt.Fprintf(csvF, "%f,%s,%f,%f\n", eps, me.name, me.bpk, fpr)
			fmt.Printf("%-6.3f | %-20s | %8.2f | %12.6f\n", eps, me.name, me.bpk, fpr)
		}
	}

	orderedSeries := []testutils.SeriesData{
		*allSeries["Theoretical"],
		*allSeries["Adaptive (Unif)"],
		*allSeries["Adaptive (Seq)"],
		*allSeries["SODA (Unif)"],
		*allSeries["SODA (Seq)"],
		*allSeries["Truncation (Unif)"],
		*allSeries["Truncation (Seq)"],
		*allSeries["Hybrid (Unif)"],
		*allSeries["Hybrid (Seq)"],
		*allSeries["CDF-ARE (Unif)"],
		*allSeries["CDF-ARE (Seq)"],
	}

	err := testutils.GenerateTradeoffSVG(
		"Range Emptiness: FPR vs Bits per Key",
		"Bits per Key (BPK)",
		"False Positive Rate (FPR)",
		orderedSeries,
		"../../bench_results/plots/are_full_comparison.svg",
	)
	if err != nil {
		t.Errorf("SVG generation failed: %v", err)
	} else {
		fmt.Println("\nSVG written to bench_results/plots/are_full_comparison.svg")
	}
}

func TestBuildTimePerKey(t *testing.T) {
	sizes := []int{1 << 10, 1 << 12, 1 << 14, 1 << 16, 1 << 18, 1 << 20}
	const (
		rangeLen  = uint64(100)
		nClusters = 5
		unifFrac  = 0.15
		eps       = 0.01
	)

	type filterDef struct {
		name  string
		color string
		build func(bs []bits.BitString, u64 []uint64) error
	}

	filters := []filterDef{
		{"Adaptive(t=0)", "#2a7fff", func(bs []bits.BitString, _ []uint64) error {
			_, err := are_optimized.NewOptimizedARE(bs, rangeLen, eps, 0)
			return err
		}},
		{"SODA", "#22a06b", func(_ []bits.BitString, u64 []uint64) error {
			_, err := are_soda_hash.NewApproximateRangeEmptinessSoda(u64, rangeLen, eps)
			return err
		}},
		{"Truncation", "#e6a800", func(bs []bits.BitString, _ []uint64) error {
			_, err := are.NewApproximateRangeEmptiness(bs, eps)
			return err
		}},
		{"Hybrid", "#9b59b6", func(bs []bits.BitString, _ []uint64) error {
			_, err := are_hybrid.NewHybridARE(bs, rangeLen, eps)
			return err
		}},
		{"CDF-ARE", "#e05d10", func(_ []bits.BitString, u64 []uint64) error {
			_, err := are_pgm.NewPGMApproximateRangeEmptiness(u64, rangeLen, eps, 64)
			return err
		}},
	}

	markers := []string{"square", "diamond", "triangle", "star", "circle"}
	var allSeries []testutils.SeriesData
	for i, f := range filters {
		allSeries = append(allSeries, testutils.SeriesData{
			Name: f.name, Color: f.color, Marker: markers[i%len(markers)],
		})
	}

	fmt.Printf("\n=== Build Time per Key (ε=%.3f, L=%d) ===\n", eps, rangeLen)
	fmt.Printf("%-16s", "Filter")
	for _, n := range sizes {
		fmt.Printf(" | %10s", fmt.Sprintf("n=%d", n))
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 16+len(sizes)*13))

	for fi, fd := range filters {
		fmt.Printf("%-16s", fd.name)
		for _, n := range sizes {
			rng := rand.New(rand.NewSource(99))
			keysU64, _ := testutils.GenerateClusterDistribution(n, nClusters, unifFrac, rng)
			keysBS := make([]bits.BitString, len(keysU64))
			for i, v := range keysU64 {
				keysBS[i] = testutils.TrieBS(v)
			}

			start := time.Now()
			err := fd.build(keysBS, keysU64)
			dur := time.Since(start)

			if err != nil {
				fmt.Printf(" | %10s", "err")
				continue
			}

			nsPerKey := float64(dur.Nanoseconds()) / float64(n)
			allSeries[fi].Points = append(allSeries[fi].Points, testutils.Point{X: float64(n), Y: nsPerKey})
			fmt.Printf(" | %8.1f ns", nsPerKey)
		}
		fmt.Println()
	}

	os.MkdirAll("../../bench_results/plots", 0755)
	err := testutils.GeneratePerformanceSVG(testutils.PlotConfig{
		Title:  fmt.Sprintf("Build Time per Key (ε=%.3f, L=%d)", eps, rangeLen),
		XLabel: "Number of Keys (n)",
		YLabel: "Build Time (ns/key)",
		XScale: testutils.Log10,
		YScale: testutils.Linear,
	}, allSeries, "../../bench_results/plots/build_time_per_key.svg")
	if err != nil {
		t.Errorf("SVG generation failed: %v", err)
	} else {
		fmt.Println("\nSVG written to bench_results/plots/build_time_per_key.svg")
	}
}

func TestQueryTimeVsRangeLen(t *testing.T) {
	rangeLens := []uint64{16, 64, 256, 1024, 4096, 16384}
	const (
		n          = 1 << 16
		queryCount = 200_000
		nClusters  = 5
		unifFrac   = 0.15
		eps        = 0.01
	)

	rng := rand.New(rand.NewSource(99))
	keysU64, clusters := testutils.GenerateClusterDistribution(n, nClusters, unifFrac, rng)
	keysBS := make([]bits.BitString, len(keysU64))
	for i, v := range keysU64 {
		keysBS[i] = testutils.TrieBS(v)
	}

	type filterDef struct {
		name  string
		color string
		build func(L uint64) (func(a, b uint64) bool, error)
	}

	filters := []filterDef{
		{"Adaptive(t=0)", "#2a7fff", func(L uint64) (func(a, b uint64) bool, error) {
			f, err := are_optimized.NewOptimizedARE(keysBS, L, eps, 0)
			if err != nil {
				return nil, err
			}
			return func(a, b uint64) bool { return f.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }, nil
		}},
		{"SODA", "#22a06b", func(L uint64) (func(a, b uint64) bool, error) {
			f, err := are_soda_hash.NewApproximateRangeEmptinessSoda(keysU64, L, eps)
			if err != nil {
				return nil, err
			}
			return func(a, b uint64) bool { return f.IsEmpty(a, b) }, nil
		}},
		{"Truncation", "#e6a800", func(_ uint64) (func(a, b uint64) bool, error) {
			f, err := are.NewApproximateRangeEmptiness(keysBS, eps)
			if err != nil {
				return nil, err
			}
			return func(a, b uint64) bool { return f.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }, nil
		}},
		{"Hybrid", "#9b59b6", func(L uint64) (func(a, b uint64) bool, error) {
			f, err := are_hybrid.NewHybridARE(keysBS, L, eps)
			if err != nil {
				return nil, err
			}
			return func(a, b uint64) bool { return f.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }, nil
		}},
		{"CDF-ARE", "#e05d10", func(L uint64) (func(a, b uint64) bool, error) {
			f, err := are_pgm.NewPGMApproximateRangeEmptiness(keysU64, L, eps, 64)
			if err != nil {
				return nil, err
			}
			return func(a, b uint64) bool { return f.IsEmpty(a, b) }, nil
		}},
	}

	markers := []string{"square", "diamond", "triangle", "star", "circle"}
	var allSeries []testutils.SeriesData
	for i, f := range filters {
		allSeries = append(allSeries, testutils.SeriesData{
			Name: f.name, Color: f.color, Marker: markers[i%len(markers)],
		})
	}

	fmt.Printf("\n=== Query Time vs Range Length (n=%d, ε=%.3f) ===\n", n, eps)
	fmt.Printf("%-16s", "Filter")
	for _, L := range rangeLens {
		fmt.Printf(" | %10s", fmt.Sprintf("L=%d", L))
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 16+len(rangeLens)*13))

	for fi, fd := range filters {
		fmt.Printf("%-16s", fd.name)
		for _, L := range rangeLens {
			qrng := rand.New(rand.NewSource(12345))
			queries := testutils.GenerateClusterQueries(queryCount, clusters, unifFrac, L, qrng)

			check, err := fd.build(L)
			if err != nil {
				fmt.Printf(" | %10s", "err")
				continue
			}

			start := time.Now()
			for _, q := range queries {
				check(q[0], q[1])
			}
			dur := time.Since(start)
			nsPerQuery := float64(dur.Nanoseconds()) / float64(queryCount)

			allSeries[fi].Points = append(allSeries[fi].Points, testutils.Point{X: float64(L), Y: nsPerQuery})
			fmt.Printf(" | %8.1f ns", nsPerQuery)
		}
		fmt.Println()
	}

	os.MkdirAll("../../bench_results/plots", 0755)
	err := testutils.GeneratePerformanceSVG(testutils.PlotConfig{
		Title:  fmt.Sprintf("Query Time vs Range Length (n=%d, ε=%.3f)", n, eps),
		XLabel: "Range Length (L)",
		YLabel: "Query Time (ns/op)",
		XScale: testutils.Log10,
		YScale: testutils.Linear,
	}, allSeries, "../../bench_results/plots/query_time_vs_rangelen.svg")
	if err != nil {
		t.Errorf("SVG generation failed: %v", err)
	} else {
		fmt.Println("\nSVG written to bench_results/plots/query_time_vs_rangelen.svg")
	}
}

// Safe size helpers to avoid nil dereference when build failed
func safeSize(f *are_optimized.OptimizedApproximateRangeEmptiness) uint64 {
	if f == nil {
		return 0
	}
	return f.SizeInBits()
}
func safeSizeSoda(f *are_soda_hash.ApproximateRangeEmptinessSoda) uint64 {
	if f == nil {
		return 0
	}
	return f.SizeInBits()
}
func safeSizeTrunc(f *are.ApproximateRangeEmptiness) uint64 {
	if f == nil {
		return 0
	}
	return f.SizeInBits()
}
func safeSizeHybrid(f *are_hybrid.HybridARE) uint64 {
	if f == nil {
		return 0
	}
	return f.SizeInBits()
}
func safeSizeCdf(f *are_pgm.PGMApproximateRangeEmptiness) uint64 {
	if f == nil {
		return 0
	}
	return f.TotalSizeInBits()
}
