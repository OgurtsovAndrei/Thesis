package rloc

import (
	"Thesis/bits"
	"Thesis/errutil"
	"Thesis/zfasttrie"
	"fmt"
)

type hzftAccessor interface {
	GetExistingPrefix(pattern bits.BitString) int64
	ByteSize() int
}

type autoLocalExactRangeLocator struct {
	hzft   hzftAccessor
	rl     RangeLocator
	widths TypeWidths
}

func (lerl *autoLocalExactRangeLocator) WeakPrefixSearch(prefix bits.BitString) (int, int, error) {
	if lerl == nil || lerl.hzft == nil || lerl.rl == nil {
		return 0, 0, nil
	}

	exitNodeLength := lerl.hzft.GetExistingPrefix(prefix)

	var exitNode bits.BitString
	if exitNodeLength == 0 {
		exitNode = bits.NewFromText("")
	} else {
		exitNode = prefix.Prefix(int(exitNodeLength))
	}

	return lerl.rl.Query(exitNode)
}

func (lerl *autoLocalExactRangeLocator) ByteSize() int {
	if lerl == nil {
		return 0
	}

	size := 0
	if lerl.hzft != nil {
		size += lerl.hzft.ByteSize()
	}
	if lerl.rl != nil {
		size += lerl.rl.ByteSize()
	}
	return size
}

func (lerl *autoLocalExactRangeLocator) TypeWidths() TypeWidths {
	if lerl == nil {
		return TypeWidths{}
	}
	return lerl.widths
}

func buildHZFastTrieWithWidth(keys []bits.BitString, eWidth int) (hzftAccessor, error) {
	switch eWidth {
	case 8:
		return zfasttrie.NewHZFastTrie[uint8](keys), nil
	case 16:
		return zfasttrie.NewHZFastTrie[uint16](keys), nil
	case 32:
		return zfasttrie.NewHZFastTrie[uint32](keys), nil
	default:
		errutil.Bug("unsupported HZFastTrie E width %d", eWidth)
		return nil, fmt.Errorf("unsupported HZFastTrie E width %d", eWidth)
	}
}
