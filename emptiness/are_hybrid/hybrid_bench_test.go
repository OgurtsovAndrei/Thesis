package are_hybrid_test

import (
	"Thesis/bits"
	"Thesis/emptiness/are"
	"Thesis/emptiness/are_hybrid"
	"Thesis/emptiness/are_optimized"
	"Thesis/emptiness/are_soda_hash"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

func trieBS(val uint64) bits.BitString {
	return bits.NewFromTrieUint64(val, 64)
}

func groundTruth(keys []uint64, a, b uint64) bool {
	idx := sort.Search(len(keys), func(i int) bool { return keys[i] >= a })
	return idx == len(keys) || keys[idx] > b
}

type clusterInfo struct {
	center uint64
	stddev float64
}

func sampleGaussian(center uint64, stddev float64, rng *rand.Rand) uint64 {
	offset := int64(rng.NormFloat64() * stddev)
	if offset >= 0 {
		v := center + uint64(offset)
		if v < center {
			return 0
		}
		return v
	}
	neg := uint64(-offset)
	if neg > center {
		return 0
	}
	return center - neg
}

func generateClusterDistribution(n int, numClusters int, unifFrac float64, rng *rand.Rand) ([]uint64, []clusterInfo) {
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

	clusters := make([]clusterInfo, numClusters)
	perCluster := (n - nUnif) / numClusters
	for c := 0; c < numClusters; c++ {
		clusters[c] = clusterInfo{
			center: rng.Uint64(),
			stddev: float64(uint64(1) << (20 + rng.Intn(10))),
		}
		generated := 0
		for generated < perCluster || (c == numClusters-1 && len(keys) < n) {
			v := sampleGaussian(clusters[c].center, clusters[c].stddev, rng)
			if v == 0 && clusters[c].center != 0 {
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

func generateClusterQueries(count int, clusters []clusterInfo, unifFrac float64, rangeLen uint64, rng *rand.Rand) [][2]uint64 {
	queries := make([][2]uint64, count)
	nUnif := int(float64(count) * unifFrac)

	for i := 0; i < nUnif; i++ {
		a := rng.Uint64()
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	for i := nUnif; i < count; i++ {
		cl := clusters[rng.Intn(len(clusters))]
		a := sampleGaussian(cl.center, cl.stddev, rng)
		if a == 0 {
			a = rng.Uint64()
		}
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}
	return queries
}

type seriesData struct {
	Name   string
	Color  string
	Dashed bool
	Marker string
	Points []point
}

type point struct {
	X, Y float64
}

func TestTradeoff_Hybrid_Cluster(t *testing.T) {
	const (
		n          = 10000
		rangeLen   = uint64(100)
		queryCount = 100_000
		nClusters  = 5
	)

	epsilons := []float64{0.1, 0.05, 0.02, 0.01, 0.005, 0.002, 0.001}

	const unifFrac = 0.15

	// Same seeds as are_optimized/tradeoff_bench_test.go for identical data
	rng := rand.New(rand.NewSource(99))
	clusterU64, clusters := generateClusterDistribution(n, nClusters, unifFrac, rng)
	clusterBS := make([]bits.BitString, len(clusterU64))
	for i, v := range clusterU64 {
		clusterBS[i] = trieBS(v)
	}

	qrng := rand.New(rand.NewSource(12345))
	queries := generateClusterQueries(queryCount, clusters, unifFrac, rangeLen, qrng)

	measureFPR := func(keys []uint64, qs [][2]uint64, check func(a, b uint64) bool) float64 {
		fp, total := 0, 0
		for _, q := range qs {
			a, b := q[0], q[1]
			if b < a {
				continue
			}
			if !groundTruth(keys, a, b) {
				continue
			}
			total++
			if !check(a, b) {
				fp++
			}
		}
		if total == 0 {
			return 0
		}
		return float64(fp) / float64(total)
	}

	tValues := []uint32{1, 2, 3, 4}
	adaptiveColors := []string{"#6495ED", "#4169E1", "#1E3A8A", "#0F1D45"}

	allSeries := map[string]*seriesData{
		"Theoretical":    {Name: "Theoretical", Color: "#ef4444", Dashed: true, Marker: "circle"},
		"Adaptive (t=0)": {Name: "Adaptive (t=0)", Color: "#2a7fff", Marker: "square"},
		"SODA":           {Name: "SODA", Color: "#22a06b", Marker: "diamond"},
		"Truncation":     {Name: "Truncation", Color: "#e6a800", Marker: "triangle"},
		"Hybrid":         {Name: "Hybrid", Color: "#9b59b6", Marker: "star"},
	}
	for i, tv := range tValues {
		name := fmt.Sprintf("Adaptive (t=%d)", tv)
		allSeries[name] = &seriesData{Name: name, Color: adaptiveColors[i], Dashed: true, Marker: "square"}
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
		allSeries["Theoretical"].Points = append(allSeries["Theoretical"].Points, point{thBPK, eps})
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
			ms = append(ms, m{name, err, bpk, func(a, b uint64) bool { return fCopy.IsEmpty(trieBS(a), trieBS(b)) }})
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
			ms = append(ms, m{"Truncation", nil, float64(fTrunc.SizeInBits()) / float64(n), func(a, b uint64) bool { return fCopy.IsEmpty(trieBS(a), trieBS(b)) }})
		} else {
			ms = append(ms, m{"Truncation", errTrunc, 0, nil})
		}
		// Hybrid
		if errHybrid == nil {
			fCopy := fHybrid
			ms = append(ms, m{"Hybrid", nil, float64(fHybrid.SizeInBits()) / float64(n), func(a, b uint64) bool { return fCopy.IsEmpty(trieBS(a), trieBS(b)) }})
		} else {
			ms = append(ms, m{"Hybrid", errHybrid, 0, nil})
		}

		for _, me := range ms {
			if me.err != nil {
				fmt.Printf("%-6.3f | %-20s | %8s | %12s (err: %v)\n", eps, me.name, "N/A", "N/A", me.err)
				continue
			}
			fpr := measureFPR(clusterU64, queries, me.check)
			allSeries[me.name].Points = append(allSeries[me.name].Points, point{me.bpk, fpr})
			fmt.Fprintf(csvF, "%f,%s,%f,%f\n", eps, me.name, me.bpk, fpr)
			fmt.Printf("%-6.3f | %-20s | %8.2f | %12.6f\n", eps, me.name, me.bpk, fpr)
		}
	}

	orderedSeries := []seriesData{
		*allSeries["Theoretical"],
		*allSeries["Adaptive (t=0)"],
	}
	for _, tv := range tValues {
		orderedSeries = append(orderedSeries, *allSeries[fmt.Sprintf("Adaptive (t=%d)", tv)])
	}
	orderedSeries = append(orderedSeries, *allSeries["SODA"], *allSeries["Truncation"], *allSeries["Hybrid"])

	err := generateTradeoffSVG(
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

func TestScalability_Hybrid(t *testing.T) {
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
			return func(a, b uint64) bool { return f.IsEmpty(trieBS(a), trieBS(b)) }, f.SizeInBits(), "-", nil
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
			return func(a, b uint64) bool { return f.IsEmpty(trieBS(a), trieBS(b)) }, f.SizeInBits(), "-", nil
		}},
		{"Hybrid", func(bs []bits.BitString, u64 []uint64) (func(a, b uint64) bool, uint64, string, error) {
			f, err := are_hybrid.NewHybridARE(bs, rangeLen, eps)
			if err != nil {
				return nil, 0, "", err
			}
			nc, nf, nt := f.Stats()
			info := fmt.Sprintf("%dc/%d%%fb", nc, 100*nf/nt)
			return func(a, b uint64) bool { return f.IsEmpty(trieBS(a), trieBS(b)) }, f.SizeInBits(), info, nil
		}},
	}

	for _, n := range sizes {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			rng := rand.New(rand.NewSource(99))
			keysU64, clusters := generateClusterDistribution(n, nClusters, unifFrac, rng)
			keysBS := make([]bits.BitString, len(keysU64))
			for i, v := range keysU64 {
				keysBS[i] = trieBS(v)
			}

			qrng := rand.New(rand.NewSource(12345))
			queries := generateClusterQueries(queryCount, clusters, unifFrac, rangeLen, qrng)

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
					if !groundTruth(keysU64, a, b) {
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

func TestTradeoff_Hybrid_Uniform(t *testing.T) {
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
		unifBS[i] = trieBS(v)
	}

	qrng := rand.New(rand.NewSource(12345))
	queries := make([][2]uint64, queryCount)
	for i := range queries {
		a := qrng.Uint64()
		queries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	measureFPR := func(keys []uint64, qs [][2]uint64, check func(a, b uint64) bool) float64 {
		fp, total := 0, 0
		for _, q := range qs {
			a, b := q[0], q[1]
			if b < a {
				continue
			}
			if !groundTruth(keys, a, b) {
				continue
			}
			total++
			if !check(a, b) {
				fp++
			}
		}
		if total == 0 {
			return 0
		}
		return float64(fp) / float64(total)
	}

	// Sequential: evenly spaced keys in a narrow range
	seqGap := uint64(1000)
	seqBase := uint64(1) << 40
	seqU64 := make([]uint64, n)
	seqBS := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		v := seqBase + uint64(i)*seqGap
		seqU64[i] = v
		seqBS[i] = trieBS(v)
	}

	qrngS := rand.New(rand.NewSource(12345))
	seqQueries := make([][2]uint64, queryCount)
	for i := range seqQueries {
		a := qrngS.Uint64()
		seqQueries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	allSeries := map[string]*seriesData{
		"Theoretical":       {Name: "Theoretical", Color: "#ef4444", Dashed: true, Marker: "circle"},
		"Adaptive (Unif)":   {Name: "Adaptive (Unif)", Color: "#2a7fff", Marker: "square"},
		"Adaptive (Seq)":    {Name: "Adaptive (Seq)", Color: "#2a7fff", Dashed: true, Marker: "square"},
		"SODA (Unif)":       {Name: "SODA (Unif)", Color: "#22a06b", Marker: "diamond"},
		"SODA (Seq)":        {Name: "SODA (Seq)", Color: "#22a06b", Dashed: true, Marker: "diamond"},
		"Truncation (Unif)": {Name: "Truncation (Unif)", Color: "#e6a800", Marker: "triangle"},
		"Truncation (Seq)":  {Name: "Truncation (Seq)", Color: "#e6a800", Dashed: true, Marker: "triangle"},
		"Hybrid (Unif)":     {Name: "Hybrid (Unif)", Color: "#9b59b6", Marker: "star"},
		"Hybrid (Seq)":      {Name: "Hybrid (Seq)", Color: "#9b59b6", Dashed: true, Marker: "star"},
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
		allSeries["Theoretical"].Points = append(allSeries["Theoretical"].Points, point{thBPK, eps})
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

		add("Adaptive (Unif)", errOptU, safeSize(fOptU), unifU64, queries, func(a, b uint64) bool { return fOptU.IsEmpty(trieBS(a), trieBS(b)) })
		add("Adaptive (Seq)", errOptS, safeSize(fOptS), seqU64, seqQueries, func(a, b uint64) bool { return fOptS.IsEmpty(trieBS(a), trieBS(b)) })
		add("SODA (Unif)", errSodaU, safeSizeSoda(fSodaU), unifU64, queries, func(a, b uint64) bool { return fSodaU.IsEmpty(a, b) })
		add("SODA (Seq)", errSodaS, safeSizeSoda(fSodaS), seqU64, seqQueries, func(a, b uint64) bool { return fSodaS.IsEmpty(a, b) })
		add("Truncation (Unif)", errTruncU, safeSizeTrunc(fTruncU), unifU64, queries, func(a, b uint64) bool { return fTruncU.IsEmpty(trieBS(a), trieBS(b)) })
		add("Truncation (Seq)", errTruncS, safeSizeTrunc(fTruncS), seqU64, seqQueries, func(a, b uint64) bool { return fTruncS.IsEmpty(trieBS(a), trieBS(b)) })
		add("Hybrid (Unif)", errHybridU, safeSizeHybrid(fHybridU), unifU64, queries, func(a, b uint64) bool { return fHybridU.IsEmpty(trieBS(a), trieBS(b)) })
		add("Hybrid (Seq)", errHybridS, safeSizeHybrid(fHybridS), seqU64, seqQueries, func(a, b uint64) bool { return fHybridS.IsEmpty(trieBS(a), trieBS(b)) })

		for _, me := range ms {
			if me.err != nil {
				fmt.Printf("%-6.3f | %-20s | %8s | %12s (err: %v)\n", eps, me.name, "N/A", "N/A", me.err)
				continue
			}
			fpr := measureFPR(me.keys, me.queries, me.check)
			allSeries[me.name].Points = append(allSeries[me.name].Points, point{me.bpk, fpr})
			fmt.Fprintf(csvF, "%f,%s,%f,%f\n", eps, me.name, me.bpk, fpr)
			fmt.Printf("%-6.3f | %-20s | %8.2f | %12.6f\n", eps, me.name, me.bpk, fpr)
		}
	}

	orderedSeries := []seriesData{
		*allSeries["Theoretical"],
		*allSeries["Adaptive (Unif)"],
		*allSeries["Adaptive (Seq)"],
		*allSeries["SODA (Unif)"],
		*allSeries["SODA (Seq)"],
		*allSeries["Truncation (Unif)"],
		*allSeries["Truncation (Seq)"],
		*allSeries["Hybrid (Unif)"],
		*allSeries["Hybrid (Seq)"],
	}

	err := generateTradeoffSVG(
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

func generateTradeoffSVG(title, xLabel, yLabel string, series []seriesData, outPath string) error {
	w, h := 960.0, 600.0
	mL, mR, mT, mB := 90.0, 40.0, 40.0, 50.0
	plotW := w - mL - mR
	plotH := h - mT - mB

	const fprFloor = 1e-6
	for i := range series {
		for j := range series[i].Points {
			if series[i].Points[j].Y <= 0 {
				series[i].Points[j].Y = fprFloor
			}
		}
	}

	minX, maxX := math.Inf(1), math.Inf(-1)
	minLogY, maxLogY := math.Inf(1), math.Inf(-1)
	for _, s := range series {
		for _, p := range s.Points {
			if p.X < minX {
				minX = p.X
			}
			if p.X > maxX {
				maxX = p.X
			}
			ly := math.Log10(p.Y)
			if ly < minLogY {
				minLogY = ly
			}
			if ly > maxLogY {
				maxLogY = ly
			}
		}
	}

	minX = 0
	maxX = math.Ceil(maxX/2) * 2
	minLogY = math.Floor(minLogY) - 0.5
	maxLogY = math.Ceil(maxLogY) + 0.5

	toX := func(x float64) float64 { return mL + plotW*(x-minX)/(maxX-minX) }
	toY := func(y float64) float64 {
		ly := math.Log10(y)
		return mT + plotH*(1-(ly-minLogY)/(maxLogY-minLogY))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f">`+"\n", w, h, w, h))
	sb.WriteString(`<style>text{font-family:Menlo,Monaco,monospace;font-size:12px;fill:#222} .axis{stroke:#333;stroke-width:1} .grid{stroke:#eee;stroke-width:0.5} .label{font-size:11px;fill:#444}</style>` + "\n")

	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="28" text-anchor="middle" style="font-size:14px;font-weight:bold">%s</text>`+"\n", w/2, title))

	sb.WriteString(fmt.Sprintf(`<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, mT+plotH, mL+plotW, mT+plotH))
	sb.WriteString(fmt.Sprintf(`<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, mT, mL, mT+plotH))

	for e := int(math.Ceil(minLogY)); e <= int(math.Floor(maxLogY)); e++ {
		py := mT + plotH*(1-(float64(e)-minLogY)/(maxLogY-minLogY))
		sb.WriteString(fmt.Sprintf(`<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, py, mL+plotW, py))
		var label string
		if e == 0 {
			label = "1"
		} else if e == -1 {
			label = "0.1"
		} else if e == -2 {
			label = "0.01"
		} else if e == -3 {
			label = "10^-3"
		} else {
			label = fmt.Sprintf("10^%d", e)
		}
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.1f" y="%.1f" text-anchor="end">%s</text>`+"\n", mL-8, py+4, label))
	}

	xStep := 2.0
	for x := math.Ceil(minX/xStep) * xStep; x <= maxX; x += xStep {
		px := toX(x)
		sb.WriteString(fmt.Sprintf(`<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", px, mT, px, mT+plotH))
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.1f" y="%.1f" text-anchor="middle">%.0f</text>`+"\n", px, mT+plotH+16, x))
	}

	drawMarker := func(sb *strings.Builder, marker, color string, cx, cy float64) {
		switch marker {
		case "square":
			sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="6" height="6" fill="%s"/>`+"\n", cx-3, cy-3, color))
		case "diamond":
			sb.WriteString(fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"/>`+"\n",
				cx, cy-4, cx+4, cy, cx, cy+4, cx-4, cy, color))
		case "triangle":
			sb.WriteString(fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s"/>`+"\n",
				cx, cy-4, cx+4, cy+3, cx-4, cy+3, color))
		case "star":
			// 5-pointed star
			r1, r2 := 5.0, 2.0
			var pts []string
			for i := 0; i < 10; i++ {
				angle := math.Pi/2 + float64(i)*math.Pi/5
				r := r1
				if i%2 == 1 {
					r = r2
				}
				px := cx + r*math.Cos(angle)
				py := cy - r*math.Sin(angle)
				pts = append(pts, fmt.Sprintf("%.1f,%.1f", px, py))
			}
			sb.WriteString(fmt.Sprintf(`<polygon points="%s" fill="%s"/>`+"\n", strings.Join(pts, " "), color))
		default:
			sb.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="3" fill="%s"/>`+"\n", cx, cy, color))
		}
	}

	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}
		var validPts []point
		for _, p := range s.Points {
			if p.Y <= 0 || p.X < 0.1 {
				continue
			}
			validPts = append(validPts, p)
		}
		if len(validPts) == 0 {
			continue
		}
		var pts []string
		for _, p := range validPts {
			pts = append(pts, fmt.Sprintf("%.1f,%.1f", toX(p.X), toY(p.Y)))
		}
		dash := ""
		if s.Dashed {
			dash = ` stroke-dasharray="8,5"`
		}
		sb.WriteString(fmt.Sprintf(`<polyline fill="none" stroke="%s" stroke-width="2"%s points="%s"/>`+"\n",
			s.Color, dash, strings.Join(pts, " ")))
		marker := s.Marker
		if marker == "" {
			marker = "circle"
		}
		for _, p := range validPts {
			drawMarker(&sb, marker, s.Color, toX(p.X), toY(p.Y))
		}
	}

	ly := mT + 20.0
	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}
		dash := ""
		if s.Dashed {
			dash = ` stroke-dasharray="8,5"`
		}
		lx := mL + plotW - 220
		sb.WriteString(fmt.Sprintf(`<line x1="%.0f" y1="%.0f" x2="%.0f" y2="%.0f" stroke="%s" stroke-width="2"%s/>`+"\n",
			lx, ly, lx+16, ly, s.Color, dash))
		marker := s.Marker
		if marker == "" {
			marker = "circle"
		}
		drawMarker(&sb, marker, s.Color, lx+8, ly)
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.0f" y="%.0f">%s</text>`+"\n", lx+22, ly+4, s.Name))
		ly += 18
	}

	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="%.0f" text-anchor="middle">%s</text>`+"\n", mL+plotW/2, h-10, xLabel))
	sb.WriteString(fmt.Sprintf(`<text transform="translate(16,%.0f) rotate(-90)" text-anchor="middle">%s</text>`+"\n", mT+plotH/2, yLabel))
	sb.WriteString("</svg>\n")

	return os.WriteFile(outPath, []byte(sb.String()), 0644)
}
