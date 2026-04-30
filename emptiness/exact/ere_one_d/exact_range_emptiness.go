package ere_one_d

import (
	"Thesis/utils/errutil"
	"Thesis/utils"
	"fmt"
	"math"
	"unsafe"

	"Thesis/bits"
	"Thesis/succinct_bit_vector/rsdic"
)

// linearScanThreshold is the bucket size below which linear scan is used instead of
// binary search. Determined experimentally: linear scan is faster for buckets up to ~128
// elements due to sequential prefetch; binary search wins above that.
// See BenchmarkBucketSearch_LinearVsBinary.
const linearScanThreshold = 128

// ExactRangeEmptiness implements the 1D range emptiness structure from SODA 2015, Section 3.2.
type ExactRangeEmptiness struct {
	D          *rsdic.RSDic
	packedData []uint64

	n         int
	numBlocks int
	KeySize   uint32
	k         uint32
	w         uint32
	suffMask  uint64
}

// NewExactRangeEmptiness builds ERE directly from sorted []uint64 keys.
// keyBits is the effective key width (≤ 64); keys must fit in their low keyBits bits.
func NewExactRangeEmptiness(keys []uint64, keyBits uint32) (*ExactRangeEmptiness, error) {
	errutil.BugOn(keyBits > 64, "keyBits must be <= 64, got %d", keyBits)

	n := len(keys)
	if n == 0 {
		return &ExactRangeEmptiness{n: 0, KeySize: keyBits}, nil
	}

	for i := 1; i < n; i++ {
		if keys[i-1] > keys[i] {
			return nil, fmt.Errorf("keys must be sorted")
		}
	}

	k := uint32(math.Floor(math.Log2(float64(n))))
	if k == 0 {
		k = 1
	}

	numBlocks := 1 << k
	if keyBits < k {
		keyBits = k
	}
	w := keyBits - k

	suffMask := uint64(0)
	if w == 64 {
		suffMask = ^uint64(0)
	} else {
		suffMask = (uint64(1) << w) - 1
	}

	D := rsdic.New()
	suffixes := make([]uint64, 0, n)

	i := 0
	for b := 0; b < numBlocks; b++ {
		D.PushBack(true)
		countInBlock := 0
		for i < n && (keys[i]>>(keyBits-k)) == uint64(b) {
			suffixes = append(suffixes, keys[i]&suffMask)
			countInBlock++
			i++
		}
		for c := 0; c < countInBlock; c++ {
			D.PushBack(false)
		}
	}
	D.PushBack(true) // sentinel

	packed := packUint64Local(suffixes, int(w))

	return &ExactRangeEmptiness{
		D:          D,
		packedData: packed,
		n:          n,
		numBlocks:  numBlocks,
		KeySize:    keyBits,
		k:          k,
		w:          w,
		suffMask:   suffMask,
	}, nil
}

func (ere *ExactRangeEmptiness) IsEmpty(a, b uint64) bool {
	if ere.n == 0 {
		return true
	}
	if a > b {
		return true
	}

	blockA := a >> ere.w
	blockB := b >> ere.w

	// Range exceeds universe
	if blockA >= uint64(ere.numBlocks) {
		return true
	}
	if blockB >= uint64(ere.numBlocks) {
		blockB = uint64(ere.numBlocks - 1)
	}

	if blockA == blockB {
		start, end := ere.getBlockRange(blockA)
		if start < end {
			suffA := a & ere.suffMask
			suffB := b & ere.suffMask
			if !ere.searchBucket(start, end, suffA, suffB) {
				return false
			}
		}
	} else {
		startA, endA, startB, endB := ere.getQueryBlockRanges(blockA, blockB)

		// Any points between the two boundary blocks imply non-emptiness immediately.
		if startB > endA {
			return false
		}

		// Check blockA for elements in [suffA, max]
		if startA < endA {
			suffA := a & ere.suffMask
			maxSuff := ere.suffMask
			if !ere.searchBucket(startA, endA, suffA, maxSuff) {
				return false
			}
		}
		// Check blockB for elements in [0, suffB]
		if startB < endB {
			suffB := b & ere.suffMask
			if !ere.searchBucket(startB, endB, 0, suffB) {
				return false
			}
		}
	}

	return true
}

