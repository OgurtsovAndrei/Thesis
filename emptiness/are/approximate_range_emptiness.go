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
type ApproximateRangeEmptiness struct {
	exact *ere.ExactRangeEmptiness
	K     uint32
}

// toKBitString ensures bit 0 is the MSB of the uint64 value for correct lexicographical order.
func toKBitString(val uint64, K uint32) bits.BitString {
	if K == 0 {
		return bits.NewBitString(0)
	}
	var reversed uint64
	for i := uint32(0); i < K; i++ {
		if (val & (uint64(1) << (63 - i))) != 0 {
			reversed |= (uint64(1) << i)
		}
	}
	return bits.NewFromUint64(reversed).Prefix(int(K))
}

func NewApproximateRangeEmptiness(keys []bits.BitString, epsilon float64) (*ApproximateRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &ApproximateRangeEmptiness{exact: nil, K: 0}, nil
	}

	val := (2.0 * float64(n)) / epsilon
	K := uint32(math.Ceil(math.Log2(val)))
	if K == 0 { K = 1 }
	if K > 64 { K = 64 }

	truncatedKeys := make([]bits.BitString, 0, n)
	var lastKey bits.BitString
	
	for i, k := range keys {
		// IMPORTANT: We must convert keys to MSB-first BitStrings of length K
		// Since input keys might be LSB-first from NewFromUint64, we need to be careful.
		// For this implementation, we assume keys are already in a consistent bit order or we re-map them.
		
		// To be safe and consistent with IsEmpty, let's use a helper if they are uint64-based.
		// But here keys is []bits.BitString. Let's assume they are standard.
		trunc := k
		if trunc.Size() > K {
			trunc = trunc.Prefix(int(K))
		}
		
		if i == 0 || trunc.Compare(lastKey) > 0 {
			truncatedKeys = append(truncatedKeys, trunc)
			lastKey = trunc
		} else if trunc.Compare(lastKey) == 0 {
			// Skip duplicates after truncation
			continue
		} else {
			return nil, fmt.Errorf("keys must be sorted")
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
	if are.exact == nil { return true }
	
	truncA := a
	if truncA.Size() > are.K { truncA = truncA.Prefix(int(are.K)) }
	truncB := b
	if truncB.Size() > are.K { truncB = truncB.Prefix(int(are.K)) }

	return are.exact.IsEmpty(truncA, truncB)
}

func (are *ApproximateRangeEmptiness) SizeInBits() uint64 {
	if are.exact == nil { return 0 }
	return are.exact.SizeInBits()
}

func (are *ApproximateRangeEmptiness) ByteSize() int {
	if are == nil || are.exact == nil { return 0 }
	return are.exact.ByteSize() + 8 
}

func (are *ApproximateRangeEmptiness) MemDetailed() utils.MemReport {
	if are == nil || are.exact == nil {
		return utils.MemReport{Name: "ApproximateRangeEmptiness", TotalBytes: 0}
	}
	return utils.MemReport{
		Name:       "ApproximateRangeEmptiness",
		TotalBytes: are.ByteSize(),
		Children:   []utils.MemReport{are.exact.MemDetailed()},
	}
}
