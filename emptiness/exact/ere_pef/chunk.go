package ere_pef

import (
	"math/bits"

	"Thesis/succinct_bit_vector/rsdic"
)

// selectCodec picks the cheapest codec for a chunk over `universe`
// positions containing `n` keys. Mirrors indexed_sequence::write
// dispatch (PISA).
func selectCodec(universe, n uint64) chunkKind {
	best := allOnesBitsize(universe, n)
	kind := kindAllOnes
	if ef := efBitsizePaper(universe, n) + codecTypeBits; ef < best {
		best = ef
		kind = kindEF
	}
	if bm := bitmapBitsizePaper(universe, n) + codecTypeBits; bm < best {
		kind = kindBitmap
	}
	return kind
}

// writeChunk dispatches the chunk to the right codec, mutating `p`
// (appending rsdics / low bits) and `c` (populating kind, rsIdx,
// efLowOff). `keys` are absolute and sorted; the caller guarantees
// keys[0] == c.base, keys[len-1] == c.last, len(keys) == int(c.n),
// and no duplicates.
func (p *PEF) writeChunk(c *chunk, keys []uint64) {
	universe := c.last - c.base + 1
	n := uint64(c.n)
	c.kind = selectCodec(universe, n)
	switch c.kind {
	case kindAllOnes:
		// no payload — values reconstructable from base..last
	case kindEF:
		p.writeEFChunk(c, keys)
	case kindBitmap:
		p.writeBitmapChunk(c, keys)
	}
}

func (p *PEF) writeEFChunk(c *chunk, keys []uint64) {
	universe := c.last - c.base + 1
	n := uint64(c.n)
	var ell uint64
	if universe > n {
		ell = uint64(bits.Len64(universe/n) - 1)
	}
	var mask uint64
	if ell > 0 {
		mask = (uint64(1) << ell) - 1
	}
	c.efLowOff = p.lowBitsN

	rs := rsdic.New()
	keyIdx := 0
	numBuckets := ((universe - 1) >> ell) + 1
	for b := uint64(0); b < numBuckets; b++ {
		rs.PushBack(true)
		for keyIdx < len(keys) {
			kr := keys[keyIdx] - c.base
			if kr>>ell != b {
				break
			}
			if ell > 0 {
				p.lowBits = ensureBitCapacity(p.lowBits, p.lowBitsN, uint8(ell))
				writeBits(p.lowBits, p.lowBitsN, uint8(ell), kr&mask)
				p.lowBitsN += ell
			}
			rs.PushBack(false)
			keyIdx++
		}
	}
	rs.PushBack(true) // sentinel

	c.rsIdx = uint32(len(p.rsdics))
	p.rsdics = append(p.rsdics, *rs)
}

func (p *PEF) writeBitmapChunk(c *chunk, keys []uint64) {
	universe := c.last - c.base + 1
	rs := rsdic.New()
	keyIdx := 0
	for u := uint64(0); u < universe; u++ {
		if keyIdx < len(keys) && keys[keyIdx]-c.base == u {
			rs.PushBack(true)
			keyIdx++
		} else {
			rs.PushBack(false)
		}
	}
	c.rsIdx = uint32(len(p.rsdics))
	p.rsdics = append(p.rsdics, *rs)
}

// chunkIntersects returns true iff chunk c contains any key in [aAbs, bAbs].
// Caller guarantees c.base <= aAbs <= bAbs <= c.last.
func (p *PEF) chunkIntersects(c *chunk, aAbs, bAbs uint64) bool {
	switch c.kind {
	case kindAllOnes:
		return true
	case kindEF:
		return p.efIntersects(c, aAbs, bAbs)
	case kindBitmap:
		return p.bitmapIntersects(c, aAbs, bAbs)
	}
	return false
}

func (p *PEF) efIntersects(c *chunk, aAbs, bAbs uint64) bool {
	aRel := aAbs - c.base
	bRel := bAbs - c.base
	universe := c.last - c.base + 1
	n := uint64(c.n)
	var ell uint64
	if universe > n {
		ell = uint64(bits.Len64(universe/n) - 1)
	}
	var mask uint64
	if ell > 0 {
		mask = (uint64(1) << ell) - 1
	}
	rs := &p.rsdics[c.rsIdx]

	highA := aRel >> ell
	highB := bRel >> ell
	var lowA, lowB uint64
	if ell > 0 {
		lowA = aRel & mask
		lowB = bRel & mask
	}

	if highA == highB {
		start, end := efBucketRange(rs, highA)
		return p.efBucketHasLow(c.efLowOff, ell, start, end, lowA, lowB)
	}

	startA, endA := efBucketRange(rs, highA)
	if p.efBucketHasLow(c.efLowOff, ell, startA, endA, lowA, mask) {
		return true
	}
	startB, endB := efBucketRange(rs, highB)
	if startB > endA {
		return true
	}
	if p.efBucketHasLow(c.efLowOff, ell, startB, endB, 0, lowB) {
		return true
	}
	return false
}

func efBucketRange(rs *rsdic.RSDic, b uint64) (start, end uint64) {
	posStart := rs.Select1(b)
	posEnd := rs.Select1(b + 1)
	return posStart - b, posEnd - (b + 1)
}

// efBucketHasLow returns true iff any low value at suffix-array index
// in [start, end) lies in [lowMin, lowMax]. For ell == 0 every stored
// low is 0, so the predicate reduces to (start < end) && lowMin == 0.
func (p *PEF) efBucketHasLow(lowOff, ell, start, end, lowMin, lowMax uint64) bool {
	if start >= end {
		return false
	}
	if ell == 0 {
		return lowMin == 0
	}
	width := uint8(ell)
	bitPos := lowOff + start*ell
	for i := start; i < end; i++ {
		low := readBits(p.lowBits, bitPos, width)
		if low > lowMax {
			return false
		}
		if low >= lowMin {
			return true
		}
		bitPos += ell
	}
	return false
}

func (p *PEF) bitmapIntersects(c *chunk, aAbs, bAbs uint64) bool {
	aRel := aAbs - c.base
	bRel := bAbs - c.base
	rs := &p.rsdics[c.rsIdx]
	return rs.Rank(bRel+1, true) > rs.Rank(aRel, true)
}
