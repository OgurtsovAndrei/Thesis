package are_dbscan

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"

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

	j := 0
	for i := 1; i < len(raw); i++ {
		if raw[i] != raw[j] {
			j++
			raw[j] = raw[i]
		}
	}
	return raw[:j+1], nil
}

// generateBenchQueries returns queryCount uniform random queries of width rangeLen.
func generateBenchQueries(queryCount int, rangeLen uint64, rng *rand.Rand) [][2]uint64 {
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

// measureBuildHybridScan builds HybridScanARE benchBuildRuns times and returns
// the built filter plus median build throughput in Mkeys/s.
func measureBuildHybridScan(keys []uint64, rangeLen uint64, eps float64) (*HybridScanARE, float64, error) {
	n := len(keys)
	K := kFromEps(n, rangeLen, eps)
	durations := make([]time.Duration, benchBuildRuns)
	var last *HybridScanARE
	for r := 0; r < benchBuildRuns; r++ {
		start := time.Now()
		f, err := NewHybridScanARE(keys, 64, Config{K: K})
		durations[r] = time.Since(start)
		if err != nil {
			return nil, 0, err
		}
		last = f
	}
	med := medianDuration(durations)
	return last, float64(n) / med.Seconds() / 1e6, nil
}

// measureQueryHybridScan runs benchQueryRuns rounds of queryCount IsEmpty calls on
// hybridScanFilter, returns median query throughput in Mqueries/s.
func measureQueryHybridScan(filter *HybridScanARE, queries [][2]uint64) float64 {
	queryCount := len(queries)
	durations := make([]time.Duration, benchQueryRuns)
	for r := 0; r < benchQueryRuns; r++ {
		start := time.Now()
		for _, q := range queries {
			filter.IsEmpty(q[0], q[1])
		}
		durations[r] = time.Since(start)
	}
	return float64(queryCount) / medianDuration(durations).Seconds() / 1e6
}

// runBench executes one (distribution × epsilon) benchmark and prints results.
func runBench(b *testing.B, distName string, ds keyDataset, eps float64) {
	b.Helper()
	keys := ds.keys
	n := len(keys)
	if n == 0 {
		b.Logf("[%s eps=%.3f] skipping: 0 keys after dedup+mask", distName, eps)
		return
	}

	qrng := rand.New(rand.NewSource(98765))
	var queries [][2]uint64
	if ds.clusters != nil {
		queries = testutils.GenerateClusterQueries(benchQueryCount, ds.clusters, 0.15, benchRangeLen, qrng)
	} else {
		queries = generateBenchQueries(benchQueryCount, benchRangeLen, qrng)
	}

	filter, buildMkps, err := measureBuildHybridScan(keys, benchRangeLen, eps)
	if err != nil {
		b.Errorf("[%s eps=%.3f] HybridScanARE build: %v", distName, eps, err)
		return
	}

	fpr := testutils.MeasureFPR(keys, queries, filter.IsEmpty)
	bpk := float64(filter.SizeInBits()) / float64(n)
	queryMqps := measureQueryHybridScan(filter, queries)
	nc, nf, _ := filter.Stats()

	fmt.Printf("\nDistribution: %s, eps=%.3f, N=%d\n", distName, eps, n)
	fmt.Printf("%-24s  %12.5f\n", "FPR:", fpr)
	fmt.Printf("%-24s  %12.2f\n", "BPK:", bpk)
	fmt.Printf("%-24s  %12.2f\n", "Build (Mkeys/s):", buildMkps)
	fmt.Printf("%-24s  %12.2f\n", "Query (Mq/s):", queryMqps)
	fmt.Printf("%-24s  %12d\n", "Clusters:", nc)
	fmt.Printf("%-24s  %12d\n", "Fallback keys:", nf)
}

// Driven by its own distribution×epsilon sweep, not by b.N — invoke with -benchtime=1x.
func BenchmarkHybridScanComparison(b *testing.B) {
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
		b.Run(dist.name, func(b *testing.B) {
			ds, ok := dist.load()
			if !ok {
				b.Skipf("dataset %q not available (file missing or unreadable)", dist.name)
			}
			b.Logf("Loaded %d keys for distribution %q", len(ds.keys), dist.name)

			for _, eps := range benchEpsilons {
				eps := eps
				b.Run(fmt.Sprintf("eps=%.3f", eps), func(b *testing.B) {
					runBench(b, dist.name, ds, eps)
				})
			}
		})
	}
}
