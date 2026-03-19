package are_dp_scan

import (
	"Thesis/bits"
	"Thesis/emptiness/are_adaptive"
	"fmt"
	"math"
	"sort"
)

type clusterFilter struct {
	filter *are_adaptive.AdaptiveApproximateRangeEmptiness
	minKey uint64
	maxKey uint64
}

// DPScanARE segments sorted keys into consecutive clusters using dynamic
// programming to minimise total estimated storage cost. It serves as a
// gold-standard for comparison against greedy segmentation strategies.
type DPScanARE struct {
	clusters []clusterFilter
	n        int
}

func NewDPScanARE(keys []bits.BitString, rangeLen uint64, epsilon float64) (*DPScanARE, error) {
	n := len(keys)
	if n == 0 {
		return &DPScanARE{}, nil
	}

	effectiveRangeLen := rangeLen + 1
	rTarget := float64(n) * float64(effectiveRangeLen) / epsilon
	K := uint32(math.Ceil(math.Log2(rTarget)))
	if K > 64 {
		K = 64
	}

	return NewDPScanAREFromK(keys, rangeLen, K)
}

func NewDPScanAREFromK(keys []bits.BitString, rangeLen uint64, K uint32) (*DPScanARE, error) {
	n := len(keys)
	if n == 0 {
		return &DPScanARE{}, nil
	}

	segments := segmentDP(keys, K)

	clusters := make([]clusterFilter, 0, len(segments))
	for _, seg := range segments {
		f, err := are_adaptive.NewAdaptiveAREFromK(seg.keys, rangeLen, K, 0)
		if err != nil {
			return nil, fmt.Errorf("cluster [%d, %d] build: %w", seg.minKey, seg.maxKey, err)
		}
		clusters = append(clusters, clusterFilter{
			filter: f,
			minKey: seg.minKey,
			maxKey: seg.maxKey,
		})
	}

	return &DPScanARE{clusters: clusters, n: n}, nil
}

func (d *DPScanARE) IsEmpty(a, b bits.BitString) bool {
	if d.n == 0 {
		return true
	}

	aVal := a.TrieUint64()
	bVal := b.TrieUint64()

	lo := sort.Search(len(d.clusters), func(i int) bool {
		return d.clusters[i].maxKey >= aVal
	})

	for i := lo; i < len(d.clusters) && d.clusters[i].minKey <= bVal; i++ {
		if !d.clusters[i].filter.IsEmpty(a, b) {
			return false
		}
	}

	return true
}

func (d *DPScanARE) SizeInBits() uint64 {
	total := uint64(0)
	for _, c := range d.clusters {
		total += c.filter.SizeInBits()
	}
	total += uint64(len(d.clusters)) * 128
	return total
}

func (d *DPScanARE) Stats() (numClusters, totalKeys int) {
	return len(d.clusters), d.n
}
