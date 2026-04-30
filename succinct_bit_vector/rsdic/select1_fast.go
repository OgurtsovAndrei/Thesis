package rsdic

// Select1Fast is an experimental Select1 implementation that addresses the
// linear-scan inner loop in the original Select1: where the original walks
// rankBlocks forward from selectOneInds[selectInd] until cumulative ones
// exceed rank, this version binary-searches rankBlocks within the bracket
// [selectOneInds[selectInd], selectOneInds[selectInd+1]] (or end of array).
//
// Motivation: on heavily-clustered bitvectors (e.g. the unary-encoded ERE
// block boundaries when SODA degenerates) 4096 ones can span millions of
// bits, making the original O(1)-amortised loop iterate thousands of
// times. The bracketed binary search caps the inner-loop work at O(log
// of the bracket size), typically ~12-14 iterations.
//
// The receiver is a pointer to avoid the ~104-byte struct copy that the
// original value-receiver Select1 incurs (visible in pprof as
// runtime.duffcopy).
//
// All other state and helper functions (selectRaw, getSlice, enumSelect1,
// kSelectBlockSize, kSmallBlockPerLargeBlock, kSmallBlockSize,
// kEnumCodeLength) are reused unchanged.
func (rs *RSDic) Select1Fast(rank uint64) uint64 {
	if rank >= rs.oneNum {
		return rs.num
	} else if rank >= rs.oneNum-rs.lastOneNum {
		lastBlockRank := uint8(rank - (rs.oneNum - rs.lastOneNum))
		return rs.lastBlockInd() + uint64(selectRaw(rs.lastBlock, lastBlockRank+1))
	}

	selectInd := rank / kSelectBlockSize
	lo := rs.selectOneInds[selectInd]
	var hi uint64
	if selectInd+1 < uint64(len(rs.selectOneInds)) {
		// +1 because the sampled block might still hold the next-hint's
		// preceding ones; widen by one to be safe.
		hi = rs.selectOneInds[selectInd+1] + 1
	} else {
		hi = uint64(len(rs.rankBlocks))
	}
	if hi > uint64(len(rs.rankBlocks)) {
		hi = uint64(len(rs.rankBlocks))
	}

	// Find the smallest lblock in [lo, hi) such that rank < rankBlocks[lblock];
	// then lblock-- gives the large block that contains the (rank+1)-th one.
	for lo < hi {
		mid := lo + (hi-lo)/2
		if rs.rankBlocks[mid] <= rank {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		// rank < rankBlocks[0]; the (rank+1)-th one is in large block 0.
		// This case can't legally occur when rankBlocks[0]==0 (which it
		// always is by construction), but guard anyway.
		lo = 1
	}
	lblock := lo - 1

	sblock := lblock * kSmallBlockPerLargeBlock
	pointer := rs.pointerBlocks[lblock]
	remain := rank - rs.rankBlocks[lblock] + 1
	for ; sblock < uint64(len(rs.rankSmallBlocks)); sblock++ {
		rankSB := rs.rankSmallBlocks[sblock]
		if remain <= uint64(rankSB) {
			break
		}
		remain -= uint64(rankSB)
		pointer += uint64(kEnumCodeLength[rankSB])
	}
	rankSB := rs.rankSmallBlocks[sblock]
	code := getSlice(rs.bits, pointer, kEnumCodeLength[rankSB])
	return sblock*kSmallBlockSize + uint64(enumSelect1(code, rankSB, uint8(remain)))
}
