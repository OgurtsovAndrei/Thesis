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

## 2. Theoretical vs. Practical Decisions

### Why no `LERLOC` (Weak Prefix Search) inside blocks?
The paper theoretically suggests using a Weak Prefix Search structure (like our `LERLOC`) for $O(1)$ worst-case queries inside each non-empty block. However, we've made a pragmatic engineering decision to use **binary search on bit-packed suffixes** for the local level:
*   **Space Overhead**: Adding an MMPH-based `LERLOC` to each block would add ~3–10 bits/key.
*   **Distribution Analysis**: With $n$ blocks for $n$ keys, the average number of keys per non-empty block is $\sim 2.24$ (Poisson distribution), with a maximum of ~12 observed for $N=10^6$.
*   **Performance**: Binary searching through 12 bit-packed suffixes is significantly faster and uses far less memory than computing a hash (MMPH) and performing two Rank-lookups (LERLOC).
*   **Result**: We achieved **~47 bits/key** (close to the 44-bit theoretical limit) instead of ~60 bits/key with full indexing.

## 3. Performance Characteristics (N = 1,000,000)

| Key Size ($L$) | Query Time | Bits per Key (Total) | Theoretical Min ($L - 19.93$) |
| :--- | :--- | :--- | :--- |
| **64 bits** | 141 ns | **47.27** | 44.07 |
| **128 bits** | 146 ns | **111.3** | 108.07 |
| **256 bits** | 141 ns | **239.3** | 236.07 |
| **512 bits** | 139 ns | **495.3** | 492.07 |

### Linear Space Growth (The "Exact" Limitation)
In this **Exact** structure, space grows linearly with $L$ because we must store enough information (the $W$-bit suffixes) to resolve boundaries exactly. 

**This is the key motivation for the next step: Approximate Range Emptiness (ARE).** By replacing these suffixes with hashed fingerprints, the $L$ dependence is removed, achieving $O(n \log(L/\epsilon))$ space.

## 4. Implementation Details
*   **Succinct BitVectors**: Provided by `github.com/hillbig/rsdic` for constant-time Rank and Select.
*   **Bit-Packing**: Centralized in `bits/bitpack.go`.
*   **Ordering**: The structure requires keys to be sorted lexicographically. `BitString.Compare` is the standard comparison metric.
