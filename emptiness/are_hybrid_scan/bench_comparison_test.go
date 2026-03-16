package are_hybrid_scan

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

	"Thesis/bits"
	"Thesis/emptiness/are_hybrid"
	"Thesis/testutils"
)

const (
	benchN          = 1_000_000
	benchRangeLen   = uint64(128)
	benchQueryCount = 200_000
	benchBuildRuns  = 3
	benchQueryRuns  = 3
	mask60bits      = (uint64(1) << 60) - 1
)

var benchEpsilons = []float64{0.1, 0.01, 0.001}

// keyDataset holds a sorted, deduplicated uint64 key slice along with optional
// cluster metadata for generating cluster-aware queries.
type keyDataset struct {
	keys     []uint64
	clusters []testutils.ClusterInfo
}

// makeBSSlice converts a sorted uint64 slice to a []bits.BitString using 64-bit trie encoding.
func makeBSSlice(keys []uint64) []bits.BitString {
	bs := make([]bits.BitString, len(keys))
	for i, k := range keys {
		bs[i] = testutils.TrieBS(k)
	}
	return bs
}

// medianDuration returns the median of a slice of durations.
func medianDuration(ds []time.Duration) time.Duration {
	cp := make([]time.Duration, len(ds))
	copy(cp, ds)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	return cp[len(cp)/2]
}

