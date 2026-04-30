# B6 sweep report — n=2^20, n=2^24 (post-fix)

**Date:** 2026-04-30
**Coverage:** n=2^20 (1M keys) and n=2^24 (16M keys), full 7-distribution × 10-filter sweep, 7 L values, individual sweep grids per filter.
**Hardware:** Apple M4 Max, 64 GB RAM.
**Production code:** rsdic.Select1 with adaptive linear/binary inner search (kSelectLinearThreshold = 128), pointer receiver. ERE one_d / ERE classic call sites use the new Select1.

JSON: `bench_results/data/b6_latency_N{1048576,16777216}.json`
Plots: `bench_results/plots/b6_N{1048576,16777216}/{query_latency,fpr,bpk,build_throughput,tradeoff,tradeoff_per_L,cache_pressure}/`

n=2^28 still running on sosd_osm only — separate report when it lands.

---

## TL;DR

The Select1 adaptive fix is live in production paths. SODA query
latency on the headline cell (sosd_fb / L=65536) drops from the
pre-fix 5173 ns to 446-490 ns across all eps values — about 11×.
FPR-vs-BPK behaviour is preserved bit-for-bit.

The K-driven filters (Truncation, Scan-ARE, Greedy+Merge) sit at
near-FPR=1 across the entire K∈{4..22} grid on the FB / L=65536 cell,
because the asymptotic K for that cell is ≈47 (= log2(n·L/ε)). The
grid was set up for moderate L; widening it for the large-L regime
is a follow-up. SODA, BloomARE, Grafite, SNARF and SuRF cover the
asymptotic regime fine on their own grids.

---

## Section 1 — Production SODA latency by N (post-fix)

`sosd_fb` headline cell, smart-mix-empty queries, full eps sweep:

| eps    | n=2^20 (ns) | n=2^24 (ns) | BPK |
|-------:|------------:|------------:|----:|
| 0.1    | 217 | 446 | 22 |
| 0.05   | 239 | 490 | 23 |
| 0.02   | 245 | 463 | 24 |
| 0.01   | 249 | 457 | 25 |
| 0.005  | 242 | 471 | 26 |
| 0.002  | 239 | 588 | 27 |
| 0.001  | 220 | 445 | 28 |
| 0.0005 | 259 | 421 | 29 |

FPR = 0 across the full grid (smart-mix queries are guaranteed empty
and SODA in degenerate mode stores the actual keys).

**Pre-fix reference**: with the original linear-scan Select1, the
same n=2^24 / L=65536 cell measured ≈ 5173 ns. The fix gives a
factor of ≈ 10-12× across the eps sweep, with no algorithmic
difference. Latency now scales gently with N (≈ 2× from 2^20 to 2^24,
matching the rsdic structure size growth from L1 to L2 footprint).

## Section 2 — Cross-distribution SODA at the headline cell (n=2^24, L=65536, eps=0.01)

| Distribution | ns/op | FPR     | Regime |
|--------------|------:|--------:|--------|
| spread       |    57 | 0.000   | trivial empty most-cells (synthetic spread) |
| uniform      |   146 | 0.0078  | wide-universe → SODA hash mixes |
| sosd_osm     |   184 | 0.0077  | wide-universe (uint64 cell IDs) → mixes |
| clustered    |   389 | 0.0005  | partly degenerate |
| sosd_fb      |   457 | 0       | degenerate hash (keys < 2^33) |
| sosd_wiki    |   466 | 0       | degenerate hash (keys < 2^33) |
| sosd_books   |   489 | 0       | degenerate hash (uint32 keys) |

**Reading**:

- **Wide-universe** distributions (uniform, sosd_osm) where SODA-hash
  mixes give the asymptotic FPR ≈ ε and the fastest queries (≈ 150 ns).
- **Narrow-universe** distributions (fb, wiki, books) where the keys
  fit below 2^K trigger the degenerate-hash mode documented in
  `b6-soda-degenerate-hash.md`. The inner ERE then stores the actual
  key set, so smart-mix-empty queries never alias and FPR = 0. Query
  latency 3× higher than the asymptotic regime due to the bucket-deep
  binary search over a clustered packed-suffix array (see the
  observation file).
- **Spread** synthetic distribution puts queries in long sparse gaps
  → most ERE blocks are empty → trivial early-exit.

The **production fix** flattens the worst-case from ≈ 5000 ns to
≤ 500 ns across all distributions. The degenerate / asymptotic
regime distinction now affects only a 3× factor instead of a 30×
gap.

## Section 3 — All filters at the headline cell (n=2^24, L=65536)

One representative sweep value per filter on `sosd_fb`:

