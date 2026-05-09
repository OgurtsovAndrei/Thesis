# Speaker Notes — Thesis Defence

---

## Slide 0 — Title + Talk Arc (0:40)

**Example text (~35s):**
> "The topic of my thesis is Range Emptiness Filters.
> A range filter answers one question: does the range [a, b] intersect the set of keys?
> Today I will show why LSM trees need filters and how to build them well."

---

## Slide 1 — LSM-tree (Log-Structured Merge-Tree) (0:50)

**Example text (~50s):**
> Range emptiness filters are used in LSM-trees.
>
> LSM is a write-optimized data structure used in RocksDB, LevelDB, Cassandra.
> In an LSM-tree, keys are stored in different levels in immutable sorted files — SSTables.
> File sizes grow geometrically across levels. Inserts go into a memtable in RAM; when it fills up,
> it is flushed to disk as a new SSTable at level zero. When too many files accumulate at a level,
> they are compacted in the background: merged together and moved to the next level.
> Because of sequential I/O, compaction is fast.
>
> On reads the picture is different: a lookup must touch every SSTable in the tree.

---

## Slide 2 — Reducing Disk I/O with Filters (1:15)

**Example text (~45s):**
> To avoid reading every SSTable from disk during a lookup, each file's metadata contains a filter.
> It answers with a one-sided error of at most ε: no false negatives, false positives are allowed.
> That is enough — if the filter says "no", the database skips the SSTable without any disk I/O.
> One of the widely used structures is the Bloom filter, which answers point queries.
>
> *(flip slide)*

---

## Slide 3 — Problem and Goswami Baseline (1:30)

**Example text (~75s):**
> Range filter is the structure for range queries. Instead of answering "does the set contain the key?"
> it answers "does the range [a, b] intersect the set?".
> The same guarantees: no false negatives, false positive rate at most ε.
> That is exactly the filter I build.
>
> Formally: given a set S of n keys, maximum query length ℒ, target false positive rate ε.
>
> Goswami proved an information-theoretic lower bound for any S  bits per key —
> and constructed a data structure that reaches this lower limit up to a constant.
> As for the computational complexity, this data structure answers intersection queries in constant time.
> Their construction:
> - a locality-preserving hash that compresses the universe, and
> - an exact range emptiness filter (ERE) which answers intersection queries exactly.
>
> Asymptotically optimal — but the constants behind the asymptotics matter.

---

## Slide 5 — Exact Range Emptiness as Elias-Fano (1:00)

