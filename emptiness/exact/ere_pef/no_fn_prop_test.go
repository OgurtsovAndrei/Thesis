package ere_pef

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"Thesis/testutils"
)

const (
	nfnTestRuns  = 1_000
	nfnMinN      = 100
	nfnMaxExtraN = 5_000
)

func setupNFNPEF(rng *rand.Rand, n int) ([]uint64, *PEF, error) {
	seen := make(map[uint64]bool, n)
	keys := make([]uint64, 0, n)
	for len(keys) < n {
		v := rng.Uint64()
		if !seen[v] {
			seen[v] = true
			keys = append(keys, v)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	p, err := NewPEF(keys, 64)
	return keys, p, err
}

func TestPEF_NoFN_PointInclusion(t *testing.T) {
	t.Parallel()
	runParallelNFN(t, func(t *testing.T, rng *rand.Rand, keys []uint64, p *PEF) {
		for j := 0; j < 50; j++ {
			k := keys[rng.Intn(len(keys))]
			if p.IsEmpty(k, k) {
				t.Errorf("false negative: key %d not found", k)
			}
		}
	})
}

func TestPEF_NoFN_TightOverhang(t *testing.T) {
	t.Parallel()
	runParallelNFN(t, func(t *testing.T, rng *rand.Rand, keys []uint64, p *PEF) {
		for j := 0; j < 50; j++ {
			k := keys[rng.Intn(len(keys))]
			if k > 0 && p.IsEmpty(k-1, k) {
				t.Errorf("false negative: range [%d, %d] is empty", k-1, k)
			}
			if k < ^uint64(0) && p.IsEmpty(k, k+1) {
				t.Errorf("false negative: range [%d, %d] is empty", k, k+1)
			}
		}
	})
}

func TestPEF_NoFN_SpanningRanges(t *testing.T) {
	t.Parallel()
	runParallelNFN(t, func(t *testing.T, rng *rand.Rand, keys []uint64, p *PEF) {
		n := len(keys)
		for j := 0; j < 20; j++ {
			idx1 := rng.Intn(n - 10)
			idx2 := idx1 + 1 + rng.Intn(min(n-idx1-1, 100))
			a, b := keys[idx1], keys[idx2]
			if p.IsEmpty(a, b) {
				t.Errorf("false negative: spanning range [%d, %d] is empty", a, b)
			}
		}
	})
}

func TestPEF_NoFN_MassiveSpan(t *testing.T) {
	t.Parallel()
	runParallelNFN(t, func(t *testing.T, rng *rand.Rand, keys []uint64, p *PEF) {
		if p.IsEmpty(keys[0], keys[len(keys)-1]) {
			t.Error("false negative: massive span [first, last] is empty")
		}
	})
}

// TestPEF_NoFN_ChunkBoundary verifies that a range straddling a chunk boundary
// is never reported empty when it spans a key in each adjacent chunk.
// Accesses p.chunks and p.chunkBaseAt (unexported) — test is in package ere_pef.
func TestPEF_NoFN_ChunkBoundary(t *testing.T) {
	t.Parallel()
	for i := 0; i < 100; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 1100)))
			n := nfnMinN + rng.Intn(nfnMaxExtraN)
			// Clustered input forces the DP to emit multiple chunk kinds.
			keys, _ := testutils.GenerateClusterDistribution(n, 4, 0.1, rng)
			if len(keys) < 2 {
				t.Skip("too few keys after dedup")
			}
			p, err := NewPEF(keys, 64)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			for ci := 0; ci+1 < len(p.chunks); ci++ {
				lastKeyCI := p.chunks[ci].last
				base := p.chunkBaseAt(ci + 1)
				idx := sort.Search(len(keys), func(k int) bool { return keys[k] >= base })
				if idx >= len(keys) {
					continue
				}
				firstKeyCIp1 := keys[idx]
				if p.IsEmpty(lastKeyCI, firstKeyCIp1) {
					t.Errorf("chunk[%d→%d] boundary: IsEmpty(%d, %d)=true want false",
						ci, ci+1, lastKeyCI, firstKeyCIp1)
				}
			}
		})
	}
}

// TestPEF_NoFN_Sequential exercises the kindAllOnes codec path on contiguous key runs.
func TestPEF_NoFN_Sequential(t *testing.T) {
	t.Parallel()
	for i := 0; i < 200; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 900)))
			n := nfnMinN + rng.Intn(nfnMaxExtraN)
			base := rng.Uint64() >> 16 // leave headroom for n sequential keys
			keys := make([]uint64, n)
			for j := range keys {
				keys[j] = base + uint64(j)
			}
			p, err := NewPEF(keys, 64)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			for j := 0; j < 20; j++ {
				k := keys[rng.Intn(n)]
				if p.IsEmpty(k, k) {
					t.Errorf("sequential: false negative at key %d", k)
				}
			}
			if p.IsEmpty(keys[0], keys[n-1]) {
				t.Error("sequential: massive span reported empty")
			}
		})
	}
}

// TestPEF_NoFN_Clustered exercises the EF and bitmap codec paths on clustered key sets.
func TestPEF_NoFN_Clustered(t *testing.T) {
	t.Parallel()
	for i := 0; i < 200; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 1300)))
			n := nfnMinN + rng.Intn(nfnMaxExtraN)
			keys, _ := testutils.GenerateClusterDistribution(n, 5, 0.15, rng)
			if len(keys) < 10 {
				t.Skip("too few keys after dedup")
			}
			p, err := NewPEF(keys, 64)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			for j := 0; j < 50; j++ {
				k := keys[rng.Intn(len(keys))]
				if p.IsEmpty(k, k) {
					t.Errorf("clustered: false negative at key %d", k)
				}
			}
			if p.IsEmpty(keys[0], keys[len(keys)-1]) {
				t.Error("clustered: massive span reported empty")
			}
		})
	}
}

func runParallelNFN(t *testing.T, testFn func(t *testing.T, rng *rand.Rand, keys []uint64, p *PEF)) {
	t.Helper()
	for i := 0; i < nfnTestRuns; i++ {
		i := i
		t.Run(fmt.Sprintf("Iter%d", i), func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewSource(int64(i + 100)))
			keys, p, err := setupNFNPEF(rng, nfnMinN+rng.Intn(nfnMaxExtraN))
			if err != nil {
				t.Fatalf("setup: %v", err)
			}
			testFn(t, rng, keys, p)
		})
	}
}
