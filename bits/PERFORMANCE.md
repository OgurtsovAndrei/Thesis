# BitString Performance Strategy

This document outlines the performance research and implementation details for `BitString` operations.

## Hashing Benchmarks (Go 1.25, 13th Gen Intel i9)

We compared three main approaches:
1. **Original FNV**: Using standard `hash/fnv` with `New64a()`, `binary.Write`, and `interface` calls.
2. **Manual FNV-1a**: Inlined FNV-1a processing 64-bit words directly with `range` loop (chosen).
3. **XXH3 (Unsafe)**: Using `github.com/zeebo/xxh3` with `unsafe.Slice` to avoid data copies.

### Hashing Results (ns/op)

| Size (bits) | Original FNV | Manual FNV-1a | XXH3 (Unsafe) | Improvement |
|-------------|--------------|---------------|---------------|-------------|
| 64          | 6.90         | **0.69**      | 4.17          | ~10x        |
| 1024        | 138.80       | **3.13**      | 8.99          | ~44x        |
| 4096        | 648.50       | **29.45**     | 43.34         | ~22x        |

## Comparison Benchmarks

The `Compare` method was optimized by removing redundant checks and using direct word access.

### Comparison Results (ns/op) - Worst Case (Difference at the end)

| Size (bits) | Original | Optimized | Improvement |
|-------------|----------|-----------|-------------|
| 64          | 3.34     | **2.51**  | ~1.3x       |
| 256         | 11.30    | **3.24**  | ~3.5x       |
| 1024        | 33.45    | **7.07**  | ~4.7x       |
| 4096        | 101.60   | **28.03** | ~3.6x       |

## Why These Optimizations?

1. **Zero Allocations**: Manual implementations avoid heap allocations.
2. **Word-wise Processing**: Processing 64-bit words directly is faster than byte-by-byte or bit-by-bit.
3. **Inlining & BCE**: Simple loops with `range` or direct indexing allow the compiler to perform Bounds Check Elimination and full inlining.
4. **Minimal Branching**: We structure loops to exit as early as possible with minimal conditions inside the hot path.

## Implementation Details

### Hashing
Standard **FNV-1a** variant adapted for 64-bit words:
```go
h ^= word
h *= prime64
```

### Comparison
Find the first differing word, then use `bits.TrailingZeros64` to identify the first differing bit.
