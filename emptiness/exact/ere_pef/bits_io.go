package ere_pef

// readBits reads `width` bits from `packed` starting at absolute bit
// position `bitPos`. Width must be in (0, 64]. Caller guarantees the
// read does not overflow `packed`.
func readBits(packed []uint64, bitPos uint64, width uint8) uint64 {
	wordIdx := bitPos / 64
	bitOffset := uint(bitPos % 64)
	val := packed[wordIdx] >> bitOffset
	if 64-int(bitOffset) < int(width) {
		val |= packed[wordIdx+1] << uint(64-int(bitOffset))
	}
	if width < 64 {
		val &= (uint64(1) << width) - 1
	}
	return val
}

// writeBits writes `value` (`width` low bits) into `packed` at absolute
// bit position `bitPos`. Caller must ensure `packed` already has enough
// words (see ensureBitCapacity).
func writeBits(packed []uint64, bitPos uint64, width uint8, value uint64) {
	wordIdx := bitPos / 64
	bitOffset := uint(bitPos % 64)
	packed[wordIdx] |= value << bitOffset
	if 64-int(bitOffset) < int(width) {
		packed[wordIdx+1] |= value >> uint(64-int(bitOffset))
	}
}

// ensureBitCapacity grows `packed` (zero-extended) so bit position
// `bitPos + width - 1` is valid. Returns the (possibly reallocated) slice.
func ensureBitCapacity(packed []uint64, bitPos uint64, width uint8) []uint64 {
	if width == 0 {
		return packed
	}
	needed := (bitPos + uint64(width) + 63) / 64
	for uint64(len(packed)) < needed {
		packed = append(packed, 0)
	}
	return packed
}
