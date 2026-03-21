package bits

// PackBitStrings takes an array of bitstrings and packs their suffixes into a dense []uint64 slice.
// Each value uses exactly 'bitWidth' bits.
func PackBitStrings(values []BitString, bitWidth uint32) []uint64 {
	if len(values) == 0 {
		return nil
	}
	if bitWidth == 0 {
		return []uint64{}
	}

	totalBits := uint64(len(values)) * uint64(bitWidth)
	numWords := (totalBits + 63) / 64
	packed := make([]uint64, numWords)

	for i, bs := range values {
		// Extract last bitWidth bits
		val := extractSuffixAsUint64Local(bs, bitWidth)
		
		bitPos := uint64(i) * uint64(bitWidth)
		wordIdx := bitPos / 64
		bitOffset := uint(bitPos % 64)

		// Write to the first word
		packed[wordIdx] |= val << bitOffset

		// Handle overflow into the next word
		bitsAvailableInWord := 64 - int(bitOffset)
		if bitsAvailableInWord < int(bitWidth) {
			bitsWritten := bitsAvailableInWord
			packed[wordIdx+1] |= val >> uint(bitsWritten)
		}
	}

	return packed
}

// UnpackToBitString extracts a value of 'bitWidth' bits from the packed []uint64 slice
// at the given 'index' and returns it as a BitString.
func UnpackToBitString(packed []uint64, index int, bitWidth uint32) BitString {
	if bitWidth == 0 {
		return BitString{}
	}

	bitPos := uint64(index) * uint64(bitWidth)
	wordIdx := bitPos / 64
	bitOffset := uint(bitPos % 64)

	// Read from the first word
	val := packed[wordIdx] >> bitOffset

	// Check if value spans across two words
	bitsAvailableInWord := 64 - int(bitOffset)
	if bitsAvailableInWord < int(bitWidth) {
		bitsRead := bitsAvailableInWord
		nextWordVal := packed[wordIdx+1]
		val |= nextWordVal << uint(bitsRead)
	}

	// Mask out the extraneous higher bits
	mask := (uint64(1) << bitWidth) - 1
	if bitWidth == 64 {
		mask = ^uint64(0)
	}
	val &= mask

	return NewFromUint64(val).Prefix(int(bitWidth))
}

// UnpackBit extracts a value of 'bitWidth' bits from the packed []uint64 slice
// at the given 'index' (which is the i-th value, not bit position).
func UnpackBit(packed []uint64, index int, bitWidth int) uint64 {
	if bitWidth == 0 {
		return 0
	}

	bitPos := uint64(index) * uint64(bitWidth)
	wordIdx := bitPos / 64
	bitOffset := uint(bitPos % 64)

	// Read from the first word
	val := packed[wordIdx] >> bitOffset

	// Check if value spans across two words
	bitsAvailableInWord := 64 - int(bitOffset)
	if bitsAvailableInWord < bitWidth {
		// Read the remaining bits from the next word
		bitsRead := bitsAvailableInWord
		nextWordVal := packed[wordIdx+1]
		val |= nextWordVal << uint(bitsRead)
	}

	// Mask out the extraneous higher bits
	mask := uint64(1<<bitWidth) - 1
	if bitWidth == 64 {
		mask = ^uint64(0)
	}
	return val & mask
}

func extractSuffixAsUint64Local(bs BitString, bitWidth uint32) uint64 {
	if bitWidth == 0 {
		return 0
	}
	size := bs.Size()
	start := uint32(0)
	if size > bitWidth {
		start = size - bitWidth
	}

	wordIdx := start / 64
	bitOffset := start % 64

	val := bs.data[wordIdx] >> bitOffset
	if bitOffset+bitWidth > 64 && int(wordIdx)+1 < len(bs.data) {
		val |= bs.data[wordIdx+1] << (64 - bitOffset)
	}

	mask := uint64(1<<bitWidth) - 1
	if bitWidth == 64 {
		mask = ^uint64(0)
	}
	return val & mask
}

// Helper to write a value directly to packed data
func SetPackedValue(packed []uint64, index int, bitWidth uint32, val uint64) {
	bitPos := uint64(index) * uint64(bitWidth)
	wordIdx := bitPos / 64
	bitOffset := uint(bitPos % 64)

	mask := (uint64(1) << bitWidth) - 1
	if bitWidth == 64 {
		mask = ^uint64(0)
	}
	val &= mask

	// Clear bits first
	packed[wordIdx] &= ^(mask << bitOffset)
	packed[wordIdx] |= val << bitOffset

	bitsAvailableInWord := 64 - int(bitOffset)
	if bitsAvailableInWord < int(bitWidth) {
		bitsWritten := bitsAvailableInWord
		packed[wordIdx+1] &= ^(mask >> uint(bitsWritten))
		packed[wordIdx+1] |= val >> uint(bitsWritten)
	}
}
