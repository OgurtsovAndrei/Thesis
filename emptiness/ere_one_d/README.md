# ERE One-D

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
`ere` always alternates between `D1` and `D2`. This is what the measured **9.1%-13.2%**
average query speedup in [ARE Benchmark Results](#are-benchmark-results) reflects,
despite the operation count occasionally going up.

## Implementation Notes

- both `ere` and `ere_one_d` use the local optimized succinct bitvector implementation:
  - `Thesis/succinct_bit_vector/rsdic`

## ARE Benchmark Results

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
| uniform | 256.79 | 258.61 | 164.00 | 133.57 | 1.23x | 22.000 | 21.000 | 1.000 |
| clustered | 174.77 | 176.75 | 331.33 | 322.27 | 1.03x | 22.000 | 21.000 | 1.000 |
| sosd_fb | 37.95 | 35.85 | 671.93 | 566.48 | 1.19x | 22.000 | 21.000 | 1.000 |
| sosd_wiki | 32.12 | 31.99 | 630.27 | 586.65 | 1.07x | 22.060 | 21.530 | 0.530 |
| sosd_osm | 212.76 | 221.79 | 365.04 | 196.08 | 1.86x | 22.000 | 21.500 | 0.500 |
| sosd_books | 49.21 | 36.75 | 590.96 | 584.60 | 1.01x | 22.000 | 21.000 | 1.000 |
| **Average** | **127.27** | **126.96** | **458.92** | **398.28** | **1.15x** | **22.010** | **21.172** | **0.838** |

Briefly:

- `one_d` improved average query time by about **13.2%**
- average memory improved by about **0.84 bpk**
- build time stayed effectively unchanged

### Greedy+Merge

| Dataset | Classic build ms | One-D build ms | Classic query ns | One-D query ns | Speedup | Classic bpk | One-D bpk | Delta bpk |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| uniform | 84.81 | 91.07 | 222.60 | 159.53 | 1.40x | 22.000 | 21.000 | 1.000 |
| clustered | 63.92 | 55.72 | 474.78 | 344.38 | 1.38x | 17.696 | 16.961 | 0.734 |
| sosd_fb | 56.46 | 56.49 | 250.56 | 331.06 | 0.76x | 12.000 | 11.000 | 1.000 |
| sosd_wiki | 51.86 | 57.75 | 299.52 | 359.13 | 0.83x | 10.061 | 9.530 | 0.530 |
| sosd_osm | 113.58 | 111.45 | 450.82 | 418.10 | 1.08x | 33.263 | 32.525 | 0.738 |
| sosd_books | 60.34 | 65.84 | 283.79 | 189.92 | 1.49x | 5.000 | 4.000 | 1.000 |
| **Average** | **71.83** | **73.05** | **330.34** | **300.35** | **1.10x** | **16.670** | **15.836** | **0.834** |

Briefly:

- `one_d` improved average query time by about **9.1%**
- average memory improved by about **0.83 bpk**
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
