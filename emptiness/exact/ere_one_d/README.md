# ERE One-D

**Production default backend.** Selected by `exactbackend.NewUint64()` since Phase C migration (2026-04-30).

This package is a variant of `Thesis/emptiness/ere` with one structural optimization:

- the original `ere` stores two succinct bitvectors, `D1` and `D2`
- `ere_one_d` stores a single succinct bitvector `D`

The full ERE design, motivation, bucket-search discussion, and general complexity notes are already documented in:

- [../ere/README.md](../ere/README.md)

This README only describes the delta.

## Optimization

Original encoding in `ere`:

- `D1[i] = 1` iff block `i` is non-empty
- `D2` stores `1 0^{n_i}` only for non-empty blocks

Encoding in `ere_one_d`:

- for every block `i`, store `1 0^{n_i}`
- append one final sentinel `1`

So:

- empty block => `1`
- block with `n_i` keys => `1 0^{n_i}`
- `D = (1 0^{n_0})(1 0^{n_1})...(1 0^{n_{B-1}})1`

## Why This Helps

With this encoding:

- block `i` starts at `select(i)`
- block `i` ends at `select(i+1)`
- the global suffix range of block `i` is:
  - `start(i) = select(i) - i`
  - `end(i) = select(i+1) - (i+1)`

That removes the need for:

- `D1.rank(i)` to map block index -> non-empty block rank
- a separate occupancy bitvector `D1`

## Expected Space Effect

Let:

- `n` = number of keys
- `B` = number of blocks
- `m` = number of non-empty blocks

Original metadata bits:

```text
|D1| + |D2| = B + (n + m + 1)
```

New metadata bits:

```text
|D| = n + B + 1
```

So the exact metadata reduction is:

```text
(B + n + m + 1) - (n + B + 1) = m bits
```

That is:

- `m / n` bits per key saved
- upper bound: almost `1 bpk`
- for roughly uniform occupancy with `lambda ~= 1`, expected savings are around `0.63 bpk`

This optimization is therefore specifically about reducing metadata overhead.

## Expected Query Effect

This version replaces `D1.Bit` / `D1.Rank` logic with direct `Select` calls on `D`.

Practical implication:

- memory should improve
- query speed is not obvious analytically
- results depend on the cost of `Select` in the local `rsdic` fork and on the query mix

That is why this package exists as a separate benchmark target instead of replacing `ere` immediately.

### Rank/Select Operation Count Per Query Case

The single-vector encoding does **not** strictly reduce the number of Rank/Select calls.
The `D1.Bit(b)` guard in the original `ere` allowed an early exit on empty boundary blocks
with zero Rank/Select. The single-vector design gives that up: it always pays at least two
`Select(D)` per boundary block, but every call hits one contiguous bitvector, and the
intermediate-block check becomes a free index comparison instead of two extra `D1.Rank`.

Counts below include only Rank and Select on succinct bitvectors (`Bit` is a single-word
read and is not counted). Bucket scan is identical in both versions and is excluded.

#### Case A — Both endpoints in the same block (`blockA == blockB`)

| Sub-case | `ere` operations | `ere_one_d` operations | Delta |
|---|---|---|---|
| Block non-empty | `1x Rank(D1) + 2x Select(D2)` = 3 | `2x Select(D)` = 2 | **-1** |
| Block empty | early exit, 0 | `2x Select(D)` = 2 | +2 |

#### Case B — Endpoints in two adjacent blocks (`blockB == blockA + 1`)

`ere_one_d` uses a specialized path: three sequential `Select(D)` calls for positions
`blockA`, `blockA+1`, `blockA+2`. The `startB > endA` short-circuit cannot fire here
because adjacent blocks satisfy `startB == endA`.

| Sub-case | `ere` operations | `ere_one_d` operations | Delta |
|---|---|---|---|
| Both blocks non-empty | `2x Rank(D1) + 4x Select(D2)` = 6 | `3x Select(D)` = 3 | **-3** |
| One block non-empty | `1x Rank(D1) + 2x Select(D2)` = 3 | `3x Select(D)` = 3 | 0 |
| Both blocks empty | early exit, 0 | `3x Select(D)` = 3 | +3 |

#### Case C — Long range with intermediate blocks (`blockB > blockA + 1`)

In `ere`, intermediate non-emptiness is detected via two `D1.Rank` calls before the
boundary blocks are touched. In `ere_one_d`, the same information is encoded in
`startB > endA` — a free integer comparison on values already produced by the two
`getBlockRange` calls.