func (ere *ExactRangeEmptiness) LinearIsEmpty(a, b uint64) bool {
	if ere.n == 0 {
		return true
	}
	if a > b {
		return true
	}

	blockA := a >> ere.w
	blockB := b >> ere.w

	if blockA >= uint64(ere.numBlocks) {
		return true
	}
	if blockB >= uint64(ere.numBlocks) {
		blockB = uint64(ere.numBlocks - 1)
	}

	if blockA == blockB {
		start, end := ere.getBlockRange(blockA)
		if start < end {
			suffA := a & ere.suffMask
			suffB := b & ere.suffMask
			if !ere.isRangeEmptyInBlockLinear(start, end, suffA, suffB) {
				return false
			}
		}
	} else {
		startA, endA, startB, endB := ere.getQueryBlockRanges(blockA, blockB)

		if startB > endA {
			return false
		}

		if startA < endA {
			suffA := a & ere.suffMask
			maxSuff := ere.suffMask
			if !ere.isRangeEmptyInBlockLinear(startA, endA, suffA, maxSuff) {
				return false
			}
		}
		if startB < endB {
			suffB := b & ere.suffMask
			if !ere.isRangeEmptyInBlockLinear(startB, endB, 0, suffB) {
				return false
			}
		}
	}

	return true
}

func (ere *ExactRangeEmptiness) getBlockRange(blockIdx uint64) (int, int) {
	posStart := ere.D.Select1(blockIdx)
	posEnd := ere.D.Select1(blockIdx + 1)
	startIndex := int(posStart - blockIdx)
	endIndex := int(posEnd - (blockIdx + 1))
	return startIndex, endIndex
}

func (ere *ExactRangeEmptiness) getQueryBlockRanges(blockA, blockB uint64) (int, int, int, int) {
	if blockB == blockA+1 {
		pos0 := ere.D.Select1(blockA)
		pos1 := ere.D.Select1(blockA + 1)
		pos2 := ere.D.Select1(blockA + 2)
		startA := int(pos0 - blockA)
		endA := int(pos1 - (blockA + 1))
		startB := endA
		endB := int(pos2 - (blockA + 2))
		return startA, endA, startB, endB
	}

	startA, endA := ere.getBlockRange(blockA)
	startB, endB := ere.getBlockRange(blockB)
	return startA, endA, startB, endB
}

func (ere *ExactRangeEmptiness) searchBucket(start, end int, minSuff, maxSuff uint64) bool {
	if end-start <= linearScanThreshold {
		return ere.isRangeEmptyInBlockLinear(start, end, minSuff, maxSuff)
	}
	return ere.isRangeEmptyInBlock(start, end, minSuff, maxSuff)
}

func (ere *ExactRangeEmptiness) isRangeEmptyInBlock(start, end int, minSuff, maxSuff uint64) bool {
	l, r := start, end
	for l < r {
		mid := l + (r-l)/2
		midVal := ere.getPackedSuffix(mid)
		if midVal < minSuff {
			l = mid + 1
		} else {
			r = mid
		}
	}

	if l < end {
		val := ere.getPackedSuffix(l)
		if val <= maxSuff {
			return false
		}
	}
	return true
}

func (ere *ExactRangeEmptiness) getPackedSuffix(idx int) uint64 {
	return bits.UnpackBit(ere.packedData, idx, int(ere.w))
}

func (ere *ExactRangeEmptiness) isRangeEmptyInBlockLinear(start, end int, minSuff, maxSuff uint64) bool {
	if start >= end {
		return true
	}
	w := int(ere.w)
	if w == 0 {
		// All suffixes are the empty string (value 0). The bucket is non-empty
		// (start < end), so the range matches iff 0 ∈ [minSuff, maxSuff].
		// Since maxSuff is uint64, the upper-bound check is always true.
		return minSuff > 0
	}
	mask := uint64(1<<w) - 1
	if w == 64 {
		mask = ^uint64(0)
	}
	bitPos := uint64(start) * uint64(w)
	for i := start; i < end; i++ {
		wordIdx := bitPos / 64
		bitOffset := uint(bitPos % 64)
		val := ere.packedData[wordIdx] >> bitOffset
		if 64-int(bitOffset) < w {
			val |= ere.packedData[wordIdx+1] << uint(64-int(bitOffset))
		}
		val &= mask
		if val >= minSuff && val <= maxSuff {
			return false
		}
		bitPos += uint64(w)
	}
	return true
}

