# Streaming Build: Iterator-Based Filter Construction

In LSM-tree databases (RocksDB, LevelDB, Pebble, etc.), SSTables are sorted files.
During compaction, keys are merged via **iterators** — each key is seen exactly once,
in sorted order. No random access, no full materialization in memory.

This document analyzes which ARE filters can be built from a streaming iterator:

```go
type KeyIterator interface {
Next() (key []byte, ok bool) // sorted ascending, one pass
}
```

SSTable metadata (footer/properties) provides $n$, $\min(S)$, $\max(S)$ before
iteration begins — these are available as build parameters.

## Two Modes of Streaming

### Mode 1: Pure Streaming

The filter is built while iterating over original keys. Only $O(1)$ or
$O(\text{segment\_size})$ memory beyond the filter itself.

### Mode 2: Compact Hash Buffer

Original keys (variable-length strings, 50–200 bytes in practice) are hashed
during iteration into $K$-bit fingerprints ($K = \lceil \log_2(n\mathcal{L}/\varepsilon) \rceil$,
typically 27–38 bits ≈ 4–5 bytes). The buffer of hashed keys is sorted after
iteration, then passed to [ERE](ere/README.md).

Memory: $O(n \cdot K/8)$ bytes — **20–50× smaller** than buffering original keys.

| $n$ | $\mathcal{L}$ | $\varepsilon$ | $K$ (bits) | Buffer per key | Buffer for $n$ keys |
|-----|---------------|---------------|------------|----------------|---------------------|
| 64K | 16            | 0.01          | 27         | 4 B            | 256 KB              |
| 1M  | 16            | 0.01          | 31         | 4 B            | 4 MB                |
| 16M | 128           | 0.01          | 38         | 5 B            | 80 MB               |

---

## Per-Filter Analysis

### [`are_bloom`](are_bloom/README.md) — Pure Streaming

**Build process:** iterate keys, call `bf.Add()` for each. No random access,
no look-ahead, no re-sorting.

**Requirements:** $n$ upfront (for Bloom filter sizing).

**Verdict:** trivially streamable. $O(\text{filter\_size})$ memory.

---

### [`are_trunc`](are_trunc/README.md) — Pure Streaming

**Build process** ([`are_trunc.go:39–82`](are_trunc/approximate_range_emptiness.go)):

1. Compute `spread = maxKey - minKey`, derive `spreadStart`.
2. Single forward pass: for each key, `normalizeToK(key, minKey, spreadStart, K)` → deduplicate → collect.
3. Build [ERE](ere/README.md) on truncated keys.

**Why it streams:** normalization is a pointwise operation on each key. Deduplication
only compares with the previous truncated key. No random access.

**ERE itself is streaming** ([`ere.go:50–72`](ere/exact_range_emptiness.go)):
iterates sorted keys left-to-right, `PushBack` bits into D1/D2
(RSDic supports sequential append), collects packed suffixes. Single pass.

**Requirements:** $\min(S)$, $\max(S)$ upfront.

**Verdict:** single-pass, $O(1)$ working memory beyond ERE output.

---

### [`are_greedy_scan`](are_greedy_scan/) — Pure Streaming ★

**Build process** ([`segment.go:27–65`](are_greedy_scan/segment.go), [
`greedy_scan_are.go:75–132`](are_greedy_scan/greedy_scan_are.go)):

1. **Greedy segmentation** — single forward scan: track segment start value,
   split when `curVal - startVal > 2^K`. Pure $O(n)$ streaming.
2. **Merge pass** — operates on segment metadata only (count, min, max).
   $O(\text{num\_segments})$, not $O(n)$.
3. Per exact-mode segment: build [Adaptive ARE](are_adaptive/README.md)
   in exact mode (streamable, FPR = 0).
4. Wide-spread segments → [Truncation ARE](are_trunc/README.md) fallback (streamable).

**Why it streams:** greedy segmentation is inherently an iterator pattern — compare
current key with segment start, decide split. Each segment's keys are buffered only
until the segment closes (spread exceeds threshold), then the filter is built and
the buffer is freed.

