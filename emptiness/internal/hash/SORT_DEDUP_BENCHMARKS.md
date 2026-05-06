# Sort+Dedup Benchmarks — uint64 in [0, 2^K)

Benchmarking seven sort-and-deduplicate strategies for the SODA-style ARE
build path, where input keys are near-uniformly distributed in `[0, 2^K)`
after pairwise hashing.

- Hardware: Apple M4 Max, 16 cores, `darwin/arm64`, Go 1.25.
- `-benchtime=1x`. Inputs: pseudo-random `uint64` masked to `K` bits.
- Per-iteration `make+copy` excluded from `ns/op` via `b.StopTimer`.

## Variants

| Name | Strategy | Scratch beyond keys |
|---|---|---|
| `sortSlice` | `sort.Slice` + linear dedup. Current baseline. | 56 B closure (2 allocs) |
| `slicesSort` | `slices.Sort` (pdqsort) + linear dedup. Drop-in. | **0 B / 0 allocs** |
| `radixFull` | 8-pass LSD radix (`dgryski/go-radixsort`) + dedup. | `8n` B (1 alloc) |
| `radixK` | K-aware LSD radix: `⌈K/8⌉` passes + dedup. | `8n` B (1 alloc) |
| `bitmap` | Single `2^K`-bit bitmap; sort-free; refuses K>32 (falls back to `slicesSort`). | `2^(K−3)` B (1 alloc) for K ≤ 32 |
| **`americanFlag`** *(new)* | K-aware MSD radix, in-place via American Flag cycles + dedup. | **0 B / 0 allocs** (only 256-int stack arrays per recursion) |
| **`msdBitmap`** *(new)* | One in-place 8-bit MSD partition (American Flag) + per-bucket bitmap dedup using a single reused `2^(K−8)`-bit bitmap. Per-bucket sparse-fallback to `slices.Sort`. | `2^(K−11)` B (1 alloc), independent of n |
| **`adaptive`** *(new)* | Dispatcher: `K ≤ 28` → `bitmap`; `29 ≤ K ≤ 36` → `msdBitmap`; `K ≥ 37` → `americanFlag`. | Inherits chosen variant's allocation. |

## Timing — full matrix, **ms/op**

### n = 2²⁰ (1 048 576 keys)

| Variant       |    K=20 |    K=24 |    K=28 |    K=32 |    K=36 |    K=40 |
|---------------|--------:|--------:|--------:|--------:|--------:|--------:|
| sortSlice     |   115.2 |   110.4 |   108.7 |   114.2 |   110.7 |   102.8 |
| slicesSort    |    68.1 |    65.7 |    67.1 |    70.7 |    67.7 |    62.0 |
| radixFull     |    19.6 |    16.9 |    16.0 |    16.4 |    14.9 |    12.9 |
| **radixK**    | **6.3** | **3.8** | **5.2** | **5.2** | **6.4** | **5.8** |
| bitmap        | **1.0** | **2.7** |    11.8 |    75.2 |   68.7¹ |   63.6¹ |
| americanFlag  |    19.0 |    22.7 |    22.1 |    24.2 |    22.0 |    20.8 |
| msdBitmap     |     7.1 |     7.7 |    14.9 |    44.4 |    51.4 |    61.0 |

### n = 2²² (4 194 304 keys)

| Variant       |    K=20 |    K=24 |     K=28 |  K=32 |     K=36 |     K=40 |
|---------------|--------:|--------:|---------:|------:|---------:|---------:|
| sortSlice     |   439.4 |   481.7 |    448.1 | 478.6 |    457.8 |    443.7 |
| slicesSort    |   301.0 |   293.1 |    298.4 | 293.6 |    300.3 |    268.4 |
| radixFull     |    76.8 |    56.9 |     65.5 |  59.5 |     58.0 |     60.4 |
| **radixK**    |    22.0 |    16.3 |     20.5 |  17.4 | **23.8** | **24.7** |
| bitmap        | **2.8** | **6.7** | **27.0** |  95.0 |   305.3¹ |   300.8¹ |
| americanFlag  |    65.3 |    82.6 |     86.9 |  94.8 |     89.0 |    106.9 |
| msdBitmap     |    28.0 |    26.9 |     50.3 |  68.1 |    225.7 |    244.1 |

