package compact_lerloc

import (
	"Thesis/bits"
	"Thesis/locators/compact_rloc"
	"Thesis/locators/rloc"
	"Thesis/trie/hzft"
	"Thesis/trie/zft"
	"Thesis/utils"
	"unsafe"
)

// CompactLocalExactRangeLocator supports weak prefix search and returns exact
// rank intervals in the sorted key set. It uses the highly compressed
// CompactRangeLocator based on LeMonHash, significantly reducing the memory
// footprint compared to the classical generic bucketing approach.
type CompactLocalExactRangeLocator[E zft.UNumber] struct {
	hzft *hzft.HZFastTrie[E]
	rl   *compact_rloc.CompactRangeLocator
}

// NewCompactLocalExactRangeLocator creates a new CompactLocalExactRangeLocator.
// It composes an HZFastTrie (to map a query prefix to an exit node) with a
// CompactRangeLocator (to map that exit node to a rank interval).
//
// The type parameter E determines the bit-width used for the internal
// length arrays in HZFastTrie.
//
// It assumes the keys provided are sorted. If not, HZFastTrie construction
// will panic.
func NewCompactLocalExactRangeLocator[E zft.UNumber](keys []bits.BitString) (*CompactLocalExactRangeLocator[E], error) {
	// Build RangeLocator first
	// We need to build a ZFastTrie to pass to NewCompactRangeLocator
	zt := zft.Build(keys)
	rl, err := compact_rloc.NewCompactRangeLocator(zt)
	if err != nil {
		return nil, err
	}

	// Build HZFastTrie
	// Note: NewHZFastTrie builds its own internal trie anyway.
	hz := hzft.NewHZFastTrie[E](keys)

	return &CompactLocalExactRangeLocator[E]{
		hzft: hz,
		rl:   rl,
	}, nil
}

// WeakPrefixSearch returns the half-open interval [i, j) of ranks of all
// elements in the original set that share the provided prefix.
func (lerl *CompactLocalExactRangeLocator[E]) WeakPrefixSearch(prefix bits.BitString) (int, int, error) {
	if lerl == nil || lerl.hzft == nil || lerl.rl == nil {
		return 0, 0, nil
	}

	// 1. Find the exit node using HZFastTrie
	// The paper says: "Let u be the node of T that is the exit node of P."
	// HZFastTrie returns the length of the exit node extent.
	exitNodeLength := lerl.hzft.GetExistingPrefix(prefix)

	var exitNode bits.BitString
	if exitNodeLength == 0 {
		exitNode = bits.NewFromText("")
	} else {
		exitNode = prefix.Prefix(int(exitNodeLength))
	}

	// 2. Query the Range Locator with the exit node name
	// The paper says: "Locate(u)"
	return lerl.rl.Query(exitNode)
}

// ByteSize returns the estimated resident size in bytes.
func (lerl *CompactLocalExactRangeLocator[E]) ByteSize() int {
	if lerl == nil {
		return 0
	}

	size := int(unsafe.Sizeof(*lerl))

	if lerl.hzft != nil {
		size += lerl.hzft.ByteSize()
	}

	if lerl.rl != nil {
		size += lerl.rl.ByteSize()
	}

	return size
}

// MemDetailed returns a detailed memory usage report.
func (lerl *CompactLocalExactRangeLocator[E]) MemDetailed() utils.MemReport {
	if lerl == nil {
		return utils.MemReport{Name: "CompactLocalExactRangeLocator", TotalBytes: 0}
	}

	headerSize := int(unsafe.Sizeof(*lerl))
	hzftSize := 0
	if lerl.hzft != nil {
		hzftSize = lerl.hzft.ByteSize()
	}
	
	var rlReport utils.MemReport
	if lerl.rl != nil {
		rlReport = lerl.rl.MemDetailed()
	}

	return utils.MemReport{
		Name:       "CompactLocalExactRangeLocator",
		TotalBytes: lerl.ByteSize(),
		Children: []utils.MemReport{
			{Name: "header", TotalBytes: headerSize},
			{Name: "hzft", TotalBytes: hzftSize},
			rlReport,
		},
	}
}

// TypeWidths returns bit-widths. S and I are hardcoded to 0 since LeMonHash
// resolves types dynamically in C++.
func (lerl *CompactLocalExactRangeLocator[E]) TypeWidths() rloc.TypeWidths {
	var e E
	return rloc.TypeWidths{
		E: int(unsafe.Sizeof(e)) * 8,
		S: 0,
		I: 0,
	}
}
