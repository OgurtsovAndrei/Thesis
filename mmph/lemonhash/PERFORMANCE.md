# LeMonHash Performance Analysis (CGO)

## 1. Benchmarks Summary (Apple M4 Max)

| Metric | 1k keys | 1M keys | 16M keys |
| :--- | :--- | :--- | :--- |
| **Space (bits/key)** | 29.2 | 3.34 | **3.32** |
| **Rank Latency (ns/key)** | ~550 | ~630 | ~700 |
| **RankBatch (ns/key)** | ~625 | ~680 | ~720 |

## 2. Overhead Breakdown

A single `Rank(key)` call currently takes ~600ns. Here is where the time goes:

1. **CGO Transition (~50-100 ns):** The fixed cost of switching the execution context from Go (segmented stack) to C (standard stack) and back.
2. **Memory Allocation (~300-400 ns):** The current implementation of `bits.BitString.Data()` creates a **new `[]byte` slice** on every call. This triggers the Go allocator and a `memcpy`.
3. **C++ Logic (< 100 ns):** LeMonHash itself is extremely fast (math-based learned index + local bit-layer lookup).
4. **CGO Pointer Validation:** Go's runtime checks if pointers passed to C are valid, which adds a small cost.

## 3. The Batch Paradox

Surprisingly, `RankBatch` is currently **slower** than single `Rank` calls.

### Why?
To pass a slice of `BitString` to C, we must pass an array of pointers (`**char`). Go's CGO rules require that any Go memory pointed to by a pointer passed to C must be **pinned**.

Since we pass an array containing $N$ pointers to $N$ different memory blocks (the keys), we have to call `runtime.Pinner.Pin()` **$N$ times**. 
* **Single Rank:** 1 implicit, highly optimized pin (done by the compiler).
* **RankBatch (1024):** 1024 explicit `Pin()` calls in a loop.

The overhead of 1024 manual pinning operations is currently greater than the overhead of 1024 CGO transitions.

## 4. Future Optimizations

To achieve < 200ns per query, we need to eliminate allocations:

1. **Zero-Alloc `BitString` Access:** Add a method to `BitString` that returns a pointer to the internal `[]uint64` storage without copying (using `unsafe.Pointer`). This removes the `Data()` allocation.
2. **Flat Buffer for Batching:** Instead of a slice of `BitString`, use a `FlatBitBuffer` where all keys are concatenated into one large `[]byte`. 
    * This allows pinning the entire batch with **one** `Pin()` call.
    * This would likely reduce `RankBatch` latency to **< 50ns per key**.
