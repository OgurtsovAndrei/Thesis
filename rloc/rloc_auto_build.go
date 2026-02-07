package rloc

import (
	"Thesis/errutil"
	"Thesis/zfasttrie"
	"fmt"
)

type widthChoice struct {
	e int
	s int
	i int
}

func autoWidthChoices(pSize int, maxBitLen int) []widthChoice {
	eStart := widthForMaxValue(maxBitLen)
	iStart := widthForIndexWithSentinel(estimatedDelimiterTrieNodes(pSize) - 1)
	sStart := suggestedSignatureWidth(pSize, maxBitLen)

	eCands := widthCandidates(eStart)
	sCands := widthCandidates(sStart)
	iCands := widthCandidates(iStart)

	choices := make([]widthChoice, 0, len(eCands)*len(sCands)*len(iCands))
	for _, e := range eCands {
		for _, s := range sCands {
			for _, i := range iCands {
				choices = append(choices, widthChoice{e: e, s: s, i: i})
			}
		}
	}
	return choices
}

func buildWithWidths(sortedItems []pItem, mmphSeed uint64, choice widthChoice) (RangeLocator, error) {
	switch choice.e {
	case 8:
		return buildWithWidthsE8(sortedItems, mmphSeed, choice)
	case 16:
		return buildWithWidthsE16(sortedItems, mmphSeed, choice)
	case 32:
		return buildWithWidthsE32(sortedItems, mmphSeed, choice)
	default:
		errutil.Bug("unsupported E width %d", choice.e)
		return nil, fmt.Errorf("unsupported E width %d", choice.e)
	}
}

func buildWithWidthsE8(sortedItems []pItem, mmphSeed uint64, choice widthChoice) (RangeLocator, error) {
	switch choice.s {
	case 8:
		switch choice.i {
		case 8:
			return safeBuildGenericRangeLocatorFromItems[uint8, uint8, uint8](sortedItems, mmphSeed)
		case 16:
			return safeBuildGenericRangeLocatorFromItems[uint8, uint8, uint16](sortedItems, mmphSeed)
		case 32:
			return safeBuildGenericRangeLocatorFromItems[uint8, uint8, uint32](sortedItems, mmphSeed)
		default:
			errutil.Bug("unsupported I width %d for E=%d S=%d", choice.i, choice.e, choice.s)
		}
	case 16:
		switch choice.i {
		case 8:
			return safeBuildGenericRangeLocatorFromItems[uint8, uint16, uint8](sortedItems, mmphSeed)
		case 16:
			return safeBuildGenericRangeLocatorFromItems[uint8, uint16, uint16](sortedItems, mmphSeed)
		case 32:
			return safeBuildGenericRangeLocatorFromItems[uint8, uint16, uint32](sortedItems, mmphSeed)
		default:
			errutil.Bug("unsupported I width %d for E=%d S=%d", choice.i, choice.e, choice.s)
		}
	case 32:
		switch choice.i {
		case 8:
			return safeBuildGenericRangeLocatorFromItems[uint8, uint32, uint8](sortedItems, mmphSeed)
		case 16:
			return safeBuildGenericRangeLocatorFromItems[uint8, uint32, uint16](sortedItems, mmphSeed)
		case 32:
			return safeBuildGenericRangeLocatorFromItems[uint8, uint32, uint32](sortedItems, mmphSeed)
		default:
			errutil.Bug("unsupported I width %d for E=%d S=%d", choice.i, choice.e, choice.s)
		}
	default:
		errutil.Bug("unsupported S width %d for E=%d", choice.s, choice.e)
	}

	return nil, fmt.Errorf("unsupported width combination E=%d S=%d I=%d", choice.e, choice.s, choice.i)
}

func buildWithWidthsE16(sortedItems []pItem, mmphSeed uint64, choice widthChoice) (RangeLocator, error) {
	switch choice.s {
	case 8:
		switch choice.i {
		case 8:
			return safeBuildGenericRangeLocatorFromItems[uint16, uint8, uint8](sortedItems, mmphSeed)
		case 16:
			return safeBuildGenericRangeLocatorFromItems[uint16, uint8, uint16](sortedItems, mmphSeed)
		case 32:
			return safeBuildGenericRangeLocatorFromItems[uint16, uint8, uint32](sortedItems, mmphSeed)
		default:
			errutil.Bug("unsupported I width %d for E=%d S=%d", choice.i, choice.e, choice.s)
		}
	case 16:
		switch choice.i {
		case 8:
			return safeBuildGenericRangeLocatorFromItems[uint16, uint16, uint8](sortedItems, mmphSeed)
		case 16:
			return safeBuildGenericRangeLocatorFromItems[uint16, uint16, uint16](sortedItems, mmphSeed)
		case 32:
			return safeBuildGenericRangeLocatorFromItems[uint16, uint16, uint32](sortedItems, mmphSeed)
		default:
			errutil.Bug("unsupported I width %d for E=%d S=%d", choice.i, choice.e, choice.s)
		}
	case 32:
		switch choice.i {
		case 8:
			return safeBuildGenericRangeLocatorFromItems[uint16, uint32, uint8](sortedItems, mmphSeed)
		case 16:
			return safeBuildGenericRangeLocatorFromItems[uint16, uint32, uint16](sortedItems, mmphSeed)
		case 32:
			return safeBuildGenericRangeLocatorFromItems[uint16, uint32, uint32](sortedItems, mmphSeed)
		default:
			errutil.Bug("unsupported I width %d for E=%d S=%d", choice.i, choice.e, choice.s)
		}
	default:
		errutil.Bug("unsupported S width %d for E=%d", choice.s, choice.e)
	}

	return nil, fmt.Errorf("unsupported width combination E=%d S=%d I=%d", choice.e, choice.s, choice.i)
}

