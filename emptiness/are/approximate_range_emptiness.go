package are

import (
	"Thesis/bits"
	"Thesis/emptiness/ere"
	"Thesis/utils"
	"fmt"
	"math"
)

// ApproximateRangeEmptiness is a probabilistic data structure that answers 1D range emptiness
// queries with a guaranteed upper bound on the false positive probability (\epsilon).
// It achieves this by truncating the keys (fingerprinting) and storing them in an ExactRangeEmptiness structure.
type ApproximateRangeEmptiness struct {
	exact *ere.ExactRangeEmptiness
	K     uint32 // Maximum bit length stored (log2(N) + log2(2/epsilon))
}

// NewApproximateRangeEmptiness builds the Approximate Range Emptiness structure.
// epsilon is the desired false positive probability (e.g., 0.01 for 1%).
func NewApproximateRangeEmptiness(keys []bits.BitString, epsilon float64) (*ApproximateRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &ApproximateRangeEmptiness{exact: nil, K: 0}, nil
	}

	// Calculate required prefix length K to satisfy epsilon.
	// We need the probability of a false positive to be <= epsilon.
	// False positives occur at the boundaries of the query range if the bucket contains a key.
	// Assuming uniform-ish distribution in the prefix space, P(FP) <= 2 * (N / 2^K).
	// We want 2 * N / 2^K <= epsilon => 2^K >= 2N / epsilon => K = ceil(log2(2N / epsilon)).
	
	val := (2.0 * float64(n)) / epsilon
	K := uint32(math.Ceil(math.Log2(val)))
	
	// Ensure K is at least 1
	if K == 0 {
		K = 1
	}

	// Truncate keys to K bits
	truncatedKeys := make([]bits.BitString, 0, n)
	var lastKey bits.BitString
	
	for i, k := range keys {
		trunc := k
		if trunc.Size() > K {
			trunc = trunc.Prefix(int(K))
		}
		
		// Remove duplicates that arise from truncation to keep ExactRangeEmptiness happy
		if i == 0 || trunc.Compare(lastKey) > 0 {
			truncatedKeys = append(truncatedKeys, trunc)
			lastKey = trunc
		} else if trunc.Compare(lastKey) < 0 {
			return nil, fmt.Errorf("keys must be sorted")
		}
	}

	// The universe size for the Exact structure is now bounded by K bits
	universe := bits.NewBitString(K)
	
	exact, err := ere.NewExactRangeEmptiness(truncatedKeys, universe)
	if err != nil {
		return nil, fmt.Errorf("failed to build underlying exact structure: %w", err)
	}

	return &ApproximateRangeEmptiness{
		exact: exact,
		K:     K,
	}, nil
}

// IsEmpty returns true if the interval [a, b] contains NO elements from the set.
// It returns false if the interval contains at least one element, or with probability <= epsilon if it's a false positive.
func (are *ApproximateRangeEmptiness) IsEmpty(a, b bits.BitString) bool {
	if are.exact == nil {
		return true
	}
	if a.Compare(b) > 0 {
		return true
	}

	// Truncate the query boundaries to match the stored prefixes
	truncA := a
	if truncA.Size() > are.K {
		truncA = truncA.Prefix(int(are.K))
	}
	
	truncB := b
	if truncB.Size() > are.K {
		truncB = truncB.Prefix(int(are.K))
	}

	return are.exact.IsEmpty(truncA, truncB)
}

// ByteSize returns the estimated resident size in bytes.
func (are *ApproximateRangeEmptiness) ByteSize() int {
	if are == nil || are.exact == nil {
		return 0
	}
	// exact size + struct overhead
	return are.exact.ByteSize() + 8 
}

// MemDetailed returns a detailed memory usage report.
func (are *ApproximateRangeEmptiness) MemDetailed() utils.MemReport {
	if are == nil || are.exact == nil {
		return utils.MemReport{Name: "ApproximateRangeEmptiness", TotalBytes: 0}
	}
	
	exactRep := are.exact.MemDetailed()
	
	return utils.MemReport{
		Name:       "ApproximateRangeEmptiness",
		TotalBytes: are.ByteSize(),
		Children: []utils.MemReport{
			exactRep,
		},
	}
}
