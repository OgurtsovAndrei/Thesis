# Monotone Minimal Perfect Hashing (MMPH)

## Overview

**Monotone Minimal Perfect Hashing (MMPH)** is a static succinct data structure that provides a bijection from a sorted
set of keys $S$ to their order indices (ranks) in the range $[0, |S|-1]$.

Unlike standard hashing, MMPH preserves lexicographic order:

$$\forall x, y \in S: x < y \implies MMPH(x) < MMPH(y)$$

## Two implementation variants

### Variant A: Time-optimized MMPH (Section 3)
- **Query time**: $O(1)$ (deterministic)
- **Space**: $O(n \log w)$ bits
- **Technique**: Bucketing with Longest Common Prefixes (LCP)
- **Use case**: When query time is critical (e.g., Range Locator)

### Variant B: Space-optimized MMPH (Section 6)
- **Query time**: $O(\log w)$
- **Space**: $O(n \log \log w)$ bits
- **Technique**: Bucketing by relative ranking + probabilistic trie
- **Use case**: When memory is critical

**Shared characteristics**:
- **Type**: Static structure (built once for an immutable dataset)
- **Build time**: $O(n \log w)$ for both variants

## API

The structure provides a minimalistic interface:

### `Build(sorted_keys)`

Builds an index over the given set of unique sorted keys.

- **Input**: Array of keys (static sorted list)
- **Complexity**: $O(n \log w)$

### `Rank(key) -> int`

Returns the order index (rank) of a key in the original set.

- **Input**: Key $x$
- **Output**: Integer $i \in [0, n-1]$
- **Complexity**: $O(1)$ for Variant A, $O(\log w)$ for Variant B

**Important**: If $key \notin S$, the result is undefined (arbitrary). Membership validation must be done externally
(e.g., Bloom Filter).

## Application

In the Range Filter architecture (Approximate Range Emptiness), MMPH is used as a low-level building block to speed up
navigation.

### Usage hierarchy

```
Approximate Range Emptiness (Top level)
            ↓
Local Exact Range Structure (Compressed key storage)
            ↓
Weak Prefix Search (Find node in implicit prefix tree)
            ↓
Range Locator (Map tree node to index interval)
            ↓
MMPH (Compute range boundaries in O(1))
```

MMPH replaces binary search ($O(\log n)$) with constant-time address computation, which is critical for overall filter
performance.

## Benchmarks & Performance

The following benchmarks compare the $O(1)$ LCP-bucketing approach (`bucket-mmph` and `rbtz-mmph`) against the space-optimized probabilistic trie (`bucket_with_approx_trie`).

### Space vs Time Trade-off

| Implementation | Query Complexity | Space Complexity | Empirical Bits/Key |
|----------------|------------------|------------------|--------------------|
| **bucket-mmph** | $O(1)$           | $O(n \log w)$    | ~42-45 bits        |
| **rbtz-mmph**   | $O(1)$           | $O(n \log n)$    | ~40 bits           |
| **approx-trie** | $O(\log w)$      | $O(n \log \log w)$| **~15 bits**      |

### Performance Visuals

#### Memory Usage
![Bits per Key](benchmarks/plots/bits_per_key_in_mem.svg)
The probabilistic trie variant (`bucket_with_approx_trie`) reduces memory footprint by ~60% compared to classical bucketing methods.

#### Lookup Latency
![Lookup Time](benchmarks/plots/lookup_time_ns.svg)
While $O(1)$ methods are faster, the $O(\log w)$ overhead of the approximate trie is minimal for practical key lengths ($w=64$).

## References

### Core algorithm (Bucketing with LCP)
- [Monotone Minimal Perfect Hashing: Searching a Sorted Table with O(1) Accesses](https://vigna.di.unimi.it/ftp/papers/MonotoneMinimalPerfectHashing.pdf) - Belazzougui D., Boldi P., Pagh R., Vigna S. (Section 3)

### Context (Weak Prefix Search)
- [Fast Prefix Search in Little Space, with Applications](https://arxiv.org/abs/1804.04720) - Belazzougui D., Boldi P., Pagh R., Vigna S.

### Parent problem (Approximate Range Emptiness)
- [Approximate Range Emptiness in Constant Time and Optimal Space](https://arxiv.org/pdf/1407.2907) - Goswami M., Gronlund A., Larsen K. G., Pagh R.
