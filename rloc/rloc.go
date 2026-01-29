package rloc

import (
	"Thesis/bits"
	bucket "Thesis/mmph/bucket_with_approx_trie"
	"Thesis/zfasttrie"
	"fmt"
	"sort"

	"github.com/hillbig/rsdic"
)

type RangeLocator struct {
	mmph        *bucket.MonotoneHashWithTrie[uint16, uint16, uint16]
	bv          *rsdic.RSDic
	totalLeaves int
}

type pItem struct {
	bs     bits.BitString
	isLeaf bool
}

func NewRangeLocator(zt *zfasttrie.ZFastTrie[bool]) (*RangeLocator, error) {
	if zt == nil {
		return &RangeLocator{totalLeaves: 0}, nil
	}

	// Use BitString directly as map key
	pMap := make(map[bits.BitString]bool)

	addToMap := func(bs bits.BitString, isLeaf bool) {
		if existingIsLeaf, exists := pMap[bs]; exists {
			// Prioritize leaf status
			if isLeaf {
				pMap[bs] = true
			} else {
				pMap[bs] = existingIsLeaf
			}
		} else {
			pMap[bs] = isLeaf
		}
	}

	it := zfasttrie.NewIterator(zt)
	for it.Next() {
		node := it.Node()
		if node == nil {
			continue
		}

		extent := node.Extent

		// Use TrimTrailingZeros instead of string conversion and trimming
		eArrowBs := extent.TrimTrailingZeros()
		addToMap(eArrowBs, node.IsLeaf)

		// Use AppendBit instead of string concatenation
		e1Bs := extent.AppendBit(true)
		addToMap(e1Bs, false)

		if !isAllOnes(extent) {
			successor := calcSuccessor(extent)
			// Use TrimTrailingZeros instead of string conversion and trimming
			succArrowBs := successor.TrimTrailingZeros()
			addToMap(succArrowBs, false)
		}
	}

	// Convert to sorted slice
	sortedItems := make([]pItem, 0, len(pMap))
	for bs, isLeaf := range pMap {
		sortedItems = append(sortedItems, pItem{
			bs:     bs,
			isLeaf: isLeaf,
		})
	}

	sort.Slice(sortedItems, func(i, j int) bool {
		return sortedItems[i].bs.TrieCompare(sortedItems[j].bs) < 0
	})

	// Build structures
	bv := rsdic.New()
	keysForMMPH := make([]bits.BitString, len(sortedItems))

	for i, item := range sortedItems {
		bv.PushBack(item.isLeaf)
		keysForMMPH[i] = item.bs
	}

	// Build MMPH - data is already sorted in TrieCompare order
	// Type parameters: E=uint16 (extent length), S=uint16 (signature), I=uint16 (delimiter index)
	mmph, err := bucket.NewMonotoneHashWithTrie[uint16, uint16, uint16](keysForMMPH)
	if err != nil {
		return nil, fmt.Errorf("failed to build MMPH for P set of size %d: %w", len(keysForMMPH), err)
	}

	totalLeaves := 0
	if bv.Num() > 0 {
		totalLeaves = int(bv.Rank(bv.Num(), true))
	}

	return &RangeLocator{
		mmph:        mmph,
		bv:          bv,
		totalLeaves: totalLeaves,
	}, nil
}

func (rl *RangeLocator) Query(nodeName bits.BitString) (int, int, error) {
	if nodeName.Size() == 0 {
		return 0, rl.totalLeaves, nil
	}

	if rl.mmph == nil {
		return 0, 0, fmt.Errorf("MMPH not initialized")
	}

	// Use TrimTrailingZeros instead of string conversion and trimming
	xArrowBs := nodeName.TrimTrailingZeros()
	lexRankLeft := rl.mmph.GetRank(xArrowBs)

	if lexRankLeft == -1 {
		return 0, 0, fmt.Errorf("key not found in structure")
	}

	i := int(rl.bv.Rank(uint64(lexRankLeft), true))

	var j int

	if isAllOnes(nodeName) {
		j = rl.totalLeaves
	} else {
		xSucc := calcSuccessor(nodeName)
		// Use TrimTrailingZeros instead of string conversion and trimming
		xSuccArrowBs := xSucc.TrimTrailingZeros()

		lexRankRight := rl.mmph.GetRank(xSuccArrowBs)
		if lexRankRight == -1 {
			return i, i, nil
		}

		j = int(rl.bv.Rank(uint64(lexRankRight), true))
	}

	return i, j, nil
}

func isAllOnes(bs bits.BitString) bool {
	return bs.IsAllOnes()
}

func calcSuccessor(bs bits.BitString) bits.BitString {
	// Use the efficient BitString method that appends '1' and computes successor
	return bs.AppendBit(true).Successor()
}

// ByteSize returns the total size of the structure in bytes.
func (rl *RangeLocator) ByteSize() int {
	if rl == nil {
		return 0
	}

	size := 0

	// Size of the MMPH (Monotone Minimal Perfect Hash function)
	if rl.mmph != nil {
		size += rl.mmph.ByteSize()
	}

	// Size of the bit vector
	if rl.bv != nil {
		// RSDic doesn't expose Size() method, approximate based on bits stored
		size += int(rl.bv.Num()/8) + 64 // rough estimate: bits/8 + overhead
	}

	// Size of totalLeaves (int)
	size += 8 // assuming 64-bit int

	return size
}
