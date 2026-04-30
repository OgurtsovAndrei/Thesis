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

| $L$    | $K$ | $w$ | populated blocks | avg bucket | bin-search depth |
|-------:|----:|----:|-----------------:|-----------:|----------------:|
| 1      | 31  | 7   | $\sim n$ (all)   | ~1         | 0 (linear)      |
| 16     | 35  | 11  | $2^{22}$         | ~4         | 0 (linear)      |
| 128    | 38  | 14  | $2^{19}$         | ~32        | 0 (≤128)        |
| 1024   | 41  | 17  | $2^{16}$         | ~256       | 8               |
| 4096   | 43  | 19  | $2^{14}$         | ~1024      | 10              |
| 16384  | 45  | 21  | $2^{12}$         | ~4096      | 12              |
| 65536  | 47  | 23  | $2^{10}$         | ~16384     | 14              |

The $\sim 16{\rm K}$ keys per populated bucket at $L=65536$ require
14-iteration binary searches over a packed-bit array of width $w=23$,
which at $n = 2^{24}$ is $\sim 48$ MB — well past L3 cache. Each
iteration costs one cache miss.

## Profile breakdown

`pprof` of the SODA L=65536 path on SOSD FB:

- `getBlockRange` (= 2× `rsdic.RSDic.Select1`): **38.24% of total**
  / 83.6% of `ExactRangeEmptiness.IsEmpty`. ~1900 ns per query, i.e.
  ~950 ns per `Select` call.
- `Thesis/bits.UnpackBit` inside the bucket binary search: ~2nd hottest.
  At depth 14, ~14 random reads × ~120 ns = ~1700 ns.
- SODA wrapper, `IsEmpty` body, branch overhead: ~1500 ns.
- **Total ≈ 5100 ns**, matches measured 5173 ns.

`rsdic.Select1` itself is two nested loops over rank-block and
small-block tables (constants `kSelectBlockSize=4096`, `kLargeBlockSize=
1024`, `kSmallBlockPerLargeBlock=16`), plus one variable-rate decode
from the bit-packed `bits` array. ~24 conditional iterations + 1–3
cache misses at 100–150 ns each gives the observed ~950 ns per call.

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

2. **The `ere_one_d` `IsEmpty` is the actual bottleneck**, not the SODA
   wrapper. Specifically `rsdic.Select` (~950 ns / call) and the bucket
   binary search.

3. **This is not a 1D-vs-classic ERE regression.** Both backends share
   the same `rsdic` Select implementation and the same packed-bit bucket
   layout. Cross-checked: on the same SODA L=65536 / SOSD FB cell, the
   classic two-vector ERE measures 5475 ns vs the one-vector 5173 ns.

4. **Optimisation opportunities** (out of scope for the defence run, but
   worth recording):
   - Replace `rsdic` with a denser, cache-friendlier rank/select
     (e.g. broadword Select on a flat bitvector). Pays ~$n$ extra bits.
   - In SODA, when the input universe fits in one super-block, fall
     through to a different mode (e.g. directly use a Truncation-style
     ERE on the natural prefix bits without the redundant identity
     hash + ERE wrap).
   - Increase ERE $k$ above $\log_2 n$ when the populated-block count
     is small, so populated buckets shrink. Trades metadata bits for
     query speed.

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
