package ere_pef

import (
	"fmt"
	"sort"
	"unsafe"

	"Thesis/succinct_bit_vector/rsdic"
)

// PEF is a Partitioned Elias-Fano range emptiness structure.
// Built once via NewPEF; queries are read-only thereafter.
type PEF struct {
	n         int
	keyBits   uint32
	firstKey  uint64
	lastKey   uint64
	chunks    []chunk
	rsdics    []rsdic.RSDic
	lowBits   []uint64
	lowBitsN  uint64
}

// chunkKind selects the codec used to encode a chunk's keys.
type chunkKind uint8

const (
	kindAllOnes chunkKind = iota
	kindEF
	kindBitmap
)

// chunk describes one partition piece. Tagged-union layout: depending
// on kind, only a subset of fields is meaningful. Sized to keep the
// per-chunk overhead small (≤ 40 bytes).
type chunk struct {
	base, last uint64 // absolute key range covered: keys ∈ [base, last]
	efLowOff   uint64 // bit offset into PEF.lowBits (kindEF only)
	n          uint32 // number of keys in this chunk (≥ 1)
	rsIdx      uint32 // index into PEF.rsdics (kindEF / kindBitmap only)
	kind       chunkKind
	_          [3]byte // pad to 8-byte alignment of next field (struct end)
}

// NewPEF builds a Partitioned Elias-Fano range emptiness structure
// from the given sorted key slice. Duplicate keys are silently
// deduplicated (set semantics is preserved for IsEmpty). keyBits is
// the effective key width used by callers; queries with values
// outside [0, 2^keyBits) are still defined (IsEmpty returns true for
// queries that do not overlap any stored key).
func NewPEF(keys []uint64, keyBits uint32) (*PEF, error) {
	if keyBits > 64 {
		return nil, fmt.Errorf("keyBits must be <= 64, got %d", keyBits)
	}
	p := &PEF{keyBits: keyBits}
	if len(keys) == 0 {
		return p, nil
	}

	deduped := make([]uint64, 0, len(keys))
	for i, k := range keys {
		if i > 0 && k < keys[i-1] {
			return nil, fmt.Errorf("keys must be sorted (idx=%d)", i)
		}
		if i > 0 && k == keys[i-1] {
			continue
		}
		deduped = append(deduped, k)
	}

	n := len(deduped)
	p.n = n
	p.firstKey = deduped[0]
	p.lastKey = deduped[n-1]

	if n == 1 {
		c := chunk{base: deduped[0], last: deduped[0], n: 1}
		p.writeChunk(&c, deduped)
		p.chunks = append(p.chunks, c)
		return p, nil
	}

	costFn := func(u, n uint64) uint64 {
		return minCodecBitsize(u, n) + defaultFixCost
	}
	superSize := superblockSize(defaultFixCost, defaultEps3)
	scratch := &partitionScratch{}
	var partitionBuf []uint32

	inputUniverse := deduped[n-1] + 1
	chunkBase := deduped[0]
	superPos := 0
	superBase := deduped[0]

	for superPos < n {
		sz := superSize
		if sz > n-superPos {
			sz = n - superPos
		}
		// Merge a tiny tail into current superblock so no superblock is
		// shorter than `superSize` (mirrors PISA compute_partition).
		if rem := n - (superPos + sz); rem > 0 && rem < superSize {
			sz = n - superPos
		}
		superKeys := deduped[superPos : superPos+sz]
		var superUniverse uint64
		if superPos+sz == n {
			superUniverse = inputUniverse
		} else {
			superUniverse = deduped[superPos+sz-1] + 1
		}

		partition, _ := scratch.compute(
			superKeys, superBase, superUniverse,
			costFn, defaultEps1, defaultEps2, partitionBuf,
		)

		prevEnd := uint32(0)
		for _, end := range partition {
			ks := superKeys[prevEnd:end]
			c := chunk{
				base: chunkBase,
				last: ks[len(ks)-1],
				n:    uint32(len(ks)),
			}
			p.writeChunk(&c, ks)
			p.chunks = append(p.chunks, c)
			chunkBase = c.last + 1
			prevEnd = end
		}
		partitionBuf = partition[:0]

		superPos += sz
		superBase = superUniverse
	}

	return p, nil
}

// IsEmpty reports whether the closed range [a, b] contains no stored key.
func (p *PEF) IsEmpty(a, b uint64) bool {
	if p.n == 0 || a > b {
		return true
	}
	if b < p.firstKey || a > p.lastKey {
		return true
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
		return false
	}
	return !p.chunkIntersects(&chunks[i], a, b)
}

// ByteSize returns total in-memory footprint in bytes (struct +
// chunks + rsdic backing + low bits buffer).
func (p *PEF) ByteSize() int {
	if p == nil || p.n == 0 {
		return 0
	}
	size := int(unsafe.Sizeof(*p))
	size += len(p.chunks) * int(unsafe.Sizeof(chunk{}))
	for i := range p.rsdics {
		size += p.rsdics[i].AllocSize()
	}
	size += len(p.lowBits) * 8
	return size
}

// SizeInBits returns the logical (information-theoretic) bit count of
// the encoded payload: every rsdic's Num() plus the EF low-bits stream.
func (p *PEF) SizeInBits() uint64 {
	if p == nil || p.n == 0 {
		return 0
	}
	var b uint64
	for i := range p.rsdics {
		b += p.rsdics[i].Num()
	}
	b += p.lowBitsN
	return b
}

// NumChunks returns the number of chunks the input was partitioned
// into. Useful as a sanity-check / characterization metric in
// benchmarks (more chunks ⇒ DP found a less-uniform-density input).
func (p *PEF) NumChunks() int {
	if p == nil {
		return 0
	}
	return len(p.chunks)
}

// MetadataAllocBits returns the rank/select dictionary backing
// allocation in bits — the structural metadata that supports O(1) rank
// and select. EF low bits are excluded (those are payload).
func (p *PEF) MetadataAllocBits() uint64 {
	if p == nil || p.n == 0 {
		return 0
	}
	var b uint64
	for i := range p.rsdics {
		b += uint64(p.rsdics[i].AllocSize()) * 8
	}
	return b
}
