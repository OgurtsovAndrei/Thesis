# Observations from B6 latency / FPR sweep

**Source**: `bench/b6_latency_test.go` at $n = 2^{24}$, $\varepsilon = 0.01$,
guaranteed-empty smart queries on 7 distributions. Data:
`bench_results/data/b6_latency.json`. Plots:
`bench_results/plots/b6_N16777216/{query_latency,fpr,bpk,build_throughput}/`.
Date: 2026-04-30. Filed during defence prep.

This file collects findings that surprised us during the run, with
enough context to defend or footnote each in the thesis text.

## Section 1 — SODA degenerate-hash mode and ERE bottleneck

## TL;DR

On SOSD FB, SODA-uint64 query latency grows from 241 ns @ L=1 to 5173 ns @
L=65536 — a 21× degradation. This is **not** the O(1) probe behavior the
SODA construction is supposed to give; the algorithmic guarantee still
holds (3 ERE calls per query worst case), but the underlying ERE call
itself becomes expensive.

Two effects compound:

1. **SODA-hash degenerates to identity** when the input universe fits in
   a single SODA super-block (i.e. when all keys satisfy `key < 2^K`).
2. **The downstream ERE inherits the original key distribution** — for
   SOSD FB this means buckets of size $2^{w-9}$ (up to ~16K elements at
   L=65536), and a 14-iteration binary search dominated by random-access
   cache misses into a 30-50 MB packed array.

The practical consequence is that we cannot quote a single "ns/op" for
SODA without specifying the distribution, and the headline filter table
should pin its "query latency" measurements to a regime where SODA's hash
actually mixes (uniform/spread synthetic with $2^{64}$-wide universe), or
explicitly note the degenerate mode for SOSD-like workloads.

## SODA construction recap

For input keys with universe $\mathcal{U}$, target FPR $\varepsilon$,
range length $L$:

- $r = n \cdot L / \varepsilon$ — codomain size.
- $K = \lceil \log_2 r \rceil$ — codomain bit-width.
- For each key $x$: $\text{block}(x) = x \gg K$, hash offset $u_x =
  \mathrm{PairwiseHash}(\text{block}(x), a, b, K)$, hashed value
  $h(x) = (u_x + x) \bmod 2^K$.
- The hashed values feed into an ERE structure parameterised by $K$.

`PairwiseHash(x, a, b, K) = ((a x + b) \bmod 2^{64}) \gg (64 - K)`.

## The degeneracy

For SOSD FB, every key satisfies $\text{key} < 2^{33}$. With
$L \in [1, 65536]$ and $\varepsilon = 0.01$, we have $K \in [31, 47]$.
Already at $L = 1$, $K = 31$ is barely above the key width, and for
$L \geq 16$ we have $K > 33 = $ effective key width, so

$$
\text{block}(x) = x \gg K = 0 \quad \forall x \in \text{keys}.
$$

Now `PairwiseHash(0, a, b, K)`:

```go
hi, lo := mbits.Mul64(a, 0)   // (0, 0)
sumLo, _ := mbits.Add64(0, b, 0)  // = b
sumHi := 0 + 0                 // = 0
return sumHi >> (64 - K)       // = 0
```

The hash offset is **identically zero** for every key (and every query
endpoint). Therefore $h(x) = (0 + x) \bmod 2^K = x$ since $x < 2^K$.

In other words: **on SOSD FB, SODA-hash is the identity function**.

The ERE underneath is built directly on the original FB keys, just
re-interpreted as $K$-bit values.

## Why this hurts

The ERE structure (`ere_one_d.ExactRangeEmptiness`) splits its $K$-bit
input into a $k$-bit prefix (`k = log2(n)`) and a $w = K - k$ bit
suffix:

- $k = 24$ for $n = 2^{24}$.
- $w \in [7, 23]$ as $L$ varies.
- Number of ERE blocks: $2^k = 2^{24} \approx$ 16M.

For uniformly distributed hashed values, each block holds on average
$n / 2^k \approx 1$ key, so binary search inside a bucket is trivial and
$\mathrm{IsEmpty}$ amortises to a couple of `rsdic.Select` calls.

But for SOSD FB the hashed values are not uniform — they are the
original 33-bit-wide FB keys. Block id is computed as $\text{key} \gg w$,
so block ids land in $[0, 2^{33-w})$:

