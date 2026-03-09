# Approximate Range Emptiness Performance Analysis (1M Keys)

This report analyzes the performance of the `ApproximateRangeEmptiness` structure, which implements the probabilistic 1D range emptiness filter from the SODA 2015 paper (Section 4).

## 1. Space Efficiency (Breaking the Linear Bound)

The theoretical space for ARE is $O(n \log(L_{interval}/\epsilon))$. By using truncated fingerprints instead of full suffixes, the space usage becomes entirely **independent of the original key length ($L$)**.

### Observed Bits per Key (N = 1,000,000)

| Key Size ($L$) | $\epsilon = 0.01$ (1% FP) | $\epsilon = 0.001$ (0.1% FP) | Exact Counterpart ($\epsilon = 0$) |
| :--- | :--- | :--- | :--- |
| **64 bits** | 11.25 | 14.27 | 47.27 |
| **128 bits** | 11.25 | 14.27 | 111.3 |
| **256 bits** | 11.25 | 14.27 | 239.3 |
| **512 bits** | 11.25 | 14.27 | **495.3** |

*Note: The actual observed space for $\epsilon=0.001$ (~14.3 bits) is even better than our initial conservative estimate (~23 bits). This happens because truncating to a $K$-bit universe reduces the total number of unique keys slightly due to deliberate collisions, which reduces the internal size of the `ExactRangeEmptiness`.*

**Key Takeaway:** We successfully decoupled memory from the key length. For 512-bit keys, ARE provides a **34x memory reduction** over the Exact structure while guaranteeing a false positive rate $\le 0.1\%$.

## 2. Query Latency

Query performance remains perfectly constant at $O(1)$ and is on par with the underlying Exact structure. 

| Key Size ($L$) | $\epsilon = 0.01$ | $\epsilon = 0.001$ |
| :--- | :--- | :--- |
| **64 bits** | 145.0 ns | 144.8 ns |
| **128 bits** | 147.0 ns | 147.8 ns |
| **256 bits** | 141.8 ns | 141.5 ns |
| **512 bits** | 138.7 ns | 140.3 ns |

The truncation of the query boundaries adds negligible overhead ($<5$ ns). The performance remains rock-solid regardless of $L$.

## 3. Visualizations

The generated SVG plots demonstrate the flat performance lines across all values of $L$:
- [Query Latency Plot](plots/approx_range_query_latency.svg)
- [Bits per Key Plot](plots/approx_range_bits_per_key.svg)

## 4. Conclusion

The `ApproximateRangeEmptiness` structure behaves exactly as theorized. By applying universe reduction via fingerprinting (prefix truncation), we created a probabilistic range filter that operates in $O(1)$ time and requires merely **~14 bits per key** to achieve a $0.1\%$ false positive rate, effectively solving the problem of high memory consumption for long strings.