### n = 2²⁴ (16 777 216 keys)

| Variant       |    K=20 |     K=24 |     K=28 |  K=32 |      K=36 |      K=40 |
|---------------|--------:|---------:|---------:|------:|----------:|----------:|
| sortSlice     |   1 781 |    2 063 |    2 043 | 2 040 |     2 024 |     2 030 |
| slicesSort    |   1 097 |    1 309 |    1 263 | 1 240 |     1 238 |     1 258 |
| radixFull     |     240 |      295 |      230 |   261 |       242 |       242 |
| **radixK**    |    69.2 |     98.1 |     93.7 |  91.1 | **109.6** | **108.2** |
| bitmap        | **8.0** | **21.5** | **56.2** | 219.8 |    1 243¹ |    1 245¹ |
| americanFlag  |     240 |      336 |      466 |   397 |       446 |       364 |
| msdBitmap     |     129 |      132 |      183 |   262 |       787 |       978 |

### n = 2²⁶ (67 108 864 keys)

| Variant       |   K=20 |   K=24 |    K=28 |  K=32 |    K=36 |    K=40 |
|---------------|-------:|-------:|--------:|------:|--------:|--------:|
| sortSlice     |  7 046 |  8 712 |   8 818 | 8 765 |   8 782 |   8 590 |
| slicesSort    |  4 242 |  5 342 |   5 368 | 5 262 |   5 363 |   5 315 |
| radixFull     |    963 |  1 115 |   1 079 | 1 041 |     992 |     962 |
| **radixK**    |    331 |    414 |     418 |   410 | **528** | **517** |
| bitmap        | **33** | **64** | **139** |   562 |  5 311¹ |  5 276¹ |
| americanFlag  |  1 004 |  1 395 |   1 749 | 1 719 |   1 909 |   1 615 |
| msdBitmap     |    655 |    628 |     740 | 1 097 |   1 424 |   4 355 |

### n = 2²⁸ (268 435 456 keys)

| Variant       |    K=20 |    K=24 |    K=28 |   K=32 |      K=36 |      K=40 |
|---------------|--------:|--------:|--------:|-------:|----------:|----------:|
| sortSlice     |  26 810 |  33 372 |  37 650 | 36 869 |    36 935 |    37 281 |
| slicesSort    |  16 155 |  20 285 |  23 208 | 22 599 |    22 577 |    22 992 |
| radixFull     |   4 231 |   4 073 |   4 469 |  3 992 |     3 643 |     3 699 |
| **radixK**    |   1 174 |   1 376 |   2 131 |  1 634 | **1 899** | **2 011** |
| bitmap        | **121** | **228** | **510** |  1 465 |   22 635¹ |   22 921¹ |
| americanFlag  |   4 200 |   5 428 |   6 286 |  8 303 |     7 161 |     8 386 |
| msdBitmap     |   2 948 |   2 764 |   2 922 |  3 102 |     5 217 |    15 258 |

¹ Bitmap above K=32 falls back internally to `slicesSort` — refused
allocation for `2^K/8` ≥ 1 GiB. Time matches `slicesSort`.

## Memory — full matrix, **B/op** (human-readable)

Same shape as the timing matrices: one table per n. Cells are *bytes
allocated per call*, formatted as B / KiB / MiB / GiB.

### n = 2²⁰ (1 048 576 keys)