**Peak memory:** $O(\max\_\text{segment\_size})$ — for clustered data (typical SSTable),
segments are much smaller than total $n$.

**Requirements:** $\min(S)$, $\max(S)$ upfront (for Truncation fallback).

**Verdict: best candidate for SSTable compaction.**
Naturally streaming, minimal memory, benefits from data clustering.

A streaming builder API would look like:

```go
type GreedyScanBuilder struct {
K         uint32
rangeLen  uint64
maxSpread uint64
segBuf    []bits.BitString // current segment buffer
segStart  uint64
segRefs   []segmentRef    // completed segment metadata
built     []clusterFilter // already-built segment filters
}

func (b *GreedyScanBuilder) Add(key bits.BitString) error { ... }
func (b *GreedyScanBuilder) Finish() (*GreedyScanARE, error) { ... }
```

---

### [`are_adaptive`](are_adaptive/README.md) — Depends on Mode

**Build process** ([`are_adaptive.go:66–159`](are_adaptive/are_adaptive.go)):

1. Compute spread $M = \lceil \log_2(\max(S) - \min(S)) \rceil$.
2. **Exact mode** ($M \leq K$): normalize keys → build ERE directly.
   **Streamable** — same as Truncation.
3. **SODA mode** ($M > K$): hash each key with pairwise-independent hash →
   `SortAndDedup(hashedKeys)` → build ERE.
   **Not streamable** — hash destroys sort order, requires re-sort.

**With compact hash buffer (Mode 2):** SODA mode becomes practical.
Hash each key during iteration into a $K$-bit value (4–5 bytes), buffer,
sort after iteration. $O(n \cdot K/8)$ memory.

**Requirements:** $\min(S)$ upfront.

**Verdict:**

- Exact mode — pure streaming, $O(1)$.
- SODA mode — needs compact hash buffer, $O(n \cdot K/8)$.

---

### [`are_soda_hash`](are_soda_hash/README.md) — Compact Hash Buffer Only

**Build process** ([`are_soda_hash.go:36–81`](are_soda_hash/are_soda_hash.go)):

1. Hash each key: `hx = (PairwiseHash(blockIdx, a, b, K) + x) mod 2^K`.
2. `SortAndDedup(hashedKeys)`.
3. Build ERE.

**Why pure streaming fails:** the pairwise hash `(u(⌊x/r⌋) + x) mod r` applies a
per-block cyclic shift. Keys from different blocks interleave unpredictably in the
hashed domain — sort order is destroyed.

**With compact hash buffer:** hash during iteration (4–5 bytes/key), sort after.
Practical for typical SSTable sizes.

**Verdict:** requires compact hash buffer, $O(n \cdot K/8)$.

---

### [`are_hybrid`](are_hybrid/README.md) — Not Streamable

**Build process** ([`cluster_detect.go:25–94`](are_hybrid/cluster_detect.go)):

1. **Cluster detection:** compute all $n-1$ gaps between consecutive keys.
   Quickselect on the gap array to find the 95th-percentile threshold.
2. Split at gaps $> \tau$, classify by size.
3. Per cluster: build Adaptive ARE. Fallback: Truncation ARE.

**Why it fails:** quickselect requires the full gap array in memory — $O(n \cdot 8)$
bytes of `uint64` values. Unlike hashed fingerprints, gaps cannot be made more compact
(they are already `uint64`). The gap array is the same size as the original key array.

**Why compact hash buffer doesn't help:** the bottleneck is cluster detection
(before any hashing), not the per-cluster filter build.

**Verdict:** not streamable. Use [Greedy+Merge](are_greedy_scan/) instead —
it achieves the same goal (segmentation + exact clusters) with a streaming-compatible
greedy scan instead of percentile-based detection.

---

### [`are_hybrid_scan`](are_hybrid_scan/README.md) — Not Streamable

**Build process** ([`dbscan_detect.go:29–144`](are_hybrid_scan/dbscan_detect.go)):

