package azft

import (
	"Thesis/bits"
)

// NewApproxZFastTrieFromIteratorStreaming creates AZFT with reduced memory overhead.
//
// The iterator MUST provide keys in sorted order.
func NewApproxZFastTrieFromIteratorStreaming[E UNumber, S UNumber, I UNumber](
	iter bits.BitStringIterator,
	seed uint64,
) (*ApproxZFastTrie[E, S, I], error) {
	return NewApproxZFastTrieFromIteratorLight[E, S, I](iter, seed)
}