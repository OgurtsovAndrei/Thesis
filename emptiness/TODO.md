# Emptiness Module — TODO

## High Priority

### Full-size SOSD benchmarks

- [ ] Run all benchmarks (BPK vs FPR, build time, query time) on full SOSD datasets at native sizes:
  - `fb_200M_uint64` (200M keys)
  - `wiki_ts_200M_uint64` (200M keys)
  - `osm_cellids_800M_uint64` (800M keys)
  - `books_200M_uint32` (200M keys, uint32)
- [ ] Compare all ARE filters + industry filters (Grafite, SNARF, SuRF) on real data
- [ ] Data path: `Thesis-Bench-industry/bench/sosd_data/`

### Testing gaps

- [x] `are_optimized`: add systematic no-FN property tests (multiple seeds/sizes, clustered dist)
- [ ] `are_optimized`: add FPR accuracy test
- [ ] `are_optimized`: add Build/N and Query/N benchmarks
- [ ] `are_soda_hash`: add Build/N and Query/N benchmarks
- [ ] `are_pgm`: add Build/N and Query/N benchmarks
- [ ] `are_bloom`: add Build/N and Query/N benchmarks
- [ ] ALL packages: add Query/L benchmark (sweep range length L) — missing everywhere
- [ ] `are_hybrid`: Query/N benchmark only at fixed N=2^20, add N-sweep
- [x] Standardize no-FN tests to use clustered distribution (all packages now have uniform + clustered)

### Hybrid cluster detection

- [ ] `are_hybrid`: `detectClusters` fails on sequential (evenly-spaced) distributions — all gaps are equal, elbow detector finds no split, 0 clusters detected, everything falls back to plain Truncation ARE. Need a more robust gap threshold algorithm (e.g. percentile-based or density-based) that handles uniform-gap distributions gracefully.

### Dead code / bugs

- [ ] `ere/exact_range_emptiness.go:97`: `extractSuffixAsUint64` — `KeySize` parameter unused, remove
- [ ] `ere_theoretical`: field `L` stored but never read — remove
- [ ] `ere_theoretical`: `locators` built and measured but never used for queries — document or remove
- [ ] `ere_theoretical`: `getBlockIndex` duplicates `ere.GetBlockIndex` with explicit loop — deduplicate
- [ ] `ere_theoretical:141-149`: scratch-pad comments left in code — clean up
- [ ] `ere_theoretical:61`: outdated comment "Store keys with prefix relative to block" — keys stored as-is

## Medium Priority

### Code duplication

- [ ] `pairwiseHash` copy-pasted in `are_soda_hash:23` and `are_optimized:26` — extract to shared `internal/hash`
- [ ] `no_fn_prop_test.go` triple-clone across `are_trunc`, `are_hybrid`, `are_soda_hash` — extract shared test harness
- [ ] Key hashing + dedup loop duplicated in `are_soda_hash` and `are_optimized` constructors

### Naming inconsistencies

- [x] Rename `are` → `are_trunc`
- [ ] Struct names: 5 different conventions across 6 packages. Consider standardizing to short form (`TruncARE`, `SodaARE`, `AdaptiveARE`, `PGMARE`)
- [ ] Constructor names: mix of verbose (`NewApproximateRangeEmptinessSoda`) and short (`NewBloomARE`)
- [ ] `IsEmpty` parameter types: split between `bits.BitString` and `uint64` — consider shared interface or standardize

### Style

- [ ] `are_optimized`: exported struct fields (`K`, `RangeLen`, `MinKey`, `TruncateBits`, `IsExactMode`) not read outside package — make private or add accessor methods
- [ ] `are_soda_hash:56`: `// odd` comment on `| 1` — either remove or explain why odd is required for 2-universal hash

### Outdated documentation

- [ ] `are_hybrid/hybrid_are.go:32`: "use the larger of the two" — no second formula exists
- [ ] `are_pgm/are_pgm.go:41-51`: doc comment attached to `Smooth` variant instead of plain
- [ ] `emptiness/README.md`: only mentions `ere` and `are`, missing 5 packages

## Low Priority / Structural

- [x] `emptiness/bench/` — migrated to `Thesis-Bench-industry/bench/performance_test.go`, deleted from submodule
- [ ] `are_trunc` does not accept `rangeLen` — implicitly hardcodes L=2 in K formula, unlike all other ARE packages
