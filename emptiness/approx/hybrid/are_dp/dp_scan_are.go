package are_dp

import (
	"Thesis/bits"
	"Thesis/emptiness/approx/are_adaptive"
	"Thesis/emptiness/approx/hybrid/hybridutil"
	"Thesis/emptiness/exact"
	"fmt"
	"math"
	"sort"
)

// DPScanARE segments sorted keys into consecutive clusters using dynamic
// programming to minimise total estimated storage cost. It serves as a
// gold-standard for comparison against greedy segmentation strategies.
type DPScanARE struct {
	clusters []hybridutil.ClusterFilter
	n        int
}

// Config holds the construction parameters for DPScanARE.
//
// EREBackend selects the underlying exact range-emptiness implementation
// (see package exact). Zero value defaults to exact.VariantAuto. Use the
// WithEREBackend option to set it explicitly without relying on the
// zero-value alias.
type Config struct {
	K          uint32
	EREBackend exact.Variant
	backendSet bool
}

// WithEREBackend returns a copy of cfg with the chosen ERE backend.
func (cfg Config) WithEREBackend(v exact.Variant) Config {
	cfg.EREBackend = v
	cfg.backendSet = true
	return cfg
}

func (cfg Config) backend() exact.Variant {
	if cfg.backendSet {
		return cfg.EREBackend
	}
	return exact.VariantAuto
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
	return NewDPScanAREWithConfig(keys, rangeLen, Config{K: K})
}

// NewDPScanAREWithConfig builds a DPScanARE using the provided Config.
// This is the canonical constructor; legacy constructors delegate here.
func NewDPScanAREWithConfig(keys []bits.BitString, rangeLen uint64, cfg Config) (*DPScanARE, error) {
	n := len(keys)
	if n == 0 {
		return &DPScanARE{}, nil
	}

	K := cfg.K
	if K == 0 {
		K = 1
	}

	segments := segmentDP(keys, K)

	clusters := make([]hybridutil.ClusterFilter, 0, len(segments))
	for _, seg := range segments {
		keys64 := bsToU64(seg.keys)
		var keyBits uint32
		if len(seg.keys) > 0 {
			keyBits = seg.keys[0].SizeBits()
		}
		f, err := are_adaptive.NewAdaptiveAREFromKWithBackend(keys64, keyBits, K, 0, cfg.backend())
		if err != nil {
			return nil, fmt.Errorf("cluster [%d, %d] build: %w", seg.minKey, seg.maxKey, err)
		}
		clusters = append(clusters, hybridutil.ClusterFilter{
			Filter: f,
			MinKey: seg.minKey,
			MaxKey: seg.maxKey,
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
		return d.clusters[i].MaxKey >= aVal
	})

	for i := lo; i < len(d.clusters) && d.clusters[i].MinKey <= bVal; i++ {
		if !d.clusters[i].Filter.IsEmpty(aVal, bVal) {
			return false
		}
	}

	return true
}

// bsToU64 converts a []bits.BitString slice to []uint64 using TrieUint64.
func bsToU64(keys []bits.BitString) []uint64 {
	out := make([]uint64, len(keys))
	for i, k := range keys {
		out[i] = k.TrieUint64()
	}
	return out
}

func (d *DPScanARE) SizeInBits() uint64 {
	total := uint64(0)
	for _, c := range d.clusters {
		total += c.Filter.SizeInBits()
	}
	total += uint64(len(d.clusters)) * 128
	return total
}

func (d *DPScanARE) Stats() (numClusters, totalKeys int) {
	return len(d.clusters), d.n
}