Empirical bucket distribution from `bench/soda_bucket_audit_test.go`
(measured on the actual SODA build, not modeled):

| $L$    | $K$ | $w$ | populated blocks | avg bucket | max bucket | binsearch depth |
|-------:|----:|----:|-----------------:|-----------:|-----------:|----------------:|
| 1      | 31  | 7   | 4 338 941        | 3.9        | 116        | 0 (linear≤128)  |
| 1024   | 41  | 17  | 48 529           | 345.7      | 2 872      | 8               |
| 65536  | 47  | 23  | 760              | 22 075     | 35 233     | 14              |

At $L=65536$ the typical bucket holds ~22K keys × 23 bits ≈ **63 KB**,
which fits in L1; the binary search there is **not** the dominant cost.
What hurts is that the surrounding `packedData` array — $n \cdot w / 8 =
\sim 48$ MB — randomly evicts the rsdic index tables during each query
(see profile breakdown below).

## Profile breakdown

Fresh `pprof` of `BenchmarkSodaIsEmptyDirect` on SODA L=65536 / sosd_fb,
$n=2^{24}$, no harness overhead (direct method call in tight loop):

| Symbol | flat % | flat ns/query (≈) |
|--------|------:|-------:|
| `rsdic.RSDic.Select1`        | **70.6%** | **3170** |
| `bits.UnpackBit` (under `searchBucket`) | 13.3% | ~600 |
| `searchBucket` / `isRangeEmptyInBlock` body | < 1% | ~10 |
| `getBlockRange` body (call+sample blame) | <1% | ~10 |
| SODA wrapper + ERE.IsEmpty body | ~13% | ~700 |
| **measured total**           | 100% | **4495** |

So `rsdic.Select1` accounts for **~70% of the entire query cost**, not
the binary search. Two Select calls per same-block query ⇒ ~1585 ns per
Select call when running under the full pipeline. The bucket binary
search at depth 14 contributes the remaining ~600 ns through 14
`bits.UnpackBit` reads on the packed-suffix array.

### Why Select1 is so slow here, and how we know it isn't the algorithm

We ran two complementary isolated benchmarks to check whether
`rsdic.Select1` is intrinsically slow.

**(a) Synthetic 50%-density scaling** —
`Thesis/succinct_bit_vector/rsdic/scaling_test.go`: Select1 / Rank /
interleaved-Select pairs on a fresh 50%-density bitvector at sizes
$2^{20}\dots 2^{28}$, M4 Max, `-benchtime=2s`:

| $N$ bits | size (MB) | Select1 (ns) | Rank (ns) | 2× Select interleaved (ns) |
|---------:|----------:|-------------:|----------:|---------------------------:|
| $2^{20}$ |    0.16   |        38.6  |     19.3  |                       70.5 |
| $2^{22}$ |    0.63   |        44.6  |     22.4  |                       82.0 |
| $2^{24}$ |    2.53   |        49.7  |     21.6  |                       83.8 |
| $2^{26}$ |   10.12   |        57.8  |     24.4  |                       93.0 |
| $2^{28}$ |   40.50   |        86.0  |     42.0  |                      152.0 |

**(b) Real SODA bitvectors** —
`bench/rsdic_isolated_bench_test.go`. We build SODA on the actual key
set ($n=2^{24}$), serialize the inner ERE rsdic via `MarshalBinary`,
load it back, and benchmark Select1 with no other working set in cache:

| Distribution | $L$    | rsdic (MB) | Select1 (ns) | 2× Select interleaved (ns) |
|--------------|-------:|-----------:|-------------:|---------------------------:|
| sosd_fb      |     1  |    3.08    |        72.6  |                      133.0 |
| sosd_fb      |  1024  |    1.10    |        66.6  |                      116.2 |
| sosd_fb      | 65536  |    1.06    |        66.8  |                      116.8 |
| uniform      | 65536  |    3.80    |        50.1  |                       78.0 |

So in isolation the SODA-FB rsdic answers Select1 in **67 ns**, flat in
$L$ and consistent with the synthetic 50%-density curve at the same
size. **The factor-of-23 inflation to ~1585 ns under the SODA pipeline
is therefore not a property of rsdic itself — it is interaction-driven
cache thrashing.**

### What is being thrashed

`rsdic.RSDic` doesn't hold a single bitvector — it holds **five
parallel arrays**, all touched on every Select1:

1. `bits` — the encoded variable-rate bit stream (~4 MB at $n=2^{24}$).
2. `rankBlocks` — large-block prefix sums (one entry per 1024 bits).
3. `selectOneInds` — pointers into rankBlocks for fast Select.
4. `rankSmallBlocks` — small-block deltas (one byte per 64 bits).
5. `pointerBlocks` — bit offsets into `bits`.

A single Select1 walks `selectOneInds` → `rankBlocks` → `rankSmallBlocks`
loop → `bits[]` decode. Four random-access fetches across four arrays
that share no cache lines. In isolation those four cache lines stay
warm in L1/L2 across iterations because nothing else is fighting them.

Under the SODA query pipeline we additionally touch the
`ExactRangeEmptiness.packedData` array, which is **~48 MB at $L=65536$**
(width $w=23$, one entry per key) — far larger than M4's 16 MB shared
L2 and aggressively random-accessed by the bucket binary search. Each
binsearch step pulls fresh 4 KB pages, evicting rsdic's working set.
The next Select1 then re-misses on all four of its index arrays.

The Apple M4 Max has a 256-entry data TLB; 48 MB of packedData spans
~12k 4 KB pages, so essentially every binsearch hop incurs a TLB walk
on top of the cache miss. ~1585 ns/Select decomposes roughly as:

- ~150 ns × 4 cache misses (one per index array, cold L2/DRAM)
- ~30 ns × 4 TLB walks
- ~40 ns × 6 small-block iterations on the bits-decode path

Reconciling the full latency budget at $L=65536$ on sosd_fb:

- 2 × Select1 under cache pressure ≈ 3170 ns (70%).
- 14 binsearch hops × `bits.UnpackBit` on the packed-suffix array ≈
  600 ns (13%).
- ERE.IsEmpty body, SODA wrapper, branch overhead ≈ 720 ns (16%).
- **Total ≈ 4490 ns**, matches measured 4495 ns / 5173 ns within margin.

### Bucket distribution check (the binary search is *not* the bottleneck)

`bench/soda_bucket_audit_test.go` dumps the actual bucket-size
distribution of the inner ERE on each SODA build. Sosd_fb at L=65536:

- 760 populated blocks (out of $2^{24}$ possible)
- avg bucket: 22 075 keys
- max: 35 233 ; p50 = 22 184 ; p99 = 30 836
- 100% of B6 smart-mix queries hit the **same-block** path (single
  `ere.IsEmpty` call per SODA call)

Each bucket spans 22 075 × 23 bits ≈ **63 KB** — fits in M4's L1 D-cache
(128 KB/core). Once the binsearch starts narrowing, the active region
collapses into one cache line within ~6 hops, leaving the last 8 hops
as L1 hits. That matches the observed 13% / 600 ns binsearch budget.

## Distinguishing distribution from algorithm

The SODA construction is correct (the FPR analysis still holds modulo
hashed-vs-identity values; for keys < $2^K$, hashing into the same
codomain doesn't change collision probabilities since the query is
range-shaped, not point-shaped). What changes is the *internal layout*
of the ERE backend. On a wide-universe distribution where SODA-hash
actually mixes:

- Hashed values are uniform in $[0, 2^K)$.
- Each ERE block holds ~1 key.
- Bucket binary search is trivial.
- `IsEmpty` reduces to ~2 `rsdic.Select` calls + O(1) packed reads.

This is the regime the SODA paper analyses asymptotically. SOSD FB,
Books, Wiki — all with key universes much smaller than $2^K$ for
practical $L \cdot 1/\varepsilon$ — fall outside it.

## Implications

1. **Single-number latency claims are misleading.** When stating "SODA
   query latency at $n=2^{24}$ is $X$ ns", we must specify the
   distribution; the same construction gives 241 ns or 5173 ns
   depending on whether the keys span a single SODA super-block.

2. **The bottleneck is `rsdic.Select1` under cache pressure**, not the
   bucket binary search. pprof attributes 70% of total query time to
   `Select1` (≈ 1585 ns/call) and only 13% to `bits.UnpackBit` inside
   the binsearch. Isolated, the same Select1 measures 67 ns — the 23×
   inflation comes from rsdic's five-array index layout being evicted
   on every binsearch step into the 48 MB packed-suffix array.

