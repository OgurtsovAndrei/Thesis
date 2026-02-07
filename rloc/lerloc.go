package rloc

import (
	"Thesis/bits"
	"Thesis/zfasttrie"
	"unsafe"
)

// LocalExactRangeLocator supports weak prefix search and returns exact rank
// intervals in the sorted key set.
//
// This is the runtime-facing interface. The generic implementation is
// GenericLocalExactRangeLocator[E, S, I].
type LocalExactRangeLocator interface {
	WeakPrefixSearch(prefix bits.BitString) (int, int, error)
	ByteSize() int
	TypeWidths() TypeWidths
}

// GenericLocalExactRangeLocator supports weak prefix search and returns exact
// rank intervals in the sorted key set.
//
// Memory analysis:
//
//   - Theoretical asymptotics (from "Fast Prefix Search in Little Space, with
//     Applications"): this structure is the sum of an exit-node locator
//     (implemented here as HZFastTrie) and a RangeLocator.
//     Space(LocalExactRangeLocator) = Space(HZFastTrie) + Space(RangeLocator).
//     Using paper-level bounds for these components gives
//     O(n log l) + O(n log log l) = O(n log l) bits total.
//
//   - Concrete field-level resident memory in this implementation (64-bit):
//     struct payload is 16 bytes (hzft pointer + rl pointer), plus the pointed
//     component memory.
//
//   - Practical estimate from fields:
//     16 + hzft.ByteSize() + rl.ByteSize() bytes.
//
//   - Empirical resident-size range from recent BenchmarkMemoryComparison runs
//     (see mmph/paramselect/study/memory_bench_v2.txt):
//     about 115.3..320.0 bits/key, with ~116..246 bits/key in the larger-key
//     regime (keys >= 8192 in that run).
//
// References:
//   - papers/Fast Prefix Search.pdf (Local Exact Range Locator decomposition)
//   - papers/Hollow-Z-Fast-Trie (Fast Prefix Search)/Section-5.md
//   - papers/Hollow-Z-Fast-Trie (Fast Prefix Search)/Section-6.md
type GenericLocalExactRangeLocator[E zfasttrie.UNumber, S zfasttrie.UNumber, I zfasttrie.UNumber] struct {
	hzft *zfasttrie.HZFastTrie[E]
	rl   *GenericRangeLocator[E, S, I]
}

// NewLocalExactRangeLocator builds a local-exact weak-prefix search structure.
//
// It composes an HZFastTrie (to map a query prefix to an exit node) with a
// RangeLocator (to convert that node into a leaf-rank interval).
//
// The constructor chooses the smallest practical type widths for E/S/I from
// input data and reuses the chosen E for HZFastTrie.
func NewLocalExactRangeLocator(keys []bits.BitString) (LocalExactRangeLocator, error) {
	if len(keys) == 0 {
		return &autoLocalExactRangeLocator{
			widths: TypeWidths{E: 8, S: 8, I: 8},
		}, nil
	}

	zt := zfasttrie.Build(keys)
	rl, err := NewRangeLocator(zt)
	if err != nil {
		return nil, err
	}

	widths := rl.TypeWidths()
	hzft, err := buildHZFastTrieWithWidth(keys, widths.E)
	if err != nil {
		return nil, err
	}

	return &autoLocalExactRangeLocator{
		hzft:   hzft,
		rl:     rl,
		widths: widths,
	}, nil
}

// NewGenericLocalExactRangeLocator builds a generic local-exact weak-prefix
// search structure.
func NewGenericLocalExactRangeLocator[E zfasttrie.UNumber, S zfasttrie.UNumber, I zfasttrie.UNumber](keys []bits.BitString) (*GenericLocalExactRangeLocator[E, S, I], error) {
	if len(keys) == 0 {
		return &GenericLocalExactRangeLocator[E, S, I]{}, nil
	}

	zt := zfasttrie.Build(keys)
	hzft := zfasttrie.NewHZFastTrie[E](keys)
	rl, err := NewGenericRangeLocator[E, S, I](zt)
	if err != nil {
		return nil, err
	}

	return &GenericLocalExactRangeLocator[E, S, I]{
		hzft: hzft,
		rl:   rl,
	}, nil
}

// WeakPrefixSearch returns the [start, end) rank interval for keys matching
// prefix.
func (lerl *GenericLocalExactRangeLocator[E, S, I]) WeakPrefixSearch(prefix bits.BitString) (int, int, error) {
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

// ByteSize returns the estimated resident size of LocalExactRangeLocator.
//
// It is the sum of HZFastTrie and RangeLocator sizes and excludes temporary
// allocations made during construction.
func (lerl *GenericLocalExactRangeLocator[E, S, I]) ByteSize() int {
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

// TypeWidths returns bit-widths of generic integer parameters used by this
// concrete instance.
func (lerl *GenericLocalExactRangeLocator[E, S, I]) TypeWidths() TypeWidths {
	return TypeWidths{
		E: int(unsafe.Sizeof(*new(E))) * 8,
		S: int(unsafe.Sizeof(*new(S))) * 8,
		I: int(unsafe.Sizeof(*new(I))) * 8,
	}
}
