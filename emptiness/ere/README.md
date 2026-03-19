# Exact Range Emptiness (Succinct, SODA 2015 §3.2)

This package implements a **succinct** 1D range emptiness data structure that answers queries $[a, b] \cap S \neq \emptyset$ in $O(1)$ expected time using $n \log(U/n) + O(n)$ space.

## 1. Architectural Foundation: SODA 2015 (§3.2)

The structure is a direct implementation of the "Range Emptiness Data Structure" described in the paper *Approximate Range Emptiness in Constant Time and Optimal Space*.

### 2-Level Hierarchy
To achieve the information-theoretic lower bound of $n \log(U/n)$ bits, the structure divides the universe $[U]$ into $n$ equal-sized blocks:
1.  **Global Level (Succinct Indexing)**:
    *   **$D_1$ (Bit array size $n$)**: A bit-vector where $D_1[i] = 1$ if the $i$-th block contains at least one point from $S$.
    *   **$D_2$ (Bit array size $\sim 2n$)**: An Elias-Fano style representation of block counts ($1$ followed by $n_i$ zeros for each non-empty block $i$). This allows mapping a block index to its starting position in the global data array in $O(1)$ time using `Select1` and `Rank0`.
2.  **Local Level (Bit-Packed Suffixes)**:
    *   **$W$-bit Suffixes**: Instead of storing full keys, we only store the suffix $x \pmod{U/n}$ for each key $x$. The length of each suffix is $W = L_{universe} - \lceil \log_2 n \rceil$.
    *   **Packed Storage**: Suffixes are stored in a single, dense `[]uint64` array where bits are packed in-flight using bit-shifts, eliminating the overhead of Go struct headers and pointers.

## 2. Practical Implementation Decisions

### Binary Search in Buckets (Local Level)
The original SODA 2015 paper theoretically suggests using a Weak Prefix Search structure (like a Hollow Z-Fast Trie) for $O(1)$ **worst-case** queries inside each non-empty block.

However, in this implementation, we use **binary search on bit-packed suffixes** for the local search. 

#### Why no LERLOC (Weak Prefix Search) inside blocks?
LERLOC (**L**ocator for **E**xact **R**ange emptiness using **LOC**ators) is our implementation
of a Weak Prefix Search structure (based on Hollow Z-Fast Trie and MMPH) that provides
$O(1)$ worst-case guarantees but comes with overhead:
*   **Space Overhead**: Adding a trie index to each block would add ~3–10 bits/key. Binary search requires **zero** extra metadata besides the sorted suffixes.
*   **Performance Trade-off**: For small datasets (like the keys within a single block), the constants of a trie/hash-based approach are higher than a simple binary search over a few CPU cache lines.

**Why Binary Search is the Better Choice Here:**
*   **Distribution Analysis**: With $n$ blocks for $n$ keys, the number of keys per block follows a Poisson distribution with $\lambda \approx 1$.
    *   **Average**: ~2.24 keys per non-empty block.
    *   **Maximum**: Observed ~12 keys for $N=10^6$ in our tests.
*   **Speed**: Binary searching through ~12 packed values is significantly faster and more cache-friendly than performing multiple rank/select lookups in a trie.
*   **Result**: This choice allowed us to achieve **~47 bits/key** (very close to the theoretical 44-bit lower bound) while maintaining $O(1)$ expected query time.

**Complexity:**
*   **Time**: $O(\log(\text{keys per block}))$ which is $O(1)$ expected for uniform distributions.
*   **Space**: $0$ extra bits per key.

---

## 3. Complexity Analysis

The structure achieves the information-theoretic lower bound for representing a subset of size $n$ in a universe of size $U$.

### Space Complexity: $O(n \log(U/n))$ bits
The total space is the sum of metadata and bit-packed suffixes:
1.  **Metadata ($D_1 + D_2$):** $O(n)$ bits. 
    *   $D_1$ (occupancy) takes $1$ bit/key.
    *   $D_2$ (counts) takes $\sim 2$ bits/key using Elias-Fano encoding.
    *   Total overhead: **$\sim 3.2$ bits/key** (including Rank/Select index overhead).
2.  **Data (Suffixes):** $O(n \cdot (L - \log n))$ bits.
    *   Each of the $n$ keys stores only its suffix of length $W = L - \log_2 n$.
3.  **Total Formula:** $Space \approx n \cdot (L - \log_2 n + 3.2)$ bits.

Measured with $n = 10^6$ keys ($\log_2 n \approx 19.93$):

| $L$ (Key Bits) | Suffix Bits ($L - 19.93$) | Metadata (Observed) | Total Bits/Key |
| :--- | :--- | :--- | :--- |
| **64** | 44.07 | + 3.2 | **47.27** |
| **128** | 108.07 | + 3.2 | **111.27** |
| **256** | 236.07 | + 3.2 | **239.27** |
| **512** | 492.07 | + 3.2 | **495.27** |

**Conclusion on Space:** Space grows **linearly with $L$**. This is unavoidable for an **Exact** structure, as we must store the information distinguishing the keys.

### Time Complexity: $O(1)$ Expected
1.  **Build:** $O(n \cdot L)$. A single pass over sorted keys to pack bits and index $D_1/D_2$.
2.  **Query:**
    *   **Global Navigation:** $O(1)$ via Rank/Select on $D_1$ and $D_2$.
    *   **Local Search:** $O(\log(\text{keys per block}))$.
    *   **Average Case:** Since we use $n$ blocks for $n$ keys, the number of keys per block follows a Poisson distribution with $\lambda \approx 1$ (if $n=m$). Expected search time is **$O(1)$** for uniform or near-uniform distributions. Note: when ERE is used inside an ARE filter, the SODA hash produces near-uniform fingerprints, so this assumption holds even for clustered input data.
    *   **Worst Case:** $O(\log n)$ if all keys collide in a single block.

## 4. Transition to Approximate Range Emptiness (ARE)

The linear dependence of space on key length ($L$) observed in this structure is the primary motivator for moving to **Approximate Range Emptiness**.

### Goal: Breaking the Linear Space Growth
In the ARE structure (SODA 2015, Section 4), we apply this Exact structure to **hashed fingerprints** instead of original keys.

1.  **Fingerprint Length**: Instead of a suffix of length $L - \log_2 n$, we use a fingerprint of length $\approx \log(L_{interval}/\epsilon)$.
2.  **Space Independence**: Since the fingerprint length is independent of the original key length $L$, the total space usage becomes $O(n \log(L_{interval}/\epsilon))$.
3.  **Expected ARE Space ($\epsilon = 0.01$)**:
    *   Fingerprint: ~14 bits.
    *   Metadata: ~3.2 bits.
    *   **Total: ~17-18 bits/key** (regardless of whether the keys are 64 or 1024 bits).

The implementation of `ApproximateRangeEmptiness` will serve as a wrapper around this `ExactRangeEmptiness` structure, managing the universe reduction (hashing) layer.