| Sub-case | `ere` operations | `ere_one_d` operations | Delta |
|---|---|---|---|
| Intermediate non-empty (early exit) | `2x Rank(D1)` = 2 | `4x Select(D)` = 4 | +2 |
| Intermediate empty, both boundaries non-empty | `4x Rank(D1) + 4x Select(D2)` = 8 | `4x Select(D)` = 4 | **-4** |
| Intermediate empty, both boundaries empty | `2x Rank(D1)` = 2 | `4x Select(D)` = 4 | +2 |

### Cross-Case Summary

All seven sub-cases collapsed into a single table. `R(D1)` is `Rank` on the occupancy
bitvector, `S(D2)` is `Select` on the count bitvector, `S(D)` is `Select` on the unified
bitvector.

In typical ARE workloads the dominant query patterns are short ranges that hit one
non-empty block or two adjacent non-empty blocks — those are the rows highlighted in
bold below, and they are precisely the rows where `ere_one_d` reduces the operation
count.

| Query type | `ere` R/S | `ere_one_d` R/S | Delta | Note |
|---|---|---|---|---|
| **Same block, non-empty** | **3 (`1x R(D1) + 2x S(D2)`)** | **2 (`2x S(D)`)** | **-1** | **Most frequent, hot path** |
| Same block, empty | 0 | 2 (`2x S(D)`) | +2 | Cold-path regression |
| **Adjacent, both non-empty** | **6 (`2x R(D1) + 4x S(D2)`)** | **3 (`3x S(D)`)** | **-3** | **Most frequent, main hot-path case** |
| Adjacent, both empty | 0 | 3 (`3x S(D)`) | +3 | Cold-path regression |
| Long range, both boundaries non-empty | 8 (`4x R(D1) + 4x S(D2)`) | 4 (`4x S(D)`) | **-4** | Op count halved |
| Long range, intermediate non-empty (early exit) | 2 (`2x R(D1)`) | 4 (`4x S(D)`) | +2 | More ops, one vector |
| Long range, all empty | 2 (`2x R(D1)`) | 4 (`4x S(D)`) | +2 | Cold-path regression |

### Summary

The hot path — queries that hit non-empty boundaries and actually carry payload — gets
cheaper everywhere: **-1** for same-block, **-3** for adjacent-block, **-4** for
full-range queries. The cold path (empty boundaries or early intermediate exit) pays
2-3 extra `Select` calls, but those queries do no further work, so the absolute overhead
is small.

