# LeMonHash Performance Analysis (CGO)

## 1. Benchmarks Summary (Apple M4 Max)

| Metric | 1k keys | 1M keys | 16M keys |
| :--- | :--- | :--- | :--- |
| **Space (bits/key)** | 29.2 | 3.34 | **3.32** |
| **Rank (Single, Zero-Alloc)** | ~88 ns | ~175 ns | ~200 ns |
| **RankPair (2 keys/call)** | **~65 ns** | **~162 ns** | ~180 ns |
| **RankBatch (1024 keys/call)** | ~190 ns | ~290 ns | ~310 ns |

## 2. Overhead Breakdown

A single `Rank(key)` call now takes ~90-200ns after removing allocations.

1. **CGO Transition (~50-80 ns):** The fixed cost of switching context.
2. **Implicit Pinning (~0 ns):** When passing arguments directly, Go's compiler handles pinning with near-zero overhead.
3. **C++ Logic (~10-100 ns):** Increases with dataset size due to PGM-index depth and cache misses.

## 3. The Pinning Trade-off

The benchmarks reveal a clear hierarchy of efficiency:

1. **Direct Arguments (RankPair):** Best performance. Amortizes the CGO transition across 2 keys while keeping pinning "implicit" and fast.
2. **Single Call (Rank):** Good performance, but pays the full transition tax for every key.
3. **Manual Pinning (RankBatch):** Worst performance. Even though it has only 1 CGO transition per 1024 keys, the cost of calling `runtime.Pinner.Pin()` 1024 times is significantly higher than 1024 CGO transitions.

## 4. Conclusion

For maximum performance through CGO:
- **Avoid `runtime.Pinner`** for large batches of small objects.
- **Use "Multi-Arg" wrappers** (like `RankPair`, `RankQuad`) to amortize transition costs if data can be passed as separate arguments.
- **Use Flat Buffers** if batching is required. By concatenating keys into one large `[]byte`, we can use `Pin()` once for the entire batch, which would likely outperform all other methods.
