package ere_pef

import (
	"math/bits"
)

// selectCodec picks the cheapest codec for a chunk whose inclusive
// relative range is [0, lastRel] and contains n keys.
// Mirrors indexed_sequence::write dispatch (PISA).
func selectCodec(lastRel, n uint64) chunkKind {
	best := allOnesBitsize(lastRel, n)
	kind := kindAllOnes
	if ef := efBitsizePaper(lastRel, n) + codecTypeBits; ef < best {
		best = ef
		kind = kindEF
	}
	bm := bitmapBitsizePaper(lastRel, n)
	// Guard against overflow when bm == MaxUint64 (infinity sentinel).
	if bm < ^uint64(0) && bm+codecTypeBits < best {
		kind = kindBitmap
	}
	return kind
}

// writeChunk dispatches the chunk to the right codec, mutating `p`
// (appending to shared rsdic / low bits / per-kind metadata) and `c`
// (populating kind, metaIdx).
//
// `keys` are absolute and sorted; the caller guarantees keys[0] == base,
// keys[len-1] == c.last, len(keys) == int(c.n()), and no duplicates.
func (p *PEF) writeChunk(c *chunk, base uint64, keys []uint64) {
	lastRel := c.last - base
	n := uint64(c.n())
	k := selectCodec(lastRel, n)
	c.nKind = packNKind(c.n(), k)
	switch k {
	case kindAllOnes:
		// no payload — values reconstructable from base..last
	case kindEF:
		p.writeEFChunk(c, base, keys)
	case kindBitmap:
		p.writeBitmapChunk(c, base, keys)
	}
}

func (p *PEF) writeEFChunk(c *chunk, base uint64, keys []uint64) {
	lastRel := c.last - base
	n := uint64(c.n())
	var ell uint64
	if lastRel >= n {
		ell = uint64(bits.Len64(lastRel/n) - 1)
	}
	var mask uint64
	if ell > 0 {
		mask = (uint64(1) << ell) - 1
	}

	meta := efChunkMeta{
		lowOff:     p.lowBitsN,
		globalOff:  p.rs.Num(),
		onesBefore: p.rs.OneNum(),
	}

	keyIdx := 0
	numBuckets := (lastRel >> ell) + 1
	for b := uint64(0); b < numBuckets; b++ {
		p.rs.PushBack(true)
		for keyIdx < len(keys) {
			kr := keys[keyIdx] - base
			if kr>>ell != b {
				break
			}
			if ell > 0 {
				p.lowBits = ensureBitCapacity(p.lowBits, p.lowBitsN, uint8(ell))
				writeBits(p.lowBits, p.lowBitsN, uint8(ell), kr&mask)
				p.lowBitsN += ell
			}
			p.rs.PushBack(false)
			keyIdx++
		}
	}
	p.rs.PushBack(true) // sentinel

	c.metaIdx = uint32(len(p.efMeta))
	p.efMeta = append(p.efMeta, meta)
}

func (p *PEF) writeBitmapChunk(c *chunk, base uint64, keys []uint64) {
	lastRel := c.last - base
	meta := bmChunkMeta{globalOff: p.rs.Num()}

	keyIdx := 0
	// Use a for-range loop over [0, lastRel] with an explicit break to
	// avoid infinite iteration when lastRel == ^uint64(0).
	for u := uint64(0); ; u++ {
		if keyIdx < len(keys) && keys[keyIdx]-base == u {
			p.rs.PushBack(true)
			keyIdx++
		} else {
			p.rs.PushBack(false)
		}
		if u == lastRel {
			break
		}
	}

	c.metaIdx = uint32(len(p.bmMeta))
	p.bmMeta = append(p.bmMeta, meta)
}

