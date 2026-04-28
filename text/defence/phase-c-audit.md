# Phase C Audit — uint64 fast path readiness

> Read-only audit performed 2026-04-28 ahead of the BitString → uint64 API
> migration discussed for Phase C. No code changes — only mapping the
> current state so the refactor scope is concrete when we start.

## n=2²⁸ commitment confirmed

Search of `Thesis/text/src/`:

| File | Line | Context |
|------|------|---------|
| `evaluation.tex` | 83 | "report headline numbers at $n \in [2^{24},\ 2^{28}]$" |
| `evaluation.tex` | 117 | $n = 2^{28}$ as a benchmark scale (paper-evaluation, SOSD) |
| `ere.tex` | 87 | "uniform 64-bit keys for $n \in \{2^{20}, 2^{24}, 2^{28}\}$" |
| `ere.tex` | 108 | metadata BPK table row for $n = 2^{28}$ |

**Conclusion**: $n = 2^{28}$ is locked into the written thesis. We need
the uint64 fast-path before running it; otherwise the alloc cost of
constructing 256M `bits.BitString` keys plus per-query trampolines
becomes prohibitive (and would not finish on the available hardware
inside the defence prep window).

## Per-package state

### Already uint64-native (no work needed)

| Package | Constructor | IsEmpty | Notes |
|---------|-------------|---------|-------|
| `are_soda_hash` | `NewSodaARE(uint64, …)`, `NewSodaAREUint64`, `NewSodaAREUint64InPlace`, `NewSodaAREFromK(uint64, …)` | `IsEmpty(a, b uint64)` | Full fast path. Multiple variants for in-place vs copy and ε vs K. |
| `are_bloom` | `NewBloomARE(uint64, …)` | `IsEmpty(a, bEnd uint64)` | Already trivially uint64. |

### BitString-only (need uint64 wrapper or full migration)

| Package | Constructor | IsEmpty | Headline-relevant? |
|---------|-------------|---------|--------------------|
| `are_trunc` | `NewTruncARE([]bits.BitString, …)`, `NewTruncAREFromK` | `IsEmpty(a, b bits.BitString)` | No (dropped from headline plot in Task 0.5) |
| `are_adaptive` | `NewAdaptiveARE([]bits.BitString, …)`, `NewAdaptiveAREFromK` | `IsEmpty(a, b bits.BitString)` | No (dropped from headline plot, but used **internally** by hybrid_scan / greedy_scan) |
| `are_hybrid_scan` (Scan-ARE) | `NewHybridScanARE`, `…FromK`, `…WithPolicy`, `…FromBPK` — all `[]bits.BitString` | `IsEmpty(a, b bits.BitString)` | **YES — headline series.** |
| `are_greedy_scan` (Greedy+Merge) | `NewGreedyScanARE`, `…FromK`, `…FromKRaw` — all `[]bits.BitString` | `IsEmpty(a, b bits.BitString)` | **YES — headline series.** |

### ERE backends

| Package | Constructor | IsEmpty | Notes |
|---------|-------------|---------|-------|
| `ere` (one-vector) | `NewExactRangeEmptiness([]bits.BitString, universe BitString)`, **`NewExactRangeEmptinessUint64([]uint64, keyBits uint32)`** | `IsEmpty(a, b bits.BitString)`, `LinearIsEmpty` (also BitString) | Build path already has uint64 fast lane. Query path is BitString-only — that's the per-query alloc. |
| `ere_one_d` | Same shape as `ere` | Same | Mirror of `ere`. |
| `ere_global` | `NewGlobalExactRangeEmptiness([]bits.BitString, …)` | `IsEmpty(a, b bits.BitString)` | No uint64 build path. |
| `ere_theoretical` | `NewTheoreticalExactRangeEmptiness([]bits.BitString, …)` | `IsEmpty(a, b bits.BitString)` | No uint64 build path. |

## Where the alloc hurts

The trampoline in `bench/comparison_test.go:274`, `289`:
```go
func(a, b uint64) bool {
    return f.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b))
}
```
allocates **two** `bits.BitString` per query. At 2¹⁸ queries × 3 seeds × 24 K-grid points × 7 L values × 9 distributions × 2 filters that go through this path
(Scan-ARE + Greedy+Merge) ≈ **1.6 billion BitString allocations** per full run. Same trampoline pattern lives in `are_hybrid_scan/are_greedy_scan` callers
through their fallback to `are_adaptive`.

`testutils.TrieBS` calls `bits.NewFromTrieUint64(val, 64)` — a fresh
heap allocation each time. There is no pool, no reuse.

At $n = 2^{28}$ the build phase is also painful: the current
`are_hybrid_scan.NewHybridScanARE` takes `[]bits.BitString`, so the
caller must convert 256M uint64 keys into 256M heap-allocated
BitStrings before calling — peak memory ≈ 256M × ~64B (slice header +
data) = ~16 GB just for the conversion.

## Refactor shape (proposal, not implemented)

Two-stage migration; can be done piece-meal:

**Stage 1 — leaf packages (`are_trunc`, `are_adaptive`)**:
- Add `NewAdaptiveAREUint64(keys []uint64, keyBits uint32, …)` etc.
- Add `IsEmptyU64(lo, hi uint64) bool` that does the BitString conversion **inside** the call (one alloc per call instead of two trampolines + two TrieBS).
- Better: do it without alloc — derive prefix/suffix bits arithmetically.

**Stage 2 — composite packages (`are_hybrid_scan`, `are_greedy_scan`)**:
- Lift their constructors to accept `[]uint64` + `keyBits uint32`.
- Internally call the `…Uint64` variants from Stage 1.
- Keep BitString constructors as thin wrappers that convert once at
  build-time (acceptable cost) and delegate.

**Stage 3 — ERE query path**:
- Add `IsEmptyUint64(lo, hi uint64) bool` to `ere` and `ere_one_d`.
- Update `are_adaptive.sodaIsEmpty` / `are_soda_hash.IsEmpty` to call
  the uint64 path on the ERE backend instead of converting back to
  BitString just to call the BitString IsEmpty.

## Risks

- **Test fixtures**: every `Thesis/emptiness/are_*/` test file constructs
  filters with `[]bits.BitString` and queries with `bits.BitString`.
  Roughly 30+ test files. Plan: keep BitString APIs as thin wrappers
  so existing tests remain green; only the bench harness and new
  benchmark tests need to use the uint64 path.
- **Cache invalidation**: behaviour identical, params hash unchanged
  → existing JSON cache should remain valid. Verify with diff after
  refactor before discarding.
- **paramsBPKSweep / paramsKGrid**: do **not** change. The fast-path
  is purely a perf rewrite, not a semantic change.

## Worktree-friendly

Every stage above is independently testable in isolation; the bench
harness can still run the BitString path while the refactor proceeds.
Spin up a `phase-c` worktree, do the migration there, run unit tests
+ a quick `n=2^20` smoke before merging back to master and running
$n = 2^{28}$.

## Recommendation (for the morning discussion)

1. Decision needed: kick this off in a worktree now, or wait until
   after the text-stream tasks for Week 1 (intro / related work) are
   drafted.
2. Scope: all four BitString-only packages (`are_trunc`, `are_adaptive`,
   `are_hybrid_scan`, `are_greedy_scan`) + `ere`/`ere_one_d` query path.
3. Time estimate: full refactor + smoke ≈ 1 working day; another day
   for the $n = 2^{28}$ measurement run.