3. **This is not a 1D-vs-classic ERE regression.** Both backends share
   the same `rsdic` Select implementation and the same packed-bit bucket
   layout. Cross-checked: on the same SODA L=65536 / SOSD FB cell, the
   classic two-vector ERE measures 5475 ns vs the one-vector 5173 ns.

4. **Optimisation opportunities** (out of scope for the defence run, but
   worth recording):
   - Co-locate rsdic's index arrays (bits, rankBlocks, rankSmallBlocks,
     selectInds, pointerBlocks) into one cache-friendly struct. Five
     parallel arrays at random offsets ⇒ five evicted lines per Select;
     a packed Select-on-flat-bitvector layout pays ~$n$ extra bits in
     exchange for ≤ 2 cache lines per Select.
   - In SODA, when the input universe fits in one super-block, fall
     through to a different mode (e.g. directly use a Truncation-style
     ERE on the natural prefix bits without the redundant identity
     hash + ERE wrap).
   - Increase ERE $k$ above $\log_2 n$ when the populated-block count
     is small, so populated buckets shrink. Trades metadata bits for
     query speed and reduces packed-suffix size below the L2 threshold,
     which would also relieve the rsdic-eviction effect.

5. **Defence-text framing.** The headline FPR-vs-BPK plots are
   unaffected — those depend only on memory and false-positive rate,
   which the SODA degenerate mode preserves. But the build/query
   latency table (§sec:eval-build-query-latency) should either:
   (a) report on a wide-universe distribution where SODA is in its
   asymptotic regime, or (b) report SOSD FB explicitly with a footnote
   on the degenerate-hash mode and what it costs in practice.

## Empirical numbers (n = 2^24, ε = 0.01, smart-mix empty queries)

SOSD FB:

| Filter        | L=1   | L=128 | L=1024 | L=65536 |
|---------------|------:|------:|-------:|--------:|
| Truncation    |  270  |  241  |   229  |   311   |
| Greedy+Merge  |  246  |  264  |   311  |   289   |
| SuRFReal(8)   |  280  |  298  |   274  |   254   |
| Scan-ARE      |  333  |  399  |   282  |   303   |
| SNARF         |  662  | 1053  |   963  |   842   |
| **SODA**      |**241**|**297**|**749** |**5173** |
| BloomARE      |   68  | 6190  | 32104  | 237954  |
| Grafite       |  175  |   —   |    —   |    —    |

Latency in nanoseconds per query.

(More distributions in `bench_results/data/b6_latency.json` — populated
by `go test -run TestB6IndustryLatency ./bench/`.)

---

## Section 2 — SuRF FPR is anti-correlated with L on sparse data

SuRF Real-suffix has a different FPR shape than the Goswami / Bloom
families: it does **not** carry an explicit $\varepsilon$ parameter, and
its FPR is determined by the trie structure plus the suffix bits, not by
$L$ in the way an ARE filter is.

Range query semantics: `lookupRange(lo, hi)` walks the trie following
the longest common byte-prefix of `lo` and `hi`, then tests whether any
key sits in the resulting prefix subtree. If the subtree is empty,
return false (true negative). Otherwise — true (FP, since smart-mix
queries are guaranteed empty).

For two query endpoints `[a, a+L-1]`, the longest common prefix shrinks
as $L$ grows: roughly $64 - \lceil \log_2 L \rceil$ shared bits. So
larger $L$ ⇒ traversal stops higher in the trie ⇒ subtree contains a
larger fraction of the address space.

The interaction with the data distribution decides whether wider
subtrees are emptier or fuller. **On sparse data the wider subtrees are
emptier**, so FPR drops with $L$.

Empirical FPR vs $L$ at $n=2^{24}$ for SuRFReal(8) (8 real bits per
leaf, equal across distributions):

| Distribution | BPK   | L=1     | L=128   | L=1024  | L=65536 |
|--------------|------:|--------:|--------:|--------:|--------:|
| sosd_books   | 13.16 | 0.0000  | 0.0000  | 0.0000  | 0.0000  |
| clustered    | 40.77 | 0.106   | 0.105   | 0.107   | 0.105   |
| spread       | 42.20 | 0.222   | 0.250   | 0.250   | 0.251   |
| sosd_osm     | 20.44 | **0.494** | **0.380** | **0.343** | **0.108** |
| uniform      | 42.83 | 0.500   | 0.500   | 0.499   | 0.472   |
| sosd_fb      |   —   |   —     |   —     |   —     |   —     |
| sosd_wiki    |   —   |   —     |   —     |   —     |   —     |