// generateUniformDataset generates n unique uint64 keys masked to 60 bits, sorted.
func generateUniformDataset(n int, rng *rand.Rand) keyDataset {
	seen := make(map[uint64]bool, n)
	keys := make([]uint64, 0, n)
	for len(keys) < n {
		v := rng.Uint64() & mask60bits
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keyDataset{keys: keys}
}

// generateClusteredDataset generates a cluster-distributed key set masked to 60 bits.
func generateClusteredDataset(n int, rng *rand.Rand) keyDataset {
	raw, clusterInfos := testutils.GenerateClusterDistribution(n, 5, 0.15, rng)
	seen := make(map[uint64]bool, len(raw))
	keys := make([]uint64, 0, len(raw))
	for _, k := range raw {
		k &= mask60bits
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keyDataset{keys: keys, clusters: clusterInfos}
}

// generateSequentialDataset generates n sequential keys: base + i*gap, masked to 60 bits.
func generateSequentialDataset(n int) keyDataset {
	const base = uint64(1000)
	const gap = uint64(1000)
	keys := make([]uint64, n)
	for i := range keys {
		keys[i] = (base + uint64(i)*gap) & mask60bits
	}
	return keyDataset{keys: keys}
}

// generateZipfianDataset generates n unique Zipfian(s=1.5, v=1, imax=2^40) keys
// masked to 60 bits, sorted.
func generateZipfianDataset(n int, rng *rand.Rand) keyDataset {
	const imax = uint64(1) << 40
	z := rand.NewZipf(rng, 1.5, 1, imax)
	seen := make(map[uint64]bool, n)
	keys := make([]uint64, 0, n)
	for len(keys) < n {
		v := z.Uint64() & mask60bits
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keyDataset{keys: keys}
}

// loadSOSD reads up to maxKeys uint64 values from a SOSD binary file.
// Format: [uint64 count (LE)][count × uint64 keys (LE)].
// Returns sorted, deduplicated keys masked to 60 bits.
func loadSOSD(path string, maxKeys int) ([]uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var count uint64
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}

	readN := int(count)
	if maxKeys > 0 && maxKeys < readN {
		readN = maxKeys
	}

	raw := make([]uint64, readN)
	if err := binary.Read(f, binary.LittleEndian, raw); err != nil {
		return nil, fmt.Errorf("read keys: %w", err)
	}

	for i := range raw {
		raw[i] &= mask60bits
	}
	sort.Slice(raw, func(i, j int) bool { return raw[i] < raw[j] })

	// Deduplicate in place.
	j := 0
	for i := 1; i < len(raw); i++ {
		if raw[i] != raw[j] {
			j++
			raw[j] = raw[i]
		}
	}
	return raw[:j+1], nil
}

// generateUniformQueries returns queryCount uniform random queries of width rangeLen.
func generateUniformQueries(queryCount int, rangeLen uint64, rng *rand.Rand) [][2]uint64 {
	queries := make([][2]uint64, queryCount)
	for i := range queries {
		a := rng.Uint64() & mask60bits
		b := a + rangeLen - 1
		if b < a {
			b = ^uint64(0)
		}
		queries[i] = [2]uint64{a, b}
	}
	return queries
}

// measureBuildHybrid builds are_hybrid.HybridARE benchBuildRuns times and returns
// the built filter plus median build throughput in Mkeys/s.
func measureBuildHybrid(keys []uint64, rangeLen uint64, eps float64) (*are_hybrid.HybridARE, float64, error) {
	n := len(keys)
	durations := make([]time.Duration, benchBuildRuns)
	var last *are_hybrid.HybridARE
	for r := 0; r < benchBuildRuns; r++ {
		bs := makeBSSlice(keys)
		start := time.Now()
		f, err := are_hybrid.NewHybridARE(bs, rangeLen, eps)
		durations[r] = time.Since(start)
		if err != nil {
			return nil, 0, err
		}
		last = f
	}
	med := medianDuration(durations)
	return last, float64(n) / med.Seconds() / 1e6, nil
}

// measureBuildHybridScan builds HybridScanARE benchBuildRuns times and returns
// the built filter plus median build throughput in Mkeys/s.
func measureBuildHybridScan(keys []uint64, rangeLen uint64, eps float64) (*HybridScanARE, float64, error) {
	n := len(keys)
	durations := make([]time.Duration, benchBuildRuns)
	var last *HybridScanARE
	for r := 0; r < benchBuildRuns; r++ {
		bs := makeBSSlice(keys)
		start := time.Now()
		f, err := NewHybridScanARE(bs, rangeLen, eps)
		durations[r] = time.Since(start)
		if err != nil {
			return nil, 0, err
		}
		last = f
	}
	med := medianDuration(durations)
	return last, float64(n) / med.Seconds() / 1e6, nil
}

// measureQueryHybrid runs benchQueryRuns rounds of queryCount IsEmpty calls on
// hybridFilter, returns median query throughput in Mqueries/s.
func measureQueryHybrid(filter *are_hybrid.HybridARE, queries [][2]uint64) float64 {
	queryCount := len(queries)
	durations := make([]time.Duration, benchQueryRuns)
	for r := 0; r < benchQueryRuns; r++ {
		start := time.Now()
		for _, q := range queries {
			filter.IsEmpty(testutils.TrieBS(q[0]), testutils.TrieBS(q[1]))
		}
		durations[r] = time.Since(start)
	}
	return float64(queryCount) / medianDuration(durations).Seconds() / 1e6
}

// measureQueryHybridScan runs benchQueryRuns rounds of queryCount IsEmpty calls on
// hybridScanFilter, returns median query throughput in Mqueries/s.
func measureQueryHybridScan(filter *HybridScanARE, queries [][2]uint64) float64 {
	queryCount := len(queries)
	durations := make([]time.Duration, benchQueryRuns)
	for r := 0; r < benchQueryRuns; r++ {
		start := time.Now()
		for _, q := range queries {
			filter.IsEmpty(testutils.TrieBS(q[0]), testutils.TrieBS(q[1]))
		}
		durations[r] = time.Since(start)
	}
	return float64(queryCount) / medianDuration(durations).Seconds() / 1e6
}

// runComparison executes one (distribution × epsilon) comparison and prints a side-by-side table.
func runComparison(t *testing.T, distName string, ds keyDataset, eps float64) {
	t.Helper()
	keys := ds.keys
	n := len(keys)
	if n == 0 {
		t.Logf("[%s eps=%.3f] skipping: 0 keys after dedup+mask", distName, eps)
		return
	}

	qrng := rand.New(rand.NewSource(98765))
	var queries [][2]uint64
	if ds.clusters != nil {
		queries = testutils.GenerateClusterQueries(benchQueryCount, ds.clusters, 0.15, benchRangeLen, qrng)
	} else {
		queries = generateUniformQueries(benchQueryCount, benchRangeLen, qrng)
	}

	hybridFilter, hybridBuildMkps, err := measureBuildHybrid(keys, benchRangeLen, eps)
	if err != nil {
		t.Errorf("[%s eps=%.3f] HybridARE build: %v", distName, eps, err)
		return
	}

	hybridScanFilter, hybridScanBuildMkps, err := measureBuildHybridScan(keys, benchRangeLen, eps)
	if err != nil {
		t.Errorf("[%s eps=%.3f] HybridScanARE build: %v", distName, eps, err)
		return
	}

	hybridFPR := testutils.MeasureFPR(keys, queries, func(a, b uint64) bool {
		return hybridFilter.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b))
	})
	hybridScanFPR := testutils.MeasureFPR(keys, queries, func(a, b uint64) bool {
		return hybridScanFilter.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b))
	})

	hybridBPK := float64(hybridFilter.SizeInBits()) / float64(n)
	hybridScanBPK := float64(hybridScanFilter.SizeInBits()) / float64(n)

	hybridQueryMqps := measureQueryHybrid(hybridFilter, queries)
	hybridScanQueryMqps := measureQueryHybridScan(hybridScanFilter, queries)

	hNC, hNF, _ := hybridFilter.Stats()
	hsNC, hsNF, _ := hybridScanFilter.Stats()

	fmt.Printf("\nDistribution: %s, eps=%.3f, N=%d\n", distName, eps, n)
	fmt.Printf("%-24s  %12s  %12s\n", "", "Hybrid", "HybridScan")
	fmt.Printf("%-24s  %12.5f  %12.5f\n", "FPR:", hybridFPR, hybridScanFPR)
	fmt.Printf("%-24s  %12.2f  %12.2f\n", "BPK:", hybridBPK, hybridScanBPK)
	fmt.Printf("%-24s  %12.2f  %12.2f\n", "Build (Mkeys/s):", hybridBuildMkps, hybridScanBuildMkps)
	fmt.Printf("%-24s  %12.2f  %12.2f\n", "Query (Mq/s):", hybridQueryMqps, hybridScanQueryMqps)
	fmt.Printf("%-24s  %12d  %12d\n", "Clusters:", hNC, hsNC)
	fmt.Printf("%-24s  %12d  %12d\n", "Fallback keys:", hNF, hsNF)
}

