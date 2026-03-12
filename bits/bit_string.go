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

func NewFromUint64WithLength(value uint64, sizeBits uint32) BitString {
	bs := NewBitString(sizeBits)
	bs.data[0] = value
	if sizeBits < 64 {
		bs.data[0] &= (uint64(1) << sizeBits) - 1
	}
	return bs
}

// Sub subtracts other from bs using trie-consistent arithmetic.
// Bit 0 is the most significant digit (consistent with Compare ordering).
// Assumes bs >= other by Compare and same size.
func (bs BitString) Sub(other BitString) BitString {
	if bs.sizeBits != other.sizeBits {
		panic("BitString sizes must match for subtraction")
	}
	result := NewBitString(bs.sizeBits)
	var borrow uint64
	for i := len(bs.data) - 1; i >= 0; i-- {
		v1 := bits.Reverse64(bs.data[i])
		v2 := bits.Reverse64(other.data[i])
		diff, nextBorrow := bits.Sub64(v1, v2, borrow)
		result.data[i] = bits.Reverse64(diff)
		borrow = nextBorrow
	}
	return result
}

// Add adds other to bs using trie-consistent arithmetic.
// Bit 0 is the most significant digit (consistent with Compare ordering).
// Assumes same size.
func (bs BitString) Add(other BitString) BitString {
	if bs.sizeBits != other.sizeBits {
		panic("BitString sizes must match for addition")
	}
	result := NewBitString(bs.sizeBits)
	var carry uint64
	for i := len(bs.data) - 1; i >= 0; i-- {
		v1 := bits.Reverse64(bs.data[i])
		v2 := bits.Reverse64(other.data[i])
		sum, nextCarry := bits.Add64(v1, v2, carry)
		result.data[i] = bits.Reverse64(sum)
		carry = nextCarry
	}
	return result
}

// TrieUint64 interprets the BitString as an integer where bit 0 is the MSB.
// The returned value is ordered consistently with Compare.
// Only valid for BitStrings with SizeBits() <= 64.
func (bs BitString) TrieUint64() uint64 {
	if bs.sizeBits == 0 {
		return 0
	}
	return bits.Reverse64(bs.Word(0)) >> (64 - bs.sizeBits)
}

// NewFromTrieUint64 creates a K-bit BitString from an integer value
// where the MSB of val corresponds to bit 0 of the BitString.
// Compare ordering on the result matches numeric ordering of val.
func NewFromTrieUint64(val uint64, K uint32) BitString {
	if K == 0 {
		return BitString{}
	}
	word := bits.Reverse64(val << (64 - K))
	return NewFromUint64WithLength(word, K)
}

// Suffix returns the last k bits of the BitString (the least significant trie characters).
func (bs BitString) Suffix(k uint32) BitString {
	if k >= bs.sizeBits {
		return bs
	}
	return bs.ShiftRight(bs.sizeBits - k)
}

func (bs BitString) ShiftRight(t uint32) BitString {
	if t == 0 {
		return bs
	}
	if t >= bs.sizeBits {
		return NewBitString(bs.sizeBits)
	}

	newSize := bs.sizeBits - t
	result := NewBitString(newSize)
	
	wordShift := t / 64
	bitShift := t % 64
	
	for i := uint32(0); i < uint32(len(result.data)); i++ {
		srcIdx := i + wordShift
		if srcIdx < uint32(len(bs.data)) {
			val := bs.data[srcIdx] >> bitShift
			if bitShift > 0 && srcIdx+1 < uint32(len(bs.data)) {
				val |= bs.data[srcIdx+1] << (64 - bitShift)
			}
			result.data[i] = val
		}
	}
	
	// Mask the last word
	if newSize % 64 != 0 {
		result.data[len(result.data)-1] &= (uint64(1) << (newSize % 64)) - 1
	}

	return result
}

func (bs BitString) BitLength() uint32 {
	if bs.sizeBits == 0 {
		return 0
	}
	for i := int(len(bs.data)) - 1; i >= 0; i-- {
		w := bs.data[i]
		// Handle masking for the last word
		if i == int(len(bs.data)-1) && bs.sizeBits%64 != 0 {
			mask := (uint64(1) << (bs.sizeBits % 64)) - 1
			w &= mask
		}
		if w != 0 {
			return uint32(i*64) + uint32(64-bits.LeadingZeros64(w))
		}
	}
	return 0
}

func (bs BitString) Word(i uint32) uint64 {
	if i >= uint32(len(bs.data)) {
		return 0
	}
	word := bs.data[i]
	// Handle masking for the last word
	if i == uint32(len(bs.data)-1) && bs.sizeBits%64 != 0 {
		mask := (uint64(1) << (bs.sizeBits % 64)) - 1
		word &= mask
	}
	return word
}