| Variant       |       K=20 |       K=24 |        K=28 |          K=32 |          K=36 |          K=40 |
|---------------|-----------:|-----------:|------------:|--------------:|--------------:|--------------:|
| sortSlice     |       56 B |       56 B |        56 B |          56 B |          88 B |          56 B |
| slicesSort    |      **0** |      **0** |       **0** |         **0** |         **0** |         **0** |
| radixFull     |      8 MiB |      8 MiB |       8 MiB |         8 MiB |         8 MiB |         8 MiB |
| radixK        |      8 MiB |      8 MiB |       8 MiB |         8 MiB |         8 MiB |         8 MiB |
| bitmap        |    128 KiB |      2 MiB |      32 MiB |       512 MiB |     0 (fb)    |     0 (fb)    |
| americanFlag  |    **16 B** |     **0** |       **0** |         **0** |         **0** |         **0** |
| msdBitmap     |    **512 B** |  **8 KiB** |  **128 KiB** |     **2 MiB** |    **32 MiB** |   **512 MiB** |

### n = 2²² (4 194 304 keys)

| Variant       |       K=20 |       K=24 |        K=28 |          K=32 |          K=36 |          K=40 |
|---------------|-----------:|-----------:|------------:|--------------:|--------------:|--------------:|
| sortSlice     |       56 B |       56 B |        56 B |          56 B |          56 B |          56 B |
| slicesSort    |      **0** |      **0** |       **0** |         **0** |         **0** |         **0** |
| radixFull     |     32 MiB |     32 MiB |      32 MiB |        32 MiB |        32 MiB |        32 MiB |
| radixK        |     32 MiB |     32 MiB |      32 MiB |        32 MiB |        32 MiB |        32 MiB |
| bitmap        |    128 KiB |      2 MiB |      32 MiB |       512 MiB |     0 (fb)    |     0 (fb)    |
| americanFlag  |      **0** |      **0** |       **0** |         **0** |         **0** |         **0** |
| msdBitmap     |    **512 B** |  **8 KiB** |  **128 KiB** |     **2 MiB** |    **32 MiB** |   **512 MiB** |

### n = 2²⁴ (16 777 216 keys)

| Variant       |       K=20 |       K=24 |        K=28 |          K=32 |          K=36 |          K=40 |
|---------------|-----------:|-----------:|------------:|--------------:|--------------:|--------------:|
| sortSlice     |       56 B |       56 B |        56 B |          56 B |          56 B |         152 B |
| slicesSort    |      96 B |      **0** |       **0** |         **0** |         **0** |         **0** |
| radixFull     |    128 MiB |    128 MiB |     128 MiB |       128 MiB |       128 MiB |       128 MiB |
| radixK        |    128 MiB |    128 MiB |     128 MiB |       128 MiB |       128 MiB |       128 MiB |
| bitmap        |    128 KiB |      2 MiB |      32 MiB |       512 MiB |     0 (fb)    |     0 (fb)    |
| americanFlag  |      **0** |      **0** |       **0** |         **0** |         **0** |         **0** |
| msdBitmap     |    **512 B** |  **8 KiB** |  **128 KiB** |     **2 MiB** |    **32 MiB** |   **512 MiB** |

### n = 2²⁶ (67 108 864 keys)

| Variant       |       K=20 |       K=24 |        K=28 |          K=32 |          K=36 |          K=40 |
|---------------|-----------:|-----------:|------------:|--------------:|--------------:|--------------:|
| sortSlice     |       56 B |       56 B |        56 B |          56 B |          56 B |          56 B |
| slicesSort    |      **0** |      **0** |       **0** |         **0** |         **0** |         **0** |
| radixFull     |    512 MiB |    512 MiB |     512 MiB |       512 MiB |       512 MiB |       512 MiB |
| radixK        |    512 MiB |    512 MiB |     512 MiB |       512 MiB |       512 MiB |       512 MiB |
| bitmap        |    128 KiB |      2 MiB |      32 MiB |       512 MiB |     0 (fb)    |     0 (fb)    |
| americanFlag  |      **0** |      96 B |       **0** |         **0** |         **0** |         **0** |
| msdBitmap     |    **512 B** |  **8 KiB** |  **128 KiB** |     **2 MiB** |    **32 MiB** |   **512 MiB** |

### n = 2²⁸ (268 435 456 keys)

