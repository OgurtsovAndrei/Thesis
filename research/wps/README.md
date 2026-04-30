# Weak Prefix Search (WPS) — Research, Not Adopted

This directory holds the implementation of the **Weak Prefix Search (WPS)**
optimization originally proposed in *Approximate Range Emptiness in Constant
Time and Optimal Space* (Goswami, Grossi, Pagh, Pătraşcu — SODA 2015) and the
auxiliary structures it relies on.

The thesis explored this direction and **rejected it in favor of binary
search on bit-packed suffixes with a linear-scan fallback for small buckets**.
See `text/src/ere.tex` § *Local Search in Buckets*:

> The original paper suggests a Weak Prefix Search structure (Hollow Z-Fast
> Trie) for O(1) worst-case local queries. We use binary search on bit-packed
> suffixes, with an adaptive fallback to linear scan for small buckets
> instead. The reason is practical. Binary search and linear scan are faster
> in practice than accessing an additional O(1) indexing structure with a big
> hidden constant, while also avoiding its memory overhead.

The code is preserved here for thesis archival and reproducibility — it is
**not** part of the production filter pipeline (`emptiness/approx/are_*` and
`emptiness/exact/{ere,ere_one_d}`).

## Contents

The WPS pipeline is a closed group of packages — the production filters
(`emptiness/approx/are_*`, `emptiness/exact/{ere,ere_one_d}`, `bench/`,
`bits/`) do **not** import any of these. They are gathered here for
archival.

### Locators (top-level WPS interfaces) — under `locators/`

| Package | Role |
|---------|------|
| `locators/rloc/` | Pure-Go RangeLocator |
| `locators/lerloc/` | Pure-Go RangeLocator built on the Hollow-Z-Fast-Trie WPS structure |
| `locators/lemon_lerloc/` | LERLOC variant backed by the LeMonHash CGo wrapper |
| `locators/lemon_rloc/` | Plain RLOC backed by LeMonHash (used internally by `lemon_lerloc`) |
| `locators/merged_lerloc/` | Experimental merged-buckets LERLOC variant |
| `locators/lemonhash/` | CGo wrapper around the LeMonHash C++ library — kept here because the only consumers are `lemon_rloc` / `lemon_lerloc` |

### Tries (the WPS data structure)

| Package | Role |
|---------|------|
| `trie/zft/` | Z-Fast Trie — base structure for all variants below |
| `trie/azft/` | Augmented Z-Fast Trie |
| `trie/hzft/` | Hollow Z-Fast Trie — the WPS core (per Goswami et al.) |
| `trie/shzft/` | Succinct Hollow Z-Fast Trie |

### Hashing infrastructure (used by tries and locators) — under `mmph/`

| Package | Role |
|---------|------|
| `mmph/bucket_mmph/` | Bucket-MMPH (Monotone Minimal Perfect Hashing) |
| `mmph/go-boomphf-bs/` | BBHash-style MMPH adapted to BitString keys |
| `mmph/paramselect/` | Hyperparameter search for MMPH variants |
| `mmph/rbtz_mmph/` | Right-to-left bucket-tree-zero MMPH |
| `mmph/relative_trie/` | Relative-trie MMPH variant |

## Known issues

- `lemon_rloc` segfaults on non-member queries (LeMonHash VL-mode crash).
  This is part of the reason WPS-via-LeMonHash was abandoned.

## What still imports this code

After the move, only the **experimental** ERE backends still reference
`lerloc`:

- `Thesis/emptiness/exact/ere_global/`
- `Thesis/emptiness/exact/ere_theoretical/`

These are **not** used by the headline filters or in the thesis evaluation;
they're kept for completeness. The production ERE backends are
`Thesis/emptiness/exact/ere/` and `Thesis/emptiness/exact/ere_one_d/` — both
WPS-free.

## Reference

See `Thesis/papers/Approximate Range Emptiness.pdf` and
`Thesis/papers/Approximate-Range-Emptiness/` for the source paper and notes.