func (bs BitString) SizeBits() uint32 {
	return bs.sizeBits
}

// NewFromUint64 creates a BitString from a single uint64 value.
func NewFromUint64(value uint64) BitString {
	bs := NewBitString(64)
	bs.data[0] = value
	return bs
}

// NewFromBinary creates a BitString from a binary string (e.g., "1011").
func NewFromBinary(text string) BitString {
	size := len(text)
	if size == 0 {
		return BitString{}
	}

	// Fast validation
	for i := 0; i < size; i++ {
		r := text[i]
		if r != '0' && r != '1' {
			errutil.Bug("invalid string format, %q", text)
		}
	}

	numWords := (uint32(size) + 63) / 64
	data := make([]uint64, numWords)

	for i := 0; i < size; i++ {
		if text[i] == '1' {
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

	fullWords := bs.sizeBits / 64
	for i := uint32(0); i < fullWords; i++ {
		if bs.data[i] != other.data[i] {
			return false
		}
	}

	if bs.sizeBits%64 != 0 {
		mask := (uint64(1) << (bs.sizeBits % 64)) - 1
		if (bs.data[fullWords] & mask) != (other.data[fullWords] & mask) {
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
		if bs.sizeBits%64 != 0 {
			mask := (uint64(1) << (bs.sizeBits % 64)) - 1
			lastWord &= mask
		}
		offset := fullWords * 8
		remainingBytes := numBytes % 8
		for j := uint32(0); j < remainingBytes; j++ {
			result[offset+j] = byte(lastWord >> (j * 8))
		}
	}

	return result
}

// AppendToBytes appends the exact byte representation of the BitString to the given buffer.
// It masks out any garbage bits beyond sizeBits. This is allocation-free if the buffer has enough capacity.
func (bs BitString) AppendToBytes(buf []byte) []byte {
	if bs.sizeBits == 0 {
		return buf
	}

	numBytes := (bs.sizeBits + 7) / 8
	
	// Ensure buffer has capacity
	if uint32(cap(buf)-len(buf)) < numBytes {
		newBuf := make([]byte, len(buf), len(buf)+int(numBytes))
		copy(newBuf, buf)
		buf = newBuf
	}
	
	start := len(buf)
	buf = buf[:start+int(numBytes)]

	fullWords := numBytes / 8
	for i := uint32(0); i < fullWords; i++ {
		binary.LittleEndian.PutUint64(buf[start+int(i*8):], bs.data[i])
	}

	if numBytes%8 != 0 {
		lastWord := bs.data[fullWords]
		if bs.sizeBits%64 != 0 {
			mask := (uint64(1) << (bs.sizeBits % 64)) - 1
			lastWord &= mask
		}
		offset := start + int(fullWords*8)
		remainingBytes := numBytes % 8
		for j := uint32(0); j < remainingBytes; j++ {
			buf[offset+int(j)] = byte(lastWord >> (j * 8))
		}
	}

	return buf
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

	numWords := (bs.sizeBits + 63) / 64
	for i := uint32(0); i < numWords; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		word := bs.data[i]
		if i == numWords-1 && bs.sizeBits%64 != 0 {
			mask := (uint64(1) << (bs.sizeBits % 64)) - 1
			word &= mask
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

	fullWords := prefixSize / 64
	for i := uint32(0); i < fullWords; i++ {
		if bs.data[i] != prefixToCheck.data[i] {
			return false
		}
	}

	if prefixSize%64 != 0 {
		mask := (uint64(1) << (prefixSize % 64)) - 1
		if (bs.data[fullWords] & mask) != (prefixToCheck.data[fullWords] & mask) {
			return false
		}
	}

	return true
}

func (bs BitString) Prefix(size int) BitString {
	if size <= 0 {
		return BitString{}
	}
	if int(bs.sizeBits) <= size {
		return bs
	}

	numWords := (uint32(size) + 63) / 64
	return BitString{
		data:     bs.data[:numWords],
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

	fullWords := bs.sizeBits / 64
	for i := uint32(0); i < fullWords; i++ {
		h ^= bs.data[i]
		h *= prime64
	}

	if bs.sizeBits%64 != 0 {
		mask := (uint64(1) << (bs.sizeBits % 64)) - 1
		h ^= (bs.data[fullWords] & mask)
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

	fullWords := bs.sizeBits / 64
	for i := uint32(0); i < fullWords; i++ {
		h ^= bs.data[i]
		h *= prime64
	}

	if bs.sizeBits%64 != 0 {
		mask := (uint64(1) << (bs.sizeBits % 64)) - 1
		h ^= (bs.data[fullWords] & mask)
		h *= prime64
	}

	return h
}

// Compare performs lexicographic comparison. Returns:
// -1 if bs < other
//
//	1 if bs > other
//	0 if bs == other
//
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
			diffBitInWord := uint32(bits.TrailingZeros64(xor))
			if (aWord & (uint64(1) << diffBitInWord)) != 0 {
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
			diffBitInWord := uint32(bits.TrailingZeros64(xor))
			if diffBitInWord < minSize%64 {
				if (aWord & (uint64(1) << diffBitInWord)) != 0 {
					return 1
				}
				return -1
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

	lastWordIdx := int32((bs.sizeBits - 1) / 64)
	lastOneBit := int32(-1)

	for i := lastWordIdx; i >= 0; i-- {
		word := bs.data[i]
		if i == lastWordIdx && bs.sizeBits%64 != 0 {
			mask := (uint64(1) << (bs.sizeBits % 64)) - 1
			word &= mask
		}

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

	newData := make([]uint64, newNumWords)
	copyWords := uint32(len(bs.data))
	if copyWords > newNumWords {
		copyWords = newNumWords
	}
	copy(newData, bs.data[:copyWords])

	if bs.sizeBits%64 != 0 {
		mask := (uint64(1) << (bs.sizeBits % 64)) - 1
		newData[bs.sizeBits/64] &= mask
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
	for i := uint32(0); i < fullWords; i++ {
		if bs.data[i] != ^uint64(0) {
			return false
		}
	}

	if bs.sizeBits%64 != 0 {
		mask := (uint64(1) << (bs.sizeBits % 64)) - 1
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
	numWords := (bs.sizeBits + 63) / 64
	newData := make([]uint64, numWords)
	copy(newData, bs.data[:numWords])

	if bs.sizeBits%64 != 0 {
		mask := (uint64(1) << (bs.sizeBits % 64)) - 1
		newData[numWords-1] &= mask
	}

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

func (bs BitString) Predecessor() BitString {
	if bs.sizeBits == 0 || bs.IsAllZeros() {
		return bs
	}

	lastWordIdx := int(bs.sizeBits-1) / 64

	// 1. Find the last '1' (highest index) using word-level scan
	lastOne := -1
	{
		w := bs.data[lastWordIdx]
		bitsInLastWord := (bs.sizeBits-1)%64 + 1
		mask := ^uint64(0)
		if bitsInLastWord < 64 {
			mask = (uint64(1) << bitsInLastWord) - 1
		}
		ones := w & mask
		if ones != 0 {
			bitIdx := 63 - bits.LeadingZeros64(ones)
			lastOne = lastWordIdx*64 + bitIdx
		}
	}
	if lastOne == -1 {
		for i := lastWordIdx - 1; i >= 0; i-- {
			if bs.data[i] != 0 {
				bitIdx := 63 - bits.LeadingZeros64(bs.data[i])
				lastOne = i*64 + bitIdx
				break
			}
		}
	}
	if lastOne == -1 {
		return bs
	}

	// 2. Create copy and mask
	numWords := (bs.sizeBits + 63) / 64
	newData := make([]uint64, numWords)
	copy(newData, bs.data[:numWords])

	if bs.sizeBits%64 != 0 {
		mask := (uint64(1) << (bs.sizeBits % 64)) - 1
		newData[numWords-1] &= mask
	}

	// 3. Clear bit lastOne, set all bits after it to 1
	wordIdx := lastOne / 64
	bitIdx := uint32(lastOne % 64)

	// In the word containing lastOne: clear lastOne, set all higher bits to 1
	newData[wordIdx] |= ^((uint64(1) << bitIdx) - 1) // set bits >= bitIdx to 1
	newData[wordIdx] ^= uint64(1) << bitIdx           // clear bitIdx (was just set to 1)

	// Set all subsequent words to all-ones
	for i := wordIdx + 1; i < len(newData); i++ {
		newData[i] = ^uint64(0)
	}

	// Mask the last word to sizeBits
	if bs.sizeBits%64 != 0 {
		mask := (uint64(1) << (bs.sizeBits % 64)) - 1
		newData[numWords-1] &= mask
	}

	return BitString{
		data:     newData,
		sizeBits: bs.sizeBits,
	}
}

func (bs BitString) IsAllZeros() bool {
	if bs.sizeBits == 0 {
		return true
	}
	fullWords := bs.sizeBits / 64
	for i := uint32(0); i < fullWords; i++ {
		if bs.data[i] != 0 {
			return false
		}
	}
	if bs.sizeBits%64 != 0 {
		mask := (uint64(1) << (bs.sizeBits % 64)) - 1
		if bs.data[fullWords]&mask != 0 {
			return false
		}
	}
	return true
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
