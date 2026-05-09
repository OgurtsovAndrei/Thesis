package ere_pef

import (
	"math/bits"
	"sort"
)

// CollectedStats holds per-path counters from a CollectQueryStats run.
type CollectedStats struct {
	Total              int
	EarlyExit          int
	MultiChunk         int
	AllOnes            int
	EF                 int
	Bitmap             int
	EFBucketCalls      int
	EFTotalBucketElems int
	EFLinearScan       int
}

// CollectQueryStats runs each query and counts which code paths were taken.
// Intended for profiling and research; not hot-path optimised.
func (p *PEF) CollectQueryStats(queries [][2]uint64) CollectedStats {
	var s CollectedStats
	for _, q := range queries {
		s.Total++
		p.isEmptyCollect(q[0], q[1], &s)
	}
	return s
}

func (p *PEF) isEmptyCollect(a, b uint64, s *CollectedStats) {
	if p.n == 0 || a > b || b < p.firstKey || a > p.lastKey {
		s.EarlyExit++
		return
	}
	if a < p.firstKey {
		a = p.firstKey
	}
	if b > p.lastKey {
		b = p.lastKey
	}

	chunks := p.chunks
	i := sort.Search(len(chunks), func(k int) bool { return chunks[k].last >= a })
	j := sort.Search(len(chunks), func(k int) bool { return chunks[k].last >= b })
	if i != j {
		s.MultiChunk++
		return
	}

	c := &chunks[i]
	switch c.kind() {
	case kindAllOnes:
		s.AllOnes++
	case kindEF:
		s.EF++
		p.collectEFBucketStats(s, c, p.chunkBaseAt(i), a, b)
	case kindBitmap:
		s.Bitmap++
	}
}

func (p *PEF) collectEFBucketStats(s *CollectedStats, c *chunk, base, aAbs, bAbs uint64) {
	aRel := aAbs - base
	bRel := bAbs - base
	lastRel := c.last - base
	n := uint64(c.n())
	var ell uint64
	if lastRel >= n {
		ell = uint64(bits.Len64(lastRel/n) - 1)
	}
	meta := &p.efMeta[c.metaIdx]

	highA := aRel >> ell
	highB := bRel >> ell

	countBucket := func(start, end uint64) {
		size := int(end - start)
		s.EFBucketCalls++
		s.EFTotalBucketElems += size
		if end-start <= linearScanThreshold {
			s.EFLinearScan++
		}
	}

	startA, endA := p.efBucketRange(meta, highA)
	countBucket(startA, endA)

	if highA == highB {
		return
	}

	startB, endB := p.efBucketRange(meta, highB)
	// efIntersects only calls efBucketHasLow on bucket B when startB <= endA.
	if startB <= endA {
		countBucket(startB, endB)
	}
}