func buildWithWidthsE32(sortedItems []pItem, mmphSeed uint64, choice widthChoice) (RangeLocator, error) {
	switch choice.s {
	case 8:
		switch choice.i {
		case 8:
			return safeBuildGenericRangeLocatorFromItems[uint32, uint8, uint8](sortedItems, mmphSeed)
		case 16:
			return safeBuildGenericRangeLocatorFromItems[uint32, uint8, uint16](sortedItems, mmphSeed)
		case 32:
			return safeBuildGenericRangeLocatorFromItems[uint32, uint8, uint32](sortedItems, mmphSeed)
		default:
			errutil.Bug("unsupported I width %d for E=%d S=%d", choice.i, choice.e, choice.s)
		}
	case 16:
		switch choice.i {
		case 8:
			return safeBuildGenericRangeLocatorFromItems[uint32, uint16, uint8](sortedItems, mmphSeed)
		case 16:
			return safeBuildGenericRangeLocatorFromItems[uint32, uint16, uint16](sortedItems, mmphSeed)
		case 32:
			return safeBuildGenericRangeLocatorFromItems[uint32, uint16, uint32](sortedItems, mmphSeed)
		default:
			errutil.Bug("unsupported I width %d for E=%d S=%d", choice.i, choice.e, choice.s)
		}
	case 32:
		switch choice.i {
		case 8:
			return safeBuildGenericRangeLocatorFromItems[uint32, uint32, uint8](sortedItems, mmphSeed)
		case 16:
			return safeBuildGenericRangeLocatorFromItems[uint32, uint32, uint16](sortedItems, mmphSeed)
		case 32:
			return safeBuildGenericRangeLocatorFromItems[uint32, uint32, uint32](sortedItems, mmphSeed)
		default:
			errutil.Bug("unsupported I width %d for E=%d S=%d", choice.i, choice.e, choice.s)
		}
	default:
		errutil.Bug("unsupported S width %d for E=%d", choice.s, choice.e)
	}

	return nil, fmt.Errorf("unsupported width combination E=%d S=%d I=%d", choice.e, choice.s, choice.i)
}

func safeBuildGenericRangeLocatorFromItems[E zfasttrie.UNumber, S zfasttrie.UNumber, I zfasttrie.UNumber](sortedItems []pItem, mmphSeed uint64) (rl *GenericRangeLocator[E, S, I], err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during generic build: %v", r)
			rl = nil
		}
	}()
	return newGenericRangeLocatorFromItems[E, S, I](sortedItems, mmphSeed)
}

func widthForMaxValue(maxValue int) int {
	switch {
	case maxValue <= 0xFF:
		return 8
	case maxValue <= 0xFFFF:
		return 16
	case maxValue <= 0xFFFFFFFF:
		return 32
	default:
		return 64
	}
}

func widthForIndexWithSentinel(maxIndex int) int {
	switch {
	case maxIndex <= 0xFE:
		return 8
	case maxIndex <= 0xFFFE:
		return 16
	case uint64(maxIndex) <= 0xFFFFFFFE:
		return 32
	default:
		return 64
	}
}

func widthCandidates(start int) []int {
	base := []int{8, 16, 32}
	out := make([]int, 0, len(base))
	for _, b := range base {
		if b >= start {
			out = append(out, b)
		}
	}
	return out
}

func estimatedDelimiterTrieNodes(pSize int) int {
	if pSize <= 0 {
		return 1
	}
	numBuckets := (pSize + 255) / 256
	// Conservative small bound for node-like entries in delimiter trie.
	return 4*numBuckets + 2
}

func suggestedSignatureWidth(pSize int, maxBitLen int) int {
	switch {
	case pSize <= 1<<10 && maxBitLen <= 0xFF:
		return 8
	case pSize <= 1<<24 && maxBitLen <= 0xFFFF:
		return 16
	default:
		return 32
	}
}
