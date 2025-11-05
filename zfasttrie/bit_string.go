package zfasttrie

import (
	"math/bits"
	"strconv"
	"strings"
)

type BitString struct {
	data     string
	sizeBits uint32
}

func NewBitString(text string) BitString {
	return BitString{
		data:     text,
		sizeBits: uint32(len(text)) * 8,
	}
}

func NewFromUint64(value uint64) BitString {
	buf := make([]byte, 8)
	buf[0] = byte(value)
	buf[1] = byte(value >> 8)
	buf[2] = byte(value >> 16)
	buf[3] = byte(value >> 24)
	buf[4] = byte(value >> 32)
	buf[5] = byte(value >> 40)
	buf[6] = byte(value >> 48)
	buf[7] = byte(value >> 56)

	return BitString{
		data:     string(buf),
		sizeBits: 64,
	}
}

func NewFromBinary(text string) BitString {
	for _, r := range text {
		BugOn(r != '0' && r != '1', "invalid string format, %q", text)
	}

	size := len(text)
	if size == 0 {
		return BitString{}
	}

	numBytes := (size + 7) / 8
	dataBytes := make([]byte, numBytes)

	for i, r := range text {
		if r == '1' {
			byteIndex := i / 8
			bitIndex := i % 8 // Corresponds to the position within the byte, 0 for LSB, 7 for MSB
			dataBytes[byteIndex] |= 1 << bitIndex
		}
	}

	return BitString{
		data:     string(dataBytes),
		sizeBits: uint32(size),
	}
}

func NewBitStringPrefix(bs BitString, size uint32) BitString {
	if size > bs.sizeBits {
		panic("size exceeds original bitstring size")
	}
	if size == bs.sizeBits {
		return bs
	}
	if size == 0 {
		return BitString{}
	}

	reqBytes := (size + 7) / 8

	dataBytes := uint32(len(bs.data))
	if reqBytes > dataBytes {
		reqBytes = dataBytes
	}

	resultData := []byte(bs.data[:reqBytes])

	// Zero out tail bits in the last byte if size is not a multiple of 8
	if size%8 != 0 && reqBytes > 0 {
		lastByteIndex := reqBytes - 1
		mask := byte((1 << (size % 8)) - 1)
		resultData[lastByteIndex] &= mask
	}

	return BitString{
		data:     string(resultData),
		sizeBits: size,
	}
}

func (bs BitString) Size() uint32 {
	return bs.sizeBits
}

func (bs BitString) IsEmpty() bool {
	return bs.sizeBits == 0
}

func (bs BitString) At(index uint32) bool {
	if index >= bs.sizeBits {
		panic("index out of bounds")
	}
	byteIndex := index / 8
	bitIndex := index % 8
	return (bs.data[byteIndex] & (1 << bitIndex)) != 0
}

func GetLCPLength(a, b BitString) uint32 {
	if a.sizeBits == 0 || b.sizeBits == 0 || len(a.data) == 0 || len(b.data) == 0 {
		return 0
	}

	result := uint32(0)
	minLengthBits := a.sizeBits
	if b.sizeBits < minLengthBits {
		minLengthBits = b.sizeBits
	}

	minByteLength := minLengthBits / 8
	i := uint32(0)

	for i < minByteLength {
		if a.data[i] != b.data[i] {
			break
		}
		i++
	}
	result = i * 8

	if result < minLengthBits {
		xorVal := a.data[i] ^ b.data[i]
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

func IsPrefix(value, prefixToCheck BitString) bool {
	if prefixToCheck.sizeBits == 0 {
		return true
	}
	return GetLCPLength(value, prefixToCheck) == prefixToCheck.sizeBits
}

func (bs BitString) Equal(a BitString) bool {
	if bs.sizeBits != a.sizeBits {
		return false
	}
	if bs.sizeBits == 0 {
		return true
	}
	return GetLCPLength(bs, a) == bs.sizeBits
}

func (bs BitString) String() string {
	// returns reversed little endian string representation
	// bites in memory are placed other way
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
