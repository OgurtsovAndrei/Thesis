package bits

// BitStringIterator iterates over a sequence of BitStrings.
type BitStringIterator interface {
	// Next advances the iterator to the next element.
	// Returns true if an element is available, false if the sequence is exhausted or an error occurred.
	Next() bool

	// Value returns the current BitString.
	// Should only be called after Next() returns true.
	Value() BitString

	// Error returns the first error encountered during iteration, if any.
	Error() error
}

// SliceBitStringIterator adapts a slice of BitStrings to the BitStringIterator interface.
type SliceBitStringIterator struct {
	keys []BitString
	idx  int
}

func NewSliceBitStringIterator(keys []BitString) *SliceBitStringIterator {
	return &SliceBitStringIterator{keys: keys, idx: -1}
}

func (it *SliceBitStringIterator) Next() bool {
	it.idx++
	return it.idx < len(it.keys)
}

func (it *SliceBitStringIterator) Value() BitString {
	return it.keys[it.idx]
}

func (it *SliceBitStringIterator) Error() error {
	return nil
}
