package are_seg

// TestRealClusterMem builds AdaptiveARE sub-filters on each detected
// segment of SOSD-Wiki and the synthetic clustered file, then reports
// actual per-cluster memory (AdaptiveARE.SizeInBits) versus the
// theoretical EF lower-part bound log2(2^M_c / n_c). Used to back the
// thesis claim that the lower-part formula underestimates real memory:
// the ERE backend at minimum stores a 1-bit-per-key bitmap plus a
// rank/select index.
//
// Run with:
//
//	go test -v -run TestRealClusterMem -timeout 15m ./Thesis/emptiness/approx/hybrid/are_seg/

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/bits"
	"os"
	"sort"
	"testing"

	"Thesis/emptiness/approx/are_adaptive"
	"Thesis/emptiness/approx/hybrid/hybridutil"
	"Thesis/emptiness/exact"
)

func loadKeysRealMem(path string, maxKeys int) ([]uint64, error) {
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

func TestRealClusterMem(t *testing.T) {
	const (
		rangeLen  = uint64(128)
		epsilon   = 0.01
		requested = 1 << 28
	)
	type dataset struct {
		name string
		path string
	}
	datasets := []dataset{
		{"sosd_wiki", benchSosdDir + "/wiki_ts_200M_uint64"},
		{"clustered", benchSyntheticDir + "/clustered_256M_uint64"},
	}

	for _, ds := range datasets {
		keys, err := loadKeysRealMem(ds.path, requested)
		if err != nil {
			t.Logf("skip %s: %v", ds.name, err)
			continue
		}
		n := len(keys)
		fullFilter, err := NewSegARE(keys, 64, rangeLen, epsilon)
		if err != nil {
			t.Fatalf("%s: NewSegARE: %v", ds.name, err)
		}
		fullBpk := float64(fullFilter.SizeInBits()) / float64(n)
		nClFull, fbFull, _ := fullFilter.Stats()
		fmt.Printf("\n>>> %s: full SegARE.SizeInBits() = %.3f BPK over n=%d (clusters=%d, fallback=%d)\n",
			ds.name, fullBpk, n, nClFull, fbFull)

		K := kFromParams(n, rangeLen, epsilon)
		eps := segEps(rangeLen, epsilon)
		thresholdFn := func(m int) uint64 {
			kLocal := hybridutil.LocalK(K, n, m)
			if kLocal >= 64 {
				return math.MaxUint64
			}
			if uint32(bits.Len64(uint64(m))) >= 64 {
				return math.MaxUint64
			}
			return uint64(1) << uint32(min(int(K), 63))
		}
		segs, fallback := detectSegments(keys, eps, thresholdFn)

		var (
			sumRealBits     uint64
			sumNc           uint64
			sumTheoryBits   float64
			minBpk          = math.MaxFloat64
			maxBpk          = 0.0
			minNc, maxNc    = math.MaxInt32, 0
			oneBitMinFloor  = 0.0
		)
		for _, s := range segs {
			nc := len(s.keys)
			if nc == 0 {
				continue
			}
			kLocal := hybridutil.LocalK(K, n, nc)
			are, err := are_adaptive.NewAdaptiveAREFromKWithBackend(s.keys, 64, kLocal, 0, exact.VariantAuto)
			if err != nil {
				t.Fatalf("%s: build cluster: %v", ds.name, err)
			}
			realBits := are.SizeInBits()
			sumRealBits += realBits
			sumNc += uint64(nc)
			span := s.maxKey - s.minKey + 1
			Mc := math.Log2(float64(span))
			theoryLower := math.Max(0, Mc-math.Log2(float64(nc)))
			sumTheoryBits += theoryLower * float64(nc)
			bpk := float64(realBits) / float64(nc)
			if bpk < minBpk {
				minBpk = bpk
			}
			if bpk > maxBpk {
				maxBpk = bpk
			}
			if nc < minNc {
				minNc = nc
			}
			if nc > maxNc {
				maxNc = nc
			}
			oneBitMinFloor += float64(nc) // 1 bit per key floor
		}
		theoryLowerPerKey := sumTheoryBits / float64(sumNc)
		realPerInClusterKey := float64(sumRealBits) / float64(sumNc)
		oneBitFloorPerKey := oneBitMinFloor / float64(sumNc)

		fmt.Printf("\n=== %s (n=%d, L=%d, eps=%.3f) ===\n", ds.name, n, rangeLen, epsilon)
		fmt.Printf("  segments               = %d  (in-cluster keys: %d, fallback keys: %d)\n",
			len(segs), sumNc, len(fallback))
		fmt.Printf("  REAL in-cluster bits   = %d  =  %.3f bpk per in-cluster key\n",
			sumRealBits, realPerInClusterKey)
		fmt.Printf("  EF lower-part theory   = %.3f bpk per in-cluster key (data part only)\n",
			theoryLowerPerKey)
		fmt.Printf("  bitmap-floor           = %.3f bpk (raw 1-bit-per-key, no rank/select)\n",
			oneBitFloorPerKey)
		fmt.Printf("  real / lower-part      = %.2fx\n",
			realPerInClusterKey/math.Max(theoryLowerPerKey, 1e-9))
		fmt.Printf("  per-cluster real BPK:    min=%.3f  max=%.3f\n", minBpk, maxBpk)
		fmt.Printf("  per-cluster n_c range:   min=%d  max=%d\n", minNc, maxNc)
	}
}
