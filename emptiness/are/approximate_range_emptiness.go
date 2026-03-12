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
		// Extract bit (63-i) from val and put it into bit i of reversed
		if (val & (uint64(1) << (63 - i))) != 0 {
			reversed |= (uint64(1) << i)
		}
	}
	// Now index 0 of BitString will contain numeric bit 63.
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
	if K > 64 {
		return nil, fmt.Errorf("K exceeds 64 bits: %d", K)
	}

	truncatedKeys := make([]bits.BitString, 0, n)
	var lastKey bits.BitString
	for i, k := range keys {
		// CRITICAL FIX: Ensure MSB-first order by using toKBitString
		// We extract the original uint64 value and re-map it to MSB-first BitString
		val := k.Word(0)
		trunc := toKBitString(val, K)
		
		if i == 0 || trunc.Compare(lastKey) > 0 {
			truncatedKeys = append(truncatedKeys, trunc)
			lastKey = trunc
		} else if trunc.Compare(lastKey) == 0 {
			continue // Skip duplicates
		} else {
			return nil, fmt.Errorf("keys must be sorted (numeric order)")
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
	
	// Ensure query keys also follow MSB-first order
	truncA := toKBitString(a.Word(0), are.K)
	truncB := toKBitString(b.Word(0), are.K)

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
	return are.exact.MemDetailed()
}
