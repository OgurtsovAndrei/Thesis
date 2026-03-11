package local_exact_range

import (
	"Thesis/bits"
	"Thesis/utils"
	"fmt"
	"math"
	"unsafe"

	"github.com/hillbig/rsdic"
)

// ExactRangeEmptiness implements the 1D range emptiness structure from SODA 2015, Section 3.2.
type ExactRangeEmptiness struct {
	D1         *rsdic.RSDic
	D2         *rsdic.RSDic
	packedData []uint64

	n         int
	numBlocks int
	L         uint32
	k         uint32
	w         uint32
}

func NewExactRangeEmptiness(keys []bits.BitString, universe bits.BitString) (*ExactRangeEmptiness, error) {
	n := len(keys)
	if n == 0 {
		return &ExactRangeEmptiness{n: 0}, nil
	}

	for i := 1; i < n; i++ {
		if keys[i-1].Compare(keys[i]) > 0 {
			return nil, fmt.Errorf("keys must be sorted")
		}
	}

	k := uint32(math.Floor(math.Log2(float64(n))))
	if k == 0 {
		k = 1
	}

	numBlocks := 1 << k
	L := universe.Size()
	if L < k {
		L = k
	}
	w := L - k

	D1 := rsdic.New()
	D2 := rsdic.New()
	suffixes := make([]uint64, 0, n)

	i := 0
	for b := 0; b < numBlocks; b++ {
		countInBlock := 0
		for i < n && getBlockIndex(keys[i], k) == uint64(b) {
			suffixes = append(suffixes, extractSuffixAsUint64(keys[i], L, w))
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
		L:          L,
		k:          k,
		w:          w,
	}, nil
}

func getBlockIndex(x bits.BitString, k uint32) uint64 {
	var idx uint64
	size := x.Size()
	// Bit 0 is MSB of index
	for i := uint32(0); i < k; i++ {
		if i < size && x.At(i) {
			idx |= (uint64(1) << (k - 1 - i))
		}
	}
	return idx
}

func extractSuffixAsUint64(bs bits.BitString, L, w uint32) uint64 {
	var val uint64
	size := bs.Size()
	k := L - w
	// Bit k is MSB of suffix
	for i := uint32(0); i < w; i++ {
		pos := k + i
		if pos < size && bs.At(pos) {
			val |= (uint64(1) << (w - 1 - i))
		}
	}
	return val
}

func (ere *ExactRangeEmptiness) IsEmpty(a, b bits.BitString) bool {
	if ere.n == 0 {
		return true
	}
	if a.Compare(b) > 0 {
		return true
	}

	blockA := getBlockIndex(a, ere.k)
	blockB := getBlockIndex(b, ere.k)

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
			suffA := extractSuffixAsUint64(a, ere.L, ere.w)
			suffB := extractSuffixAsUint64(b, ere.L, ere.w)
			if !ere.isRangeEmptyInBlock(start, end, suffA, suffB) {
				return false
			}
		}
	} else {
		// Check blockA for elements in [suffA, max]
		if ere.D1.Bit(blockA) {
			start, end := ere.getBlockRange(blockA)
			suffA := extractSuffixAsUint64(a, ere.L, ere.w)
			maxSuff := (uint64(1) << ere.w) - 1
			if ere.w == 64 {
				maxSuff = ^uint64(0)
			}
			if !ere.isRangeEmptyInBlock(start, end, suffA, maxSuff) {
				return false
			}
		}
		// Check blockB for elements in [0, suffB]
		if ere.D1.Bit(blockB) {
			start, end := ere.getBlockRange(blockB)
			suffB := extractSuffixAsUint64(b, ere.L, ere.w)
			if !ere.isRangeEmptyInBlock(start, end, 0, suffB) {
				return false
			}
		}
	}

	return true
}

func (ere *ExactRangeEmptiness) getBlockRange(blockIdx uint64) (int, int) {
	numNonEmptyBefore := int(ere.D1.Rank(blockIdx, true))
	posInD2 := ere.D2.Select(uint64(numNonEmptyBefore), true)
	startIndex := int(posInD2 - uint64(numNonEmptyBefore))
	posEndInD2 := ere.D2.Select(uint64(numNonEmptyBefore+1), true)
	endIndex := int(posEndInD2 - uint64(numNonEmptyBefore+1))
	return startIndex, endIndex
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

type Stats struct {
	N               int
	NumBlocks       int
	NonEmptyBlocks  int
	EmptyBlocks     int
	AvgKeysPerBlock float64
	MaxKeysInBlock  int
	EmptyBlockPct   float64
}

func (ere *ExactRangeEmptiness) GetStats() Stats {
	nonEmpty := int(ere.D1.Rank(uint64(ere.numBlocks), true))
	maxKeys := 0
	for b := uint64(0); b < uint64(ere.numBlocks); b++ {
		if ere.D1.Bit(b) {
			start, end := ere.getBlockRange(b)
			count := end - start
			if count > maxKeys {
				maxKeys = count
			}
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
	}
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
