package compact_lerloc

import (
	"Thesis/bits"
)

// NewAutoCompactLocalExactRangeLocator returns a CompactLocalExactRangeLocator.
// It is kept for backwards compatibility with test and benchmark files that
// expected the generic auto-selection behavior.
func NewAutoCompactLocalExactRangeLocator(keys []bits.BitString) (interface {
	WeakPrefixSearch(prefix bits.BitString) (int, int, error)
	ByteSize() int
}, error) {
	return NewCompactLocalExactRangeLocator(keys)
}