The structural win matters more than the op count: every Rank/Select now hits one
contiguous `RSDic` instead of two. The second `Select(D)` in a query is very likely to
share the auxiliary index pages and cache lines loaded by the first, while the original
`ere` always alternates between `D1` and `D2`. This is what the measured **7.6%-12.7%**
average query speedup in [ARE Benchmark Results](#are-benchmark-results) reflects,
despite the operation count occasionally going up.

## Implementation Notes

- both `ere` and `ere_one_d` use the local optimized succinct bitvector implementation:
  - `Thesis/succinct_bit_vector/rsdic`

## ARE Benchmark Results

The previous version of these tables was generated with a buggy `SizeInBits()`
formula in `ere` that over-counted `D2` by `numBlocks - m - 1` bits (it used a
hardcoded length expression instead of `D2.Num()`). The numbers below reflect
the corrected formula, which calls `D1.Num() + D2.Num()` directly. As a
consequence the per-distribution `Delta bpk` now tracks the theoretical value
`m / n` (number of non-empty blocks per key) instead of an inflated
`(B - 1) / n`, so reductions are smaller, especially on dense distributions
where `m` is close to `B`.

Measured with `ere_one_d` used as the exact backend inside:

1. `SODA`
2. `Greedy+Merge`

Workload:

- `n = 2^20`
- `rangeLen = 4096`
- `epsilon = 0.01`
- mixed queries: 32768 queries, 3 rounds
- datasets:
  - `uniform`
  - `clustered`
  - `sosd_fb`
  - `sosd_wiki`
  - `sosd_osm`
  - `sosd_books`

Hardware:

- Apple M4 Max
- `darwin/arm64`

### SODA

| Dataset | Classic build ms | One-D build ms | Classic query ns | One-D query ns | Speedup | Classic bpk | One-D bpk | Delta bpk |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| uniform | 252.07 | 254.47 | 156.24 | 140.47 | 1.11x | 21.632 | 21.000 | 0.632 |
| clustered | 191.94 | 186.50 | 370.00 | 314.24 | 1.18x | 21.110 | 21.000 | 0.110 |
| sosd_fb | 41.03 | 40.71 | 703.01 | 618.96 | 1.14x | 21.001 | 21.000 | 0.001 |
| sosd_wiki | 33.18 | 33.69 | 562.60 | 533.66 | 1.05x | 21.530 | 21.530 | 0.000 |
| sosd_osm | 199.25 | 220.08 | 306.16 | 162.44 | 1.88x | 21.930 | 21.500 | 0.430 |
| sosd_books | 56.84 | 39.83 | 656.68 | 635.21 | 1.03x | 21.000 | 21.000 | 0.000 |
| **Average** | **129.05** | **129.21** | **459.12** | **400.83** | **1.15x** | **21.367** | **21.172** | **0.196** |

Briefly:

- `one_d` improved average query time by about **12.7%**
- average memory improved by about **0.20 bpk**; the saving tracks `m / n`,
  where `m` is the count of non-empty blocks. Uniform input gives the Poisson
  value `1 - e^{-1} ~= 0.63 bpk` (at `lambda = 1` about `e^{-1} ~= 37%` of
  blocks are empty). Heavily clustered inputs (`sosd_fb`, `sosd_wiki`,
  `sosd_books`) leave most blocks empty after the locality-preserving hash,
  so `m << B` and the saving collapses toward zero. The theoretical upper
  bound is `B / n = 1 bpk`, attained only when every block holds at least
  one key
- build time stayed effectively unchanged

### Greedy+Merge

| Dataset | Classic build ms | One-D build ms | Classic query ns | One-D query ns | Speedup | Classic bpk | One-D bpk | Delta bpk |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| uniform | 102.91 | 91.73 | 199.72 | 169.96 | 1.18x | 21.632 | 21.000 | 0.632 |
| clustered | 65.56 | 62.24 | 523.28 | 389.84 | 1.34x | 17.201 | 16.961 | 0.240 |
| sosd_fb | 70.49 | 60.95 | 251.06 | 322.66 | 0.78x | 11.232 | 11.000 | 0.232 |
| sosd_wiki | 64.10 | 50.30 | 276.41 | 346.92 | 0.80x | 9.708 | 9.530 | 0.177 |
| sosd_osm | 119.76 | 115.91 | 453.06 | 427.85 | 1.06x | 32.817 | 32.525 | 0.293 |
| sosd_books | 67.22 | 70.36 | 315.90 | 208.96 | 1.51x | 4.517 | 4.000 | 0.517 |
| **Average** | **81.67** | **75.25** | **336.57** | **311.03** | **1.08x** | **16.184** | **15.836** | **0.349** |

Briefly:

- `one_d` improved average query time by about **7.6%**
- average memory improved by about **0.35 bpk**; same `m / n` rule as above.
  Inputs that fill more blocks (`uniform` and `sosd_books`, which after the
  Greedy+Merge fingerprinting still leave many blocks populated) approach
  the Poisson `~0.63 bpk` saving, while the heavily clustered SOSD inputs
  (`sosd_fb`, `sosd_wiki`) leave most blocks empty (`m << B`) and save very
  little
- there are still bad latency cases on `sosd_fb` and `sosd_wiki`

### Short Takeaway

- for `SODA`, `one_d` looks strictly better on the measured setup:
  - same build cost
  - lower average query latency
  - lower bits/key
- for `Greedy+Merge`, `one_d` is better on average, but not uniformly better:
  - average query latency improves
  - average bits/key improves
  - `fb` and `wiki` remain counterexamples on latency

## What To Benchmark

Head-to-head against `ere`:

1. build time
2. query latency
3. bits per key
4. memory breakdown of metadata vs packed suffixes

Useful workloads:

1. uniform keys
2. clustered keys
3. heavy-bucket / adversarial distributions

## TODO: Validate Real Prod Memory Reduction In ARE

This optimization is only useful if the metadata reduction survives contact with the full
Approximate Range Emptiness stack, not just standalone ERE microbenchmarks.

TODO:

1. add end-to-end benchmarks for ARE with `ere` vs `ere_one_d` as the exact backend
2. measure `bits_per_key` and resident memory, not only theoretical `SizeInBits()`
3. run on at least two data distributions:
   - uniform
   - clustered
4. separate build-time and query-time measurements
5. report memory breakdown:
   - hash / fingerprint layer
   - exact backend metadata
   - packed suffix / payload storage
6. confirm whether the saved `m/n` bpk at ERE level produces a meaningful total BPK reduction at ARE level

Practical benchmark idea:

1. generate the same keysets for both implementations
2. build ARE twice:
   - once with `ere`
   - once with `ere_one_d`
3. use the same query workload for both:
   - hit-heavy
   - miss-heavy
   - mixed
4. run on:
   - uniform synthetic keys
   - clustered synthetic keys
5. compare:
   - build ns/op
   - query ns/op
   - allocs/op
   - total bits/key
   - backend-only bits/key

## Status

This package is an experimental optimized ERE variant.

Use `ere` as the baseline implementation and `ere_one_d` as the candidate with:

- one bitvector instead of two
- lower metadata footprint
- potentially different query-time tradeoff