**Example text (~55s):**
> To keep things simple, we will only consider an example on this slide.
> Here we have eight 8-bit keys.
> We split each key into a 3-bit prefix (log 8) and a 5-bit suffix — the prefix determines the bucket,
> the suffix is stored in it. All suffixes live in a packed array A.
>
[//]: # (> This is the Elias-Fano encoding of the sorted key set.)
> For navigation we need to know where each bucket starts.
>
> To answer a range query :
> The intersection of the range with S reduces to checking at most two boundary buckets.
> We navigate to them using Select(D, i) in O(1), then run adaptive binary/linear search
> on packed suffixes inside each bucket.
> Goswami uses Weak Prefix Search instead.
>
> Goswami instead stored two vectors:
> - D₁ — which buckets are non-empty,
> - D₂ — sizes of non-empty buckets in unary.
    > Navigation used both D₁ and D₂ together.
    > I replaced them with one succinct vector which encodes bucket lengths in unary, including empty buckets.


---

## Slide 6 — Succinct Backend (1:00) — core slide

**Example text (~60s):**
> How do these two improvements work on the ERE backend?
>
> First: Goswami stored two-level metadata D₁+D₂.
> I replaced them with a single contiguous vector D — saving 24% of metadata.
>
> For search in the bucket, Goswami proposed Weak Prefix Search — which works in O(1) in theory.
> But how O(1) translates to nanoseconds depends on two factors:
> (1) working set size — whether the structure fits in CPU caches;
> (2) memory access pattern — sequential or random.
>
> The practical dataset sizes are limited by 2²⁴ or 2²⁸ keys.
> Thus, buckets sizes are also limited by thousands of keys.
>
> Look at the table: Facebook median [k=27] — WPS fits in L1, still 307 ns. 
> Larger buckets escape L1 — 514–556 ns. 
> At these bucket sizes Adaptive binary/linear search outperforms WPS by 13–30×,
> with no auxiliary index required.
>
> The same principle was used in D vector replacement - one contiguous vector
> means more sequential reads — less pointer chasing between two separate structures.

---

## Slide 7 — Real Distributions Often Happen To Be Clustered (0:40) — bridge

**Example text (~40s):**
> The performance of approximate range filters depends on two aspects:
> the ERE backend performance, and the choice of locality-preserving hash that reduces
> the approximate emptiness problem to multiple exact emptiness problems.
> On the previous slide I covered the ERE backend.
> Now let's discuss how to choose the locality-preserving hash based on the data distribution.
>
> Goswami's bound works for any distribution of keys.
> But in many practical workloads, keys tend to be clustered. [point to histogram]
> File paths, URLs, S2 cell IDs — dense clusters alternate with sparse regions.
>
> The idea: divide the universe into parts — dense clusters and sparse gaps — and handle them differently.

---

## Slide 8 — Two Regimes: Dense Clusters and Sparse Tails (~0:55)

**Example text (~55s):**
> There are two different scenarios.
>
> Dense cluster: many keys in a narrow window.
> Goswami's bound for any approximate range filter — is independent of U.
> But, ERE cost — shrinks with the size of the Univ.
>
> If the compressed universe in Goswami's hash is larger than the original cluster window,
> we don't need a hash at all — we build an exact range emptiness filter directly on the window.
> It costs fewer bits per key and gives zero false positive rate.
> This is exact mode.
>
> "In sparse regions where we have very few points, we can simply truncate some lower bits
> to obtain our locality-preserving hash. And this hash will still have few collisions."

---

## Slide 9 — Detection: 1D-DBSCAN on Sorted Keys (~0:40)

**Example text (~40s):**
> One question remains: how do we find the partition?
> We apply the DBSCAN clustering algorithm.
> On sorted one Dimential keys, DBSCAN runs in linear time.
> The division threshold is derived from the filter parameters.
>
> Each dense cluster becomes an ERE filter.
> Sparse points that DBSCAN marks as noise go to the truncation fallback.

---

## Slide 10 — Headline Result: SOSD Facebook, ℒ=128, n=2²⁴ (1:15)

**Example text (~55s):**
> Headline result.
> The graph shows the FPR vs BPK trade-off.
> SNARF, SuRF and Rosetta are other solutions to the same ARE problem.
> The structures are built on Facebook user IDs from the SOSD (Search on Sorted Data) dataset.
> The n = 2²⁴ — the same scale at which SuRF shows its best results.
> DB-scan Range Filter (purple line) reaches zero FPR at 11 bits per key.
> - SNARF saturates at 10⁻3 — no matter how much memory you give it, it cannot go lower.
> - SuRF uses 19 BPK.
> - Rosetta just struggles.
> 
> At the same time, благодаря всем CPU оптимизациям которые мы сделали, мы не только достигаем рекордный FPR но 
> we also have much lower query latency.
> 
---

## Slide 11 — Limitations: Uniform, ℒ=1, n=2²⁴ (0:30)

**Example text (~30s):**
> Honest limitation: where we do not win.
> Uniform keys, point queries.
> No clusters to segment and no ranges to benefit from truncation.
> We pay the segmentation overhead without anything to segment.
> Our advantage - working with the real data.

---

## Slide 12 — Conclusion: Three Results (1:00)

**Example text (~50s):**
> Three main results.
> We reduced metadata usage by 24% — through single-vector Elias-Fano ERE.
> We reduced query latency by 13–30× — by replacing WPS with adaptive binary/linear search.
> We achieved almost 0 FPR at 11 bits per key on Facebook user IDs — using DB-scan Range Filter.
> Future work: dynamic updates, Partial Elias-Fano encoding, RocksDB integration. Thank you.
