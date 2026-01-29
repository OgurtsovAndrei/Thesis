package rloc

import (
	"Thesis/bits"
	"Thesis/zfasttrie"
)

type LocalExactRangeLocator struct {
	hzft *zfasttrie.HZFastTrie[uint32]
	rl   *RangeLocator
}

func NewLocalExactRangeLocator(keys []bits.BitString) (*LocalExactRangeLocator, error) {
	if len(keys) == 0 {
		return &LocalExactRangeLocator{}, nil
	}

	zt := zfasttrie.Build(keys)
	hzft := zfasttrie.NewHZFastTrie[uint32](keys)
	rl, err := NewRangeLocator(zt)
	if err != nil {
		return nil, err
	}

	return &LocalExactRangeLocator{
		hzft: hzft,
		rl:   rl,
	}, nil
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

// ByteSize returns the total size of the structure in bytes.
func (lerl *LocalExactRangeLocator) ByteSize() int {
	if lerl == nil {
		return 0
	}

	size := 0

	// Size of the HZFastTrie
	if lerl.hzft != nil {
		size += lerl.hzft.ByteSize()
	}

	// Size of the RangeLocator
	if lerl.rl != nil {
		size += lerl.rl.ByteSize()
	}

	return size
}
