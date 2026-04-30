package ere

import (
	"fmt"
	"math"
	"unsafe"

	"Thesis/utils/errutil"
	"Thesis/succinct_bit_vector/rsdic"
	"Thesis/utils"
)

// linearScanThreshold is the bucket size below which linear scan is used instead of
// binary search. Determined experimentally: linear scan is faster for buckets up to ~128
// elements due to sequential prefetch; binary search wins above that.
// See BenchmarkBucketSearch_LinearVsBinary.
const linearScanThreshold = 128

// ExactRangeEmptiness implements the 1D range emptiness structure from SODA 2015, Section 3.2.
type ExactRangeEmptiness struct {
	D1         *rsdic.RSDic
	D2         *rsdic.RSDic
	packedData []uint64

	n         int
	numBlocks int
	KeySize   uint32
	k         uint32
	w         uint32
	suffMask  uint64
}

// NewExactRangeEmptiness builds ERE from sorted []uint64 keys.
// keyBits is the effective key width (e.g. 64); must be <= 64.
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

	var suffMask uint64
	if w == 64 {
		suffMask = ^uint64(0)
	} else {
		suffMask = (uint64(1) << w) - 1
	}

	D1 := rsdic.New()
	D2 := rsdic.New()
	suffixes := make([]uint64, 0, n)

	i := 0
	for b := 0; b < numBlocks; b++ {
		countInBlock := 0
		for i < n && (keys[i]>>(keyBits-k)) == uint64(b) {
			suffixes = append(suffixes, keys[i]&suffMask)
			countInBlock++
			i++
		}

		if countInBlock > 0 {
			D1.PushBack(true)
			D2.PushBack(true)
			for c := 0; c < countInBlock; c++ {
				D2.PushBack(false)
			}
		} else {
			D1.PushBack(false)
		}
	}
	D2.PushBack(true) // sentinel

	packed := packUint64Local(suffixes, int(w))

	return &ExactRangeEmptiness{
		D1:         D1,
		D2:         D2,
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

	// 1. Check intermediate full blocks
	if blockB > blockA+1 {
		onesBeforeB := ere.D1.Rank(blockB, true)
		onesBeforeA1 := ere.D1.Rank(blockA+1, true)
		if onesBeforeB > onesBeforeA1 {
			return false
		}
	}

	// 2. Check boundary blocks
	if blockA == blockB {
		if ere.D1.Bit(blockA) {
			start, end := ere.getBlockRange(blockA)
			suffA := a & ere.suffMask
			suffB := b & ere.suffMask
			if !ere.searchBucket(start, end, suffA, suffB) {
				return false
			}
		}
	} else {
		// Check blockA for elements in [suffA, max]
		if ere.D1.Bit(blockA) {
			start, end := ere.getBlockRange(blockA)
			suffA := a & ere.suffMask
			maxSuff := ere.suffMask
			if !ere.searchBucket(start, end, suffA, maxSuff) {
				return false
			}
		}
		// Check blockB for elements in [0, suffB]
		if ere.D1.Bit(blockB) {
			start, end := ere.getBlockRange(blockB)
			suffB := b & ere.suffMask
			if !ere.searchBucket(start, end, 0, suffB) {
				return false
			}
		}
	}

	return true
}

