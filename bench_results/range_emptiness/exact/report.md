# Exact Range Emptiness Performance Analysis (1M Keys)

This report analyzes the performance of the succinct `ExactRangeEmptiness` implementation based on the SODA 2015 paper (Section 3.2).

## 1. Space Efficiency

The space usage follows the formula: $Space \approx (L - \log_2 n) + O(1)$ bits per key.
For $n = 1,000,000$, $\log_2 n \approx 19.93$.

| Key Size ($L$) | Bits/Key (Theoretical $L - 19.93$) | Bits/Key (Observed) | Overhead |
| :--- | :--- | :--- | :--- |
| 64 | 44.07 | **47.27** | ~3.2 bits |
| 128 | 108.07 | **111.3** | ~3.2 bits |
| 256 | 236.07 | **239.3** | ~3.2 bits |
| 512 | 492.07 | **495.3** | ~3.2 bits |

The constant overhead of **~3.2 bits/key** accounts for:
- `D1`: 1 bit/key (non-empty block indicator)
- `D2`: ~2 bits/key (Elias-Fano for block counts)
- Struct metadata and `rsdic` internal overhead.

## 2. Query Latency

Query performance is nearly constant across bit lengths, as the structure performs $O(1)$ Rank/Select operations on succinct vectors and a small-scale binary search on boundary blocks.

| Key Size ($L$) | Query Time (ns/op) |
| :--- | :--- |
| 64 | 141.9 ns |
| 128 | 146.5 ns |
| 256 | 141.3 ns |
| 512 | 139.3 ns |

## 3. Visualizations

The following plots were generated from the benchmark data:

- [Query Latency Plot](exact_range_query_latency.svg)
- [Bits per Key Plot](exact_range_bits_per_key.svg)

## 4. Conclusion

The implementation is **space-optimal** and achieves **constant-time queries**. It uses roughly **$L - \log_2 n + 3.2$ bits per key**, which is a significant improvement over the previous pointer-based representation.
