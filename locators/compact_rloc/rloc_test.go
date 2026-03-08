package compact_rloc

import (
	"Thesis/locators/rloc"
	"Thesis/trie/zft"
	"fmt"
	"testing"
	"time"
)

const (
	testRuns  = 50
	maxKeys   = 200
	maxBitLen = 64
)

func TestCompactRangeLocator_Correctness(t *testing.T) {
	for run := 0; run < testRuns; run++ {
		t.Run(fmt.Sprintf("run=%d", run), func(t *testing.T) {
			t.Parallel()
			seed := time.Now().UnixNano()
			keys := rloc.GenUniqueBitStrings(seed, maxKeys, maxBitLen)

			zt := zft.Build(keys)
			
			// Build baseline RangeLocator
			rlBaseline, err := rloc.NewRangeLocator(zt)
			if err != nil {
				t.Fatalf("NewRangeLocator (baseline) failed (seed: %d): %v", seed, err)
			}
			
			// Build CompactRangeLocator
			crl, err := NewCompactRangeLocator(zt)
			if err != nil {
				t.Fatalf("NewCompactRangeLocator failed (seed: %d): %v", seed, err)
			}

			it := zft.NewIterator(zt)
			for it.Next() {
				node := it.Node()
				if node == nil {
					continue
				}

				expectedI, expectedJ, _ := rlBaseline.Query(node.Extent)
				actualI, actualJ, err := crl.Query(node.Extent)

				if err != nil {
					t.Fatalf("CompactQuery error for node %s (seed: %d): %v", node.Extent, seed, err)
				}

				if actualI != expectedI || actualJ != expectedJ {
					t.Fatalf("Mismatch for node %s (seed: %d).\nExpected interval [%d, %d)\nGot [%d, %d)",
						node.Extent, seed, expectedI, expectedJ, actualI, actualJ)
				}
			}
		})
	}
}
