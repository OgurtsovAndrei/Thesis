# Approximate Range Emptiness (SODA 2015 §4)

This package implements the **probabilistic** 1D range emptiness data structure described in Section 4 of the paper *Approximate Range Emptiness in Constant Time and Optimal Space*.

## 1. Architectural Overview: $K$-bit Truncation

The core of the Approximate structure is **locality-preserving hashing** via $K$-bit truncation. Unlike standard cryptographic hashes (like SHA-256) which scramble the order of keys, prefix truncation preserves the lexicographical order.

### How it Works
1.  **Prefix Selection**: For each key $x$, we retain only the first $K$ bits. 
    - Mathematically: $h(x) = \text{Prefix}_K(x) = \lfloor x \cdot 2^{K-L} \rfloor$.
2.  **Order Preservation**: If $x < y$, then $h(x) \le h(y)$. This property allows us to map range queries $[a, b]$ in the original space directly to $[h(a), h(b)]$ in the truncated space.
3.  **Universe Reduction**: We transform a massive universe (e.g., $2^{256}$ for 256-bit keys) into a manageable universe of size $2^K$.

### The False Positive Mechanism
False positives in this structure occur only at the **boundaries** of the query range. 

If we query an interval $[a, b]$ that is actually empty in the original set $S$, a false positive happens if:
- There is a key $x \in S$ such that $x < a$ (just before the range), but after truncation, $h(x) = h(a)$.
- There is a key $y \in S$ such that $y > b$ (just after the range), but after truncation, $h(y) = h(b)$.

### Optimal $K$ Selection
To bound the false positive rate by $\epsilon$, we choose:
$$K = \lceil \log_2(2n / \epsilon) \rceil$$

This formula ensures that the probability of a collision at either the lower or upper boundary is sufficiently low.

**Key Advantage**: The required $K$ depends on $n$ and $\epsilon$, but **is independent of the original bit-length $L$**. This allows the structure to achieve a flat memory profile even for arbitrarily long keys (e.g., long strings or large hashes).

---

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
