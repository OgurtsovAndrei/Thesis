package compact_lerloc

import (
	"Thesis/bits"
)

// NewAutoCompactLocalExactRangeLocator analyzes the maximum key length and chooses
// the smallest uint type (E) that can accommodate it to minimize the memory
// overhead of the HZFastTrie component.
func NewAutoCompactLocalExactRangeLocator(keys []bits.BitString) (interface {
	WeakPrefixSearch(prefix bits.BitString) (int, int, error)
	ByteSize() int
}, error) {
	if len(keys) == 0 {
		return NewCompactLocalExactRangeLocator[uint8](keys)
	}

	maxLen := 0
	for _, key := range keys {
		if l := int(key.Size()); l > maxLen {
			maxLen = l
		}
	}

	if maxLen <= 255 {
		return NewCompactLocalExactRangeLocator[uint8](keys)
	} else if maxLen <= 65535 {
		return NewCompactLocalExactRangeLocator[uint16](keys)
	} else if maxLen <= 4294967295 {
		return NewCompactLocalExactRangeLocator[uint32](keys)
	} else {
		return NewCompactLocalExactRangeLocator[uint64](keys)
	}
}