# B6 sweep — final morning report

**Date:** 2026-05-01.
**Coverage:** N ∈ {2²⁰, 2²⁴, 2²⁸} × P ∈ {1, 2, 4, 8, 16} on the
B6 latency suite.
**Hardware:** Apple M4 Max (12 perf + 4 eff cores), 64 GB RAM.

JSON data: `bench_results/data/b6_latency_N{1048576,16777216,268435456}.json`
Plots: `bench_results/plots/b6_N{N}/{query_latency,fpr,bpk,build_throughput,
tradeoff,tradeoff_per_L,cache_pressure}/`.
Progress log: `bench_results/b6_progress.log`.

---

## TL;DR

Production rsdic.Select1 fix is live and validated end-to-end across the
whole sweep. SODA query latency at the headline cell (sosd_fb / L=65536,
n=2²⁴, K=36, FPR=0) measures **255 ns at P=1** and scales to **23 ns at
P=16** (11× parallel speedup, near-linear to P=8 then sublimited by
M4's 12 perf cores).

Adaptive K-grid + build-once-per-K + L-as-query-only architecture cut
the sweep from initial estimate of 8–15 hours per N down to 60 min on
average. All 5 P values together completed in 4h 52min wall time.

---

## Sweep coverage (rows per (N, P))

| N      | P=1  | P=2  | P=4  | P=8  | P=16 | total |
|-------:|-----:|-----:|-----:|-----:|-----:|------:|
| 2²⁰    | 5077 | 3358 | 3274 | 3274 | 3274 | 18257 |
| 2²⁴    | 5140 | 3453 | 3453 | 3453 | 3453 | 18952 |
| 2²⁸    |  839 |  580 |  580 |  580 |  580 |  3159 |
| total  |11056 |7391  |7307  |7307  |7307  |40368  |

**P=1 has more rows** because CGo filters (Grafite, SNARF, SuRFNone,
SuRFHash, SuRFReal) hold C++ filter state that is not concurrent-safe;
P>1 skips them and measures only the K-driven Go filters
(Truncation, Scan-ARE, Greedy+Merge, SODA, BloomARE).

**N=2²⁸ is sosd_osm only** — all other distributions have < 268M
keys available, so they auto-skip.

---

## Section 1 — SODA cross-distribution at the headline cell

**n=2²⁴, L=65536, P=1, FPR-saturated K** (highest K measured per dist
before adaptive break or BPK>25):

| Distribution | Saturating K | ns/op | BPK | FPR     | Regime |
|--------------|-------------:|------:|----:|--------:|--------|
| spread       |       36     |    16 |   0 | 0.000   | trivial empty |
| sosd_books   |       26     |   119 |   4 | 0.000   | degenerate-hash, low BPK saturates |
| uniform      |       48     |   149 |  26 | 0.0039  | wide-universe asymptotic ε |
| sosd_osm     |       48     |   192 |  26 | 0.0039  | wide-universe asymptotic ε |
| sosd_wiki    |       28     |   235 |   6 | 0.000   | degenerate-hash |
| sosd_fb      |       36     |   255 |  14 | 0.000   | degenerate-hash |
| clustered    |       48     |   387 |  26 | 0.0002  | wide-universe asymptotic ε |

Pre-fix legacy reference for sosd_fb / L=65536: ≈ 5173 ns. The current
255 ns is a **20× improvement** end-to-end on this cell.

The "regime" column matches the SODA degenerate-hash analysis in
`b6-soda-degenerate-hash.md`. Wide-universe distributions converge at
ε ≈ 0.004 (matches the 0.01 target eps modulo bucket variance);
narrow-universe ones store the actual key set so smart-mix queries
never alias and FPR collapses to 0.

---

## Section 2 — Parallel scaling (the headline new finding)

SODA on **sosd_fb / L=65536 / K=36** (the FPR=0 saturation point on FB):

| P  | ns/op | speedup | comment |
|---:|------:|--------:|---------|
| 1  |   255 |   1.0×  | baseline (after rsdic Select1 fix) |
| 2  |   120 |   2.13× | near-linear |
| 4  |    69 |   3.70× | near-linear |
| 8  |    33 |   7.73× | near-linear |
| 16 |    23 |  11.1×  | sublinear past 12 perf cores |

Speedup curve is near-linear up to P=8, then bends at P=16 — Apple
M4 Max has 12 perf cores plus 4 efficiency cores, so 11× at P=16
matches the perf-core ceiling.

The fact that this scales near-linearly through P=8 is informative:
the rsdic + packedData read pattern fits comfortably within the 64 GB
RAM at n=2²⁴, so contention is on instruction throughput rather than
memory bandwidth. (Cache_pressure plots in `bench_results/plots/
b6_N{N}/cache_pressure/` show ns/op vs P per filter; see for sub-K
granularity.)

### Cross-N parallel saturation

SODA at FB L=65536, **highest K with all P measured**:

| N    | P=1 | P=2 | P=4 | P=8 | P=16 |
|-----:|----:|----:|----:|----:|-----:|
| 2²⁰  | (saturates earlier) |
| 2²⁴  | 255 | 120 |  69 |  33 |   23 |
| 2²⁸  | 272 (osm only) |
|      |     |     |     |     |      |

n=2²⁸ data is sosd_osm-only; FB / wiki / books / synthetic don't have
enough keys.

---

## Section 3 — Bench harness improvements that landed during this run

The sweep itself drove a stack of architectural fixes; recording
because they're now permanent:

1. **rsdic.Select1 adaptive linear/binary** with kSelectLinearThreshold
   = 128 (commits `43d4c45`, `43ce7bb`, `ca9fe2c` and the perf-thread
   leading up). Production swap done; legacy 5173 ns/cell is gone.
2. **L-as-query-parameter, not filter-parameter**. Refactor at
   `b924ed2` and `16f7196`. Filter built once per K, queries at all L
   reuse the same filter. Cuts build time by 7× on the L-grid.
3. **BloomARE rewritten as L-independent** via `NewBloomAREFromPointFPR`
   (`3ad161d`). Sweep now BPK-driven with grid {4..64}; range FPR =
   1 - (1-pointFPR)^L computed per query, never baked in.
4. **SODA Seed parameter** instead of leaked rangeLen (`8872ebf`,
   `e806d40`). Caller-controlled seed; no spurious dependency on L.
5. **Adaptive K early-exit** with per-L saturation tracking
   (`b924ed2`). Once an L hits FPR=0 we skip its query at higher K.
   Sweep breaks when bpk_used > 25 (memory budget) or every L has
   saturated.
6. **Per-(filter, L) skip via skipLs** (`eed160c`). BloomARE skips
   L≥4096 since query time is O(L) and the analytical formula
   covers larger L anyway.
7. **CGo filters skip at P>1** (`<latest>`). SuRF / SNARF / Grafite
   wrappers hold non-concurrent C++ state; we skip them at P>1
   instead of risking SIGSEGV mid-sweep.
8. **B6_N multi-N env**, per-N JSON files, plotter glob (`2efb862`).
9. **t.Cleanup-safe progress logger** at
   `bench_results/b6_progress.log` (`ac35a6b`, `00433d5`). Real-time
   per-cell visibility regardless of `go test`'s buffering.
10. **Adaptive K-grid extended to {4, 6, …, 60, 64}** to reach
    asymptotic K = log₂(n·L/ε) for the largest cells.

---

## Section 4 — What's where

- **Plots**: 3 directories per N, 7 metrics each
  (`query_latency`, `fpr`, `bpk`, `build_throughput`, `tradeoff`,
  `tradeoff_per_L`, `cache_pressure`).
- **Cache-pressure plots** (`cache_pressure/`) show ns/op as a
  function of P at fixed (filter, K, L). The headline FPR-vs-BPK
  curves are in `tradeoff/<dist>.svg` and `tradeoff_per_L/<dist>/L<L>.svg`.
- **Raw data**: `bench_results/data/b6_latency_N{N}.json`. Schema
  in `bench/b6_latency_test.go::b6Row`. paramsHash de-dups; `note`
  non-empty marks envelope-rejected cells.
- **Legacy comparison** (pre-fix Select1):
  `bench_results/data/legacy/b6_latency_N16777216_slow_select.json`
  preserved for sanity comparison.
- **SODA observation file**:
  `Thesis/text/defence/b6-soda-degenerate-hash.md`. Full mechanism
  + fix story, rsdic profiling, threshold sweep.

---

## Section 4.5 — Truncation FPR=0.5 plateau on `uniform` (diagnosed)

You flagged "where is Truncation on `tradeoff_per_L/uniform/L128.svg`?"
Agent dispatched, here's the verdict.

**Not a filter bug.** The plateau is workload-induced by `smart_mix`'s
near-key branch.

### Mechanism

`generateSmartQueries` (in `bench/sosd_test.go`, ~line 70–169) splits the
query workload as:

| Branch     | weight | range placement                       |
|------------|-------:|---------------------------------------|
| near-key   |   50%  | offset ∈ [-5L, +5L] from a stored key |
| in-gap     |   30%  | uniform inside a between-keys gap     |
| out-of-univ|   20%  | outside [minK, maxK]                  |

For Truncation at K=48 on 60-bit keys, t = W − K = 12, so each stored key
covers a phantom interval of size `2^t = 4096` keys. Smart-mix picks
near-key offsets up to ±5·L = ±640. Since 640 ≪ 4096, **every near-key
query lands inside the phantom of the picked stored key** ⇒ forced FP.

Numerical fit (diagnostic, n=2²⁰, K=48):
- near-key FPR: **0.9316**
- in-gap FPR: **0.0000** (gaps ≈ 2³⁶ ≫ phantom 2¹²)
- out-of-univ FPR: ≈ 0
- Predicted smart-mix FPR: 0.50·0.93 + 0.30·0 ≈ **0.466**
- Observed in B6 at K=48: **0.467** ✓ exact match

For K=44 (phantom 2¹⁶) and K=36 (phantom 2²⁴), the phantom intervals of
neighboring keys overlap each other (avg gap ≈ 2³⁶), so the FPR floor
inflates to ≈ 0.498/0.501 — also matching B6.

### Why this isn't a contradiction with `phantom_comparison.svg`

`are_trunc/README.md` lines 44–58 spell out exactly this: "All false
positives are concentrated near stored keys [...] Only uniformly spread
keys with **gaps ≫ 2^t** give the theoretical FPR." The README also
specifies "uniformly distributed queries" as a precondition. Smart-mix
violates the latter by design — the 50% near-key weight is adversarial
against prefix-truncation.

`phantom_comparison.svg` is correct in its own scope: it depicts a
single-key universe under uniform queries.

### What to do

Two options, both legitimate:

1. **Add `pure_uniform` query mode** for the `uniform` distribution cell
   (`nNear = nGap = 0`, all weight to uniform-over-[minK, maxK]).
   Apples-to-apples comparison; invalidates current uniform cache (~1/9
   of the sweep).
2. **Annotate the existing plot** with a caption explaining the phantom-
   vs-near-key interaction. Reframes the data as empirical confirmation
   of the README's qualitative claim. No re-run needed.

**Recommended:** do both — one B6 column with `smart_mix` (= adversarial
near-key, what we have), one with `pure_uniform` (= classical uniform
workload, where Truncation should shine). Lets the thesis show both
"Truncation excels under uniform-uniform" *and* "Truncation degenerates
under near-key adversarial" with the same primitive.

Pinging here for your call — autonomous mode didn't take this action
because it changes the workload definition for ~1/3 of the sweep, which
is a thesis-narrative decision, not a bench-harness fix.

---

## Section 5 — What's next (if you want to extend)

- **More N points**. n=2¹⁶ for "small filter" calibration; n=2³⁰ if
  data file exists (only sosd_osm scales that far, and it has 800M
  keys).
- **Synthetic-distribution larger files**. uniform/spread/clustered
  are pinned at 16M keys; regenerating them at 256M would let
  n=2²⁸ have non-osm distributions too.
- **CGo concurrent-safe wrapping**. Currently we skip Grafite/SNARF/
  SuRF at P>1; a per-instance mutex around IsEmpty would let them
  participate in cache-pressure analysis at the cost of contention
  noise.
- **Cross-validation against pre-fix SODA**. Run one cell with the
  pre-rsdic-fix Select1 path, compare ns/op directly. The legacy
  JSON has it but with old query strategy; a clean apples-to-apples
  re-measurement would bolster the headline number.
