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
- [x] Add industry results cache, as they are consistent, and no need to rerun them every time. (selective rebuild with ONLY/SKIP env vars)

### Testing gaps

- [x] `are_adaptive`: add systematic no-FN property tests (multiple seeds/sizes, clustered dist)
- [ ] `are_adaptive`: add FPR accuracy test
- [ ] `are_adaptive`: add Build/N and Query/N benchmarks
- [ ] `are_soda_hash`: add Build/N and Query/N benchmarks
- [ ] `are_pgm`: add Build/N and Query/N benchmarks
- [ ] `are_bloom`: add Build/N and Query/N benchmarks
- [ ] ALL packages: add Query/L benchmark (sweep range length L) — missing everywhere
- [ ] `are_hybrid`: Query/N benchmark only at fixed N=2^20, add N-sweep
- [x] Standardize no-FN tests to use clustered distribution (all packages now have uniform + clustered) 

### Hybrid cluster detection

- [x] `are_hybrid`: `detectClusters` fails on sequential — fixed with `>` instead of `>=`. New `are_hybrid_scan` package with 1D DBSCAN addresses all limitations (merging, equidistant, dual fallback).

### Dead code / bugs

- [ ] `ere/exact_range_emptiness.go:97`: `extractSuffixAsUint64` — `KeySize` parameter unused, remove
- [ ] `ere_theoretical`: field `L` stored but never read — remove
- [ ] `ere_theoretical`: `locators` built and measured but never used for queries — document or remove
- [ ] `ere_theoretical`: `getBlockIndex` duplicates `ere.GetBlockIndex` with explicit loop — deduplicate
- [ ] `ere_theoretical:141-149`: scratch-pad comments left in code — clean up
- [ ] `ere_theoretical:61`: outdated comment "Store keys with prefix relative to block" — keys stored as-is

## Medium Priority

### Code duplication

- [x] `pairwiseHash` copy-pasted in `are_soda_hash:23` and `are_adaptive:26` — extract to shared `internal/hash`
- [x] `no_fn_prop_test.go` triple-clone across `are_trunc`, `are_hybrid`, `are_soda_hash` — extract shared test harness
- [x] Key hashing + dedup loop duplicated in `are_soda_hash` and `are_adaptive` constructors

### Naming inconsistencies

- [x] Rename `are` → `are_trunc`
- [x] Struct names: 5 different conventions across 6 packages. Consider standardizing to short form (`TruncARE`, `SodaARE`, `AdaptiveARE`, `PGMARE`)
- [x] Constructor names: mix of verbose (`NewApproximateRangeEmptinessSoda`) and short (`NewBloomARE`)
- [ ] `IsEmpty` parameter types: split between `bits.BitString` and `uint64` — consider shared interface or standardize

### Style

- [ ] `are_adaptive`: exported struct fields (`K`, `RangeLen`, `MinKey`, `TruncateBits`, `IsExactMode`) not read outside package — make private or add accessor methods
- [ ] `are_soda_hash:56`: `// odd` comment on `| 1` — either remove or explain why odd is required for 2-universal hash

### Outdated documentation

- [x] `are_hybrid/hybrid_are.go:32`: "use the larger of the two" — no second formula exists
- [x] `are_pgm/are_pgm.go:41-51`: doc comment attached to `Smooth` variant instead of plain
- [x] `emptiness/README.md`: updated — now lists all packages including are_hybrid_scan

## Low Priority / Structural

- [x] `emptiness/bench/` — migrated to `Thesis-Bench-industry/bench/performance_test.go`, deleted from submodule
- [ ] `are_trunc` does not accept `rangeLen` — implicitly hardcodes L=2 in K formula, unlike all other ARE packages
