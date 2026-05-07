package exact

import (
	"Thesis/emptiness/exact/ere"
	"Thesis/emptiness/exact/ere_one_d"
	"Thesis/emptiness/exact/ere_pef"
	"Thesis/utils"
	"fmt"
	"strings"
	"sync/atomic"
)

type Filter interface {
	IsEmpty(a, b uint64) bool
	SizeInBits() uint64
	ByteSize() int
	MemDetailed() utils.MemReport
}

type Stats struct {
	N               int
	NumBlocks       int
	NonEmptyBlocks  int
	EmptyBlocks     int
	AvgKeysPerBlock float64
	MaxKeysInBlock  int
	EmptyBlockPct   float64
	SumSquaredKeys  uint64
}

type StatsProvider interface {
	GetStats() Stats
}

type BucketSizesProvider interface {
	NonEmptyBlockSizes() []int
}

func NonEmptyBlockSizesOf(f Filter) []int {
	if bp, ok := f.(BucketSizesProvider); ok {
		return bp.NonEmptyBlockSizes()
	}
	return nil
}

type Variant uint32

const (
	VariantClassic Variant = iota
	VariantOneD
	VariantPEF
	VariantAuto
)

// AutoPEFThreshold is the inclusive upper bound on len(keys) at which
// VariantAuto picks PEF over OneD. Above this size, OneD is preferred
// because its query path is 2-3× faster while bpk is on par.
const AutoPEFThreshold = 1 << 24

// AutoPEFMaxKeyBits is the inclusive upper bound on keyBits at which
// VariantAuto considers PEF safe to dispatch. Above this width, the
// PEF chunk codec degrades on extremely sparse universes (the upstream
// parity tests cover up to 60-bit keys), so we fall back to OneD.
const AutoPEFMaxKeyBits = 60

func (v Variant) String() string {
	switch v {
	case VariantClassic:
		return "classic"
	case VariantOneD:
		return "one_d"
	case VariantPEF:
		return "pef"
	case VariantAuto:
		return "auto"
	default:
		return fmt.Sprintf("unknown(%d)", uint32(v))
	}
}

func ParseVariant(s string) (Variant, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "classic", "ere":
		return VariantClassic, nil
	case "one_d", "oned", "ere_one_d":
		return VariantOneD, nil
	case "pef", "ere_pef":
		return VariantPEF, nil
	case "", "auto":
		return VariantAuto, nil
	default:
		return VariantAuto, fmt.Errorf("unknown exact backend variant %q", s)
	}
}

var defaultVariant atomic.Uint32

func init() {
	defaultVariant.Store(uint32(VariantAuto))
}

func SetVariant(v Variant) error {
	switch v {
	case VariantClassic, VariantOneD, VariantPEF, VariantAuto:
		defaultVariant.Store(uint32(v))
		return nil
	default:
		return fmt.Errorf("unsupported exact backend variant %v", v)
	}
}

func SetVariantByName(name string) error {
	v, err := ParseVariant(name)
	if err != nil {
		return err
	}
	return SetVariant(v)
}

func CurrentVariant() Variant {
	return Variant(defaultVariant.Load())
}

func NewUint64(keys []uint64, keyBits uint32) (Filter, error) {
	return NewUint64WithVariant(keys, keyBits, CurrentVariant())
}

// resolveAutoVariant picks a concrete backend for VariantAuto based on
// |keys| and keyBits: PEF for small/medium inputs in narrow universes
// (better bpk on clustered data), OneD above AutoPEFThreshold (faster
// queries at parity bpk) or for keyBits > AutoPEFMaxKeyBits (PEF's
// chunk codec is exercised only up to 60-bit keys upstream).
func resolveAutoVariant(n int, keyBits uint32) Variant {
	if n <= AutoPEFThreshold && keyBits <= AutoPEFMaxKeyBits {
		return VariantPEF
	}
	return VariantOneD
}

func NewUint64WithVariant(keys []uint64, keyBits uint32, variant Variant) (Filter, error) {
	if variant == VariantAuto {
		variant = resolveAutoVariant(len(keys), keyBits)
	}
	switch variant {
	case VariantClassic:
		f, err := ere.NewExactRangeEmptiness(keys, keyBits)
		if err != nil {
			return nil, err
		}
		return classicFilter{f}, nil
	case VariantOneD:
		f, err := ere_one_d.NewExactRangeEmptiness(keys, keyBits)
		if err != nil {
			return nil, err
		}
		return oneDFilter{f}, nil
	case VariantPEF:
		f, err := ere_pef.NewPEF(keys, keyBits)
		if err != nil {
			return nil, err
		}
		return pefFilter{f}, nil
	default:
		return nil, fmt.Errorf("unsupported exact backend variant %v", variant)
	}
}

