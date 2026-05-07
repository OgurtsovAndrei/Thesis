package ere_pef

import (
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
