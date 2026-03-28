# Exact Range Emptiness (Succinct, SODA 2015 §3.2)

This package implements a **succinct** 1D range emptiness data structure that answers
queries $[a, b] \cap S \neq \emptyset$ in $O(1)$ expected time using $n \log(U/n) + O(n)$ space.

## 1. Architectural Foundation: SODA 2015 (§3.2)

The structure is a direct implementation of the "Range Emptiness Data Structure" described in the paper *Approximate
Range Emptiness in Constant Time and Optimal Space*.

### 2-Level Hierarchy

To achieve the information-theoretic lower bound of $n \log(U/n)$ bits, the structure divides the universe $[U]$
into $n$ equal-sized blocks:

1. **Global Level (Succinct Indexing)**:
    * **$D_1$ (Bit array size $n$)**: A bit-vector where $D_1[i] = 1$ if the $i$-th block contains at least one point
      from $S$.
    * **$D_2$ (Bit array size $\sim 2n$)**: An Elias-Fano style representation of block counts ($1$ followed by $n_i$
      zeros for each non-empty block $i$). This allows mapping a block index to its starting position in the global data
      array in $O(1)$ time using `Select1` and `Rank0`.
2. **Local Level (Bit-Packed Suffixes)**:
    * **$W$-bit Suffixes**: Instead of storing full keys, we only store the suffix $x \pmod{U/n}$ for each key $x$. The
      length of each suffix is $W = L_{universe} - \lceil \log_2 n \rceil$.
    * **Packed Storage**: Suffixes are stored in a single, dense `[]uint64` array where bits are packed in-flight using
      bit-shifts, eliminating the overhead of Go struct headers and pointers.

## 2. Practical Implementation Decisions

### Search Strategy in Buckets (Local Level)

The original SODA 2015 paper suggests a Weak Prefix Search structure (Hollow Z-Fast Trie)
for $O(1)$ worst-case queries inside each block. We use **binary search on bit-packed
suffixes** instead, and also provide a **linear scan** alternative.

#### Why no LERLOC (Weak Prefix Search) inside blocks?

LERLOC costs 50–80 bits/key (Hollow Z-Fast Trie + MMPH overhead) and up to ~500 ns per
query (pointer chasing through trie + hash lookups). When used inside ARE, suffixes are
only $w = 7$–$12$ bits/key — LERLOC would cost **4–10x more space than the data itself**,
and its query is **~50x slower** than binary search on a ~12-element bucket (~9 ns).
Binary search requires **zero** extra metadata beyond the sorted suffixes.

#### Bucket Occupancy Analysis

The number of blocks is $m = 2^{\lfloor\log_2 n\rfloor}$, so the expected occupancy per block
is $\lambda = n/m \in [1, 2)$. When $n$ is a power of two, $m = n$ and $\lambda = 1$ exactly.
Each key falls into a uniformly random block, so the occupancy of a fixed block follows
$\mathrm{Bin}(n, 1/m) \to \mathrm{Poisson}(\lambda)$ as $n \to \infty$
(Mitzenmacher & Upfal, *Probability and Computing*, Ch. 5).

**Measured bucket stats (uniform distribution, $\lambda = 1$, `TestExactRangeEmptiness_BucketStatsUint64`):**

| $n$      | Blocks   | Empty % | Avg keys/non-empty block | Max keys/block |
|----------|----------|---------|--------------------------|----------------|
| $2^{20}$ | $2^{20}$ | 36.75%  | 1.58                     | 9              |
| $2^{24}$ | $2^{24}$ | 36.79%  | 1.58                     | 9              |
| $2^{27}$ | $2^{27}$ | 36.79%  | 1.58                     | 10             |
| $2^{30}$ | $2^{30}$ | 36.79%  | 1.58                     | 13             |

Empty fraction converges to $e^{-1} \approx 36.79\%$, average occupancy of non-empty blocks
to $1/(1 - e^{-1}) \approx 1.58$ — both matching $\mathrm{Poisson}(1)$ theory exactly.
Maximum grows as $\Theta(\log n / \log \log n)$.

**Non-uniform distributions break the Poisson model.** When ERE is used inside SODA ARE,
the locality-preserving hash maps clusters to clusters — it does not break density.
Measured via `TestEREBucketStats_SodaARE` ($n = 2^{20}$, $\varepsilon = 0.01$):

| Dataset                | L    | Avg keys/block | Max keys/block |
|------------------------|------|----------------|----------------|
| uniform                | 16   | 2.31           | 12             |
| clustered (8 Gaussian) | 256  | 6.15           | 137            |
| sosd_fb                | 4096 | 1466           | 4561           |
| sosd_wiki              | 4096 | 14126          | 58214          |
| sosd_osm               | 4096 | 2.32           | 29             |
| sosd_books             | 4096 | 209715         | 218469         |

However, the **absolute bound** on bucket size is $2^w$ where $w = K - \lfloor\log_2 n\rfloor$
is the suffix length. For industry-standard BPK $\leq 15$, we have $w \leq 12$, so
max bucket $\leq 4096$. In practice at BPK 10–12 (the sweet spot), $w = 7$–$9$ and
max bucket $\leq 512$.

#### Binary Search vs Linear Scan

Both are implemented: `isRangeEmptyInBlock` (binary) and `isRangeEmptyInBlockLinear`.

