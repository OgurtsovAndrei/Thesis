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
	String() string
	GetLCPLength(other BitString) uint32
	HasPrefix(prefixToCheck BitString) bool
	Prefix(size int) BitString
	Data() []byte
	Hash() uint64
	Compare(other BitString) int
}

type BitStringImpl int

const SelectedImpl = Uint64ArrayString

const (
	CharString BitStringImpl = iota
	Uint64String
	Uint64ArrayString
)

func NewBitString(text string) BitString {
	switch SelectedImpl {
	case CharString:
		return NewCharBitString(text)
	case Uint64String:
		return NewUint64StringFromText(text)
	case Uint64ArrayString:
		// Convert text to binary representation for array implementation
		size := uint32(len(text)) * 8
		data := []byte(text)
		return NewUint64ArrayFromDataAndSize(data, size)
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
		return NewUint64BitString(value, 64)
	case Uint64ArrayString:
		bs := NewUint64ArrayBitString(64)
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
		return NewUint64ArrayFromDataAndSize(data, size)
	default:
		errutil.Bug("Unexpected Impl selected")
	}
	return nil
}
