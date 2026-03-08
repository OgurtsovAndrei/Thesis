# Future Refactoring Plan: Big-Endian `BitString` for Zero-Copy CGO

## 1. Problem Statement

The current `BitString` implementation uses a **Little-Endian (LE)** bit order stored in a `[]uint64` slice. 
- **Bit 0** corresponds to the least significant bit (LSB) of the first `uint64`.
- **Lexicographical Comparison** requires non-trivial logic using `TrailingZeros64` and bit-masking.
- **CGO Integration** with LeMonHash (and most C++ libraries) is inefficient because C++ `std::string` and `memcmp` expect **byte-lexicographical order**, which corresponds to a **Big-Endian (BE)** bit order (where Bit 0 is the most significant bit of the first byte).

### Current Workaround (CGO Query Path)
To support LeMonHash without changing the whole project, we currently:
1. Copy the `BitString` to a stack buffer (`[32]byte`).
2. Perform an in-place `Reverse8` on every byte to align the bit order with `memcmp`.
3. **Limitation:** This is $O(L)$ where $L$ is key length. It works well for small keys (< 256 bits) but becomes a bottleneck for keys of 1024-4096 bits.

## 2. Proposed Solution: Big-Endian `[]byte` Storage

Transform `BitString` to use a `[]byte` slice as the underlying storage with a **Big-Endian bit order**.

### A. Memory Layout
- **Bit 0:** Most Significant Bit (MSB) of `data[0]`.
- **Bit 1:** `(data[0] >> 6) & 1`.
- **Bit 7:** Least Significant Bit (LSB) of `data[0]`.
- **Bit 8:** MSB of `data[1]`.

### B. Advantages
1. **True Zero-Copy CGO:** A simple pointer cast `(*C.char)(unsafe.Pointer(&ds.data[0]))` will be enough. No copies, no `Reverse8`.
2. **Faster Comparisons:** The `Compare` and `TrieCompare` methods become a simple byte-by-byte loop (or word-by-loop) using `LeadingZeros64`.
3. **Standard Consistency:** Matches the behavior of `bytes.Compare`, `sort.Strings`, and common index formats (RocksDB, etc.).

## 3. Implementation Details

- **`At(i uint32)`**: `(data[i/8] >> (7 - (i%8))) & 1`.
- **`Set(i uint32, val bool)`**: Standard bit manipulation with `7 - (i%8)` shift.
- **`Compare`**:
    ```go
    // Conceptual loop for BE compare
    for i := range data {
        if data[i] != other.data[i] {
            diff := data[i] ^ other.data[i]
            firstDiffBit := mathbits.LeadingZeros8(diff)
            // ... return based on that bit
        }
    }
    ```
- **Arithmetic (`Successor`)**: Needs careful re-implementation as the "carry" now moves from right to left across byte boundaries.

## 4. Risks & Costs

- **Refactoring Scope:** High. Almost every method in `bits/bit_string.go` needs to be updated.
- **Performance Trade-off:** 
    - **Win:** Comparisons, CGO, Prefix/LCP logic.
    - **Loss:** Heavy bitwise arithmetic (addition/subtraction) might become slightly more complex.
- **Verification:** Requires a complete pass of the `bits/dumb_test.go` and `trie/` property tests.

## 5. Conclusion

This refactoring is **required** if the project aims to support learned indexes (like LeMonHash) on **large keys** (1024-4096 bits) with sub-100ns query latency. For current small-key use cases, the stack-copy workaround is sufficient but non-optimal.
