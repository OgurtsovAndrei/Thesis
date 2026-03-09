# Exact Range Emptiness

This package implements the Exact 1D Range Emptiness data structure, based on Section 3 of the paper *"Approximate Range Emptiness in Constant Time and Optimal Space"* (SODA 2015).

## Theoretical Background

The exact range emptiness problem asks whether an interval $[a, b]$ contains any points from a static set $S \subset [U]$.
The structure proposed in the paper divides the universe $[U]$ into $n$ blocks (where $n = |S|$) and maintains a summary bit vector of non-empty blocks.

## Implementation Details

In our implementation, the keys are of type `bits.BitString` and can be of arbitrary length. The data structure maps the lexicographic prefix of length $k$ (where $k = \lfloor \log_2 n \rfloor$) to an integer bucket index. 
Because `bits.BitString` uses bit 0 as the most significant bit for lexicographic sorting, we extract the first $k$ bits of the key and reverse their significance to form a bucket index $b \in [0, 2^k - 1]$. 

This mapping ensures that if $b_1 > b_2$, then any key in bucket $b_1$ is strictly lexicographically greater than any key in bucket $b_2$.

### Space Usage
- The summary bit vector $H$ is constructed using Elias-Fano encoding. It stores $n$ ones and $2^k \le n$ zeros, bounded by $2n$ bits. We use the `rsdic` Succinct Bit Vector library for $O(1)$ Rank/Select operations on this bit vector.
- The explicitly stored keys take $O(n \times W)$ bits. For simplicity and robustness, the full `BitString` slice is kept.

### Query Time
- The interval $[a, b]$ is mapped to bucket interval $[B(a), B(b)]$.
- If $B(b) > B(a) + 1$, we use the succinct vector $H$ to verify in $O(1)$ time if there are any elements in intermediate buckets.
- At the boundaries (buckets $B(a)$ and $B(b)$), we perform binary searches. Since $k \approx \log_2 n$, the average bucket size is 1. The binary search operates in expected $O(1)$ time, yielding an overall expected $O(1)$ query time.
