# Speaker Notes — Thesis Defence

---

## Slide 0 — Title + Talk Arc (0:40)

**Example text (~25s):**
> "The topic of my thesis is Range Emptiness Filters.
> A range filter is a compact structure that tells whether a given range intersects the stored set.
> The approximate version allows false positives but no false negatives — which makes it much smaller in memory."

---

## Slide 1 — Teaser: Three Headline Results (0:30)

[//]: # (todo: data sizes are limited, we choose structure which performs best on real-world data)

**Example text (~30s):**
> Today I will show how optimizing the structure's memory layout
> lets us reduce metadata size and speed up filter queries.
> And on top of that, how we built the Segmented-ARE filter, which achieves better FPR than competing algorithms on some datasets like FB.

---

## Slide 2 — LSM-tree (Log-Structured Merge-Tree) (0:50)

**Example text (~50s):**
> Af first I will tell you how the Range filters are used in LSM-trees:
>
> LSM is a write-optimized data structure used in modern DBs.
> In an LSM-tree, keys are stored in different levels in immutable sorted files — SSTables.
> File sizes grow geometrically across levels. Inserts go into a memtable in RAM; when it fills up,
> it is flushed to disk as a new SSTable at level zero. When too many files accumulate at a level,
> they are compacted in the background: merged together and moved to the next level.
> Because of sequential I/O, all disk writes are fast.
>
> On reads the picture is different: a lookup must touch every SSTable in the tree.

---

## Slide 3 — Reducing Disk I/O with Filters (1:15)

**Example text (~45s):**
> To avoid reading every SSTable from disk during a lookup, each file's metadata contains a filter.
> It answers with a one-sided error: no false negatives, false positives are allowed.
> That is enough — if the filter says "no", the database skips the SSTable without any disk I/O.
> One of the widely used structures is the Bloom filter, which answers point queries.


---

## Slide 4 — Problem and Goswami Baseline (1:30)

**Example text (~75s):**
> Range filter is the structure for range queries. Instead of answering "does the set contain the key?"
> it answers "does the range [a, b] intersect the set?".
> The same guarantees: no false negatives, false positive rate at most ε.
> That is exactly the filter I build.
>
> Goswami proved information-theoretic lower bound on the memory usage for any ARE filter.
> 
> As for the computational complexity, this data structure answers intersection queries in constant time.
> Their construction:
> - a locality-preserving hash that compresses the universe, and
> - an exact range emptiness filter (ERE) which answers intersection queries exactly.
>
> Asymptotically optimal — but the constants behind the asymptotics matter.

---

## Slide 5 — Exact Range Emptiness as Elias-Fano (0:30)


**Example text (~25s):**
> ERE splits each key into a prefix and a suffix.
> Suffixes are grouped into buckets; prefixes drive the metadata that navigates to the right bucket.
> Goswami stores metadata as two separate vectors, We managed to collapse them into one, these gives us better mem layout.
> In bucket search also was optimized, let's explain how.

---

## Slide 6 — Succinct Backend (1:00) — core slide

**Example text (~60s):**

> For search in the bucket, Goswami proposed Weak Prefix Search — which works in "constant" number of steps, 
> however each step requires hash calculation and pointer chasing, which are expensive operations.
> 
[//]: # (> translates to nanoseconds depends on two factors:)
[//]: # (> &#40;1&#41; working set size — whether the structure fits in CPU caches;)
[//]: # (> &#40;2&#41; memory access pattern — sequential or random.)
>
> Look at the table: on small bucket size — WPS fits in L1, still 307 ns. 
> Larger buckets escape L1 — 514–556 ns.
> 
> On all these cases WPS does same 
> 
> 
> Usually, DB tries to keep SSTable size manageable;
> OR
> The practical dataset sizes are limited.
> Thus, buckets sizes are typically limited by thousands of keys.
> 
> At these bucket sizes Adaptive binary/linear search outperforms WPS by 13–30×,
> with no auxiliary index required.
>
> Also, Goswami stored two vectors with metadata D₁+D₂.
> I replaced them with a single contiguous vector D — saving 24% of metadata.
> Contiguous vector also means more sequential reads — 
> less pointer chasing between two separate structures.
> Hardware performance counters confirm this:
> 14 to 56% fewer L1 cache misses across real SOSD distributions.

---

## Slide 7 — Real Distributions Often Happen To Be Clustered (0:40) — bridge

**Example text (~40s):**
> The performance of approximate range filters depends on two aspects:
> the ERE backend performance, and the choice of locality-preserving hash.

[//]: # (> that reduces the approximate emptiness problem to multiple exact emptiness problems.)

> On the previous slide I covered the ERE backend.
> Now let's discuss how to choose the locality-preserving hash based on the data distribution.
>
> Goswami's bound works for any distribution of keys.
> But in many practical workloads, keys tend to be clustered. [point to histogram]
> File paths, URLs, S2 cell IDs — dense clusters alternate with sparse regions.
>
> The idea: divide the universe into parts — dense clusters and sparse gaps — and handle them differently.

---

## Slide 8 — Two Mode: Dense Clusters and Sparse Tails (~0:55)

**Example text (~55s):**
> There are two different scenarios.
>
> Dense cluster: many keys in a narrow window.
> Goswami's bound for any approximate range filter — is independent of U.
> But, ERE cost — shrinks with the size of the Univ.
>
> Thus, if keys are dense enough, 
> It costs fewer bits to build ERE and this will give us zero false positive rate.
> This is exact mode.
>
> "In sparse regions where we have very few points, we can simply truncate some lower bits
> to obtain our locality-preserving hash. And this hash will still have few collisions."

---

## Slide 9 — Detection: 1D-DBSCAN on Sorted Keys (~0:40)

**Example text (~40s):**
> One question remains: how do we find the partition?
> We apply the **DBSCAN clustering** algorithm.
> 
> For this we apply **1D-DBSCAN**.
> **1D-DBSCAN** is a clustering algorithm that runs on sorted one-dimensional keys in linear time.
> The division threshold is derived from the filter parameters.
>
> On Each dense cluster we will build Exact filter.
> While, sparse gaps that DBSCAN marks as noise go to the truncation fallback.

---

## Slide 10 — Headline Result: SOSD Facebook, ℒ=128, n=2²⁴ (1:15)

**Example text (~55s):**
> Headline result.
> The graph shows the FPR vs BPK trade-off.
> SNARF, SuRF and Rosetta are other solutions to the same ARE problem.
> The structures are built on Facebook user IDs from the SOSD (Search on Sorted Data) dataset.
> Segmented-ARE (purple line) reaches zero FPR at 11 bits per key.
> 
> At the same time, thanks to all the CPU optimizations we made, 
> we not only reach the record FPR but also have much lower query latency.

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
> We achieved almost 0 FPR at 11 bits per key on Facebook user IDs — using Segmented-ARE.
> Future work: dynamic updates, Partial Elias-Fano encoding, RocksDB integration. Thank you.
