package bits

import (
	"Thesis/errutil"
	"encoding/binary"
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

// BitString represents a sequence of bits stored in a slice of uint64 words.
type BitString struct {
	data     []uint64
	sizeBits uint32
}

// NewBitString initializes a BitString with the given number of bits.
func NewBitString(sizeBits uint32) BitString {
	numWords := (sizeBits + 63) / 64
	return BitString{
		data:     make([]uint64, numWords),
		sizeBits: sizeBits,
	}
}

// NewFromUint64 creates a BitString from a single uint64 value.
func NewFromUint64(value uint64) BitString {
	bs := NewBitString(64)
	bs.data[0] = value
	return bs
}

// NewFromBinary creates a BitString from a binary string (e.g., "1011").
func NewFromBinary(text string) BitString {
	for _, r := range text {
		errutil.BugOn(r != '0' && r != '1', "invalid string format, %q", text)
	}

	size := len(text)
	if size == 0 {
		return BitString{}
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

	return BitString{
		data:     data,
		sizeBits: uint32(size),
	}
}

// NewFromText creates a BitString from a string (converts to binary representation).
func NewFromText(text string) BitString {
	size := uint32(len(text)) * 8
	data := []byte(text)
	return NewFromDataAndSize(data, size)
}

// NewFromDataAndSize creates a BitString from raw bytes and a size in bits.
func NewFromDataAndSize(data []byte, size uint32) BitString {
	if size == 0 {
		return BitString{}
	}

	numWords := (size + 63) / 64
	result := make([]uint64, numWords)

	numBytes := (size + 7) / 8
	if uint32(len(data)) < numBytes {
		panic("data length is insufficient for the specified size")
	}

	// Process full words (8 bytes each)
	fullWords := numBytes / 8
	for i := uint32(0); i < fullWords; i++ {
		result[i] = binary.LittleEndian.Uint64(data[i*8:])
	}

	// Handle remaining bytes
	if numBytes%8 != 0 {
		lastWordIdx := fullWords
		remainingBytes := numBytes % 8
		offset := fullWords * 8
		var lastWord uint64
		for j := uint32(0); j < remainingBytes; j++ {
			lastWord |= uint64(data[offset+j]) << (j * 8)
		}
		result[lastWordIdx] = lastWord
	}

	if size%64 != 0 {
		lastWordIndex := numWords - 1
		mask := (uint64(1) << (size % 64)) - 1
		result[lastWordIndex] &= mask
	}

	return BitString{
		data:     result,
		sizeBits: size,
	}
}

func (bs BitString) Size() uint32 {
	return bs.sizeBits
}

func (bs BitString) IsEmpty() bool {
	return bs.sizeBits == 0
}

func (bs BitString) HasValue() bool {
	return bs.sizeBits != 0
}

func (bs BitString) At(index uint32) bool {
	if index >= bs.sizeBits {
		panic(fmt.Sprintf("index out of bounds: %d >= %d", index, bs.sizeBits))
	}
	wordIndex := index / 64
	bitIndex := index % 64
	return (bs.data[wordIndex] & (uint64(1) << bitIndex)) != 0
}

func (bs BitString) Equal(other BitString) bool {
	if bs.sizeBits != other.sizeBits {
		return false
	}
	if bs.IsEmpty() {
		return true
	}
	if len(bs.data) != len(other.data) {
		return false
	}
	for i := range bs.data {
		if bs.data[i] != other.data[i] {
			return false
		}
	}
	return true
}

func (bs BitString) Eq(other BitString) bool {
	return bs.Equal(other)
}

func (bs BitString) Data() []byte {
	if bs.sizeBits == 0 {
		return []byte{}
	}

	numBytes := (bs.sizeBits + 7) / 8
	result := make([]byte, numBytes)

	fullWords := numBytes / 8
	for i := uint32(0); i < fullWords; i++ {
		binary.LittleEndian.PutUint64(result[i*8:], bs.data[i])
	}

	if numBytes%8 != 0 {
		lastWord := bs.data[fullWords]
		offset := fullWords * 8
		remainingBytes := numBytes % 8
		for j := uint32(0); j < remainingBytes; j++ {
			result[offset+j] = byte(lastWord >> (j * 8))
		}
	}

	return result
}

func (bs BitString) PrettyString() string {
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

func (bs BitString) String() string {
	return bs.PrettyString()
}

func (bs BitString) GetLCPLength(other BitString) uint32 {
	aSize := bs.sizeBits
	bSize := other.sizeBits

	if aSize == 0 || bSize == 0 {
		return 0
	}

	minLengthBits := aSize
	if bSize < minLengthBits {
		minLengthBits = bSize
	}

	fullWords := minLengthBits / 64
	for i := uint32(0); i < fullWords; i++ {
		wA := bs.data[i]
		wB := other.data[i]
		if wA != wB {
			return i*64 + uint32(bits.TrailingZeros64(wA^wB))
		}
	}

	if minLengthBits%64 != 0 {
		i := fullWords
		wA := bs.data[i]
		wB := other.data[i]
		xor := wA ^ wB
		if xor != 0 {
			lcp := i*64 + uint32(bits.TrailingZeros64(xor))
			if lcp < minLengthBits {
				return lcp
			}
		}
	}

	return minLengthBits
}

func (bs BitString) HasPrefix(prefixToCheck BitString) bool {
	prefixSize := prefixToCheck.sizeBits
	if prefixSize == 0 {
		return true
	}
	if prefixSize > bs.sizeBits {
		return false
	}

	prefixWords := (prefixSize + 63) / 64

	for i := uint32(0); i < prefixWords-1; i++ {
		wordA := uint64(0)
		wordB := uint64(0)

		if i < uint32(len(bs.data)) {
			wordA = bs.data[i]
		}
		if i < uint32(len(prefixToCheck.data)) {
			wordB = prefixToCheck.data[i]
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
		if lastWordIndex < uint32(len(prefixToCheck.data)) {
			wordB = prefixToCheck.data[lastWordIndex]
		}

		mask := (uint64(1) << bitsInLastWord) - 1
		return (wordA & mask) == (wordB & mask)
	}

	return true
}

func (bs BitString) Prefix(size int) BitString {
	if size <= 0 {
		return BitString{}
	}
	if int(bs.sizeBits) == size {
		return bs
	}

	numWords := (uint32(size) + 63) / 64

	// If size is a multiple of 64, we can just slice the data.
	// This is safe because BitString is immutable and we never modify the underlying array.
	if uint32(size)%64 == 0 {
		// Ensure we don't go out of bounds if requested size is larger than actual data
		endWord := numWords
		if endWord > uint32(len(bs.data)) {
			endWord = uint32(len(bs.data))
		}

		// If we need more words than we have, we still need to allocate to provide zeros.
		if numWords > uint32(len(bs.data)) {
			newData := make([]uint64, numWords)
			copy(newData, bs.data)
			return BitString{
				data:     newData,
				sizeBits: uint32(size),
			}
		}

		return BitString{
			data:     bs.data[:endWord],
			sizeBits: uint32(size),
		}
	}

	newData := make([]uint64, numWords)

	copyWords := numWords
	if copyWords > uint32(len(bs.data)) {
		copyWords = uint32(len(bs.data))
	}

	copy(newData, bs.data[:copyWords])

	lastWordIndex := numWords - 1
	mask := (uint64(1) << (uint32(size) % 64)) - 1
	newData[lastWordIndex] &= mask

	return BitString{
		data:     newData,
		sizeBits: uint32(size),
	}
}

// Hash returns a 64-bit hash of the BitString using an optimized manual FNV-1a.
// See PERFORMANCE.md for performance benchmarks and rationale.
func (bs BitString) Hash() uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h := uint64(offset64)
	h ^= uint64(bs.sizeBits)
	h *= prime64

	for _, word := range bs.data {
		h ^= word
		h *= prime64
	}
	return h
}

// HashWithSeed returns a 64-bit hash of the BitString using an optimized manual FNV-1a.
// See PERFORMANCE.md for performance benchmarks and rationale.
func (bs BitString) HashWithSeed(seed uint64) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h := uint64(offset64) ^ seed
	h *= prime64
	h ^= uint64(bs.sizeBits)
	h *= prime64

	for _, word := range bs.data {
		h ^= word
		h *= prime64
	}
	return h
}

// Compare performs lexicographic comparison. Returns:
// -1 if bs < other
//  1 if bs > other
//  0 if bs == other
// See PERFORMANCE.md for performance benchmarks and rationale.
func (bs BitString) Compare(other BitString) int {
	aSize := bs.sizeBits
	bSize := other.sizeBits

	if aSize == 0 {
		if bSize == 0 {
			return 0
		}
		return -1
	}
	if bSize == 0 {
		return 1
	}

	minWords := len(bs.data)
	if len(other.data) < minWords {
		minWords = len(other.data)
	}

	for i := 0; i < minWords; i++ {
		aWord := bs.data[i]
		bWord := other.data[i]
		if aWord != bWord {
			xor := aWord ^ bWord
			diffBitInWord := uint32(bits.TrailingZeros64(xor))
			if (aWord & (uint64(1) << diffBitInWord)) != 0 {
				return 1
			}
			return -1
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

// TrieCompare implements trie-traversal-consistent comparison where trailing zeros come before trimmed
func (bs BitString) TrieCompare(other BitString) int {
	aSize := bs.sizeBits
	bSize := other.sizeBits

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

	fullWords := minSize / 64
	for i := uint32(0); i < fullWords; i++ {
		aWord := bs.data[i]
		bWord := other.data[i]
		if aWord != bWord {
			xor := aWord ^ bWord
			diffBit := uint32(bits.TrailingZeros64(xor))
			if (aWord & (uint64(1) << diffBit)) != 0 {
				return 1
			}
			return -1
		}
	}

	if minSize%64 != 0 {
		i := fullWords
		aWord := bs.data[i]
		bWord := other.data[i]
		if aWord != bWord {
			xor := aWord ^ bWord
			diffBit := uint32(bits.TrailingZeros64(xor))
			if diffBit < minSize%64 {
				if (aWord & (uint64(1) << diffBit)) != 0 {
					return 1
				}
				return -1
			}
		}
	}

	if aSize < bSize {
		if other.At(aSize) {
			return -1
		} else {
			return 1
		}
	}
	if aSize > bSize {
		if bs.At(bSize) {
			return 1
		} else {
			return -1
		}
	}
	return 0
}

func (bs BitString) TrimTrailingZeros() BitString {
	if bs.sizeBits == 0 {
		return bs
	}

	lastOneBit := int32(-1)

	for i := int32(len(bs.data) - 1); i >= 0; i-- {
		word := bs.data[i]
		if word != 0 {
			highestBit := 63 - bits.LeadingZeros64(word)
			lastOneBit = i*64 + int32(highestBit)
			break
		}
	}

	if lastOneBit < 0 {
		return BitString{}
	}

	newSize := uint32(lastOneBit + 1)
	if newSize > bs.sizeBits {
		newSize = bs.sizeBits
	}

	return bs.Prefix(int(newSize))
}

func (bs BitString) AppendBit(bit bool) BitString {
	newSize := bs.sizeBits + 1
	newNumWords := (newSize + 63) / 64

	var newData []uint64
	if newNumWords > uint32(len(bs.data)) {
		newData = make([]uint64, newNumWords)
		copy(newData, bs.data)
	} else {
		newData = make([]uint64, len(bs.data))
		copy(newData, bs.data)
	}

	if bit {
		wordIndex := bs.sizeBits / 64
		bitIndex := bs.sizeBits % 64
		newData[wordIndex] |= uint64(1) << bitIndex
	}

	return BitString{
		data:     newData,
		sizeBits: newSize,
	}
}

func (bs BitString) IsAllOnes() bool {
	if bs.sizeBits == 0 {
		return false
	}

	fullWords := bs.sizeBits / 64
	remainingBits := bs.sizeBits % 64

	for i := uint32(0); i < fullWords; i++ {
		if i >= uint32(len(bs.data)) || bs.data[i] != ^uint64(0) {
			return false
		}
	}

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

func (bs BitString) Successor() BitString {
	if bs.sizeBits == 0 {
		return NewFromBinary("1")
	}

	// 1. Find the highest index i such that At(i) == false
	lastWordIdx := int(bs.sizeBits-1) / 64
	lastZero := -1

	// Check last word first (might be partial)
	{
		w := bs.data[lastWordIdx]
		bitsInLastWord := (bs.sizeBits-1)%64 + 1
		mask := ^uint64(0)
		if bitsInLastWord < 64 {
			mask = (uint64(1) << bitsInLastWord) - 1
		}

		zeros := (^w) & mask
		if zeros != 0 {
			bitIdx := 63 - bits.LeadingZeros64(zeros)
			lastZero = lastWordIdx*64 + bitIdx
		}
	}

	if lastZero == -1 {
		// Check previous words
		for i := lastWordIdx - 1; i >= 0; i-- {
			w := bs.data[i]
			if w != ^uint64(0) {
				bitIdx := 63 - bits.LeadingZeros64(^w)
				lastZero = i*64 + bitIdx
				break
			}
		}
	}

	if lastZero == -1 {
		// All ones: "11" -> "100"
		newSize := bs.sizeBits + 1
		numWords := (newSize + 63) / 64
		newData := make([]uint64, numWords)
		newData[0] = 1
		return BitString{
			data:     newData,
			sizeBits: newSize,
		}
	}

	// 2. Create copy and apply change
	newData := make([]uint64, len(bs.data))
	copy(newData, bs.data)

	wordIdx := lastZero / 64
	bitIdx := uint32(lastZero % 64)

	// Set bitIdx to 1 and clear all bits to its right in this word (indices > lastZero)
	newData[wordIdx] &= (uint64(1) << bitIdx) - 1
	newData[wordIdx] |= (uint64(1) << bitIdx)

	// Clear all subsequent words
	for i := wordIdx + 1; i < len(newData); i++ {
		newData[i] = 0
	}

	return BitString{
		data:     newData,
		sizeBits: bs.sizeBits,
	}
}

func BugIfNotSortedOrHaveDuplicates(bss []BitString) {
	i := 1
	for i < len(bss) {
		if bss[i-1].Compare(bss[i]) >= 0 {
			errutil.Bug("BitStrings are not sorted")
		}
		i++
	}
}