**Benchmark results** (Apple M4 Max, `BenchmarkBucketSearch_LinearVsBinary`):

| Bucket size | Linear (ns) | Binary (ns) | Winner      |
|-------------|-------------|-------------|-------------|
| 2           | 1.3         | 5.1         | linear 3.9x |
| 8           | 2.2         | 6.8         | linear 3.1x |
| 12          | 2.7         | 9.0         | linear 3.3x |
| 32          | 6.0         | 12.1        | linear 2.0x |
| 64          | 10.0        | 15.0        | linear 1.5x |
| 128         | 18.0        | 18.0        | **tie**     |
| 256         | 35.0        | 21.0        | binary 1.7x |

Crossover: **~128 elements**. Linear scan wins on small buckets due to sequential
prefetch and no branch misprediction. Values are stable across $w \in \{8, 11, 16, 20\}$.

**We use binary search as the default** despite linear being faster on typical buckets,
because:

1. **Robustness**: binary search degrades gracefully — 5 ns at n=2, 21 ns at n=256.
   Linear goes from 1.3 ns to 35 ns. On adversarial distributions, binary stays bounded.
2. **Marginal impact**: bucket search is a small fraction of total ERE query time
   (which includes Rank/Select on $D_1$, $D_2$). The absolute difference between
   linear and binary on typical buckets (~6 ns) is unlikely to dominate.
3. **Predictability**: $O(\log k)$ worst case vs $O(k)$ worst case. For a data structure
   that may face unknown distributions, predictable performance matters more than
   best-case speed.

Linear scan (`LinearIsEmpty`) is available for specialized use cases where the distribution
is known to be uniform and every nanosecond counts.

**Complexity:**

* **Binary search**: $O(\log(\text{keys per block}))$ — $O(1)$ expected for uniform distributions.
* **Linear scan**: $O(\text{keys per block})$ — $O(1)$ expected for uniform, but $O(2^w)$ worst case.
* **Space**: $0$ extra bits per key (both strategies).

---

## 3. Complexity Analysis

The structure achieves the information-theoretic lower bound for representing a subset of size $n$ in a universe of
size $U$.

### Space Complexity: $O(n \log(U/n))$ bits

The total space is the sum of metadata and bit-packed suffixes:

1. **Metadata ($D_1 + D_2$):** $O(n)$ bits.
    * $D_1$ (occupancy) takes $1$ bit/key.
    * $D_2$ (counts) takes $\sim 2$ bits/key using Elias-Fano encoding.
    * Total overhead: **$\sim 3.2$ bits/key** (including Rank/Select index overhead).
2. **Data (Suffixes):** $O(n \cdot (L - \log n))$ bits.
    * Each of the $n$ keys stores only its suffix of length $W = L - \log_2 n$.
3. **Total Formula:** $Space \approx n \cdot (L - \log_2 n + 3.2)$ bits.

Measured with $n = 10^6$ keys ($\log_2 n \approx 19.93$):

| $L$ (Key Bits) | Suffix Bits ($L - 19.93$) | Metadata (Observed) | Total Bits/Key |
|:---------------|:--------------------------|:--------------------|:---------------|
| **64**         | 44.07                     | + 3.2               | **47.27**      |
| **128**        | 108.07                    | + 3.2               | **111.27**     |
| **256**        | 236.07                    | + 3.2               | **239.27**     |
| **512**        | 492.07                    | + 3.2               | **495.27**     |

**Conclusion on Space:** Space grows **linearly with $L$**. This is unavoidable for an **Exact** structure, as we must
store the information distinguishing the keys.

### Time Complexity: $O(1)$ Expected

1. **Build:** $O(n \cdot L)$. A single pass over sorted keys to pack bits and index $D_1/D_2$.
2. **Query:**
    * **Global Navigation:** $O(1)$ via Rank/Select on $D_1$ and $D_2$.
    * **Local Search:** $O(\log(\text{keys per block}))$.
    * **Average Case:** For uniform distributions, bucket occupancy is $\mathrm{Poisson}(1)$ — expected search time
      is $O(1)$. For non-uniform inputs (clustered, SOSD), buckets can be heavily skewed (see Section 2 above), but are
      always bounded by $2^w$.
    * **Worst Case:** $O(\log n)$ if all keys collide in a single block.

## 4. Transition to Approximate Range Emptiness (ARE)

The linear dependence of space on key length ($L$) observed in this structure is the primary motivator for moving to *
*Approximate Range Emptiness**.

### Goal: Breaking the Linear Space Growth

In the ARE structure (SODA 2015, Section 4), we apply this Exact structure to **hashed fingerprints** instead of
original keys.

1. **Fingerprint Length**: Instead of a suffix of length $L - \log_2 n$, we use a fingerprint of
   length $\approx \log(L_{interval}/\epsilon)$.
2. **Space Independence**: Since the fingerprint length is independent of the original key length $L$, the total space
   usage becomes $O(n \log(L_{interval}/\epsilon))$.
3. **Expected ARE Space ($\epsilon = 0.01$)**:
    * Fingerprint: ~14 bits.
    * Metadata: ~3.2 bits.
    * **Total: ~17-18 bits/key** (regardless of whether the keys are 64 or 1024 bits).

The implementation of `ApproximateRangeEmptiness` will serve as a wrapper around this `ExactRangeEmptiness` structure,
managing the universe reduction (hashing) layer.
