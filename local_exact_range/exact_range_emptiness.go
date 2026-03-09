package local_exact_range

import (
	"Thesis/bits"
	"fmt"
	"github.com/hillbig/rsdic"
)

// ExactRangeEmptiness is a succinct data structure that answers exact 1D range emptiness
// queries in O(1) expected time (with binary search over small buckets).
// It divides the lexicographic space into buckets based on the most significant bits.
type ExactRangeEmptiness struct {
	keys []bits.BitString
	H    *rsdic.RSDic // Header bitvector for Elias-Fano encoding of bucket sizes
	k    uint32       // number of prefix bits used for bucketing
	n    int
}

// NewExactRangeEmptiness builds the exact range emptiness structure.
// `keys` must be sorted in ascending lexicographic order.
// `universe` is an optional parameter defining the maximum possible value/length.
func NewExactRangeEmptiness(keys []bits.BitString, universe bits.BitString) (*ExactRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &ExactRangeEmptiness{
			keys: keys,
			H:    rsdic.New(),
			k:    0,
			n:    0,
		}, nil
	}

	// Verify keys are sorted
	for i := 1; i < n; i++ {
		if keys[i-1].Compare(keys[i]) > 0 {
			return nil, fmt.Errorf("keys must be sorted")
		}
	}

	// Calculate number of bits k to use for bucket index
	// We want approx n buckets. Let k = floor(log2(n))
	k := uint32(0)
	for temp := n; temp > 1; temp >>= 1 {
		k++
	}
	// max k is 60 because we shift into a uint64
	if k > 60 {
		k = 60
	}

	m := uint64(1) << k

	H := rsdic.New()

	// Build the unary encoded bucket sizes
	currentBucket := uint64(0)
	for _, key := range keys {
		b := getBucketIndex(key, k)
		if b < currentBucket {
			return nil, fmt.Errorf("bucket index inversion: sort order violation or prefix bug")
		}
		for currentBucket < b {
			H.PushBack(false)
			currentBucket++
		}
		H.PushBack(true)
	}
	// Finish the remaining buckets up to m
	for currentBucket < m {
		H.PushBack(false)
		currentBucket++
	}

	return &ExactRangeEmptiness{
		keys: keys,
		H:    H,
		k:    k,
		n:    n,
	}, nil
}

// getBucketIndex extracts the first k bits of x and returns it as a uint64 bucket index.
// Bit 0 of x becomes the most significant bit (k-1) of the bucket index.
func getBucketIndex(x bits.BitString, k uint32) uint64 {
	var bucket uint64 = 0
	size := x.Size()
	for i := uint32(0); i < k; i++ {
		if i < size && x.At(i) {
			bucket |= (uint64(1) << (k - 1 - i))
		}
	}
	return bucket
}

// IsEmpty returns true if the interval [a, b] contains no elements from S.
func (ere *ExactRangeEmptiness) IsEmpty(a, b bits.BitString) bool {
	if ere.n == 0 {
		return true
	}

	// If a > b, empty interval
	if a.Compare(b) > 0 {
		return true
	}

	bucketA := getBucketIndex(a, ere.k)
	bucketB := getBucketIndex(b, ere.k)

	// If there's an intermediate bucket, check if any of them have elements
	if bucketB > bucketA+1 {
		posB := ere.H.Select(bucketB, false)
		onesBeforeB := ere.H.Rank(posB, true)

		posA1 := ere.H.Select(bucketA+1, false)
		onesBeforeA1 := ere.H.Rank(posA1, true)

		if onesBeforeB > onesBeforeA1 {
			return false // There is at least one element in an intermediate bucket
		}
	}

	// Check elements in bucketA
	startA, endA := ere.getBucketRange(bucketA)
	if !isBucketRangeEmpty(ere.keys[startA:endA], a, b) {
		return false
	}

	// If bucketB != bucketA, check elements in bucketB
	if bucketB != bucketA {
		startB, endB := ere.getBucketRange(bucketB)
		if !isBucketRangeEmpty(ere.keys[startB:endB], a, b) {
			return false
		}
	}

	return true
}

func (ere *ExactRangeEmptiness) getBucketRange(bucket uint64) (uint64, uint64) {
	if bucket == 0 {
		posZero := ere.H.Select(0, false)
		return 0, ere.H.Rank(posZero, true)
	}

	posZeroBefore := ere.H.Select(bucket-1, false)
	start := ere.H.Rank(posZeroBefore, true)

	posZeroThis := ere.H.Select(bucket, false)
	end := ere.H.Rank(posZeroThis, true)

	return start, end
}

// isBucketRangeEmpty checks if the given sorted slice has any elements in [a, b]
func isBucketRangeEmpty(subKeys []bits.BitString, a, b bits.BitString) bool {
	if len(subKeys) == 0 {
		return true
	}
	// Binary search to find the first element >= a
	l, r := 0, len(subKeys)
	for l < r {
		mid := l + (r-l)/2
		if subKeys[mid].Compare(a) < 0 {
			l = mid + 1
		} else {
			r = mid
		}
	}

	// l is the index of the first element >= a
	if l < len(subKeys) && subKeys[l].Compare(b) <= 0 {
		return false // Found an element in [a, b]
	}

	return true
}