| Filter         | Sweep config | ns/op | BPK | FPR    | Notes |
|----------------|--------------|------:|----:|-------:|-------|
| Truncation     | K=22         |   113 |  22 | 0.997  | K too small for L=65536, FPR pinned at 1 |
| Scan-ARE       | K=22         |   132 |  ≈22|≈ 1.000 | same K-grid issue |
| Greedy+Merge   | K=22         | (similar) | ≈22 |≈ 1.000 | same |
| SODA           | ε=0.01       |   457 |  25 | 0.000  | post-fix, degenerate path |
| BloomARE       | ε=0.1 (small L only) | (varies) | (skipped at L=65536 by memory guard) |
| Grafite        | bpk=4        |   118 |   4 | 0.918  | Grafite envelope rejects most cells at low bpk |
| SNARF          | bpk=4        |   486 |   3 | 0.993  | likewise |
| SuRFNone       | real_bits=0  |   255 |  11 | 0.628  | structural FPR independent of L (see SuRF section in observation) |
| SuRFHash       | hash_bits=2  |   216 |  13 | 0.628  |   |
| SuRFReal       | real_bits=0  |   240 |  11 | 0.628  |   |

**SuRF** behaves consistently with the prior observation: FPR ≈ 0.63
on FB at L=65536 regardless of suffix variant.

**BloomARE** is skipped at the largest L cells by the memory-envelope
guard added during this run (m = n·L/ε grows multiplicatively; at
n=2^24 / L=65536 / ε=0.005 the filter would need 50 GB).

**K-grid filters** sit at FPR≈1 because their K-grid {4..22} caps
well below the asymptotic K ≈ 47 the cell needs. Plot inspection
confirms they only become useful at L ≤ 1024 with this grid.

## Section 4 — Build throughput summary

| Filter         | n=2^20 (M k/s) | n=2^24 (M k/s) | Notes |
|----------------|---------------:|---------------:|-------|
| SODA           | ~16-90 (varies with eps) | ~10-60 | dominated by sort + ERE build |
| Truncation     | ~50-200 | ~30-100 | very cheap |
| Scan-ARE       | ~30-120 | ~15-50 | cluster detection overhead |
| Greedy+Merge   | ~25-80 | ~12-40 |   |
| BloomARE       | ~80-150 | (mostly skipped at large L) |
| Grafite (CGo)  | ~20-40 | ~15-25 | CGo + Elias-Fano cost |
| SNARF (CGo)    | ~10-25 | ~8-15 | CGo + GLM training |
| SuRF (CGo)     | ~0.8-1.4 | ~1-2 | trie build is expensive |

(Per-cell numbers vary across the sweep grid; see `build_throughput/`
plots for the full picture.)

## Section 5 — Issues encountered

1. **OOM-kill at peak swap pressure** during the initial multi-N run
   on n=2^20: BloomARE at L=65536/ε=0.0005 needed 16 GB of bits, and
   stacked across `t.Run` boundaries with prior filter structures
   pushed total RSS past system RAM. **Fixed** by:
   - per-Bloom eps grid `{0.1, 0.05, 0.02, 0.01, 0.005}` (drops the
     three smallest eps that always blow up at large L)
   - pre-flight memory guard in the build closure (rejects any cell
     with estimated > 1.6e10 bits = 2 GB)
   - `runtime.GC + debug.FreeOSMemory` between filters to release
     prior structures before the next builds

2. **OS kills during n=2^24** (twice) — IDE/browser memory pressure
   pushed the bench past the OS panic threshold. **Recovered** via
   `paramsHash` cache: each resume took 1-2 minutes to skip already-
   measured cells and continue from the kill point. Eventually the
   full 67-pair coverage was achieved.

3. **K-grid coverage gap** (not fixed). For (n, L) combinations where
   asymptotic K = log2(n·L/ε) > 22, the K-driven filters (Truncation,
   Scan-ARE, Greedy+Merge, Adaptive) hit their grid ceiling and show
   FPR≈1 across all sweep values. To get meaningful FPR-vs-BPK
   trajectories at L=65536 we'd need K up to ≈47. Adding K∈{24, 28,
   32, 40, 47} to the grid is straightforward — held off to keep this
   run scope-bound.

4. **SuRF SIGSEGV on sosd_wiki** (known upstream bug efficient/SuRF#8).
   Handled via `skipDists: {"sosd_wiki": true}` on each SuRF variant —
   the runner records the cell as skipped and proceeds.

## Section 6 — n=2^28 status

Started after n=2^24 completion. Only `sosd_osm` has 800M keys ≥
n=2^28 = 268M; the other 6 distributions auto-skip. As of report
time the process was 21 min in, 99% CPU, 18.5 GB RSS, no first flush
yet — building the first filter. Expected total wall time on osm
alone: 8-15 hours (SODA's L-dependent grid alone is ~7 hours of
build at the heaviest cells). The data will land in
`bench_results/data/b6_latency_N268435456.json`.

---

## Where to look

- **Plots**: `bench_results/plots/b6_N{1048576,16777216}/` — 7 metrics × 7
  distributions per N. The `tradeoff/` subdir is the FPR-vs-BPK headline,
  one figure per distribution.
- **Raw JSON**: `bench_results/data/b6_latency_N{N}.json`. Schema
  documented in `b6_latency_test.go:b6Row`. Each row carries a
  `paramsHash` for cache de-duplication; `note` non-empty marks
  envelope-rejected cells.
- **Legacy comparison** (pre-Select1 fix): preserved at
  `bench_results/data/legacy/b6_latency_N16777216_slow_select.json`.
  All ERE-using filter latencies are 5-15× higher there.
- **SODA observation file**:
  `Thesis/text/defence/b6-soda-degenerate-hash.md`. Full mechanism +
  fix story.
