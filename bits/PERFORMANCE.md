# BitString Performance Strategy

This document outlines the performance research and implementation details for `BitString` operations.

## Prefix Search Benchmarks (O(1) Refactor)

The most significant optimization in the latest version is making `Prefix` an $O(1)$ operation by relaxing the zero-tail invariant. Previously, taking a prefix with a length not aligned to 64 bits required a new allocation and bitwise masking.

### Prefix Results (ns/op, 0 Allocs)

| Size (max bits) | Old (O(N) + Allocs) | **New (O(1) No Allocs)** | Improvement |
|-----------------|---------------------|--------------------------|-------------|
| 64              | 14.28               | **2.62**                 | ~5.4x       |
| 256             | 21.05               | **3.65**                 | ~5.7x       |
| 1024            | 26.21               | **3.64**                 | ~7.2x       |
| 4096            | 45.33               | **4.21**                 | **~10.7x**  |

*Note: The new implementation is entirely allocation-free for all sizes.*

## Hashing Benchmarks (Go 1.25, 13th Gen Intel i9 P-Core 0)

We use a manual **FNV-1a** implementation. After relaxing the zero-tail invariant, we added mandatory masking for the final word to ensure "junk bits" don't affect the hash.

### Hashing Results (ns/op)

| Size (bits) | Manual FNV-1a (Clean Tail) | **Manual FNV-1a (With Masking)** | Difference |
|-------------|----------------------------|-----------------------------------|------------|
| 64          | 0.81                       | **0.87**                          | +7%        |
| 1024        | 3.22                       | **3.80**                          | +18%       |
| 4096        | 23.10                      | **29.70**                         | +28%       |

*The slight trade-off in hashing speed is heavily offset by the $O(1)$ Prefix performance and zero allocations in tree traversals.*

## Comparison Benchmarks

`Compare` and `Equal` now explicitly ignore bits beyond `sizeBits` by masking the final 64-bit word.

### Comparison Results (ns/op) - 4096 bits

| Method        | Before (Clean Tail) | **After (With Masking)** | Difference |
|---------------|---------------------|--------------------------|------------|
| **Equal**     | 32.50               | **36.20**                | +11%       |
| **Compare**   | 35.20               | **36.50**                | +3%        |

## Why These Optimizations?

1. **Zero Allocations**: `Prefix` no longer triggers heap allocations regardless of length. This drastically reduces GC pressure in hot paths like Trie traversal.
2. **Logical vs Physical Length**: By decoupling `sizeBits` from the underlying array's zero-state, we enable $O(1)$ slicing similar to Go's native slices.
3. **Word-wise Processing**: Even with masking, we process 64 bits at a time. The CPU overhead of a single `AND` operation on the final word is negligible compared to a memory allocation.

## Implementation Details

### Prefixing
`Prefix(n)` simply returns a new `BitString` header pointing to the same underlying `data` slice, but with a reduced `sizeBits` and a sub-sliced `data` array:
```go
return BitString{
    data:     bs.data[:numWords],
    sizeBits: uint32(size),
}
```

### Masking for Junk Bits
In all terminal operations (Hash, Compare, Equal, Data), we apply a mask to the final word:
```go
if bs.sizeBits % 64 != 0 {
    mask := (uint64(1) << (bs.sizeBits % 64)) - 1
    word &= mask
}
```
This ensures that any bits existing in the underlying array beyond `sizeBits` (the "junk") are ignored.
