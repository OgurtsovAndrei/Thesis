package rloc

import (
	"Thesis/bits"
	boomphf "Thesis/mmph/go-boomphf-bs"
	"Thesis/zfasttrie"
	"fmt"
	"sort"

	"github.com/hillbig/rsdic"
)

type RangeLocator struct {
	mph         *boomphf.H
	perm        []uint32
	bv          *rsdic.RSDic
	totalLeaves int
}

type pItem struct {
	bs     bits.BitString
	isLeaf bool
}

func NewRangeLocator(zt *zfasttrie.ZFastTrie[bool]) *RangeLocator {
	if zt == nil {
		return &RangeLocator{totalLeaves: 0}
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
		return sortedItems[i].bs.Compare(sortedItems[j].bs) < 0
	})

	// Build structures
	bv := rsdic.New()
	keysForMPH := make([]bits.BitString, len(sortedItems))

	for i, item := range sortedItems {
		bv.PushBack(item.isLeaf)
		keysForMPH[i] = item.bs
	}

	mph := boomphf.New(boomphf.Gamma, keysForMPH)

	perm := make([]uint32, len(sortedItems))
	for rank, item := range sortedItems {
		idx := mph.Query(item.bs) - 1
		perm[idx] = uint32(rank)
	}

	totalLeaves := 0
	if bv.Num() > 0 {
		totalLeaves = int(bv.Rank(bv.Num(), true))
	}

	return &RangeLocator{
		mph:         mph,
		perm:        perm,
		bv:          bv,
		totalLeaves: totalLeaves,
	}
}

func (rl *RangeLocator) Query(nodeName bits.BitString) (int, int, error) {
	if nodeName.Size() == 0 {
		return 0, rl.totalLeaves, nil
	}

	// Use TrimTrailingZeros instead of string conversion and trimming
	xArrowBs := nodeName.TrimTrailingZeros()
	idxLeft := rl.mph.Query(xArrowBs)

	if idxLeft == 0 {
		return 0, 0, fmt.Errorf("key not found in structure")
	}

	lexRankLeft := rl.perm[idxLeft-1]
	i := int(rl.bv.Rank(uint64(lexRankLeft), true))

	var j int

	if isAllOnes(nodeName) {
		j = rl.totalLeaves
	} else {
		xSucc := calcSuccessor(nodeName)
		// Use TrimTrailingZeros instead of string conversion and trimming
		xSuccArrowBs := xSucc.TrimTrailingZeros()

		idxRight := rl.mph.Query(xSuccArrowBs)
		if idxRight == 0 {
			return i, i, nil
		}

		lexRankRight := rl.perm[idxRight-1]
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
