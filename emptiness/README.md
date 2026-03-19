# Range Emptiness Filters

Data structures for answering 1D range emptiness queries: **"Does $[a, b]$ contain any keys from $S$?"**

Based on: *["Approximate Range Emptiness in Constant Time and Optimal Space"](https://arxiv.org/abs/1407.2907)*,
Goswami, Pagh, Silvestri, Sivertsen (SODA 2015).

> **Note:** To the best of our knowledge (as of March 2025), this is the first practical implementation
> of the SODA 2015 algorithm. A search of GitHub, artifact links in citing papers
> ([Grafite](https://arxiv.org/abs/2311.15380), [Rosetta](https://dl.acm.org/doi/10.1145/3389400.3389431),
> [SuRF](https://dl.acm.org/doi/10.1145/3183713.3196759)), and the authors' own repositories found
> no public implementation. The paper is widely cited as the theoretical gold standard for range filters
> (e.g., Grafite, SIGMOD 2024 calls it "the information-theoretically optimal solution"),
> but the complexity of the underlying machinery (succinct bit vectors, Elias-Fano coding,
> monotone minimal perfect hashing, hollow tries) appears to have prevented prior implementations.

**Use case:** Range filters for LSM-tree key-value stores (e.g., RocksDB). Before reading an SST file from disk, the
filter answers whether a queried key range *might* intersect the file's key set, avoiding unnecessary I/O with a bounded
false positive probability.

## Problem Statement

Let $U = [0, 2^L)$ be a universe of $L$-bit keys. We are given:

- A sorted set $S \subset U$ of $n$ keys stored in the structure.
- A maximum query range length $\mathcal{L}$.
- A target false positive rate $\varepsilon$.

Let $Y = U \setminus S$ be all keys **not** in the structure.

The goal: build a data structure that answers "$[a, b] \cap S = \emptyset$?" for intervals of length $\leq \mathcal{L}$,
with:

- **No false negatives:** if $[a, b] \cap S \neq \emptyset$, always answer "not empty".
- **Bounded false positives:** if $[a, b] \cap S = \emptyset$, answer "not empty" with probability $\leq \varepsilon$.

## Asymptotics ([see paper](https://arxiv.org/pdf/1407.2907))

**Lower bound (§2):** any data structure solving this problem requires at
least $n \log_2(\mathcal{L} / \varepsilon) - O(n)$ bits.

**Upper bound (§3):** achieved via two layers:

1. **Locality-preserving hash** $h: U \to U'$ where $|U'| = r = n\mathcal{L}/\varepsilon$.
   A hash is locality-preserving if it maps any interval $[a,b]$ in $U$ to a bounded number
   of contiguous intervals in $U'$ — ensuring that range queries are always fully checked,
   preventing false negatives.
   Projects $S \mapsto S' = h(S)$ and $[a,b] \mapsto h([a,b])$.

2. **Exact Range Emptiness (ERE)** over $S' \subset [r]$: succinct structure with zero error and
   space $n \log_2(r/n) + O(n)$ bits.

**Resulting space:** $n \log_2(\mathcal{L}/\varepsilon) + O(n)$ bits = **$\log_2(\mathcal{L}/\varepsilon) + O(1)$ bits
per key** — matching the lower bound.

The fingerprint length $K = \log_2(r) = \log_2(n\mathcal{L}/\varepsilon)$ controls accuracy: increasing $K$ by 1 bit
halves the FPR. After ERE compression (subtracting $\log_2 n$ for block indexing), the effective cost per key
is $K - \log_2 n = \log_2(\mathcal{L}/\varepsilon)$.

## The Role of the Hash Function

The hash $h$ projects the universe $U$ down to $U' = [r]$. Under this projection:

- $S$ maps to $S' = h(S)$ — the stored fingerprints.
- $Y = U \setminus S$ maps to $Y' = h(Y)$ — fingerprints of non-keys.

**A false positive occurs when $Y'$ overlaps with $S'$:** a query point $y \in Y$ has $h(y) \in S'$, so the structure
cannot distinguish it from a real key. We call the pre-images of a stored fingerprint $x' \in S'$ its **phantom
points** — all keys $y$ such that $h(y) = x'$. The fewer phantoms overlap with $Y'$, the lower the FPR.

The ideal hash would map $S'$ and $Y'$ to completely disjoint regions of $U'$ — zero overlap, zero false positives. In
practice this is impossible within the space budget, but **the choice of $h$ determines how well $S'$ and $Y'$ separate
for a given data distribution.**

This is why we experiment with different hash functions: each makes different trade-offs between space, speed, and how
well it separates $S'$ from $Y'$ across different key distributions. See each package's README for details.

## Packages

ERE stores a compressed sorted set and answers "$[a,b] \cap S = \emptyset$?" with zero error.
All approximate ARE packages use ERE as their inner layer, storing hashed fingerprints instead of raw keys.

Recommended reading order (top to bottom):

| Package                                        | Description                                                          |
|------------------------------------------------|----------------------------------------------------------------------|
| [`ere`](ere/README.md)                         | Exact Range Emptiness (Layer 2). Zero error, space $O(n \log(U/n))$. |
| [`are_soda_hash`](are_soda_hash/README.md)     | ARE via the SODA 2015 pairwise-independent hash (paper baseline).    |
| [`are_pgm`](are_pgm/README.md)                 | ARE via CDF mapping (PGM index). Experimental.                       |
| [`are_trunc`](are_trunc/README.md)             | ARE via prefix truncation.                                           |
| [`are_adaptive`](are_adaptive/README.md)       | ARE via adaptive prefix truncation with threshold $t$.               |
| [`are_hybrid`](are_hybrid/README.md)           | ARE with per-cluster segmentation (gap-percentile).                  |
| [`are_hybrid_scan`](are_hybrid_scan/README.md) | **Best implementation.** 1D DBSCAN segmentation + dual fallback.     |
| [`are_bloom`](are_bloom/README.md)             | Bloom filter baseline.                                               |

### Advanced topics

The following packages require familiarity with the [SODA 2015 paper](https://arxiv.org/abs/1407.2907)
and the references therein (succinct bit vectors, Elias-Fano coding, monotone minimal perfect hashing,
hollow tries, Z-fast tries):

| Package                                                   | Description                                                 |
|-----------------------------------------------------------|-------------------------------------------------------------|
| [`lerloc`](../locators/lerloc/)                           | LERLOC — Range Locator via Weak Prefix Search (MMPH + Hollow Z-Fast Trie). |
| [`ere_theoretical`](ere_theoretical/)                     | Theoretical ERE baseline using LERLOC for $O(1)$ worst-case queries. |
