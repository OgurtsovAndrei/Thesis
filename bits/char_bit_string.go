package bits

import (
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

var _ BitString = CharBitString{}

type CharBitString struct {
	data     string
	sizeBits uint32
}

func NewCharBitString(text string) CharBitString {
	return CharBitString{
		data:     text,
		sizeBits: uint32(len(text)) * 8,
	}
}

func NewCharFromUint64(value uint64) CharBitString {
	buf := make([]byte, 8)
	buf[0] = byte(value)
	buf[1] = byte(value >> 8)
	buf[2] = byte(value >> 16)
	buf[3] = byte(value >> 24)
	buf[4] = byte(value >> 32)
	buf[5] = byte(value >> 40)
	buf[6] = byte(value >> 48)
	buf[7] = byte(value >> 56)

	return CharBitString{
		data:     string(buf),
		sizeBits: 64,
	}
}

func NewCharFromBinary(text string) CharBitString {
	for _, r := range text {
		if r != '0' && r != '1' {
			panic(fmt.Sprintf("invalid string format, %q", text))
		}
	}

	size := len(text)
	if size == 0 {
		return CharBitString{}
	}

	numBytes := (size + 7) / 8
	dataBytes := make([]byte, numBytes)

	for i, r := range text {
		if r == '1' {
			byteIndex := i / 8
			bitIndex := i % 8
			dataBytes[byteIndex] |= 1 << bitIndex
		}
	}

	return CharBitString{
		data:     string(dataBytes),
		sizeBits: uint32(size),
	}
}

func NewCharBitStringPrefix(bs BitString, size uint32) BitString {
	if size > bs.Size() {
		panic("size exceeds original bitstring size")
	}
	if size == bs.Size() {
		return bs
	}
	if size == 0 {
		return CharBitString{}
	}

	return bs.(CharBitString).Prefix(int(size))

}

func NewCharBitStringFromDataAndSize(data []byte, size uint32) CharBitString {
	if size == 0 {
		return CharBitString{}
	}

	numBytes := (size + 7) / 8
	if uint32(len(data)) < numBytes {
		panic("data length is insufficient for the specified size")
	}

	resultData := make([]byte, numBytes)
	copy(resultData, data[:numBytes])

	if size%8 != 0 {
		lastByteIndex := numBytes - 1
		mask := byte((1 << (size % 8)) - 1)
		resultData[lastByteIndex] &= mask
	}

	return CharBitString{
		data:     string(resultData),
		sizeBits: size,
	}
}

func (bs CharBitString) Size() uint32 {
	return bs.sizeBits
}

func (bs CharBitString) IsEmpty() bool {
	return bs.sizeBits == 0
}

func (bs CharBitString) At(index uint32) bool {
	if index >= bs.sizeBits {
		panic("index out of bounds")
	}
	byteIndex := index / 8
	bitIndex := index % 8
	return (bs.data[byteIndex] & (1 << bitIndex)) != 0
}

func (bs CharBitString) GetLCPLength(other BitString) uint32 {
	aSize := bs.Size()
	bSize := other.Size()
	aData := bs.data
	bData := other.(CharBitString).data

	if aSize == 0 || bSize == 0 || len(aData) == 0 || len(bData) == 0 {
		return 0
	}

	result := uint32(0)
	minLengthBits := aSize
	if bSize < minLengthBits {
		minLengthBits = bSize
	}

	minByteLength := minLengthBits / 8
	i := uint32(0)

	for i < minByteLength {
		if aData[i] != bData[i] {
			break
		}
		i++
	}
	result = i * 8

	if result < minLengthBits {
		xorVal := aData[i] ^ bData[i]
		if xorVal == 0 {
			result += 8
		} else {
			result += uint32(bits.TrailingZeros8(xorVal))
		}
	}

	if minLengthBits < result {
		result = minLengthBits
	}

	return result
}

func (bs CharBitString) HasPrefix(prefixToCheck BitString) bool {
	if prefixToCheck.Size() == 0 {
		return true
	}
	return bs.GetLCPLength(prefixToCheck) == prefixToCheck.Size()
}

func (bs CharBitString) Equal(a BitString) bool {
	if bs.Size() != a.Size() {
		return false
	}
	if bs.Size() == 0 {
		return true
	}
	return bs.GetLCPLength(a) == bs.Size()
}

func (bs CharBitString) Prefix(size int) BitString {
	reqBytes := (size + 7) / 8
	dataBytes := len(bs.data)
	if reqBytes > dataBytes {
		reqBytes = dataBytes
	}

	resultData := []byte(bs.data[:reqBytes])

	if size%8 != 0 && reqBytes > 0 {
		lastByteIndex := reqBytes - 1
		mask := byte((1 << (size % 8)) - 1)
		resultData[lastByteIndex] &= mask
	}

	return CharBitString{
		data:     string(resultData),
		sizeBits: uint32(size),
	}
}

func (bs CharBitString) Data() []byte {
	return []byte(bs.data)
}

func (bs CharBitString) String() string {
	if bs.sizeBits == 0 {
		return "<empty>"
	}

	var sb strings.Builder
	sb.Grow(int(bs.sizeBits))

	for i := uint32(0); i < bs.sizeBits; i++ {
		if bs.At(i) {
			sb.WriteByte('1')
		} else {
			sb.WriteByte('0')
		}
	}
	sb.WriteString(": (")
	sb.WriteString(strconv.Itoa(int(bs.sizeBits)))
	sb.WriteString(" bits) [")
	sb.WriteString(bs.data)
	sb.WriteString("]")

	return sb.String()
}
