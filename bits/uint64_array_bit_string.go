package bits

import (
	"Thesis/errutil"
	"encoding/binary"
	"hash/fnv"
	"math/bits"
	"strconv"
	"strings"

	"github.com/zeebo/xxh3"
)

var _ BitString = Uint64ArrayBitString{}

type Uint64ArrayBitString struct {
	data     []uint64
	sizeBits uint32
}

func NewUint64ArrayBitString(sizeBits uint32) Uint64ArrayBitString {
	numWords := (sizeBits + 63) / 64
	return Uint64ArrayBitString{
		data:     make([]uint64, numWords),
		sizeBits: sizeBits,
	}
}

func NewUint64ArrayFromBinaryText(text string) Uint64ArrayBitString {
	for _, r := range text {
		errutil.BugOn(r != '0' && r != '1', "invalid string format, %q", text)
	}

	size := len(text)
	if size == 0 {
		return Uint64ArrayBitString{}
	}

	numWords := (uint32(size) + 63) / 64
	data := make([]uint64, numWords)

	for i, r := range text {
		if r == '1' {
			wordIndex := uint32(i) / 64
			bitIndex := uint32(i) % 64
			data[wordIndex] |= uint64(1) << bitIndex
		}
	}

	return Uint64ArrayBitString{
		data:     data,
		sizeBits: uint32(size),
	}
}

func NewUint64ArrayFromDataAndSize(data []byte, size uint32) Uint64ArrayBitString {
	if size == 0 {
		return Uint64ArrayBitString{}
	}

	numWords := (size + 63) / 64
	result := make([]uint64, numWords)

	numBytes := (size + 7) / 8
	if uint32(len(data)) < numBytes {
		panic("data length is insufficient for the specified size")
	}

	for i := uint32(0); i < numBytes; i++ {
		wordIndex := i / 8
		byteOffsetInWord := i % 8
		if wordIndex < uint32(len(result)) {
			result[wordIndex] |= uint64(data[i]) << (byteOffsetInWord * 8)
		}
	}

	if size%64 != 0 {
		lastWordIndex := numWords - 1
		mask := (uint64(1) << (size % 64)) - 1
		result[lastWordIndex] &= mask
	}

	return Uint64ArrayBitString{
		data:     result,
		sizeBits: size,
	}
}

func NewUint64ArrayBitStringPrefix(bs BitString, size uint32) BitString {
	errutil.BugOn(size > bs.Size(), "size exceeds original bitstring size")
	otherBs, ok := bs.(Uint64ArrayBitString)
	errutil.BugOn(!ok, "NewUint64ArrayBitStringPrefix can only be called on Uint64ArrayBitString")

	return otherBs.Prefix(int(size))
}

func (bs Uint64ArrayBitString) Size() uint32 {
	return bs.sizeBits
}

func (bs Uint64ArrayBitString) IsEmpty() bool {
	return bs.sizeBits == 0
}

func (bs Uint64ArrayBitString) At(index uint32) bool {
	if index >= bs.sizeBits {
		panic("index out of bounds")
	}
	wordIndex := index / 64
	bitIndex := index % 64
	return (bs.data[wordIndex] & (uint64(1) << bitIndex)) != 0
}

func (bs Uint64ArrayBitString) Equal(a BitString) bool {
	if bs.Size() != a.Size() {
		return false
	}
	if bs.IsEmpty() {
		return true
	}

	if otherBs, ok := a.(Uint64ArrayBitString); ok {
		if len(bs.data) != len(otherBs.data) {
			return false
		}
		for i := 0; i < len(bs.data); i++ {
			if bs.data[i] != otherBs.data[i] {
				return false
			}
		}
		return true
	}

	return bs.GetLCPLength(a) == bs.Size()
}