(`sosd_wiki` triggers a SIGSEGV in SuRF's trie builder — see Section 3
below; `sosd_fb` was deferred from this run.)

Reading row by row:

- **`sosd_books`** is dense uint32 keys in $[0, 2^{25})$. Smart-mix
  queries are forced into key-adjacent positions; the BE-encoding
  shares 7 of 8 bytes with neighbours and the last byte differs by
  exactly the smart-query offset. The 8 real-suffix bits cover that
  difference, so FPR collapses to literally zero.

- **`clustered`** keeps a flat 0.105 across $L$. The 5-cluster
  distribution has uniform-density inside each cluster; smart queries
  near cluster boundaries always land in a non-empty subtree, and the
  real-suffix gamble settles at the same fraction regardless of $L$.

- **`spread`** sits at 0.22–0.25 (essentially 1/4) flat. The synthetic
  spread structure makes every smart query share a fixed-depth common
  prefix with two real keys; SuRF's two-bit gamble sees the same
  ambiguity at all $L$.

- **`sosd_osm`** is the headline case. OSM cell-IDs are H3-style
  hierarchical indices: nearby keys share long byte prefixes (the
  region geohash), but globally the universe is sparse ($2^{60}$
  addressable, $2^{24}$ keys). At small $L$, smart-near-key queries
  sit in deep, dense subtrees of the regional trie; at large $L$, the
  query straddles regional boundaries and lands in shallower subtrees
  that are almost entirely outside the populated 1/$2^{36}$ fraction
  of the space, so most queries get a clean trivial-empty answer.
  $0.494 \to 0.108$ is exactly this transition.

- **`uniform`** sits at $\approx 0.5$ — essentially a coin flip. With
  16M random uint64 keys spread uniformly in $[0, 2^{64})$, the
  top-level trie branches densely on the first byte (nearly all 256
  branches populated), and the real-suffix bits don't recover enough
  signal to push the FP rate down. SuRF is poorly matched to this
  distribution.

### Why this matters in the thesis

1. **SuRF cannot be reported as "FPR = $\varepsilon$" alongside the
   Goswami filters.** Its FPR is structural and ranges from 0 to 50%
   depending on key distribution. Use a per-distribution column.

2. **The "ARE filters' FPR grows with L" intuition does not transfer
   to SuRF.** On sparse hierarchical data SuRF actually *improves*
   with L, opposite to BloomARE / Truncation / SODA.

3. **`sosd_osm` is a particularly informative test bed** for SuRF
   because it stresses the geohash-locality regime where the trie's
   strengths (deep prefix sharing) and weaknesses (short range
   queries getting trapped in dense subtrees) both surface clearly.

4. **`sosd_books` shows SuRF can be perfect** in the right regime
   (dense, narrow universe, smart-near-key queries cleanly resolved
   by suffix bits). FPR=0 over $7 \times 256K \approx 1.8M$ queries.

---

## Section 3 — SuRF crashes on sosd_wiki (timestamps with shared BE prefix)

`surf::SuRF` constructor SIGSEGVs on SOSD `wiki_ts_200M` at $n=2^{24}$.
Reproducer: `bench/b6_latency_test.go` without `SKIP_FILTERS`. Fault
during `surf_new` in `_cgo_gotypes.go:79`; address `0x834000000` looks
like a wiki timestamp value (~ 35 × 10⁹) being mis-dereferenced as a
pointer in the trie builder.

Likely match in upstream:
[**efficient/SuRF#8** "Buffer overrun in bitvector"](https://github.com/efficient/SuRF/issues/8) — out-of-bounds read up
to 8 bytes past `bits_` buffer when there is a long run of 0s near the
end of the bit vector. Author confirmed; partial patch in
`6b36fc4f`. Wiki-ts in BE encoding has 4 zero high bytes for every
key (timestamps fit in 33 bits), giving exactly the long-zero-run
pattern the patched code-path was supposed to handle.

Upstream is dormant (last push 2022-03-11). For this thesis we work
around by passing `SKIP_FILTERS='SuRFReal(8)'` on the wiki cell
specifically; other distributions in the suite are unaffected.

Filed for Section 3 of the limitations list in the evaluation chapter.