1. **Forward sweep:** two-pointer scan to identify core points.
2. **Backward sweep:** reverse two-pointer scan — needs the full key array
   and traverses it right-to-left.
3. Cluster formation, border expansion, size filtering.

**Why it fails:** the backward sweep is fundamentally incompatible with a
single-pass forward iterator. It requires bidirectional access to the key array.

**Verdict:** not streamable. The DBSCAN algorithm requires $O(n)$ memory
and two passes over the data.

---

### [`are_pgm`](are_pgm/README.md) — Not Streamable

**Build process** ([`are_pgm.go:59–189`](are_pgm/are_pgm.go)):

1. Build PGM index on all keys (needs full key array).
2. Query PGM for each key's position.
3. Monotonicity fix (running max — forward pass, OK).
4. CDF sampling, max density computation over all CDF segments.
5. Map keys through CDF, build ERE.

**Why it fails:** PGM index construction is inherently batch — it computes
a piecewise-linear approximation of the CDF that requires seeing all data
points. Max density computation also requires all CDF points.

**Verdict:** not streamable. PGM is a batch algorithm.

---

## Summary

| Filter                                     | Pure Streaming | Compact Hash Buffer | Peak Memory         | SSTable-Ready |
|--------------------------------------------|:--------------:|:-------------------:|---------------------|:-------------:|
| [BloomARE](are_bloom/README.md)            |       ✓        |          —          | $O(\text{filter})$  |       ✓       |
| [Truncation](are_trunc/README.md)          |       ✓        |          —          | $O(1)$ + ERE output |       ✓       |
| [**Greedy+Merge**](are_greedy_scan/)       |     **✓**      |          —          | $O(\text{segment})$ |     **★**     |
| [Adaptive (exact)](are_adaptive/README.md) |       ✓        |          —          | $O(1)$ + ERE output |       ✓       |
| [Adaptive (SODA)](are_adaptive/README.md)  |       ✗        |          ✓          | $O(n \cdot K/8)$    |       ✓       |
| [SODA Hash](are_soda_hash/README.md)       |       ✗        |          ✓          | $O(n \cdot K/8)$    |       ✓       |
| [Hybrid (gap)](are_hybrid/README.md)       |       ✗        |          ✗          | $O(n \cdot 8)$ gaps |       ✗       |
| [Hybrid Scan](are_hybrid_scan/README.md)   |       ✗        |          ✗          | $O(n)$ bidir        |       ✗       |
| [CDF-ARE (PGM)](are_pgm/README.md)         |       ✗        |          ✗          | $O(n)$ batch        |       ✗       |

### Recommendation for SSTable Integration

**[Greedy+Merge](are_greedy_scan/)** is the natural fit for SSTable compaction:

1. Greedy segmentation is already a forward scan — it maps directly to an iterator.
2. Each segment is buffered only until it closes, then its filter is built and memory freed.
3. Merge operates on $O(\text{segments})$ metadata, not $O(n)$ keys.
4. Dense clusters (typical in real-world SSTables) trigger exact mode (FPR = 0).
5. The Truncation fallback for wide-spread segments is also streaming.

**Alternative:** if distribution-independent FPR guarantees are needed (adversarial
workloads), [SODA Hash](are_soda_hash/README.md) with compact hash buffer provides
a predictable $O(n \cdot 4\text{–}5\ \text{bytes})$ build at any distribution.

### Required Metadata from SSTable

All streaming-compatible filters need these values before iteration begins:

| Parameter                    | Source                         | Used By                               |
|------------------------------|--------------------------------|---------------------------------------|
| $n$ (key count)              | SSTable properties / footer    | All (ERE block sizing, Bloom sizing)  |
| $\min(S)$                    | SSTable properties / first key | Truncation, Adaptive, Greedy fallback |
| $\max(S)$                    | SSTable properties / last key  | Truncation, Greedy fallback           |
| $\mathcal{L}$ (range length) | Configuration                  | Adaptive, SODA, Greedy                |
| $\varepsilon$ (target FPR)   | Configuration                  | All                                   |