func TestBenchComparison(t *testing.T) {
	const sosdDir = "/Users/andrei.ogurtsov/Thesis-Bench-industry/bench/sosd_data"

	type distEntry struct {
		name string
		load func() (keyDataset, bool)
	}

	distributions := []distEntry{
		{
			name: "uniform",
			load: func() (keyDataset, bool) {
				return generateUniformDataset(benchN, rand.New(rand.NewSource(42))), true
			},
		},
		{
			name: "clustered",
			load: func() (keyDataset, bool) {
				return generateClusteredDataset(benchN, rand.New(rand.NewSource(77))), true
			},
		},
		{
			name: "sequential",
			load: func() (keyDataset, bool) {
				return generateSequentialDataset(benchN), true
			},
		},
		{
			name: "zipfian",
			load: func() (keyDataset, bool) {
				return generateZipfianDataset(benchN, rand.New(rand.NewSource(13))), true
			},
		},
		{
			name: "sosd_facebook",
			load: func() (keyDataset, bool) {
				keys, err := loadSOSD(sosdDir+"/fb_200M_uint64", benchN)
				if err != nil {
					return keyDataset{}, false
				}
				return keyDataset{keys: keys}, true
			},
		},
		{
			name: "sosd_wiki_ts",
			load: func() (keyDataset, bool) {
				keys, err := loadSOSD(sosdDir+"/wiki_ts_200M_uint64", benchN)
				if err != nil {
					return keyDataset{}, false
				}
				return keyDataset{keys: keys}, true
			},
		},
		{
			name: "sosd_osm",
			load: func() (keyDataset, bool) {
				keys, err := loadSOSD(sosdDir+"/osm_cellids_200M_uint64", benchN)
				if err != nil {
					return keyDataset{}, false
				}
				return keyDataset{keys: keys}, true
			},
		},
	}

	for _, dist := range distributions {
		dist := dist
		t.Run(dist.name, func(t *testing.T) {
			ds, ok := dist.load()
			if !ok {
				t.Skipf("dataset %q not available (file missing or unreadable)", dist.name)
			}
			t.Logf("Loaded %d keys for distribution %q", len(ds.keys), dist.name)

			for _, eps := range benchEpsilons {
				eps := eps
				t.Run(fmt.Sprintf("eps=%.3f", eps), func(t *testing.T) {
					runComparison(t, dist.name, ds, eps)
				})
			}
		})
	}
}
