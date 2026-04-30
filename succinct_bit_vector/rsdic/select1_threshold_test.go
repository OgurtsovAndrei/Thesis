package rsdic

import (
	"fmt"
	"math/rand"
	"testing"
)

// select1Linear forces the original linear-scan inner loop (no bracket
// upper bound). Used to measure the pure-linear baseline.
func (rs *RSDic) select1Linear(rank uint64) uint64 {
	if rank >= rs.oneNum {
		return rs.num
	} else if rank >= rs.oneNum-rs.lastOneNum {
		lastBlockRank := uint8(rank - (rs.oneNum - rs.lastOneNum))
		return rs.lastBlockInd() + uint64(selectRaw(rs.lastBlock, lastBlockRank+1))
	}
	selectInd := rank / kSelectBlockSize
	lblock := rs.selectOneInds[selectInd]
	for ; lblock < uint64(len(rs.rankBlocks)); lblock++ {
		if rank < rs.rankBlocks[lblock] {
			break
		}
	}
	lblock--
	return rs.select1FinishUpper(lblock, rank)
}

// select1Binary forces a binary search bracketed by selectOneInds.
func (rs *RSDic) select1Binary(rank uint64) uint64 {
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
		hi = rs.selectOneInds[selectInd+1] + 1
	} else {
		hi = uint64(len(rs.rankBlocks))
	}
	if hi > uint64(len(rs.rankBlocks)) {
		hi = uint64(len(rs.rankBlocks))
	}
	l, r := lo, hi
	for l < r {
		mid := l + (r-l)/2
		if rs.rankBlocks[mid] <= rank {
			l = mid + 1
		} else {
			r = mid
		}
	}
	if l == 0 {
		l = 1
	}
	return rs.select1FinishUpper(l-1, rank)
}

// select1FinishUpper is the post-large-block phase: small-block scan +
// enumSelect1. Shared by select1Linear and select1Binary.
func (rs *RSDic) select1FinishUpper(lblock uint64, rank uint64) uint64 {
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

// buildRSDicWithBracket constructs an rsdic where the average bracket
// size (selectOneInds[i+1] - selectOneInds[i]) equals targetBracket
// large blocks. Achieved by spacing 1-bits by targetBracket/4 bits on
// average (since one selectOneInds entry covers 4096 ones, and 1 large
// block = 1024 bits).
func buildRSDicWithBracket(targetBracket int, totalOnes int) *RSDic {
	zerosBetweenOnes := targetBracket/4 - 1
	if zerosBetweenOnes < 0 {
		zerosBetweenOnes = 0
	}
	rs := New()
	for i := 0; i < totalOnes; i++ {
		for j := 0; j < zerosBetweenOnes; j++ {
			rs.PushBack(false)
		}
		rs.PushBack(true)
	}
	return rs
}

// TestSelect1ThresholdEquivalence cross-checks Linear, Binary and the
// production adaptive Select1 over a sweep of bracket sizes.
func TestSelect1ThresholdEquivalence(t *testing.T) {
	for _, bracket := range []int{1, 4, 8, 16, 32, 64, 256, 1024} {
		bracket := bracket
		t.Run(fmt.Sprintf("bracket=%d", bracket), func(t *testing.T) {
			rs := buildRSDicWithBracket(bracket, 16384)
			ones := rs.OneNum()
			rng := rand.New(rand.NewSource(42))
			for trial := 0; trial < 200; trial++ {
				rank := uint64(rng.Int63n(int64(ones)))
				want := rs.Select1(rank)
				gotL := rs.select1Linear(rank)
				gotB := rs.select1Binary(rank)
				if gotL != want || gotB != want {
					t.Fatalf("rank=%d (bracket=%d): adaptive=%d linear=%d binary=%d",
						rank, bracket, want, gotL, gotB)
				}
			}
		})
	}
}

// BenchmarkSelect1ThresholdSweep measures linear vs binary vs adaptive
// Select1 latency as a function of bracket size. The crossover point
// (if any) shows where adaptive should switch.
func BenchmarkSelect1ThresholdSweep(b *testing.B) {
	const totalOnes = 1 << 14 // 16K ones — enough for stable bracket sampling
	brackets := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 1024, 4096}
	for _, bracket := range brackets {
		bracket := bracket
		rs := buildRSDicWithBracket(bracket, totalOnes)
		ones := rs.OneNum()
		rng := rand.New(rand.NewSource(7))
		const kIters = 200_000
		ranks := make([]uint64, kIters)
		for i := range ranks {
			ranks[i] = uint64(rng.Int63n(int64(ones)))
		}

		b.Run(fmt.Sprintf("bracket=%-4d/Linear", bracket), func(b *testing.B) {
			var sink uint64
			for i := 0; i < b.N; i++ {
				sink ^= rs.select1Linear(ranks[i%kIters])
			}
			if sink == 0xDEADBEEF {
				b.Log("sink trick")
			}
		})
		b.Run(fmt.Sprintf("bracket=%-4d/Binary", bracket), func(b *testing.B) {
			var sink uint64
			for i := 0; i < b.N; i++ {
				sink ^= rs.select1Binary(ranks[i%kIters])
			}
			if sink == 0xDEADBEEF {
				b.Log("sink trick")
			}
		})
		b.Run(fmt.Sprintf("bracket=%-4d/Adaptive", bracket), func(b *testing.B) {
			var sink uint64
			for i := 0; i < b.N; i++ {
				sink ^= rs.Select1(ranks[i%kIters])
			}
			if sink == 0xDEADBEEF {
				b.Log("sink trick")
			}
		})
	}
}
