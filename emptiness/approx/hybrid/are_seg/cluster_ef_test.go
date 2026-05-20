package are_seg

// TestClusterEF measures the per-cluster Elias-Fano cost
// log2(2^M_c / n_c) versus the universe-wide log2(U/n) on real and
// synthetic distributions at the headline scale n = 2^28. Used to
// quantify the Seg-ARE vs Grafite EF decomposition in the thesis text.
//
// Run with:
//
//	go test -v -run TestClusterEF -timeout 10m ./Thesis/emptiness/approx/hybrid/are_seg/

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
	"os"
	"sort"
	"testing"
)

const benchSosdDir = "/Users/andrei.ogurtsov/Thesis-Bench-industry/bench/sosd_data"
const benchSyntheticDir = "/Users/andrei.ogurtsov/Thesis-Bench-industry/bench/synthetic_data"

func loadUint64File(path string, maxKeys int) ([]uint64, error) {
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

func clusterLocalBits(minKey, maxKey uint64) float64 {
	span := maxKey - minKey + 1
	if span < 2 {
		return 0
	}
	return math.Log2(float64(span))
}

func TestClusterEF(t *testing.T) {
	const (
		rangeLen  = uint64(1)
		epsilon   = 0.01
		requested = 1 << 28
	)

	type dataset struct {
		name string
		load func() ([]uint64, error)
	}
	datasets := []dataset{
		{"sosd_wiki", func() ([]uint64, error) {
			return loadUint64File(benchSosdDir+"/wiki_ts_200M_uint64", requested)
		}},
		{"clustered", func() ([]uint64, error) {
			return loadUint64File(benchSyntheticDir+"/clustered_256M_uint64", requested)
		}},
	}

	eps := segEps(rangeLen, epsilon)
	fmt.Printf("\n=== Cluster EF decomposition ===\n")
	fmt.Printf("L=%d  eps=%.3f  delta=%d  minPts=%d\n\n", rangeLen, epsilon, eps, segMinPts)

	for _, ds := range datasets {
		keys, err := ds.load()
		if err != nil {
			t.Logf("skip %s: %v", ds.name, err)
			continue
		}
		n := len(keys)
		if n < 2 {
			continue
		}

		minKey := keys[0]
		maxKey := keys[n-1]
		globalSpan := maxKey - minKey + 1
		globalBits := math.Log2(float64(globalSpan))
		globalLogUn := math.Log2(float64(globalSpan) / float64(n))
		rawUnivBits := math.Log2(float64(maxKey) + 1)
		rawLogUn := math.Log2((float64(maxKey) + 1) / float64(n))

		// segDelta for n = 2^28 at L=128, eps=0.001.
		thresholdFn := func(m int) uint64 {
			K := kFromParams(n, rangeLen, epsilon)
			kLocal := uint32(K)
			_ = kLocal
			if uint32(bits.Len64(uint64(m))) >= 64 {
				return math.MaxUint64
			}
			// Same threshold the production code uses for the merge step.
			return uint64(1) << uint32(min(K, 63))
		}
		segs, fallback := detectSegments(keys, eps, thresholdFn)
		_ = thresholdFn

		var sumNcLogBits, totalNc float64
		var sumMcMinusLogNc float64
		var minMc, maxMc = math.MaxFloat64, 0.0
		var minNc, maxNc = int(math.MaxInt32), 0
		for _, s := range segs {
			Mc := clusterLocalBits(s.minKey, s.maxKey)
			nc := float64(len(s.keys))
			perKey := Mc - math.Log2(nc)
			sumNcLogBits += nc * perKey
			sumMcMinusLogNc += perKey
			totalNc += nc
			if Mc < minMc {
				minMc = Mc
			}
			if Mc > maxMc {
				maxMc = Mc
			}
			if len(s.keys) < minNc {
				minNc = len(s.keys)
			}
			if len(s.keys) > maxNc {
				maxNc = len(s.keys)
			}
		}

		var perKeyWeighted float64
		if totalNc > 0 {
			perKeyWeighted = sumNcLogBits / totalNc
		}
		var perKeyMean float64
		if len(segs) > 0 {
			perKeyMean = sumMcMinusLogNc / float64(len(segs))
		}

		fbCount := len(fallback)
		fmt.Printf("--- %s ---\n", ds.name)
		fmt.Printf("  n=%d  min=%d  max=%d\n", n, minKey, maxKey)
		fmt.Printf("  local_span=%d  log2(span)=%.2f  log2(U_local/n)=%.2f  (Seg-ARE universe)\n",
			globalSpan, globalBits, globalLogUn)
		fmt.Printf("  raw_max=%d  log2(max)=%.2f   log2(U_raw/n)  =%.2f  (Grafite universe)\n",
			maxKey, rawUnivBits, rawLogUn)
		_ = minKey
		_ = globalSpan
		_ = globalBits
		_ = globalLogUn
		_ = rawUnivBits
		_ = rawLogUn
		fmt.Printf("  segments=%d  in_cluster_keys=%.0f (%.1f%%)  fallback=%d (%.1f%%)\n",
			len(segs), totalNc, 100*totalNc/float64(n), fbCount, 100*float64(fbCount)/float64(n))
		if len(segs) > 0 {
			fmt.Printf("  per-cluster M_c bits: min=%.2f  max=%.2f\n", minMc, maxMc)
			fmt.Printf("  per-cluster n_c:      min=%d   max=%d\n", minNc, maxNc)
			fmt.Printf("  key-weighted  avg log2(2^M_c / n_c) = %.2f  bits\n", perKeyWeighted)
			fmt.Printf("  unweighted    avg log2(2^M_c / n_c) = %.2f  bits\n", perKeyMean)
			fmt.Printf("  savings vs universe-wide EF:  %.2f - %.2f = %.2f  bits/key\n",
				globalLogUn, perKeyWeighted, globalLogUn-perKeyWeighted)
		}
		fmt.Println()
	}
}
