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
// Uses simple prefix truncation: keys are truncated to K bits via Prefix(K).
type ApproximateRangeEmptiness struct {
	exact *ere.ExactRangeEmptiness
	K     uint32
}

func NewApproximateRangeEmptiness(keys []bits.BitString, epsilon float64) (*ApproximateRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &ApproximateRangeEmptiness{exact: nil, K: 0}, nil
	}

	val := (2.0 * float64(n)) / epsilon
	K := uint32(math.Ceil(math.Log2(val)))
	if K == 0 {
		K = 1
	}

	// Truncate to K bits using Prefix — preserves trie (Compare) ordering
	truncatedKeys := make([]bits.BitString, 0, n)
	var lastKey bits.BitString
	for i, k := range keys {
		trunc := k.Prefix(int(K))

		if i == 0 || trunc.Compare(lastKey) > 0 {
			truncatedKeys = append(truncatedKeys, trunc)
			lastKey = trunc
		} else if trunc.Compare(lastKey) == 0 {
			continue
		} else {
			return nil, fmt.Errorf("keys must be sorted by Compare")
		}
	}

	universe := bits.NewBitString(K)
	exact, err := ere.NewExactRangeEmptiness(truncatedKeys, universe)
	if err != nil {
		return nil, err
	}

	return &ApproximateRangeEmptiness{exact: exact, K: K}, nil
}

func (are *ApproximateRangeEmptiness) IsEmpty(a, b bits.BitString) bool {
	if are.exact == nil {
		return true
	}
	truncA := a.Prefix(int(are.K))
	truncB := b.Prefix(int(are.K))
	return are.exact.IsEmpty(truncA, truncB)
}

func (are *ApproximateRangeEmptiness) SizeInBits() uint64 {
	if are.exact == nil {
		return 0
	}
	return are.exact.SizeInBits()
}

func (are *ApproximateRangeEmptiness) ByteSize() int {
	if are == nil || are.exact == nil {
		return 0
	}
	return are.exact.ByteSize() + 8
}

func (are *ApproximateRangeEmptiness) MemDetailed() utils.MemReport {
	if are == nil || are.exact == nil {
		return utils.MemReport{Name: "ApproximateRangeEmptiness", TotalBytes: 0}
	}
	return are.exact.MemDetailed()
}
