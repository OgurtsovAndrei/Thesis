# BitString Hashing Strategy

This document outlines the performance research and implementation details for `BitString.Hash` and `BitString.HashWithSeed`.

## Benchmarks Summary (Go 1.25, 13th Gen Intel i9)

We compared three main approaches:
1. **Original FNV**: Using standard `hash/fnv` with `New64a()`, `binary.Write`, and `interface` calls.
2. **Manual FNV-1a**: Inlined FNV-1a processing 64-bit words directly with `range` loop (chosen).
3. **XXH3 (Unsafe)**: Using `github.com/zeebo/xxh3` with `unsafe.Slice` to avoid data copies.

### Results (ns/op)

| Size (bits) | Original FNV | Manual FNV-1a | XXH3 (Unsafe) | Improvement |
|-------------|--------------|---------------|---------------|-------------|
| 64          | 6.90         | **0.69**      | 4.17          | ~10x        |
| 1024        | 138.80       | **3.13**      | 8.99          | ~44x        |
| 4096        | 648.50       | **29.45**     | 43.34         | ~22x        |

## Why Manual FNV-1a?

1. **Zero Allocations**: Standard `hash/fnv` requires `New64a()` which allocates on the heap. Manual implementation is stack-only.
2. **Word-wise Processing**: Standard FNV in Go works byte-by-byte. Since `BitString` stores data in `uint64` words, processing 8 bytes at once is significantly faster.
3. **Inlining**: The Go compiler can fully inline the manual loop, eliminating function call overhead.
4. **Bounds Check Elimination (BCE)**: Using `for _, word := range bs.data` allows the compiler to prove that index checks are unnecessary.
5. **Small Data Performance**: While XXH3 is faster for very large buffers (kilobytes/megabytes), FNV-1a wins on small-to-medium keys (up to 4096 bits) due to lower initialization latency.

## Implementation Details

The algorithm used is a standard **FNV-1a** variant adapted for 64-bit words:

```go
h ^= word
h *= prime64
```

For `HashWithSeed`, the seed is XORed with the initial FNV offset to influence the entire hashing chain.
