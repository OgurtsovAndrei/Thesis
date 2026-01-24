package rloc

import (
	"Thesis/bits"
	boomphf "Thesis/mmph/go-boomphf-bs"
	"Thesis/zfasttrie"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/hillbig/rsdic"
)

type RangeLocator struct {
	mph         *boomphf.H
	perm        []uint32
	bv          *rsdic.RSDic
	totalLeaves int
}

type pItem struct {
	s      string
	bs     bits.BitString
	isLeaf bool
}

func toBinary(bs bits.BitString) string {
	var sb strings.Builder
	for i := uint32(0); i < bs.Size(); i++ {
		if bs.At(i) {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	return sb.String()
}

func NewRangeLocator(zt *zfasttrie.ZFastTrie[bool]) *RangeLocator {
	if zt == nil {
		return &RangeLocator{totalLeaves: 0}
	}

	pMap := make(map[string]bool)

	addToMap := func(s string, isLeaf bool) {
		if val, exists := pMap[s]; exists {
			if isLeaf {
				pMap[s] = true
			} else {
				pMap[s] = val
			}
		} else {
			pMap[s] = isLeaf
		}
	}

	it := zfasttrie.NewIterator(zt)
	for it.Next() {
		node := it.Node()
		if node == nil {
			continue
		}

		extent := node.Extent
		binExtent := toBinary(extent)

		eArrow := strings.TrimRight(binExtent, "0")
		addToMap(eArrow, node.IsLeaf)

		e1 := binExtent + "1"
		addToMap(e1, false)

		if !isAllOnes(extent) {
			successor := calcSuccessor(extent)
			succArrow := strings.TrimRight(toBinary(successor), "0")
			addToMap(succArrow, false)
		}
	}

	sortedP := make([]pItem, 0, len(pMap))
	for s, isLeaf := range pMap {
		sortedP = append(sortedP, pItem{
			s:      s,
			bs:     bits.NewFromBinary(s),
			isLeaf: isLeaf,
		})
	}

	sort.Slice(sortedP, func(i, j int) bool {
		return sortedP[i].s < sortedP[j].s
	})

	bv := rsdic.New()
	keysForMPH := make([]bits.BitString, len(sortedP))

	for i, item := range sortedP {
		bv.PushBack(item.isLeaf)
		keysForMPH[i] = item.bs
	}

	mph := boomphf.New(boomphf.Gamma, keysForMPH)

	perm := make([]uint32, len(sortedP))
	for rank, item := range sortedP {
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

	binX := toBinary(nodeName)
	xArrow := strings.TrimRight(binX, "0")
	idxLeft := rl.mph.Query(bits.NewFromBinary(xArrow))

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
		xSuccArrow := strings.TrimRight(toBinary(xSucc), "0")

		idxRight := rl.mph.Query(bits.NewFromBinary(xSuccArrow))
		if idxRight == 0 {
			return i, i, nil
		}

		lexRankRight := rl.perm[idxRight-1]
		j = int(rl.bv.Rank(uint64(lexRankRight), true))
	}

	return i, j, nil
}

func isAllOnes(bs bits.BitString) bool {
	sz := bs.Size()
	for k := uint32(0); k < sz; k++ {
		if !bs.At(k) {
			return false
		}
	}
	return true
}

func calcSuccessor(bs bits.BitString) bits.BitString {
	s := toBinary(bs) + "1"
	i := new(big.Int)
	i.SetString(s, 2)
	i.Add(i, big.NewInt(1))
	resStr := i.Text(2)

	if len(resStr) < len(s) {
		resStr = strings.Repeat("0", len(s)-len(resStr)) + resStr
	} else if len(resStr) > len(s) {
		resStr = resStr[len(resStr)-len(s):]
	}

	return bits.NewFromBinary(resStr)
}