func (bs Uint64ArrayBitString) Data() []byte {
	if bs.sizeBits == 0 {
		return []byte{}
	}

	numBytes := (bs.sizeBits + 7) / 8
	result := make([]byte, numBytes)

	for i := uint32(0); i < numBytes; i++ {
		wordIndex := i / 8
		byteOffsetInWord := i % 8
		if wordIndex < uint32(len(bs.data)) {
			result[i] = byte(bs.data[wordIndex] >> (byteOffsetInWord * 8))
		}
	}

	return result
}

func (bs Uint64ArrayBitString) String() string {
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
	for i, word := range bs.data {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(strconv.FormatUint(word, 10))
	}
	sb.WriteString("]")

	return sb.String()
}

func (bs Uint64ArrayBitString) GetLCPLength(other BitString) uint32 {
	aSize := bs.Size()
	bSize := other.Size()

	if aSize == 0 || bSize == 0 {
		return 0
	}

	minLengthBits := aSize
	if bSize < minLengthBits {
		minLengthBits = bSize
	}

	if otherBs, ok := other.(Uint64ArrayBitString); ok {
		minWords := (minLengthBits + 63) / 64

		for i := uint32(0); i < minWords; i++ {
			wordA := uint64(0)
			wordB := uint64(0)

			if i < uint32(len(bs.data)) {
				wordA = bs.data[i]
			}
			if i < uint32(len(otherBs.data)) {
				wordB = otherBs.data[i]
			}

			if wordA != wordB {
				xor := wordA ^ wordB
				lcp := i*64 + uint32(bits.TrailingZeros64(xor))
				if lcp < minLengthBits {
					return lcp
				}
				return minLengthBits
			}
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

func (bs Uint64ArrayBitString) HasPrefix(prefixToCheck BitString) bool {
	prefixSize := prefixToCheck.Size()
	if prefixSize == 0 {
		return true
	}
	if prefixSize > bs.Size() {
		return false
	}

	if otherBs, ok := prefixToCheck.(Uint64ArrayBitString); ok {
		prefixWords := (prefixSize + 63) / 64

		for i := uint32(0); i < prefixWords-1; i++ {
			wordA := uint64(0)
			wordB := uint64(0)

			if i < uint32(len(bs.data)) {
				wordA = bs.data[i]
			}
			if i < uint32(len(otherBs.data)) {
				wordB = otherBs.data[i]
			}

			if wordA != wordB {
				return false
			}
		}

		if prefixWords > 0 {
			lastWordIndex := prefixWords - 1
			bitsInLastWord := prefixSize % 64
			if bitsInLastWord == 0 {
				bitsInLastWord = 64
			}

			wordA := uint64(0)
			wordB := uint64(0)

			if lastWordIndex < uint32(len(bs.data)) {
				wordA = bs.data[lastWordIndex]
			}
			if lastWordIndex < uint32(len(otherBs.data)) {
				wordB = otherBs.data[lastWordIndex]
			}

			mask := (uint64(1) << bitsInLastWord) - 1
			return (wordA & mask) == (wordB & mask)
		}

		return true
	}

	return bs.GetLCPLength(prefixToCheck) == prefixSize
}

func (bs Uint64ArrayBitString) Prefix(size int) BitString {
	if size == 0 {
		return Uint64ArrayBitString{}
	}
	if int(bs.sizeBits) == size {
		return bs
	}

	numWords := (uint32(size) + 63) / 64
	newData := make([]uint64, numWords)

	copyWords := numWords
	if copyWords > uint32(len(bs.data)) {
		copyWords = uint32(len(bs.data))
	}

	copy(newData, bs.data[:copyWords])

	if uint32(size)%64 != 0 {
		lastWordIndex := numWords - 1
		mask := (uint64(1) << (uint32(size) % 64)) - 1
		newData[lastWordIndex] &= mask
	}

	return Uint64ArrayBitString{
		data:     newData,
		sizeBits: uint32(size),
	}
}

func (bs Uint64ArrayBitString) Hash() uint64 {
	h := fnv.New64a()

	// Write sizeBits first to avoid collisions
	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, bs.sizeBits)
	h.Write(sizeBuf)

	numWords := (bs.sizeBits + 63) / 64
	buf := make([]byte, 8)
	for i := uint32(0); i < numWords && i < uint32(len(bs.data)); i++ {
		binary.LittleEndian.PutUint64(buf, bs.data[i])
		h.Write(buf)
	}

	return h.Sum64()
}

func (bs Uint64ArrayBitString) HashWithSeed(seed uint64) uint64 {
	h := xxh3.New()

	// Write seed as bytes
	seedBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(seedBuf, seed)
	h.Write(seedBuf)

	// Write sizeBits first to avoid collisions
	sizeBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBuf, bs.sizeBits)
	h.Write(sizeBuf)

	numWords := (bs.sizeBits + 63) / 64
	buf := make([]byte, 8)
	for i := uint32(0); i < numWords && i < uint32(len(bs.data)); i++ {
		binary.LittleEndian.PutUint64(buf, bs.data[i])
		h.Write(buf)
	}

	return h.Sum64()
}

func (bs Uint64ArrayBitString) Eq(other BitString) bool {
	return bs.Equal(other)
}

func (bs Uint64ArrayBitString) Compare(other BitString) int {
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

	if otherBs, ok := other.(Uint64ArrayBitString); ok {
		minWords := (minSize + 63) / 64

		for i := uint32(0); i < minWords; i++ {
			aWord := uint64(0)
			bWord := uint64(0)

			if i < uint32(len(bs.data)) {
				aWord = bs.data[i]
			}
			if i < uint32(len(otherBs.data)) {
				bWord = otherBs.data[i]
			}

			if aWord != bWord {
				xor := aWord ^ bWord
				firstDiffBit := i*64 + uint32(bits.TrailingZeros64(xor))
				if firstDiffBit < minSize {
					if (aWord & (uint64(1) << (firstDiffBit % 64))) != 0 {
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

func (bs Uint64ArrayBitString) TrimTrailingZeros() BitString {
	if bs.sizeBits == 0 {
		return bs
	}

	// Find the last non-zero bit
	lastOneBit := int32(-1)

	for i := int32(len(bs.data) - 1); i >= 0; i-- {
		word := bs.data[i]
		if word != 0 {
			// Find the highest bit position in this word
			highestBit := 63 - bits.LeadingZeros64(word)
			lastOneBit = i*64 + int32(highestBit)
			break
		}
	}

	if lastOneBit < 0 {
		// All bits are zero
		return Uint64ArrayBitString{}
	}

	newSize := uint32(lastOneBit + 1)
	if newSize > bs.sizeBits {
		newSize = bs.sizeBits
	}

	return bs.Prefix(int(newSize))
}

func (bs Uint64ArrayBitString) AppendBit(bit bool) BitString {
	newSize := bs.sizeBits + 1
	newNumWords := (newSize + 63) / 64

	var newData []uint64
	if newNumWords > uint32(len(bs.data)) {
		// Need to allocate a new word
		newData = make([]uint64, newNumWords)
		copy(newData, bs.data)
	} else {
		// Can reuse existing capacity
		newData = make([]uint64, len(bs.data))
		copy(newData, bs.data)
	}

	if bit {
		wordIndex := bs.sizeBits / 64
		bitIndex := bs.sizeBits % 64
		newData[wordIndex] |= uint64(1) << bitIndex
	}

	return Uint64ArrayBitString{
		data:     newData,
		sizeBits: newSize,
	}
}

func (bs Uint64ArrayBitString) IsAllOnes() bool {
	if bs.sizeBits == 0 {
		return false
	}

	fullWords := bs.sizeBits / 64
	remainingBits := bs.sizeBits % 64

	// Check full words
	for i := uint32(0); i < fullWords; i++ {
		if i >= uint32(len(bs.data)) || bs.data[i] != ^uint64(0) {
			return false
		}
	}

	// Check remaining bits in the last word
	if remainingBits > 0 {
		if fullWords >= uint32(len(bs.data)) {
			return false
		}
		mask := (uint64(1) << remainingBits) - 1
		if (bs.data[fullWords] & mask) != mask {
			return false
		}
	}

	return true
}

func (bs Uint64ArrayBitString) Successor() BitString {
	// Convert to Uint64BitString if possible, compute successor, convert back
	// This ensures consistent behavior across all implementations

	if bs.sizeBits == 0 {
		newData := []uint64{1}
		return Uint64ArrayBitString{
			data:     newData,
			sizeBits: 1,
		}
	}

	// For small BitStrings (≤64 bits), use Uint64BitString logic
	if bs.sizeBits <= 64 {
		// Convert to uint64 value
		value := uint64(0)
		if len(bs.data) > 0 {
			value = bs.data[0]
		}

		// Create temporary Uint64BitString and compute successor
		tempBs := Uint64BitString{value: value, len: int8(bs.sizeBits)}
		successorBs := tempBs.Successor()

		// Convert back to Uint64ArrayBitString
		tempUint64, ok := successorBs.(Uint64BitString)
		if !ok {
			return bs // Fallback
		}

		newSize := uint32(tempUint64.len)
		numWords := (newSize + 63) / 64
		newData := make([]uint64, numWords)
		if numWords > 0 {
			newData[0] = tempUint64.value
		}

		return Uint64ArrayBitString{
			data:     newData,
			sizeBits: newSize,
		}
	}

	// For larger BitStrings, implement full multi-word successor logic
	// We need to convert the LSB-first representation to a normal number,
	// increment it, then convert back to LSB-first

	// First, convert to big-endian byte array for easier arithmetic
	numBytes := (bs.sizeBits + 7) / 8
	bigEndianBytes := make([]byte, numBytes)

	// Copy LSB-first bits to big-endian bytes
	for bitIdx := uint32(0); bitIdx < bs.sizeBits; bitIdx++ {
		if bs.At(bitIdx) {
			byteIdx := numBytes - 1 - (bitIdx / 8)
			bitInByte := 7 - (bitIdx % 8)
			bigEndianBytes[byteIdx] |= 1 << bitInByte
		}
	}

	// Add 1 to the big-endian number
	carry := byte(1)
	for i := len(bigEndianBytes) - 1; i >= 0 && carry > 0; i-- {
		sum := uint16(bigEndianBytes[i]) + uint16(carry)
		bigEndianBytes[i] = byte(sum)
		if sum > 255 {
			carry = 1
		} else {
			carry = 0
		}
	}

	// Determine result size
	var resultSize uint32
	var resultBytes []byte
	if carry > 0 {
		// Need one more bit
		resultSize = bs.sizeBits + 1
		newNumBytes := (resultSize + 7) / 8
		resultBytes = make([]byte, newNumBytes)

		if newNumBytes > numBytes {
			// Need new byte
			resultBytes[0] = 1
			copy(resultBytes[1:], bigEndianBytes)
		} else {
			// Can fit in existing bytes
			copy(resultBytes, bigEndianBytes)
			resultBytes[0] |= 0x80 // Set MSB
		}
	} else {
		resultSize = bs.sizeBits
		resultBytes = bigEndianBytes
	}

	// Convert back to LSB-first BitString
	resultNumWords := (resultSize + 63) / 64
	resultData := make([]uint64, resultNumWords)

	for bitIdx := uint32(0); bitIdx < resultSize; bitIdx++ {
		// Map from LSB-first bit index to big-endian bit position
		bigEndianByteIdx := len(resultBytes) - 1 - int(bitIdx/8)
		bitInByte := 7 - (bitIdx % 8)

		if bigEndianByteIdx >= 0 && (resultBytes[bigEndianByteIdx]&(1<<bitInByte)) != 0 {
			wordIdx := bitIdx / 64
			bitInWord := bitIdx % 64
			resultData[wordIdx] |= uint64(1) << bitInWord
		}
	}

	return Uint64ArrayBitString{
		data:     resultData,
		sizeBits: resultSize,
	}
}
