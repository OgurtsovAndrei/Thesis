package are_optimized

import (
	"Thesis/bits"
	"Thesis/emptiness/are"
	"Thesis/emptiness/are_soda_hash"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
)

// trieBS converts uint64 to a 64-bit BitString where integer order = trie (Compare) order.
func trieBS(val uint64) bits.BitString {
	return bits.NewFromTrieUint64(val, 64)
}

// groundTruth returns true if [a,b] is empty in sorted uint64 keys.
func groundTruth(keys []uint64, a, b uint64) bool {
	idx := sort.Search(len(keys), func(i int) bool { return keys[i] >= a })
	return idx == len(keys) || keys[idx] > b
}

type seriesData struct {
	Name   string
	Color  string
	Dashed bool
	Marker string // "circle", "square", "diamond", "triangle"
	Points []point
}

type point struct {
	X, Y float64
}

func TestTradeoff_FPR_vs_BPK(t *testing.T) {
	const (
		n          = 10000
		rangeLen   = uint64(100)
		queryCount = 100_000
	)

	epsilons := []float64{0.1, 0.05, 0.02, 0.01, 0.005, 0.002, 0.001}

	// --- Generate keys (once, reused across epsilons) ---
	rng := rand.New(rand.NewSource(42))

	// Uniform: random uint64 keys
	unifU64 := make([]uint64, 0, n)
	seen := make(map[uint64]bool)
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

	// Sequential: evenly spaced keys in a narrow range
	seqGap := uint64(1000)
	seqBase := uint64(1) << 40
	seqU64 := make([]uint64, n)
	seqBS := make([]bits.BitString, n)
	for i := 0; i < n; i++ {
		v := seqBase + uint64(i)*seqGap
		seqU64[i] = v
		seqBS[i] = trieBS(v) // NewFromTrieUint64 preserves integer order → already sorted by Compare
	}

	// Pre-generate query sets (shared across epsilons for consistency)
	qrngU := rand.New(rand.NewSource(12345))
	unifQueries := make([][2]uint64, queryCount)
	for i := range unifQueries {
		a := qrngU.Uint64()
		unifQueries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	// Sequential queries: same random ranges as uniform (fair comparison — same queries, different keys)
	qrngS := rand.New(rand.NewSource(12345))
	seqQueries := make([][2]uint64, queryCount)
	for i := range seqQueries {
		a := qrngS.Uint64()
		seqQueries[i] = [2]uint64{a, a + rangeLen - 1}
	}

	// Measure FPR with ground truth verification
	measureFPR := func(keys []uint64, queries [][2]uint64, check func(a, b uint64) bool) float64 {
		fp, total := 0, 0
		for _, q := range queries {
			a, b := q[0], q[1]
			if b < a {
				continue // overflow
			}
			if !groundTruth(keys, a, b) {
				continue // skip non-empty ranges
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

	// Collect results per series
	allSeries := map[string]*seriesData{
		"Theoretical":       {Name: "Theoretical", Color: "#ef4444", Dashed: true, Marker: "circle"},
		"Adaptive (Unif)":   {Name: "Adaptive (Unif)", Color: "#2a7fff", Marker: "square"},
		"Adaptive (Seq)":    {Name: "Adaptive (Seq)", Color: "#2a7fff", Dashed: true, Marker: "square"},
		"SODA (Unif)":       {Name: "SODA (Unif)", Color: "#22a06b", Marker: "diamond"},
		"SODA (Seq)":        {Name: "SODA (Seq)", Color: "#22a06b", Dashed: true, Marker: "diamond"},
		"Truncation (Unif)": {Name: "Truncation (Unif)", Color: "#e6a800", Marker: "triangle"},
		"Truncation (Seq)":  {Name: "Truncation (Seq)", Color: "#e6a800", Dashed: true, Marker: "triangle"},
	}

	// CSV output
	os.MkdirAll("../../bench_results/plots", 0755)
	csvF, _ := os.Create("../../bench_results/plots/are_tradeoff_data.csv")
	defer csvF.Close()
	fmt.Fprintln(csvF, "Epsilon,Series,BPK,FPR")

	// Header
	fmt.Printf("\n%-6s | %-20s | %8s | %12s\n", "Eps", "Series", "BPK", "FPR")
	fmt.Println(strings.Repeat("-", 55))

	for _, eps := range epsilons {
		// Theoretical: BPK = lg(L/eps), FPR = eps
		thBPK := math.Log2(float64(rangeLen) / eps)
		allSeries["Theoretical"].Points = append(allSeries["Theoretical"].Points, point{thBPK, eps})
		fmt.Fprintf(csvF, "%f,Theoretical,%f,%f\n", eps, thBPK, eps)
		fmt.Printf("%-6.3f | %-20s | %8.2f | %12.6f\n", eps, "Theoretical", thBPK, eps)

		// --- Build filters ---
		fOptU, errOptU := NewOptimizedARE(unifBS, rangeLen, eps, 0)
		fOptS, errOptS := NewOptimizedARE(seqBS, rangeLen, eps, 0)
		fSodaU, errSodaU := are_soda_hash.NewApproximateRangeEmptinessSoda(unifU64, rangeLen, eps)
		fSodaS, errSodaS := are_soda_hash.NewApproximateRangeEmptinessSoda(seqU64, rangeLen, eps)
		fTruncU, errTruncU := are.NewApproximateRangeEmptiness(unifBS, eps)
		fTruncS, errTruncS := are.NewApproximateRangeEmptiness(seqBS, eps)

		// --- Measure each filter ---
		type measurement struct {
			name    string
			err     error
			bpk     float64
			fpr     float64
			keys    []uint64
			queries [][2]uint64
		}

		measures := []measurement{
			{name: "Adaptive (Unif)", err: errOptU, keys: unifU64, queries: unifQueries},
			{name: "Adaptive (Seq)", err: errOptS, keys: seqU64, queries: seqQueries},
			{name: "SODA (Unif)", err: errSodaU, keys: unifU64, queries: unifQueries},
			{name: "SODA (Seq)", err: errSodaS, keys: seqU64, queries: seqQueries},
			{name: "Truncation (Unif)", err: errTruncU, keys: unifU64, queries: unifQueries},
			{name: "Truncation (Seq)", err: errTruncS, keys: seqU64, queries: seqQueries},
		}

		// Set BPK and measure function per filter
		if errOptU == nil {
			measures[0].bpk = float64(fOptU.SizeInBits()) / float64(n)
		}
		if errOptS == nil {
			measures[1].bpk = float64(fOptS.SizeInBits()) / float64(n)
		}
		if errSodaU == nil {
			measures[2].bpk = float64(fSodaU.SizeInBits()) / float64(n)
		}
		if errSodaS == nil {
			measures[3].bpk = float64(fSodaS.SizeInBits()) / float64(n)
		}
		if errTruncU == nil {
			measures[4].bpk = float64(fTruncU.SizeInBits()) / float64(n)
		}
		if errTruncS == nil {
			measures[5].bpk = float64(fTruncS.SizeInBits()) / float64(n)
		}

		// FPR measurements
		checkFns := []func(a, b uint64) bool{
			func(a, b uint64) bool { return fOptU.IsEmpty(trieBS(a), trieBS(b)) },
			func(a, b uint64) bool { return fOptS.IsEmpty(trieBS(a), trieBS(b)) },
			func(a, b uint64) bool { return fSodaU.IsEmpty(a, b) },
			func(a, b uint64) bool { return fSodaS.IsEmpty(a, b) },
			func(a, b uint64) bool { return fTruncU.IsEmpty(trieBS(a), trieBS(b)) },
			func(a, b uint64) bool { return fTruncS.IsEmpty(trieBS(a), trieBS(b)) },
		}

		for i := range measures {
			m := &measures[i]
			if m.err != nil {
				fmt.Printf("%-6.3f | %-20s | %8s | %12s (err: %v)\n", eps, m.name, "N/A", "N/A", m.err)
				continue
			}
			m.fpr = measureFPR(m.keys, m.queries, checkFns[i])

			allSeries[m.name].Points = append(allSeries[m.name].Points, point{m.bpk, m.fpr})
			fmt.Fprintf(csvF, "%f,%s,%f,%f\n", eps, m.name, m.bpk, m.fpr)
			fmt.Printf("%-6.3f | %-20s | %8.2f | %12.6f\n", eps, m.name, m.bpk, m.fpr)
		}
	}

	// Generate SVG
	orderedSeries := []seriesData{
		*allSeries["Theoretical"],
		*allSeries["Adaptive (Unif)"],
		*allSeries["Adaptive (Seq)"],
		*allSeries["SODA (Unif)"],
		*allSeries["SODA (Seq)"],
		*allSeries["Truncation (Unif)"],
		*allSeries["Truncation (Seq)"],
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

type clusterInfo struct {
	center uint64
	stddev float64
}

// generateClusterDistribution creates n keys: unifFrac uniform + rest from Gaussian clusters.
// Returns sorted keys and cluster metadata (for query generation).
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
				continue // overflow/underflow sentinel
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

// sampleGaussian returns center + NormFloat64()*stddev, with overflow protection.
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

// generateClusterQueries generates queries matching the cluster distribution:
// unifFrac fully random, rest near cluster centers.
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

func TestTradeoff_FPR_vs_BPK_Cluster(t *testing.T) {
	const (
		n          = 10000
		rangeLen   = uint64(100)
		queryCount = 100_000
		nClusters  = 5
	)

	epsilons := []float64{0.1, 0.05, 0.02, 0.01, 0.005, 0.002, 0.001}

	const unifFrac = 0.15 // 15% uniform background, 85% from clusters

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
			f, err := NewOptimizedARE(clusterBS, rangeLen, eps, tv)
			var bpk float64
			if err == nil {
				bpk = float64(f.SizeInBits()) / float64(n)
			}
			fCopy := f
			ms = append(ms, m{name, err, bpk, func(a, b uint64) bool { return fCopy.IsEmpty(trieBS(a), trieBS(b)) }})
		}

		// SODA and Truncation
		if errSoda == nil {
			fCopy := fSoda
			ms = append(ms, m{"SODA", nil, float64(fSoda.SizeInBits()) / float64(n), func(a, b uint64) bool { return fCopy.IsEmpty(a, b) }})
		} else {
			ms = append(ms, m{"SODA", errSoda, 0, nil})
		}
		if errTrunc == nil {
			fCopy := fTrunc
			ms = append(ms, m{"Truncation", nil, float64(fTrunc.SizeInBits()) / float64(n), func(a, b uint64) bool { return fCopy.IsEmpty(trieBS(a), trieBS(b)) }})
		} else {
			ms = append(ms, m{"Truncation", errTrunc, 0, nil})
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
	orderedSeries = append(orderedSeries, *allSeries["SODA"], *allSeries["Truncation"])

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

func generateTradeoffSVG(title, xLabel, yLabel string, series []seriesData, outPath string) error {
	w, h := 960.0, 600.0
	mL, mR, mT, mB := 90.0, 40.0, 40.0, 50.0
	plotW := w - mL - mR
	plotH := h - mT - mB

	// Clamp FPR=0 to a minimum value for log-scale plotting
	const fprFloor = 1e-6
	for i := range series {
		for j := range series[i].Points {
			if series[i].Points[j].Y <= 0 {
				series[i].Points[j].Y = fprFloor
			}
		}
	}

	// Find data ranges
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

	// Set axis ranges
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

	// Title
	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="28" text-anchor="middle" style="font-size:14px;font-weight:bold">%s</text>`+"\n", w/2, title))

	// Axes
	sb.WriteString(fmt.Sprintf(`<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, mT+plotH, mL+plotW, mT+plotH))
	sb.WriteString(fmt.Sprintf(`<line class="axis" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", mL, mT, mL, mT+plotH))

	// Y grid (log scale)
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

	// X grid
	xStep := 2.0
	for x := math.Ceil(minX/xStep) * xStep; x <= maxX; x += xStep {
		px := toX(x)
		sb.WriteString(fmt.Sprintf(`<line class="grid" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`+"\n", px, mT, px, mT+plotH))
		sb.WriteString(fmt.Sprintf(`<text class="label" x="%.1f" y="%.1f" text-anchor="middle">%.0f</text>`+"\n", px, mT+plotH+16, x))
	}

	// Helper to draw a marker at (cx, cy)
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
		default: // circle
			sb.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="3" fill="%s"/>`+"\n", cx, cy, color))
		}
	}

	// Series
	for _, s := range series {
		if len(s.Points) == 0 {
			continue
		}
		// Filter valid points (skip degenerate BPK < 0.1)
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

	// Legend
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

	// Axis labels
	sb.WriteString(fmt.Sprintf(`<text x="%.0f" y="%.0f" text-anchor="middle">%s</text>`+"\n", mL+plotW/2, h-10, xLabel))
	sb.WriteString(fmt.Sprintf(`<text transform="translate(16,%.0f) rotate(-90)" text-anchor="middle">%s</text>`+"\n", mT+plotH/2, yLabel))
	sb.WriteString("</svg>\n")

	return os.WriteFile(outPath, []byte(sb.String()), 0644)
}
