# Approximate Range Emptiness (SODA 2015 §4)

This package implements the **probabilistic** 1D range emptiness data structure described in Section 4 of the paper *Approximate Range Emptiness in Constant Time and Optimal Space*.

## 1. Architectural Overview

The core purpose of this structure is to answer interval queries $[a, b] \cap S \neq \emptyset$ in $O(1)$ time while using memory that depends only on the false positive rate $\epsilon$, **not** on the original length of the keys ($L$).

It achieves this by acting as a wrapper around the `ExactRangeEmptiness` structure:
1.  **Universe Reduction (Hashing)**: The structure uses lexicographical truncation as a locality-preserving hash function. It reduces the $L$-bit keys to $K$-bit fingerprints, where $K = \lceil \log_2(2n / \epsilon) \rceil$.
2.  **Deduplication**: Because $K \ll L$, some keys will inevitably collide and map to the same fingerprint. These duplicates are removed to satisfy the strict ordering requirement of the underlying Exact structure.
3.  **Exact Storage**: The unique, truncated fingerprints are fed into the `ExactRangeEmptiness` structure, which treats them as a complete universe of size $2^K$.
4.  **Query Delegation**: When queried with interval $[a, b]$, the structure simply truncates the boundaries to $a_{[:K]}$ and $b_{[:K]}$ and queries the underlying Exact structure.

## 2. False Positive Bounds

A false positive occurs if the query interval $[a, b]$ is empty, but the underlying exact structure reports it is not. 
Because the mapping preserves order (it's just a prefix truncation), the only way a false positive can happen is if:
1. There is a key $x \in S$ just outside the interval $[a, b]$ (e.g., $x < a$ or $x > b$).
2. The truncation of $x$ is identical to the truncation of $a$ or $b$.

By setting $K = \lceil \log_2(2n / \epsilon) \rceil$, the probability of such a collision at the boundaries is bounded by $\epsilon$.

## 3. Performance Characteristics

The most important characteristic of this structure is the **flat memory footprint** relative to key length.

### Memory Profile (N = 1,000,000)

| $\epsilon$ | Fingerprint Length ($K$) | Total Bits/Key | Time (ns/op) |
| :--- | :--- | :--- | :--- |
| **0.01** (1%) | ~28 bits | **~11.3 bits** | ~145 ns |
| **0.001** (0.1%) | ~31 bits | **~14.3 bits** | ~145 ns |

*Note: The actual `Bits/Key` is lower than $K$ because the Exact structure compresses the fingerprints further using blocks, and truncation inherently reduces the number of unique entries.*

### Comparison to Exact Structure
For 512-bit keys, the Exact structure requires **~495 bits/key**.
By accepting a $0.1\%$ false positive rate, this Approximate structure requires only **~14.3 bits/key**—a **~34x reduction in memory**.
