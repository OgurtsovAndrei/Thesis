package bits

import (
	"Thesis/errutil"
	"log"
	"math/bits"
	"strconv"
	"strings"
)

const unsafe = false

var _ BitString = Uint64BitString{}

type Uint64BitString struct {
	value uint64
	len   int8
}

func NewUint64StringFromText(text string) Uint64BitString {
	var val uint64
	l := len(text)
	errutil.BugOn(l > 8, "Illegal length: %q", text)

	for i := 0; i < l; i++ {
		val |= uint64(text[i]) << (i * 8)
	}

	return Uint64BitString{
		value: val,
		len:   int8(l * 8),
	}
}

func NewUint64FromBinaryText(text string) Uint64BitString {
	for _, r := range text {
		errutil.BugOn(r != '0' && r != '1', "invalid string format, %q", text)
	}

	size := len(text)
	if size == 0 {
		return Uint64BitString{}
	}

	if size > 64 {
		log.Panicf("length too big: %d", size)
	}

	var val uint64
	for i, r := range text {
		if i >= size {
			break
		}
		if r == '1' {
			val |= (uint64(1) << i)
		}
	}

	return Uint64BitString{
		value: val,
		len:   int8(size),
	}
}

func NewUint64BitStringPrefix(bs BitString, size uint32) BitString {
	errutil.BugOn(size > bs.Size(), "size exceeds original bitstring size")
	errutil.BugOn(size > 64, "size cannot exceed 64 for Uint64BitString")
	otherBs, ok := bs.(Uint64BitString)
	errutil.BugOn(!ok, "NewUint64BitStringPrefix can only be called on Uint64BitString")

	return otherBs.Prefix(int(size))
}

func NewUint64BitString(value uint64, length int8) Uint64BitString {
	if !unsafe {
		if length < 0 || length > 64 {
			panic("length must be between 0 and 64")
		}
	}

	mask := ^uint64(0)
	if length < 64 {
		mask = (uint64(1) << length) - 1
	}

	return Uint64BitString{
		value: value & mask,
		len:   length,
	}
}

func NewUint64BitStringFromDataAndSize(data []byte, size uint32) Uint64BitString {
	if !unsafe {
		if size > 64 {
			panic("size cannot exceed 64 for Uint64BitString")
		}
	}
	if size == 0 {
		return Uint64BitString{}
	}

	numBytes := (size + 7) / 8
	if uint32(len(data)) < numBytes {
		panic("data length is insufficient for the specified size")
	}

	var value uint64
	for i := uint32(0); i < numBytes && i < 8; i++ {
		value |= uint64(data[i]) << (i * 8)
	}

	mask := ^uint64(0)
	if size < 64 {
		mask = (uint64(1) << size) - 1
	}

	return Uint64BitString{
		value: value & mask,
		len:   int8(size),
	}
}

func (bs Uint64BitString) Size() uint32 {
	return uint32(bs.len)
}

func (bs Uint64BitString) IsEmpty() bool {
	return bs.len == 0
}

func (bs Uint64BitString) At(index uint32) bool {
	if !unsafe {
		if index >= uint32(bs.len) {
			log.Panicf("index out of bounds index: %d >= len: %d", index, bs.len)
		}
	}
	return (bs.value & (uint64(1) << index)) != 0
}

func (bs Uint64BitString) Equal(a BitString) bool {
	if bs.Size() != a.Size() {
		return false
	}
	if bs.IsEmpty() {
		return true
	}

	if otherBs, ok := a.(Uint64BitString); ok {
		return bs.value == otherBs.value && bs.len == otherBs.len
	}

	return bs.GetLCPLength(a) == bs.Size()
}

func (bs Uint64BitString) Data() []byte {
	if bs.len == 0 {
		return []byte{}
	}
	numBytes := (int(bs.len) + 7) / 8

	data := make([]byte, numBytes)
	val := bs.value

	for i := 0; i < numBytes; i++ {
		data[i] = byte(val & 0xFF)
		val >>= 8
	}

	return data
}

