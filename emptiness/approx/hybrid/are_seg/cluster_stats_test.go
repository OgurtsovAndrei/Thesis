package are_seg

// TestClusterStats measures how many segments detectSegments finds on real and
// synthetic distributions for minPts ∈ {256, 512}. Run with:
//
//	go test -v -run TestClusterStats -timeout 2m

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"

	"Thesis/testutils"
)

const sosdDir = "/Users/andrei.ogurtsov/Thesis-Bench-industry/bench/sosd_data"

func detectSegmentsN(keys []uint64, eps uint64, minPts int) ([]segment, []uint64) {
	n := len(keys)
	if n < 2 {
		return nil, append([]uint64(nil), keys...)
	}

	type run struct{ start, end int }
	var runs []run

	left := 0
	runStart := -1
	for right := 0; right < n; right++ {
		for keys[right]-keys[left] > eps {
			left++
		}
		if right-left+1 >= minPts {
			if runStart < 0 {
				runStart = left
			}
		} else if runStart >= 0 {
			runs = append(runs, run{runStart, right - 1})
			runStart = -1
		}
	}
	if runStart >= 0 {
		runs = append(runs, run{runStart, n - 1})
	}

	merged := make([]run, 0, len(runs))
	for _, r := range runs {
		if len(merged) > 0 && keys[r.start]-keys[merged[len(merged)-1].end] <= eps {
			merged[len(merged)-1].end = r.end
		} else {
			merged = append(merged, r)
		}
	}

	assigned := make([]bool, n)
	segs := make([]segment, 0, len(merged))
	for _, r := range merged {
		segs = append(segs, segment{
			keys:   keys[r.start : r.end+1],
			minKey: keys[r.start],
			maxKey: keys[r.end],
		})
		for j := r.start; j <= r.end; j++ {
			assigned[j] = true
		}
	}

	var fallback []uint64
	for i, k := range keys {
		if !assigned[i] {
			fallback = append(fallback, k)
		}
	}
	return segs, fallback
}

func loadSOSDStats(path string, maxKeys int) ([]uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var count uint64
	if err := binary.Read(f, binary.LittleEndian, &count); err != nil {
		return nil, err
	}
	readN := int(count)
	if maxKeys > 0 && maxKeys < readN {
		readN = maxKeys
	}
	raw := make([]uint64, readN)
	if err := binary.Read(f, binary.LittleEndian, raw); err != nil {
		return nil, err
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

func printStats(name string, keys []uint64, segs []segment, fallback []uint64) {
	n := len(keys)
	nc := len(segs)
	nInCluster := n - len(fallback)

	var minSize, maxSize int = math.MaxInt32, 0
	var totalInCluster int
	for _, s := range segs {
		sz := len(s.keys)
		if sz < minSize {
			minSize = sz
		}
		if sz > maxSize {
			maxSize = sz
		}
		totalInCluster += sz
	}
	avgSize := 0.0
	if nc > 0 {
		avgSize = float64(totalInCluster) / float64(nc)
		_ = minSize
	} else {
		minSize = 0
	}

	fmt.Printf("  %-14s  clusters=%3d  in_cluster=%7d (%4.0f%%)  fallback=%7d  avg_sz=%6.0f  min=%5d  max=%7d\n",
		name, nc, nInCluster, 100*float64(nInCluster)/float64(n),
		len(fallback), avgSize, minSize, maxSize)
}

func TestClusterStats(t *testing.T) {
	const N = 1_000_000
	const rangeLen = uint64(65536)
	const epsilon = 0.01

	type dataset struct {
		name string
		load func() []uint64
	}

	rng := rand.New(rand.NewSource(42))

	datasets := []dataset{
		{
			name: "sosd_fb",
			load: func() []uint64 {
				keys, err := loadSOSDStats(sosdDir+"/fb_200M_uint64", N)
				if err != nil {
					t.Logf("skip sosd_fb: %v", err)
					return nil
				}
				return keys
			},
		},
		{
			name: "sosd_wiki",
			load: func() []uint64 {
				keys, err := loadSOSDStats(sosdDir+"/wiki_ts_200M_uint64", N)
				if err != nil {
					t.Logf("skip sosd_wiki: %v", err)
					return nil
				}
				return keys
			},
		},
		{
			name: "sosd_osm",
			load: func() []uint64 {
				keys, err := loadSOSDStats(sosdDir+"/osm_cellids_800M_uint64", N)
				if err != nil {
					t.Logf("skip sosd_osm: %v", err)
					return nil
				}
				return keys
			},
		},
		{
			name: "sosd_books",
			load: func() []uint64 {
				keys, err := loadSOSDStats(sosdDir+"/books_800M_uint64", N)
				if err != nil {
					t.Logf("skip sosd_books: %v", err)
					return nil
				}
				return keys
			},
		},
		{
			name: "clustered",
			load: func() []uint64 {
				raw, _ := testutils.GenerateClusterDistribution(N, 5, 0.15, rng)
				sort.Slice(raw, func(i, j int) bool { return raw[i] < raw[j] })
				j := 0
				for i := 1; i < len(raw); i++ {
					if raw[i] != raw[j] {
						j++
						raw[j] = raw[i]
					}
				}
				return raw[:j+1]
			},
		},
	}

	eps := segEps(rangeLen, epsilon)
	fmt.Printf("\nL=%d  ε=%.3f  δ=%d\n\n", rangeLen, epsilon, eps)

	for _, minPts := range []int{256, 512} {
		fmt.Printf("=== minPts = %d ===\n", minPts)
		for _, ds := range datasets {
			keys := ds.load()
			if keys == nil {
				continue
			}
			segs, fallback := detectSegmentsN(keys, eps, minPts)
			printStats(ds.name, keys, segs, fallback)
		}
		fmt.Println()
	}
}