| Variant       |       K=20 |       K=24 |        K=28 |          K=32 |          K=36 |          K=40 |
|---------------|-----------:|-----------:|------------:|--------------:|--------------:|--------------:|
| sortSlice     |       56 B |       56 B |        56 B |          56 B |          56 B |          56 B |
| slicesSort    |      **0** |      **0** |       **0** |         **0** |         **0** |         **0** |
| radixFull     |      2 GiB |      2 GiB |       2 GiB |         2 GiB |         2 GiB |         2 GiB |
| radixK        |      2 GiB |      2 GiB |       2 GiB |         2 GiB |         2 GiB |         2 GiB |
| bitmap        |    128 KiB |      2 MiB |      32 MiB |       512 MiB |     0 (fb)    |     0 (fb)    |
| americanFlag  |      **0** |      **0** |       **0** |         **0** |         **0** |         **0** |
| msdBitmap     |    **512 B** |  **8 KiB** |  **128 KiB** |     **2 MiB** |    **32 MiB** |   **512 MiB** |

`(fb)` = bitmap above K=32 falls back to `slicesSort`, which allocates
nothing.

### Allocation count (`allocs/op`) — independent of n and K

| Variant | allocs/op |
|---|---:|
| sortSlice | **2** (closure box inside `sort.Slice`) |
| slicesSort | **0** |
| radixFull | **1** (scratch) |
| radixK | **1** (scratch) |
| bitmap (K ≤ 32) | **1** (the bitmap) |
| bitmap (K ≥ 36) | **0** (fallback path is in-place) |
| **americanFlag** | **0** (truly in-place) |
| **msdBitmap** | **1** (single reused inner bitmap) |

### Allocation rule, in one line

| Variant | Rule | Grows with |
|---|---|---|
| sortSlice | const 56 B | — |
| slicesSort | none | — |
| radixFull / radixK | `8 · n` B | **n only** |
| bitmap (K ≤ 32) | `2^(K−3)` B | **K only** |
| **americanFlag** | none | — |
| **msdBitmap** | `2^(K−11)` B | **K only** |

Three things are immediately visible from the matrices:

1. `radixK` / `radixFull` allocations grow column-uniform (only with n) —
   the scratch is independent of the key bitwidth.
2. `bitmap` and `msdBitmap` allocations grow row-uniform (only with K) —
   the input size doesn't matter.
3. `msdBitmap` is **8× smaller** than `bitmap` at every K where both work
   (the inner bitmap addresses only `2^(K−8)` slots vs the full `2^K`).
   At n=2²⁸, K=32 this is 2 MiB vs 512 MiB.

## In-place vs 2× memory — head-to-head at n = 2²⁸

| K | radixK time | radixK B/op | americanFlag time | americanFlag B/op | msdBitmap time | msdBitmap B/op |
|---:|---:|---:|---:|---:|---:|---:|
| 20 | 1.17 s | 2.00 GiB | 4.20 s | **0** | 2.95 s | 512 B |
| 24 | 1.38 s | 2.00 GiB | 5.43 s | **0** | 2.76 s | 8 KiB |
| 28 | 2.13 s | 2.00 GiB | 6.29 s | **0** | 2.92 s | 128 KiB |
| 32 | 1.63 s | 2.00 GiB | 8.30 s | **0** | 3.10 s | 2 MiB |
| 36 | 1.90 s | 2.00 GiB | 7.16 s | **0** | 5.22 s | 32 MiB |
| 40 | 2.01 s | 2.00 GiB | 8.39 s | **0** | 15.26 s | 512 MiB |

Headline: at K ≤ 32 you can match within ~1.5–2× of `radixK` while
spending **at most 2 MiB** of scratch (vs 2 GiB).

## Speedup over current `sort.Slice` — best variant per cell