func packUint64Local(values []uint64, bitWidth int) []uint64 {
	if len(values) == 0 || bitWidth == 0 {
		return nil
	}
	totalBits := uint64(len(values)) * uint64(bitWidth)
	numWords := (totalBits + 63) / 64
	packed := make([]uint64, numWords)
	for i, val := range values {
		bitPos := uint64(i) * uint64(bitWidth)
		wordIdx := bitPos / 64
		bitOffset := uint(bitPos % 64)
		packed[wordIdx] |= val << bitOffset
		if 64-int(bitOffset) < bitWidth {
			packed[wordIdx+1] |= val >> uint(64-int(bitOffset))
		}
	}
	return packed
}

func (ere *ExactRangeEmptiness) ByteSize() int {
	if ere == nil || ere.n == 0 {
		return 0
	}
	size := int(unsafe.Sizeof(*ere))
	size += ere.D.AllocSize()
	size += len(ere.packedData) * 8
	return size
}

func (ere *ExactRangeEmptiness) SizeInBits() uint64 {
	if ere == nil || ere.n == 0 {
		return 0
	}
	dBits := ere.D.Num()
	suffixBits := uint64(ere.n) * uint64(ere.w)
	return dBits + suffixBits
}

// MetadataNumBits returns the logical (information-theoretic) metadata bit
// count, namely the number of bits actually pushed into D (excluding the
// packed suffix array A_ds and any rank/select auxiliary indices).
func (ere *ExactRangeEmptiness) MetadataNumBits() uint64 {
	if ere == nil || ere.n == 0 {
		return 0
	}
	return ere.D.Num()
}

// MetadataAllocBits returns the actual allocated metadata size in bits, namely
// the rsdic backing storage of D (raw bit array + rank/select indices).
// Excludes the packed suffix array A_ds.
func (ere *ExactRangeEmptiness) MetadataAllocBits() uint64 {
	if ere == nil || ere.n == 0 {
		return 0
	}
	return uint64(ere.D.AllocSize()) * 8
}

type Stats struct {
	N               int
	NumBlocks       int
	NonEmptyBlocks  int
	EmptyBlocks     int
	AvgKeysPerBlock float64
	MaxKeysInBlock  int
	EmptyBlockPct   float64
	SumSquaredKeys  uint64
}

func (ere *ExactRangeEmptiness) GetStats() Stats {
	nonEmpty := 0
	maxKeys := 0
	var sumSq uint64
	for b := uint64(0); b < uint64(ere.numBlocks); b++ {
		start, end := ere.getBlockRange(b)
		count := end - start
		if count > 0 {
			nonEmpty++
		}
		if count > maxKeys {
			maxKeys = count
		}
		sumSq += uint64(count) * uint64(count)
	}

	return Stats{
		N:               ere.n,
		NumBlocks:       ere.numBlocks,
		NonEmptyBlocks:  nonEmpty,
		EmptyBlocks:     ere.numBlocks - nonEmpty,
		AvgKeysPerBlock: float64(ere.n) / float64(nonEmpty),
		MaxKeysInBlock:  maxKeys,
		EmptyBlockPct:   float64(ere.numBlocks-nonEmpty) / float64(ere.numBlocks) * 100,
		SumSquaredKeys:  sumSq,
	}
}

// NonEmptyBlockSizes returns k_b for every non-empty block, in block-index
// order.
func (ere *ExactRangeEmptiness) NonEmptyBlockSizes() []int {
	if ere == nil || ere.n == 0 {
		return nil
	}
	out := make([]int, 0, ere.numBlocks)
	for b := uint64(0); b < uint64(ere.numBlocks); b++ {
		start, end := ere.getBlockRange(b)
		count := end - start
		if count > 0 {
			out = append(out, count)
		}
	}
	return out
}

func (ere *ExactRangeEmptiness) MemDetailed() utils.MemReport {
	if ere == nil || ere.n == 0 {
		return utils.MemReport{Name: "ExactRangeEmptiness", TotalBytes: 0}
	}
	return utils.MemReport{
		Name:       "ExactRangeEmptiness",
		TotalBytes: ere.ByteSize(),
		Children: []utils.MemReport{
			{Name: "metadata", TotalBytes: int(unsafe.Sizeof(*ere))},
			{Name: "D_blocks", TotalBytes: ere.D.AllocSize()},
			{Name: "suffixes_packed", TotalBytes: len(ere.packedData) * 8},
		},
	}
}
