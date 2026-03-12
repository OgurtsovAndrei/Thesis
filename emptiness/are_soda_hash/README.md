# ARE Robust: SODA 2015 Locality-Preserving Hash

This package implements the **Robust Approximate Range Emptiness** filter using the locality-preserving hash function described in Section 3.1 of the SODA 2015 paper.

## 1. The Limitation of Simple Truncation
In the basic `are` package, we use simple prefix truncation. While efficient, it suffers from **100% False Positive Rate** on sequential data (e.g., $x$ and $x+1$). 

If a key $x$ and its neighbor $x+1$ share the same prefix after truncation, the filter cannot distinguish between them. This creates "blind spots" where any query for a gap between keys will always return a false positive.

## 2. The SODA 2015 Solution: Randomized Locality
The Robust version solves this by applying a randomized, but order-preserving, shift to each block of the universe.

### The Hash Function
For a key $x$ in a universe divided into blocks of size $r$, the hash $h(x)$ is defined as:
$$h(x) = (\text{hash}(\lfloor x/r \rfloor) + x) \pmod r$$

- **Within a block**: The order of keys is preserved perfectly (it's just a cyclic shift).
- **Across blocks**: Each block is shifted by a random value $u = \text{hash}(\text{block\_idx})$, effectively "scrambling" the relative positions of prefixes across the global universe.

### Why it Works
This transformation ensures that the probability of two neighbors ($x$ and $x+1$) colliding after hashing is exactly $1/r$, regardless of their original prefix similarity. This eliminates the adversarial patterns that break simple truncation.

---

## 3. Features & Guarantees

### Range Sensitivity ($RangeLen$)
Unlike simple truncation, the Robust ARE is built with a target **maximum range length** ($L$ or `RangeLen`). 
- It guarantees an upper bound on the False Positive Rate ($\epsilon$) for any query of length $\le L$.
- Memory consumption scales logarithmically with $L$: adding **1 bit/key** for every doubling of $L$.

### Multi-Block Range Support
This implementation supports queries of **any length**, even those exceeding $L$ or spanning multiple SODA blocks ($2^K$):
1.  **Small Ranges**: Handled with 1 or 2 ERE queries (high precision).
2.  **Large Ranges**: Handled by checking block boundaries and intermediate full blocks.
    - *Note*: For ranges exceeding the block size $2^K$, the filter conservatively reports "Not Empty" if the filter contains any keys.

### Space-Optimal Succinctness
The structure is **independent of the original key size** (e.g., 64, 128, or 256 bits). It only stores the "entropy" needed to distinguish $n$ keys with error $\epsilon$ over range $L$.

---

## 4. Performance & Accuracy Comparison

| Metric | Fast ARE (Truncation) | Robust ARE (SODA Hash) |
| :--- | :--- | :--- |
| **Uniform Data FPR** | ~0.3% | ~0.6% |
| **Sequential Data FPR** | **100% (Fail)** | **0% (Pass)** |
| **Memory (L=100)** | ~11 bits/key | ~16 bits/key |
| **Query Time** | ~150 ns | ~250-400 ns |

## 5. Usage Guidelines
- Use **`are` (Fast)** if your data is naturally high-entropy (UUIDs, random hashes) and query speed is paramount.
- Use **`are_soda_hash` (Robust)** for structured data (Auto-increment IDs, timestamps, sorted sequences) or when range-query reliability is critical.
