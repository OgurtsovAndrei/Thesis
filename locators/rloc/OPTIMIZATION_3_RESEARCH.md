# Research: Asymptotic Memory Optimization for HZFT (Optimization 3)

This document explores the feasibility, theoretical background, and practical Go implementation of the asymptotic $O(N \log \log L)$ space optimization for the Heavy Z-Fast Trie (HZFT), as described in Section 3.2 of the "Fast Prefix Search" paper.

## 1. The Core Problem with Current HZFT

Currently, HZFT generates **Descriptors** (true internal nodes) and **Pseudo-descriptors** (fake nodes to guide Fat Binary Search). 
- For $N$ keys, there are $N-1$ true descriptors.
- For $L=64$, it generates $\approx 4.5 	imes N$ pseudo-descriptors.
- Total entries in MPH = $\approx 5.5 	imes N$.

**Current Memory Layout (64 bits/key):**
1. `BoomPHF` indexes all $5.5N$ entries $	o \approx 19$ bits/key.
2. `[]HNodeData[E]` stores a 1-byte (`uint8`) extent length for **all** $5.5N$ entries $	o 44$ bits/key.
   - True descriptors store their actual length ($0..64$).
   - Pseudo-descriptors store `255` (infinity).

The massive inefficiency here is that we are allocating a full byte of data for $4.5N$ pseudo-descriptors just to store the constant value `255`.

## 2. The Theoretical Solution: Relative Dictionary

The paper states:
> "We can store the internal node descriptors in a relative dictionary. The dictionary will store $n-1$ strings out of a universe of $O(n \log l)$ strings, using $O(n \log \log l)$ bits."

**How it works mathematically:**
Instead of storing $5.5N$ values, we separate the "routing" (topology) from the "payload" (extent lengths).
1. We build an MPH over all $5.5N$ strings.
2. We maintain a Bitvector `B` of length $5.5N$. We set `B[i] = 1` if the $i$-th MPH entry is a true descriptor, and `0` if it's a pseudo-descriptor.
3. We add a `Rank` index to `B`.
4. We maintain a dense `data` array of size exactly $N-1$.
5. To query a prefix:
   - $idx \gets MPH(prefix)$
   - If `B[idx] == 0`, return $\infty$ (it's a pseudo-descriptor).
   - If `B[idx] == 1`, return `data[Rank(B, idx)]`.

## 3. Further Compression: Relative Mapping ($\Delta$)

The paper also suggests compressing the payload:
> "The mapping $h \mapsto |e|$ can be reformulated to $h \mapsto |e| - |h|$."

For a descriptor $h$, its extent $e$ is strictly longer than $h$. The difference $\Delta = |e| - |h|$ is often very small. 
Instead of storing the absolute length in a full `uint8` or `uint16`, we can bit-pack $\Delta$. Since $L=64$, $\Delta$ strictly requires $\le 6$ bits. For $L=1024$, it requires $\le 10$ bits.

## 4. Projected Memory Savings in Go

Let's calculate the expected memory of this new architecture for $N=32,768, L=64$:

| Component | Calculation | Expected Bits/Key |
| :--- | :--- | :--- |
| **MPH (BoomPHF)** | Indexes $5.5N$ strings | **$\approx 19.5$ bits** |
| **Bitvector `B`** | $5.5N$ bits long | **$5.5$ bits** |
| **Rank Index** | `rsdic` overhead (~25% of bits) | **$\approx 1.5$ bits** |
| **Dense Data Array**| $N-1$ entries $	imes 6$ bits (bit-packed) | **$\approx 6.0$ bits** |
| **Total Expected** | | **$\approx 32.5$ bits/key** |

**Conclusion:** By implementing the Relative Dictionary and $\Delta$-packing, we can cut the HZFT memory consumption exactly in half (from ~64 bits/key down to ~32 bits/key). For $L=1024$, the savings are even more dramatic (dropping from ~186 bits/key to ~50 bits/key).

## 5. Feasibility and Trade-offs

### Pros:
- **Massive Memory Reduction**: Achieves the theoretical $O(N \log \log L)$ bound.
- **Solves the Long Key Paradox**: Makes LERLOC highly viable for long strings ($L \ge 1024$) by capping the payload array size strictly at $N-1$.

### Cons (Performance Impact):
- **Query Latency**: Currently, `GetExistingPrefix` does an MPH lookup + array access ($O(1)$). With this optimization, it must do:
  1. MPH lookup.
  2. Bitvector check.
  3. `Rank` operation on the bitvector (involves popcount on multiple cache lines).
  4. Bit-unpacking from the dense array.
  5. Addition ($|h| + \Delta$).
- The `Rank` operation inside the tight `while` loop of the Fat Binary Search (which executes $\approx \log L$ times) will likely cause a **$2x - 3x$ performance degradation** in query speed.

## 6. Final Recommendation

Is this optimization worth it? 
**Yes, but as a configurable toggle.**

If the primary goal of LERLOC is to be the fastest possible exact range locator, the current 64 bits/key implementation is ideal. However, if the goal is to compete for the "smallest possible index footprint" (especially for long keys), implementing the Relative Dictionary + Bit-packed $\Delta$ array is the mathematically proven path forward.
