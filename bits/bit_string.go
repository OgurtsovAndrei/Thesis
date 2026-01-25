package bits

import (
	"Thesis/errutil"
)

type BitString interface {
	Size() uint32
	IsEmpty() bool
	At(index uint32) bool
	Equal(a BitString) bool
	Eq(other BitString) bool
	PrettyString() string
	GetLCPLength(other BitString) uint32
	HasPrefix(prefixToCheck BitString) bool
	Prefix(size int) BitString
	Data() []byte
	Hash() uint64
	HashWithSeed(seed uint64) uint64
	Compare(other BitString) int

	TrimTrailingZeros() BitString
	AppendBit(bit bool) BitString
	IsAllOnes() bool
	Successor() BitString
}

const benchmarkParallelism = 4

type BitStringImpl int

const SelectedImpl = CharString

const (
	CharString BitStringImpl = iota
	Uint64String
	Uint64ArrayString
)

func NewFromText(text string) BitString {
	switch SelectedImpl {
	case CharString:
		return NewCharFromText(text)
	case Uint64String:
		return NewUint64FromText(text)
	case Uint64ArrayString:
		// Convert text to binary representation for array implementation
		size := uint32(len(text)) * 8
		data := []byte(text)
		return NewUint64ArrFromDataAndSize(data, size)
	default:
		errutil.Bug("Unexpected Impl selected")
	}
	return nil
}

func NewFromUint64(value uint64) BitString {
	switch SelectedImpl {
	case CharString:
		return NewCharFromUint64(value)
	case Uint64String:
		return NewUint64FromUint64(value, 64)
	case Uint64ArrayString:
		bs := NewUint64ArrFromUint64(64)
		bs.data[0] = value
		return bs
	default:
		errutil.Bug("Unexpected Impl selected")
	}
	return nil
}

func NewFromBinary(text string) BitString {
	switch SelectedImpl {
	case CharString:
		return NewCharFromBinary(text)
	case Uint64String:
		return NewUint64FromBinaryText(text)
	case Uint64ArrayString:
		return NewUint64ArrayFromBinaryText(text)
	default:
		errutil.Bug("Unexpected Impl selected")
	}
	return nil
}

func NewBitStringPrefix(bs BitString, size uint32) BitString {
	switch SelectedImpl {
	case CharString:
		return NewCharBitStringPrefix(bs, size)
	case Uint64String:
		return NewUint64BitStringPrefix(bs, size)
	case Uint64ArrayString:
		return NewUint64ArrayBitStringPrefix(bs, size)
	default:
		errutil.Bug("Unexpected Impl selected")
	}
	return nil
}

func NewBitStringFormDataAndSize(data []byte, size uint32) BitString {
	switch SelectedImpl {
	case CharString:
		return NewCharBitStringFromDataAndSize(data, size)
	case Uint64String:
		return NewUint64BitStringFromDataAndSize(data, size)
	case Uint64ArrayString:
		return NewUint64ArrFromDataAndSize(data, size)
	default:
		errutil.Bug("Unexpected Impl selected")
	}
	return nil
}
