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
