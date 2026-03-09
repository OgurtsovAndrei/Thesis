# Range Emptiness Filters

This package provides space-efficient data structures for answering 1D range emptiness queries: **"Does the interval $[a, b]$ contain any points from set $S$?"**

The implementations are based on the SODA 2015 paper: *["Approximate Range Emptiness in Constant Time and Optimal Space"](https://arxiv.org/abs/1407.2907)* by Goswami, Pagh, Silvestri, and Sivertsen.

> **Note:** This is the first known practical implementation of the Goswami et al. SODA 2015 algorithm.
> The paper is widely cited as the theoretical gold standard for range filters
> (e.g., [Grafite, SIGMOD 2024](https://arxiv.org/abs/2311.15380) calls it "the information-theoretically optimal solution"),
> but prior to this work no public implementation existed due to the complexity of the underlying machinery
> (succinct bit vectors, Elias-Fano coding, monotone minimal perfect hashing, hollow tries).

**Use case:** Range filters for LSM-tree key-value stores (e.g., RocksDB). Before reading an SST file from disk, the filter answers whether a queried key range *might* intersect the file's key set, avoiding unnecessary I/O with a bounded false positive probability.

## Data Structures

### 1. [Exact Range Emptiness](exact_range_emptiness.md)
A succinct structure that answers range queries with **100% accuracy**.
- **Space:** $O(n \log(U/n))$ bits. Achieving the information-theoretic lower bound.
- **Technique:** Uses a 2-level hierarchy with Elias-Fano indexing ($D_1, D_2$ bitvectors) and bit-packed suffixes.
- **Performance:** $O(1)$ expected query time (~140ns for 1M keys).

### 2. [Approximate Range Emptiness (ARE)](approximate_range_emptiness.md)
A probabilistic filter that allows for a small false positive probability $\epsilon$.
- **Space:** $O(n \log(1/\epsilon))$ bits. The memory footprint is **independent of key length ($L$)**.
- **Technique:** Universe reduction via locality-preserving fingerprinting (prefix truncation) coupled with the Exact structure.
- **Performance:** $O(1)$ constant time query.
- **Efficiency:** Achieves 0.1% FP rate at only **~6.5 bits per key** (empirical).

## Benchmarks & Reports

Comprehensive performance analysis and visualizations are available in the [bench_results](../bench_results/range_emptiness/) directory:

- **[Exact Structure Report](../bench_results/range_emptiness/exact/report.md)**: Analysis of space growth relative to key length $L$.
- **[Approximate Structure Report](../bench_results/range_emptiness/approx/report.md)**: Scale testing with up to 16 million keys and 1024-bit lengths.
- **[Accuracy Trade-off Study](../bench_results/range_emptiness/tradeoff/report.md)**: High-resolution empirical analysis of FPR vs. Space.

## Usage Example

```go
import "Thesis/local_exact_range"

// 1. Prepare sorted keys
keys := []bits.BitString{...}

// 2. Build Approximate Filter (0.1% error rate)
are, _ := local_exact_range.NewApproximateRangeEmptiness(keys, 0.001)

// 3. Query interval [a, b]
if are.IsEmpty(a, b) {
    fmt.Println("Range is definitely empty")
} else {
    fmt.Println("Range might contain elements")
}
```
