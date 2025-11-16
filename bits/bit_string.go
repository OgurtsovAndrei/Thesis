package bits

import (
	"Thesis/errutil"
)

type BitString interface {
	Size() uint32
	IsEmpty() bool
	At(index uint32) bool
	Equal(a BitString) bool
	String() string
	GetLCPLength(other BitString) uint32
	HasPrefix(prefixToCheck BitString) bool
	Prefix(size int) BitString
	Data() []byte
}

type BitStringImpl int

const SelectedImpl = CharString

const (
	CharString BitStringImpl = iota
	Uint64String
)

func NewBitString(text string) BitString {
	switch SelectedImpl {
	case CharString:
		return NewCharBitString(text)
	case Uint64String:
		return NewUint64StringFromText(text)
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
	default:
		errutil.Bug("Unexpected Impl selected")
	}
	return nil
}