| n | K=20 | K=24 | K=28 | K=32 | K=36 | K=40 |
|---|---|---|---|---|---|---|
| 2²⁰ | bitmap **115×** | bitmap **41×** | radixK **21×** | radixK **22×** | radixK **17×** | radixK **18×** |
| 2²² | bitmap **157×** | bitmap **72×** | bitmap **17×** | radixK **27×** | radixK **19×** | radixK **18×** |
| 2²⁴ | bitmap **223×** | bitmap **96×** | bitmap **36×** | radixK **22×** | radixK **18×** | radixK **19×** |
| 2²⁶ | bitmap **214×** | bitmap **136×** | bitmap **63×** | radixK **21×** | radixK **17×** | radixK **17×** |
| 2²⁸ | bitmap **222×** | bitmap **146×** | bitmap **74×** | bitmap **25×** ≈ radixK | radixK **19×** | radixK **19×** |

`slicesSort` alone (free 1-line swap) is ~1.6× across the whole matrix.

## Observations

1. **`radixK` keeps the absolute time crown** for K > 32 (where simple
   bitmap can't fit), at the cost of `8n` scratch — 2 GiB on n=2²⁸.

2. **`americanFlag` truly is in-place** (0 B / 0 allocs) but pays
   3–4× the time of `radixK` due to swap-based scattering (poor
   prefetcher locality, branch mispredictions on bucket selection).
   Useful when memory is the absolute hard constraint.

3. **`msdBitmap` is a partial win**: it's ~1.5× of `radixK` and fits in
   ≤ 32 MiB of scratch through K=36 (vs `radixK`'s 2 GiB). However at
   **K=40 / n=2²⁸ it collapses to 15.3 s** because random scatter into a
   512 MiB inner bitmap is fundamentally DRAM-bound — each `bm[v>>6] |= …`
   is a cache-cold round-trip. `msdBitmap` is therefore useful in the
   K ∈ [33, ~36] range and not above.

4. **Bitmap remains absolute-best up to K = 28**, with peaks of 220×
   over `sort.Slice` at n = 2²⁸. At K = 32 it ties `radixK` but allocates
   512 MiB. Above K = 32 it cannot run at all.

5. **Memory profile, side-by-side**:

   | Variant | scratch grows with | Magnitude at extreme |
   |---|---|---|
   | `radixK` | n only (8n) | **2 GiB** at n=2²⁸ |
   | `bitmap` | K only (2^(K−3)) | **512 MiB** at K=32, refused above |
   | `msdBitmap` | K only (2^(K−11)) — **8× smaller than bitmap** | 512 MiB at K=40, **but slow there** |
   | `americanFlag` | nothing | **0** |

6. **None of the new in-place variants beat `bitmap` for K ≤ 32.** The
   small bitmap fits in cache and dominates everything else, including
   `msdBitmap` (which essentially performs the same scatter under a 256-
   way partition prelude).

7. **For K > 32 the field splits cleanly into three points on the
   memory-time frontier**:
   - `radixK`: 2 GiB scratch, ~1.9 s on n=2²⁸ K=36.
   - `msdBitmap`: 32 MiB scratch (K=36), ~5.2 s. Useful K ∈ [33, ≤36].
   - `americanFlag`: 0 scratch, ~7.2 s on n=2²⁸ K=36. Always works.

## Adaptive — measured (10× per cell, mean ± CV)

`SortAndDedupUint64Adaptive(keys, K)` dispatches to:

- `K ≤ 28` → `SortAndDedupUint64Bitmap`
- `29 ≤ K ≤ 36` → `SortAndDedupUint64MSDBitmap`
- `K ≥ 37` → `SortAndDedupUint64AmericanFlag`

The numbers below are 10 independent runs of `-benchtime=1x` aggregated
with `benchstat`. CV column is the relative standard deviation across the
10 samples (uses sample stddev, n=10).

### Time (mean, ms) and CV

| n / K | K=20 | K=24 | K=28 | K=32 | K=36 | K=40 |
|---|---|---|---|---|---|---|
| 2²⁰ | 1.19 ms ±7.8% | 2.86 ms ±4.8% | 10.3 ms ±4.8% | 39.9 ms ±5.3% | 49.9 ms ±1.9% | 23.8 ms ±2.6% |
| 2²² | 2.80 ms ±3.0% | 7.44 ms ±2.2% | 29.0 ms ±4.0% | 70.7 ms ±3.6% | 215 ms ±2.4% | 105 ms ±2.7% |
| 2²⁴ | 8.30 ms ±2.0% | 20.9 ms ±1.9% | 60.4 ms ±4.4% | 245 ms ±4.0% | 762 ms ±1.7% | 365 ms ±4.7% |
| 2²⁶ | 32.8 ms ±2.7% | 68.9 ms ±2.5% | 150 ms ±4.9% | 1 048 ms ±3.6% | 1 436 ms ±3.3% | 1 639 ms ±1.1% |
| 2²⁸ | 128 ms ±4.4% | 233 ms ±4.7% | 484 ms ±4.1% | 3 101 ms ±1.1% | 5 297 ms ±1.5% | 8 465 ms ±**0.7%** |

### Memory (B/op) — independent of n, depends only on K

| K | adaptive B/op | dispatch target | allocs/op |
|---:|---:|---|---:|
| 20 | 128 KiB | bitmap | 1 |
| 24 |   2 MiB | bitmap | 1 |
| 28 |  32 MiB | bitmap | 1 |
| 32 |   2 MiB | msdBitmap | 1 |
| 36 |  32 MiB | msdBitmap | 1 |
| 40 |   **0**  | americanFlag | **0** |

Peak scratch across the entire workload (any n, K ∈ [20..40]) is **32 MiB**,
**64× less** than `radixK`'s 2 GiB on n=2²⁸.

### Adaptive vs radixK on n=2²⁸ (the relevant operating point)

| K | adaptive time | radixK time | factor | adaptive B/op | radixK B/op | memory ratio |
|---:|---:|---:|---:|---:|---:|---:|
| 20 | **128 ms** | 1 174 ms | adaptive **9.2× faster** | 128 KiB | 2.0 GiB | **16 384× less** |
| 24 | **233 ms** | 1 376 ms | adaptive **5.9× faster** |   2 MiB | 2.0 GiB | **1 024× less** |
| 28 | **484 ms** | 2 131 ms | adaptive **4.4× faster** |  32 MiB | 2.0 GiB | **64× less** |
| 32 | 3 101 ms | **1 634 ms** | adaptive 1.9× slower |   2 MiB | 2.0 GiB | **1 024× less** |
| 36 | 5 297 ms | **1 899 ms** | adaptive 2.8× slower |  32 MiB | 2.0 GiB | **64× less** |
| 40 | 8 465 ms | **2 011 ms** | adaptive 4.2× slower |    **0** | 2.0 GiB | **∞ less** |

Adaptive is the right default whenever doubling RAM is unacceptable. For
K ≤ 28 it is also the absolute-fastest variant in the table — no
trade-off, just a strict win over `radixK`.

### Adaptive vs current `sort.Slice` baseline on n=2²⁸

| K | sortSlice | adaptive | speedup |
|---:|---:|---:|---:|
| 20 | 26 810 ms | 128 ms | **209×** |
| 24 | 33 372 ms | 233 ms | **143×** |
| 28 | 37 650 ms | 484 ms | **78×** |
| 32 | 36 869 ms | 3 101 ms | **12×** |
| 36 | 36 935 ms | 5 297 ms | **7×** |
| 40 | 37 281 ms | 8 465 ms | **4.4×** |

## Reproducing

```bash
cd Thesis
go test -run TestSortAndDedup    -timeout 5m  ./emptiness/internal/hash/

# Full matrix (one shot per cell):
go test -run NoTest -bench BenchmarkSortAndDedup$ \
        -benchtime=1x -timeout 90m ./emptiness/internal/hash/

# Adaptive only, 10x for tight CIs:
go test -run NoTest -bench 'BenchmarkSortAndDedup/adaptive' \
        -benchtime=1x -count=10 -timeout 60m ./emptiness/internal/hash/ \
        | benchstat -
```

Raw output: `/tmp/bench_full2.txt` (657 s, full 7-variant matrix) and
`/tmp/bench_adaptive_10x.txt` (~9 min, adaptive × 10).
