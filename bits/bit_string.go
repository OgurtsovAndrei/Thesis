package bits

import (
	"Thesis/errutil"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math/bits"
	"strconv"
	"strings"

	"github.com/zeebo/xxh3"
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

	for i := uint32(0); i < numBytes; i++ {
		wordIndex := i / 8
		byteOffsetInWord := i % 8
		if wordIndex < uint32(len(bs.data)) {
			result[i] = byte(bs.data[wordIndex] >> (byteOffsetInWord * 8))
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

	minWords := (minLengthBits + 63) / 64

	for i := uint32(0); i < minWords; i++ {
		wordA := uint64(0)
		wordB := uint64(0)

		if i < uint32(len(bs.data)) {
			wordA = bs.data[i]
		}
		if i < uint32(len(other.data)) {
			wordB = other.data[i]
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
	if size == 0 {
		return BitString{}
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

	return BitString{
		data:     newData,
		sizeBits: uint32(size),
	}
}

func (bs BitString) Hash() uint64 {
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

func (bs BitString) HashWithSeed(seed uint64) uint64 {
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

func (bs BitString) Compare(other BitString) int {
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

	minWords := (minSize + 63) / 64

	for i := uint32(0); i < minWords; i++ {
		aWord := uint64(0)
		bWord := uint64(0)

		if i < uint32(len(bs.data)) {
			aWord = bs.data[i]
		}
		if i < uint32(len(other.data)) {
			bWord = other.data[i]
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

	// Standard lexicographic comparison: shorter < longer when prefixes match
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

	minWords := (minSize + 63) / 64

	for i := uint32(0); i < minWords; i++ {
		aWord := uint64(0)
		bWord := uint64(0)

		if i < uint32(len(bs.data)) {
			aWord = bs.data[i]
		}
		if i < uint32(len(other.data)) {
			bWord = other.data[i]
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

	// Handle different sizes - in trie in-order traversal:
	// - Left children (extension starts with 0) come before parent
	// - Parent comes in the middle
	// - Right children (extension starts with 1) come after parent
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

	// Big-Endian increment: start from the last bit (index sizeBits-1)
	// If it's 0, set to 1 and we're done.
	// If it's 1, set to 0 and carry over to the left.

	// Find the rightmost zero bit
	lastZero := int32(-1)
	for i := int32(bs.sizeBits) - 1; i >= 0; i-- {
		if !bs.At(uint32(i)) {
			lastZero = i
			break
		}
	}

	if lastZero == -1 {
		// All bits are 1 (e.g., "11" -> "100")
		// New size is current size + 1. New bit 0 is 1, rest are 0.
		newSize := bs.sizeBits + 1
		result := NewBitString(newSize)
		// Set bit at index 0 to 1
		result.data[0] |= 1
		return result
	}

	// Create a copy and increment
	result := NewBitString(bs.sizeBits)
	copy(result.data, bs.data)

	// Set the last zero to 1
	wordIdx := uint32(lastZero) / 64
	bitIdx := uint32(lastZero) % 64
	result.data[wordIdx] |= uint64(1) << bitIdx

	// Set all bits to the right of it to 0
	for i := uint32(lastZero) + 1; i < bs.sizeBits; i++ {
		wIdx := i / 64
		bIdx := i % 64
		result.data[wIdx] &= ^(uint64(1) << bIdx)
	}

	return result
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

// BitMap is a high-performance map using BitString as keys.
// It uses an underlying map of hashes to handle collisions via slices of entries.
type BitMap[V any] struct {
	m     map[uint64][]entry[V]
	count int
}

type entry[V any] struct {
	key   BitString
	value V
}

func NewBitMap[V any]() *BitMap[V] {
	return &BitMap[V]{
		m: make(map[uint64][]entry[V]),
	}
}

func (bm *BitMap[V]) Put(key BitString, value V) {
	h := key.Hash()
	entries := bm.m[h]
	for i := range entries {
		if entries[i].key.Equal(key) {
			entries[i].value = value
			return
		}
	}
	bm.m[h] = append(entries, entry[V]{key, value})
	bm.count++
}

func (bm *BitMap[V]) Get(key BitString) (V, bool) {
	h := key.Hash()
	entries := bm.m[h]
	for i := range entries {
		if entries[i].key.Equal(key) {
			return entries[i].value, true
		}
	}
	var zero V
	return zero, false
}

func (bm *BitMap[V]) Delete(key BitString) {
	h := key.Hash()
	entries := bm.m[h]
	for i := range entries {
		if entries[i].key.Equal(key) {
			bm.m[h] = append(entries[:i], entries[i+1:]...)
			if len(bm.m[h]) == 0 {
				delete(bm.m, h)
			}
			bm.count--
			return
		}
	}
}

func (bm *BitMap[V]) Len() int {
	return bm.count
}

func (bm *BitMap[V]) Range(f func(key BitString, value V) bool) {
	for _, entries := range bm.m {
		for _, e := range entries {
			if !f(e.key, e.value) {
				return
			}
		}
	}
}