func (bs Uint64BitString) String() string {
	if bs.len == 0 {
		return "<empty>"
	}

	var sb strings.Builder
	sb.Grow(int(bs.len))

	for i := uint32(0); i < uint32(bs.len); i++ {
		if bs.At(i) {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	sb.WriteString(": (")
	sb.WriteString(strconv.Itoa(int(bs.len)))
	sb.WriteString(" bits) [val=")
	sb.WriteString(strconv.FormatUint(bs.value, 10))
	sb.WriteString("]")

	return sb.String()
}

func (bs Uint64BitString) GetLCPLength(other BitString) uint32 {
	aSize := bs.Size()
	bSize := other.Size()

	if aSize == 0 || bSize == 0 {
		return 0
	}

	minLengthBits := aSize
	if bSize < minLengthBits {
		minLengthBits = bSize
	}

	if otherBs, ok := other.(Uint64BitString); ok {
		xor := bs.value ^ otherBs.value
		lcp := uint32(bits.TrailingZeros64(xor))

		if lcp < minLengthBits {
			return lcp
		}
		return minLengthBits
	}

	lcp := uint32(0)
	for lcp < minLengthBits {
		if bs.At(lcp) != other.At(lcp) {
			break
		}
		lcp++
	}
	return lcp
}

func (bs Uint64BitString) HasPrefix(prefixToCheck BitString) bool {
	prefixSize := prefixToCheck.Size()
	if prefixSize == 0 {
		return true
	}
	if prefixSize > bs.Size() {
		return false
	}

	if otherBs, ok := prefixToCheck.(Uint64BitString); ok {
		mask := ^uint64(0)
		if otherBs.len < 64 {
			mask = (uint64(1) << otherBs.len) - 1
		}
		return (bs.value & mask) == otherBs.value
	}

	return bs.GetLCPLength(prefixToCheck) == prefixSize
}

func (bs Uint64BitString) Prefix(size int) BitString {
	if size == 0 {
		return Uint64BitString{}
	}
	if int(bs.len) == size {
		return bs
	}

	mask := (uint64(1) << size) - 1
	return Uint64BitString{
		value: bs.value & mask,
		len:   int8(size),
	}
}

func (bs Uint64BitString) Hash() uint64 {
	// Combine value with length to avoid collisions
	return bs.value ^ (uint64(bs.len) << 56)
}

func (bs Uint64BitString) HashWithSeed(seed uint64) uint64 {
	// Combine value with length and seed to avoid collisions
	return bs.value ^ (uint64(bs.len) << 56) ^ seed
}

func (bs Uint64BitString) Eq(other BitString) bool {
	return bs.Equal(other)
}

func (bs Uint64BitString) Compare(other BitString) int {
	aSize := bs.Size()
	bSize := other.Size()

	if aSize == 0 && bSize == 0 {
		return 0
	}
	if aSize == 0 {
		return -1
	}
	if bSize == 0 {
		return 1
	}

	minSize := aSize
	if bSize < minSize {
		minSize = bSize
	}

	if otherBs, ok := other.(Uint64BitString); ok {
		xor := bs.value ^ otherBs.value
		if xor != 0 {
			firstDiffBit := uint32(bits.TrailingZeros64(xor))
			if firstDiffBit < minSize {
				if (bs.value & (uint64(1) << firstDiffBit)) != 0 {
					return 1
				}
				return -1
			}
		}
	} else {
		for i := uint32(0); i < minSize; i++ {
			aBit := bs.At(i)
			bBit := other.At(i)
			if !aBit && bBit {
				return -1
			}
			if aBit && !bBit {
				return 1
			}
		}
	}

	if aSize < bSize {
		return -1
	}
	if aSize > bSize {
		return 1
	}
	return 0
}