// chunkIntersects returns true iff chunks[i] contains any key in [aAbs, bAbs].
// Caller guarantees chunkBaseAt(i) <= aAbs <= bAbs <= chunks[i].last.
func (p *PEF) chunkIntersects(i int, aAbs, bAbs uint64) bool {
	c := &p.chunks[i]
	switch c.kind() {
	case kindAllOnes:
		return true
	case kindEF:
		return p.efIntersects(c, p.chunkBaseAt(i), aAbs, bAbs)
	case kindBitmap:
		return p.bitmapIntersects(c, p.chunkBaseAt(i), aAbs, bAbs)
	}
	return false
}

func (p *PEF) efIntersects(c *chunk, base, aAbs, bAbs uint64) bool {
	aRel := aAbs - base
	bRel := bAbs - base
	lastRel := c.last - base
	n := uint64(c.n())
	var ell uint64
	if lastRel >= n {
		ell = uint64(bits.Len64(lastRel/n) - 1)
	}
	var mask uint64
	if ell > 0 {
		mask = (uint64(1) << ell) - 1
	}
	meta := &p.efMeta[c.metaIdx]

	highA := aRel >> ell
	highB := bRel >> ell
	var lowA, lowB uint64
	if ell > 0 {
		lowA = aRel & mask
		lowB = bRel & mask
	}

	if highA == highB {
		start, end := p.efBucketRange(meta, highA)
		return p.efBucketHasLow(meta.lowOff, ell, start, end, lowA, lowB)
	}

	startA, endA := p.efBucketRange(meta, highA)
	if p.efBucketHasLow(meta.lowOff, ell, startA, endA, lowA, mask) {
		return true
	}
	startB, endB := p.efBucketRange(meta, highB)
	if startB > endA {
		return true
	}
	if p.efBucketHasLow(meta.lowOff, ell, startB, endB, 0, lowB) {
		return true
	}
	return false
}

// efBucketRange returns the suffix-array range [start, end) of low
// values stored under bucket index `b` for the EF chunk described by
// `meta`. Local Select1(k) is computed against the shared rsdic as
// `rs.Select1(meta.onesBefore+k) - meta.globalOff`.
func (p *PEF) efBucketRange(meta *efChunkMeta, b uint64) (start, end uint64) {
	posStart := p.rs.Select1(meta.onesBefore+b) - meta.globalOff
	posEnd := p.rs.Select1(meta.onesBefore+b+1) - meta.globalOff
	return posStart - b, posEnd - (b + 1)
}

// linearScanThreshold mirrors ere_one_d: for buckets up to this size,
// sequential prefetch beats branchy binary search. Value taken from
// BenchmarkBucketSearch_LinearVsBinary in ere_one_d (~128).
const linearScanThreshold = 128

// efBucketHasLow returns true iff any low value at suffix-array index
// in [start, end) lies in [lowMin, lowMax]. For ell == 0 every stored
// low is 0, so the predicate reduces to (start < end) && lowMin == 0.
// For larger buckets (size > linearScanThreshold) switches to binary
// search on lows for predecessor of lowMin, then a single bound check.
func (p *PEF) efBucketHasLow(lowOff, ell, start, end, lowMin, lowMax uint64) bool {
	if start >= end {
		return false
	}
	if ell == 0 {
		return lowMin == 0
	}
	width := uint8(ell)
	if end-start <= linearScanThreshold {
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
	// Binary search for the first idx with low >= lowMin; check it ≤ lowMax.
	l, r := start, end
	for l < r {
		mid := l + (r-l)/2
		low := readBits(p.lowBits, lowOff+mid*ell, width)
		if low < lowMin {
			l = mid + 1
		} else {
			r = mid
		}
	}
	if l >= end {
		return false
	}
	low := readBits(p.lowBits, lowOff+l*ell, width)
	return low <= lowMax
}

func (p *PEF) bitmapIntersects(c *chunk, base, aAbs, bAbs uint64) bool {
	aRel := aAbs - base
	bRel := bAbs - base
	off := p.bmMeta[c.metaIdx].globalOff
	return p.rs.Rank(off+bRel+1, true) > p.rs.Rank(off+aRel, true)
}
