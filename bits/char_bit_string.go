package bits

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math/bits"
	"strconv"
	"strings"

	"github.com/zeebo/xxh3"
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

func (bs CharBitString) Hash() uint64 {
	h := fnv.New64a()

	// Write sizeBits first to avoid collisions
	sizeBytes := make([]byte, 4)
	sizeBytes[0] = byte(bs.sizeBits)
	sizeBytes[1] = byte(bs.sizeBits >> 8)
	sizeBytes[2] = byte(bs.sizeBits >> 16)
	sizeBytes[3] = byte(bs.sizeBits >> 24)
	h.Write(sizeBytes)

	data := []byte(bs.data)
	numBytes := (bs.sizeBits + 7) / 8

	if numBytes > 0 && uint32(len(data)) >= numBytes {
		h.Write(data[:numBytes])
	}

	return h.Sum64()
}

func (bs CharBitString) HashWithSeed(seed uint64) uint64 {
	h := xxh3.New()

	// Write seed as bytes
	seedBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(seedBuf, seed)
	h.Write(seedBuf)

	// Write sizeBits first to avoid collisions
	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, bs.sizeBits)
	h.Write(sizeBuf)

	data := []byte(bs.data)
	numBytes := (bs.sizeBits + 7) / 8

	if numBytes > 0 && uint32(len(data)) >= numBytes {
		h.Write(data[:numBytes])
	}

	return h.Sum64()
}

func (bs CharBitString) Eq(other BitString) bool {
	return bs.Equal(other)
}

func (bs CharBitString) Compare(other BitString) int {
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

	if otherBs, ok := other.(CharBitString); ok {
		aData := []byte(bs.data)
		bData := []byte(otherBs.data)
		minBytes := (minSize + 7) / 8

		for i := uint32(0); i < minBytes; i++ {
			aByte := byte(0)
			bByte := byte(0)

			if i < uint32(len(aData)) {
				aByte = aData[i]
			}
			if i < uint32(len(bData)) {
				bByte = bData[i]
			}

			if aByte != bByte {
				xor := aByte ^ bByte
				firstDiffBit := i*8 + uint32(bits.TrailingZeros8(xor))
				if firstDiffBit < minSize {
					if (aByte & (1 << (firstDiffBit % 8))) != 0 {
						return 1
					}
					return -1
				}
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

func (bs CharBitString) TrimTrailingZeros() BitString {
	if bs.sizeBits == 0 {
		return bs
	}

	data := []byte(bs.data)
	lastOneBit := int32(-1)

	// Find the last bit that is 1
	for i := int32(len(data) - 1); i >= 0; i-- {
		b := data[i]
		if b != 0 {
			// Find the highest bit in this byte
			highestBit := 7 - bits.LeadingZeros8(b)
			lastOneBit = i*8 + int32(highestBit)
			break
		}
	}

	if lastOneBit < 0 {
		// All bits are zero
		return CharBitString{}
	}

	newSize := uint32(lastOneBit + 1)
	if newSize > bs.sizeBits {
		newSize = bs.sizeBits
	}

	return bs.Prefix(int(newSize))
}

func (bs CharBitString) AppendBit(bit bool) BitString {
	newSize := bs.sizeBits + 1
	numBytes := (newSize + 7) / 8

	var newData []byte
	if numBytes > uint32(len(bs.data)) {
		// Need to allocate a new byte
		newData = make([]byte, numBytes)
		copy(newData, []byte(bs.data))
	} else {
		// Can reuse existing capacity
		newData = make([]byte, len(bs.data))
		copy(newData, []byte(bs.data))
	}

	if bit {
		byteIndex := bs.sizeBits / 8
		bitIndex := bs.sizeBits % 8
		newData[byteIndex] |= 1 << bitIndex
	}

	return CharBitString{
		data:     string(newData),
		sizeBits: newSize,
	}
}

func (bs CharBitString) IsAllOnes() bool {
	if bs.sizeBits == 0 {
		return false
	}

	data := []byte(bs.data)
	fullBytes := bs.sizeBits / 8
	remainingBits := bs.sizeBits % 8

	// Check full bytes
	for i := uint32(0); i < fullBytes; i++ {
		if i >= uint32(len(data)) || data[i] != 0xFF {
			return false
		}
	}

	// Check remaining bits in the last byte
	if remainingBits > 0 {
		if fullBytes >= uint32(len(data)) {
			return false
		}
		mask := byte((1 << remainingBits) - 1)
		if (data[fullBytes] & mask) != mask {
			return false
		}
	}

	return true
}

func (bs CharBitString) Successor() BitString {
	// Convert to Uint64BitString, compute successor, convert back
	// This ensures consistent behavior across all implementations

	if bs.sizeBits == 0 {
		return CharBitString{
			data:     string([]byte{1}),
			sizeBits: 1,
		}
	}

	// Convert to Uint64BitString format
	if bs.sizeBits > 64 {
		// For large BitStrings, we need to implement the full logic
		// For now, fall back to a simpler approach
		return bs // TODO: Implement for large strings
	}

	// Convert to uint64 value
	value := uint64(0)
	for i := uint32(0); i < bs.sizeBits; i++ {
		if bs.At(i) {
			value |= uint64(1) << i
		}
	}

	// Create temporary Uint64BitString and compute successor
	tempBs := Uint64BitString{value: value, len: int8(bs.sizeBits)}
	successorBs := tempBs.Successor()

	// Convert back to CharBitString
	tempUint64, ok := successorBs.(Uint64BitString)
	if !ok {
		return bs // Fallback
	}

	newSize := uint32(tempUint64.len)
	numBytes := (newSize + 7) / 8
	newData := make([]byte, numBytes)

	for i := uint32(0); i < newSize; i++ {
		if (tempUint64.value & (uint64(1) << i)) != 0 {
			byteIndex := i / 8
			bitIndex := i % 8
			newData[byteIndex] |= 1 << bitIndex
		}
	}

	return CharBitString{
		data:     string(newData),
		sizeBits: newSize,
	}
}