func (ere *ExactRangeEmptiness) getBlockRange(blockIdx uint64) (int, int) {
	numNonEmptyBefore := int(ere.D1.Rank(blockIdx, true))
	posInD2 := ere.D2.Select1Fast(uint64(numNonEmptyBefore))
	startIndex := int(posInD2 - uint64(numNonEmptyBefore))
	posEndInD2 := ere.D2.Select1Fast(uint64(numNonEmptyBefore + 1))
	endIndex := int(posEndInD2 - uint64(numNonEmptyBefore+1))
	return startIndex, endIndex
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
	w := int(ere.w)
	if w == 0 {
		return 0
	}
	bitPos := uint64(idx) * uint64(w)
	wordIdx := bitPos / 64
	bitOffset := uint(bitPos % 64)
	val := ere.packedData[wordIdx] >> bitOffset
	if 64-int(bitOffset) < w {
		val |= ere.packedData[wordIdx+1] << uint(64-int(bitOffset))
	}
	mask := uint64(1<<w) - 1
	if w == 64 {
		mask = ^uint64(0)
	}
	return val & mask
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

	// 1. Check intermediate full blocks
	if blockB > blockA+1 {
		onesBeforeB := ere.D1.Rank(blockB, true)
		onesBeforeA1 := ere.D1.Rank(blockA+1, true)
		if onesBeforeB > onesBeforeA1 {
			return false
		}
	}

	// 2. Check boundary blocks
	if blockA == blockB {
		if ere.D1.Bit(blockA) {
			start, end := ere.getBlockRange(blockA)
			suffA := a & ere.suffMask
			suffB := b & ere.suffMask
			if !ere.isRangeEmptyInBlockLinear(start, end, suffA, suffB) {
				return false
			}
		}
	} else {
		if ere.D1.Bit(blockA) {
			start, end := ere.getBlockRange(blockA)
			suffA := a & ere.suffMask
			maxSuff := ere.suffMask
			if !ere.isRangeEmptyInBlockLinear(start, end, suffA, maxSuff) {
				return false
			}
		}
		if ere.D1.Bit(blockB) {
			start, end := ere.getBlockRange(blockB)
			suffB := b & ere.suffMask
			if !ere.isRangeEmptyInBlockLinear(start, end, 0, suffB) {
				return false
			}
		}
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
	size += ere.D1.AllocSize()
	size += ere.D2.AllocSize()
	size += len(ere.packedData) * 8
	return size
}

func (ere *ExactRangeEmptiness) SizeInBits() uint64 {
	if ere == nil || ere.n == 0 {
		return 0
	}
	d1Bits := ere.D1.Num()
	d2Bits := ere.D2.Num()
	suffixBits := uint64(ere.n) * uint64(ere.w)
	return d1Bits + d2Bits + suffixBits
}

// MetadataNumBits returns the logical (information-theoretic) metadata bit
// count, namely the number of bits actually pushed into D1 and D2 (excluding
// the packed suffix array A_ds and any rank/select auxiliary indices).
func (ere *ExactRangeEmptiness) MetadataNumBits() uint64 {
	if ere == nil || ere.n == 0 {
		return 0
	}
	return ere.D1.Num() + ere.D2.Num()
}

// MetadataAllocBits returns the actual allocated metadata size in bits, summing
// the rsdic backing storage of D1 and D2 (raw bit array + rank/select indices).
// Excludes the packed suffix array A_ds.
func (ere *ExactRangeEmptiness) MetadataAllocBits() uint64 {
	if ere == nil || ere.n == 0 {
		return 0
	}
	return uint64(ere.D1.AllocSize()+ere.D2.AllocSize()) * 8
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
	nonEmpty := int(ere.D1.Rank(uint64(ere.numBlocks), true))
	maxKeys := 0
	var sumSq uint64
	for b := uint64(0); b < uint64(ere.numBlocks); b++ {
		if ere.D1.Bit(b) {
			start, end := ere.getBlockRange(b)
			count := end - start
			if count > maxKeys {
				maxKeys = count
			}
			sumSq += uint64(count) * uint64(count)
		}
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
// order. The returned slice has length equal to GetStats().NonEmptyBlocks.
func (ere *ExactRangeEmptiness) NonEmptyBlockSizes() []int {
	if ere == nil || ere.n == 0 {
		return nil
	}
	out := make([]int, 0, ere.numBlocks)
	for b := uint64(0); b < uint64(ere.numBlocks); b++ {
		if ere.D1.Bit(b) {
			start, end := ere.getBlockRange(b)
			out = append(out, end-start)
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
			{Name: "D1_blocks", TotalBytes: ere.D1.AllocSize()},
			{Name: "D2_counts", TotalBytes: ere.D2.AllocSize()},
			{Name: "suffixes_packed", TotalBytes: len(ere.packedData) * 8},
		},
	}
}
