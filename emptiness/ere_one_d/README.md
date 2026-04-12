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

- `ere_one_d` uses the local optimized succinct bitvector implementation:
  - `Thesis/succinct_bit_vector/rsdic`
- the original `ere` currently still imports the external library version

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
