package ere_one_d

// getBlockRangeFast mirrors getBlockRange but calls rsdic.Select1Fast
// (bracketed binary search over rankBlocks) instead of Select. Used by
// IsEmptyFast for end-to-end validation of the rsdic fix.
func (ere *ExactRangeEmptiness) getBlockRangeFast(blockIdx uint64) (int, int) {
	posStart := ere.D.Select1Fast(blockIdx)
	posEnd := ere.D.Select1Fast(blockIdx + 1)
	return int(posStart - blockIdx), int(posEnd - (blockIdx + 1))
}

// getQueryBlockRangesFast mirrors getQueryBlockRanges but uses
// Select1Fast.
func (ere *ExactRangeEmptiness) getQueryBlockRangesFast(blockA, blockB uint64) (int, int, int, int) {
	if blockB == blockA+1 {
		// Same as IsEmpty's optimisation: 3 contiguous ones-positions
		// give us 4 packedData boundaries with one fewer Select call.
		pos0 := ere.D.Select1Fast(blockA)
		pos1 := ere.D.Select1Fast(blockA + 1)
		pos2 := ere.D.Select1Fast(blockA + 2)
		startA := int(pos0 - blockA)
		endA := int(pos1 - (blockA + 1))
		startB := int(pos1 - (blockA + 1))
		endB := int(pos2 - (blockA + 2))
		return startA, endA, startB, endB
	}
	startA, endA := ere.getBlockRangeFast(blockA)
	startB, endB := ere.getBlockRangeFast(blockB)
	return startA, endA, startB, endB
}

// IsEmptyFast is functionally identical to IsEmpty but routes through
// the rsdic.Select1Fast bracket-binary-search path, eliminating the
// linear-scan pathology on clustered bitvectors.
//
// Diagnostic-only — meant for end-to-end validation of the fix's effect
// on SODA query latency. Not yet wired into the production path.
func (ere *ExactRangeEmptiness) IsEmptyFast(a, b uint64) bool {
	if ere.n == 0 {
		return true
	}
	if a > b {
		return true
	}

	blockA := a >> ere.w
	blockB := b >> ere.w

	if blockA >= uint64(ere.numBlocks) {
		return true
	}
	if blockB >= uint64(ere.numBlocks) {
		blockB = uint64(ere.numBlocks - 1)
	}

	if blockA == blockB {
		start, end := ere.getBlockRangeFast(blockA)
		if start < end {
			suffA := a & ere.suffMask
			suffB := b & ere.suffMask
			if !ere.searchBucket(start, end, suffA, suffB) {
				return false
			}
		}
	} else {
		startA, endA, startB, endB := ere.getQueryBlockRangesFast(blockA, blockB)

		if startB > endA {
			return false
		}

		if startA < endA {
			suffA := a & ere.suffMask
			maxSuff := ere.suffMask
			if !ere.searchBucket(startA, endA, suffA, maxSuff) {
				return false
			}
		}
		if startB < endB {
			suffB := b & ere.suffMask
			if !ere.searchBucket(startB, endB, 0, suffB) {
				return false
			}
		}
	}

	return true
}