func StatsOf(f Filter) Stats {
	if sp, ok := f.(StatsProvider); ok {
		return sp.GetStats()
	}
	return Stats{}
}

type classicFilter struct {
	*ere.ExactRangeEmptiness
}

func (f classicFilter) GetStats() Stats {
	s := f.ExactRangeEmptiness.GetStats()
	return Stats{
		N:               s.N,
		NumBlocks:       s.NumBlocks,
		NonEmptyBlocks:  s.NonEmptyBlocks,
		EmptyBlocks:     s.EmptyBlocks,
		AvgKeysPerBlock: s.AvgKeysPerBlock,
		MaxKeysInBlock:  s.MaxKeysInBlock,
		EmptyBlockPct:   s.EmptyBlockPct,
		SumSquaredKeys:  s.SumSquaredKeys,
	}
}

type oneDFilter struct {
	*ere_one_d.ExactRangeEmptiness
}

// Unwrap exposes the embedded *ere_one_d.ExactRangeEmptiness to callers
// that need direct access (e.g. bench diagnostics that read the inner
// rsdic). Returns nil on a zero-value receiver.
func (f oneDFilter) Unwrap() *ere_one_d.ExactRangeEmptiness {
	return f.ExactRangeEmptiness
}

// UnwrapOneD returns the underlying ere_one_d filter when f is the
// OneD variant; otherwise returns nil. Backwards-compat hook for older
// callers that depend on direct access to the OneD internals.
func UnwrapOneD(f Filter) *ere_one_d.ExactRangeEmptiness {
	if w, ok := f.(interface {
		Unwrap() *ere_one_d.ExactRangeEmptiness
	}); ok {
		return w.Unwrap()
	}
	return nil
}

func (f oneDFilter) GetStats() Stats {
	s := f.ExactRangeEmptiness.GetStats()
	return Stats{
		N:               s.N,
		NumBlocks:       s.NumBlocks,
		NonEmptyBlocks:  s.NonEmptyBlocks,
		EmptyBlocks:     s.EmptyBlocks,
		AvgKeysPerBlock: s.AvgKeysPerBlock,
		MaxKeysInBlock:  s.MaxKeysInBlock,
		EmptyBlockPct:   s.EmptyBlockPct,
		SumSquaredKeys:  s.SumSquaredKeys,
	}
}

// pefFilter adapts *ere_pef.PEF to the Filter interface. PEF lacks a
// native MemDetailed/GetStats — we synthesize coarse equivalents:
// GetStats reports the chunk count as the "non-empty block" proxy
// (PEF chunks are by construction non-empty after partitioning), and
// MemDetailed returns a flat report rooted at ByteSize.
//
// The Filter contract takes SizeInBits to mean "all-in encoded size";
// PEF's bare SizeInBits omits chunk-descriptor metadata and reports 0
// for all-ones chunks. We therefore override it to add the metadata
// allocation, mirroring how OneD includes its rsdic backing.
type pefFilter struct {
	*ere_pef.PEF
}

func (f pefFilter) SizeInBits() uint64 {
	if f.PEF == nil {
		return 0
	}
	// PEF.SizeInBits reports payload-only bits, which collapses to 0
	// for all-ones chunks. The Filter contract treats SizeInBits as the
	// total encoded size, so we fall back to ByteSize*8 (struct +
	// chunks + metadata + payload) when the payload alone is empty.
	payload := f.PEF.SizeInBits()
	allocated := uint64(f.PEF.ByteSize()) * 8
	if payload < allocated {
		return allocated
	}
	return payload
}

func (f pefFilter) MemDetailed() utils.MemReport {
	if f.PEF == nil {
		return utils.MemReport{Name: "PEF", TotalBytes: 0}
	}
	return utils.MemReport{Name: "PEF", TotalBytes: f.PEF.ByteSize()}
}

func (f pefFilter) GetStats() Stats {
	if f.PEF == nil {
		return Stats{}
	}
	n := f.PEF.NumChunks()
	// PEF chunks are non-empty by construction; N and per-bucket
	// distribution are not exposed by the underlying type, so we
	// surface only the aggregate.
	return Stats{
		NumBlocks:      n,
		NonEmptyBlocks: n,
	}
}
