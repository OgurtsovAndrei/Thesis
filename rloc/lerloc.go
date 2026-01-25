package rloc

import (
	"Thesis/bits"
	"Thesis/zfasttrie"
)

type LocalExactRangeLocator struct {
	hzft *zfasttrie.HZFastTrie[uint32]
	rl   *RangeLocator
}

func NewLocalExactRangeLocator(keys []bits.BitString) *LocalExactRangeLocator {
	if len(keys) == 0 {
		return &LocalExactRangeLocator{}
	}

	zt := zfasttrie.Build(keys)
	hzft := zfasttrie.NewHZFastTrie[uint32](keys)
	rl := NewRangeLocator(zt)

	return &LocalExactRangeLocator{
		hzft: hzft,
		rl:   rl,
	}
}

func (lerl *LocalExactRangeLocator) WeakPrefixSearch(prefix bits.BitString) (int, int, error) {
	if lerl.hzft == nil || lerl.rl == nil {
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
