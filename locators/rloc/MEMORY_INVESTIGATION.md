# Memory Usage Investigation: MMPH Buckets Overhead

This document summarizes the investigation into why the `MMPH_Buckets` component in `RangeLocator` (RLOC) and `LocalExactRangeLocator` (LERLOC) reports significantly higher memory usage (~48 bits/key) compared to standalone MMPH benchmarks (~15 bits/key).

## 1. The Core Discrepancy: $N$ vs. $|P|$

The most critical finding is that while standalone MMPH indexes original keys ($N$), the `RangeLocator` indexes an internal **boundary set $P$** derived from the Z-Fast Trie structure to support exact range mapping.

### Theoretical vs. Experimental Multipliers
For a set of $N$ unique keys:
- **Trie Nodes ($U$)**: In a compacted binary trie, the number of nodes is always $2N - 1$. As $N \to \infty$, the ratio **$U/N \to 2.0$**.
- **Boundary Set ($P$)**: Each node generates up to 3 boundary strings (trimmed extent, extension with '1', and successor). After deduplication, experimental results confirm a stable ratio:
  $$\mathbf{|P|/N \approx 3.3}$$

### Impact on Memory (bits/key)
Since the benchmarking pipeline normalizes all metrics to **bits per original key ($N$)**, the MMPH contribution is multiplied by the $|P|/N$ ratio:
$$15 \text{ bits/item in } P \times 3.3 \text{ items/key} \approx \mathbf{49.5 \text{ bits/key}}$$

This matches the observed **~48.7 bits/key** in LERLOC benchmarks.

## 2. Component Breakdown of `MMPH_Buckets`

At $N=32,768$ ($|P| \approx 108,000$), the MMPH uses a bucket size of 256. The overhead is attributed as follows:

| Component | Bits per Item in $P$ | Contribution to Bits per Key $N$ | Description |
|-----------|-----------------------|----------------------------------|-------------|
| **Local Ranks** | 8.0 bits | **26.4 bits/key** | `[]uint8` array (1 byte per item in $P$). |
| **MPHF (BoomPHF)** | ~3.5 bits | **~11.5 bits/key** | Minimal Perfect Hash Function for local indexing. |
| **Headers & Delims** | ~1.5 bits | **~5.0 bits/key** | Go struct headers and bucket delimiter BitStrings. |
| **Padding & Other** | ~1.5 bits | **~5.0 bits/key** | Memory alignment and slice overhead. |
| **Total** | **~14.5 bits/item** | **~48 bits/key** | |

## 3. Consistency Across Key Lengths ($L$)

Experiments with $L=64, 256, 1024$ show that these constants remain **stable**. The Z-Fast Trie ensures that the number of internal nodes and boundary strings depends only on $N$, not on the length of the keys.

## 4. Conclusion

The ~48 bits/key reported for `MMPH_Buckets` is an accurate reflection of the current architecture. The "inflated" number compared to standalone MMPH is primarily due to the **$3.3\times$ expansion** of the indexed set $P$ relative to the key set $N$.

## 5. Optimization Opportunities
- **Reduce $|P|$**: Investigate if all three boundary strings per node are strictly necessary for correctness.
- **Succinct Ranks**: Replacing `[]uint8` with a bit-packed representation (e.g., 4 or 5 bits per item) could save **~13-16 bits/key**.
