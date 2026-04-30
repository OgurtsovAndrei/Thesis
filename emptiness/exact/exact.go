package exact

import (
	"Thesis/emptiness/exact/ere"
	"Thesis/emptiness/exact/ere_one_d"
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
)

func (v Variant) String() string {
	switch v {
	case VariantClassic:
		return "classic"
	case VariantOneD:
		return "one_d"
	default:
		return fmt.Sprintf("unknown(%d)", uint32(v))
	}
}

func ParseVariant(s string) (Variant, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "classic", "ere":
		return VariantClassic, nil
	case "one_d", "oned", "ere_one_d":
		return VariantOneD, nil
	default:
		return VariantClassic, fmt.Errorf("unknown exact backend variant %q", s)
	}
}

var defaultVariant atomic.Uint32

func init() {
	defaultVariant.Store(uint32(VariantOneD))
}

func SetVariant(v Variant) error {
	switch v {
	case VariantClassic, VariantOneD:
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

func NewUint64WithVariant(keys []uint64, keyBits uint32, variant Variant) (Filter, error) {
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
