package lerloc

import (
	"Thesis/bits"
	"Thesis/errutil"
	"Thesis/locators/rloc"
	"Thesis/trie/hzft"
	"Thesis/trie/zft"
	"Thesis/utils"
	"fmt"
	"unsafe"
)

type hzftAccessor interface {
	GetExistingPrefix(pattern bits.BitString) int64
	ByteSize() int
}

type autoLocalExactRangeLocator struct {
	hzft   hzftAccessor
	rl     rloc.RangeLocator
	widths rloc.TypeWidths
}

// NewLocalExactRangeLocator constructs a LocalExactRangeLocator with automatically
// selected bit-widths for internal fields to minimize memory usage.
//
// It first builds a RangeLocator (which selects optimal widths), then uses the
// selected 'E' width for the HZFastTrie component.
func NewLocalExactRangeLocator(keys []bits.BitString) (LocalExactRangeLocator, error) {
	zt := zft.Build(keys)

	rl, err := rloc.NewRangeLocator(zt)
	if err != nil {
		return nil, err
	}

	widths := rl.TypeWidths()
	hzftComp, err := buildHZFastTrieWithWidth(keys, widths.E)
	if err != nil {
		return nil, err
	}

	return &autoLocalExactRangeLocator{
		hzft:   hzftComp,
		rl:     rl,
		widths: widths,
	}, nil
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

// MemDetailed returns a detailed memory usage report for autoLocalExactRangeLocator.
func (lerl *autoLocalExactRangeLocator) MemDetailed() utils.MemReport {
	if lerl == nil {
		return utils.MemReport{Name: "autoLocalExactRangeLocator", TotalBytes: 0}
	}

	headerSize := int(unsafe.Sizeof(*lerl))
	hzftSize := 0
	if lerl.hzft != nil {
		hzftSize = lerl.hzft.ByteSize()
	}
	rlReport := lerl.rl.MemDetailed()

	return utils.MemReport{
		Name:       "autoLocalExactRangeLocator",
		TotalBytes: lerl.ByteSize(),
		Children: []utils.MemReport{
			{Name: "header", TotalBytes: headerSize},
			{Name: "hzft", TotalBytes: hzftSize},
			rlReport,
		},
	}
}

func (lerl *autoLocalExactRangeLocator) TypeWidths() rloc.TypeWidths {
	if lerl == nil {
		return rloc.TypeWidths{}
	}
	return lerl.widths
}

func buildHZFastTrieWithWidth(keys []bits.BitString, eWidth int) (hzftAccessor, error) {
	switch eWidth {
	case 8:
		return hzft.NewHZFastTrie[uint8](keys), nil
	case 16:
		return hzft.NewHZFastTrie[uint16](keys), nil
	case 32:
		return hzft.NewHZFastTrie[uint32](keys), nil
	default:
		errutil.Bug("unsupported HZFastTrie E width %d", eWidth)
		return nil, fmt.Errorf("unsupported HZFastTrie E width %d", eWidth)
	}
}
