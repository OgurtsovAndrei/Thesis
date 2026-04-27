# Thesis Lock-In Plan (2026-04-27 → 2026-05-20 text submission; defence shortly after)

> Source spec: `Thesis/text/defence/talk-structure.md`
> Adapted from `superpowers:writing-plans` for non-code deliverables.

**Goal:** submit a finished thesis text on 2026-05-20 and present an 11–12 minute defence with modular DLC support, on the structure agreed in `talk-structure.md`.

**Architecture:** three parallel streams that converge in week 4.
- **Text stream** — finish missing chapters (intro, related work, conclusion, abstract, limitations); fill 9 empty BPK tables; remove all `\colorbox{orange}{TODO}` markers.
- **Measurement stream** — confirm n=2²⁴ SOSD coverage suffices; run B6 (build throughput + query latency vs Grafite/SNARF/SuRF).
- **Slide stream** — build Beamer deck in `defence/slides/` matching `talk-structure.md` (10 core + 4 DLC + 10 backup).

**Critical path:** framework fixes → kick off full rerun (background) → fill BPK tables → finalize evaluation chapter → conclusion text → slide 10 → pre-defence → DLC selection → final talk.

**Why Week 0 exists:** all current data in `bench_results/` was generated **before** the one-vector ERE optimization landed. It's stale and must be regenerated. Before kicking off a multi-day rerun we make targeted framework fixes so the new plots are publication-quality (otherwise we re-run twice).

**Compile commands:**
- Thesis: `cd Thesis/text && make thesis`
- Beamer deck: `cd Thesis/text/defence/slides && latexmk -pdf defence.tex`

**Scale decision (locked):** SOSD tables at $n=2^{24}$; synthetic tables at $n=2^{20}$. Caption explicit about the difference. Avoids ~1 week of synthetic-at-$2^{24}$ runs that would not change the headline.

---

## Week 0: Framework fixes + kick off mass rerun (Apr 27 – Apr 30)

All current `bench_results/` data is stale: it predates the one-vector ERE optimization. Cache invalidation does **not** trigger automatically for implementation-only changes (`framework_test.go:426` compares params, not impl). Before kicking off the multi-day rerun we make targeted plot-quality fixes — otherwise we run twice.

**Cache invalidation strategy:** clean slate. Rename `bench_results/` to `bench_results_obsolete_pre_1d_opt/` and start with an empty cache. The existing `ONLY=` / `SKIP=` env vars in `framework_test.go:441` remain intact for future point-selective reruns; we just don't fight them for this one-time event (overengineering a `FORCE` flag would invert the cost/benefit).

### Task 0.1 — X-axis cap at 25 BPK (no rerun, render only)

**Files:**
- Modify: `Thesis/testutils/plot.go:104–115` — add `XMax` field to `PlotConfig`; in linear scale mode clamp `axMaxX = min(axMaxX, XMax)`.
- Modify: callers in `bench/comparison_test.go`, `bench/sosd_test.go` — pass `XMax: 25`.

- [ ] **Step 1:** Read `Thesis/testutils/plot.go` lines 100–130 to confirm the auto-scaling block.
- [ ] **Step 2:** Add `XMax float64` to `PlotConfig`. In the linear branch: `if cfg.XMax > 0 && axMaxX > cfg.XMax { axMaxX = cfg.XMax }`.
- [ ] **Step 3:** In all `GenerateTradeoffSVG` call sites, set `XMax: 25` (or `30`; pick one and stick).
- [ ] **Step 4:** Re-render existing plots in plot-only mode: `PLOT_ONLY=1 go test -run TestComparison -v ./bench/ -timeout 5m`. Verify visual change in one SVG.
- [ ] **Step 5:** Commit: `git add testutils/plot.go bench/comparison_test.go bench/sosd_test.go && git commit -m "feat(plot): cap X-axis at 25 BPK to focus on production-relevant range"`. (Parent repo this time, not Thesis submodule — `bench/` lives in parent.)

### Task 0.2 — Audit `YFloor` consistency across plot call sites (no rerun)

**Mechanism is already in place:** `plot.go:39 PlotConfig.YFloor` is a configurable field; lines 119–121 use it when set, fallback `1e-8` otherwise. The "3e-07" we see in the L65536 SVG is a caller passing `1.0/queryCount ≈ 3.3e-7`. This is correct behaviour. The risk is **inconsistency** across callers — some may pass 0 (silent fallback to `1e-8`) or a magic constant.

- [ ] **Step 1:** `grep -rn "GenerateTradeoffSVG\|GeneratePerformanceSVG" bench/ Thesis/` — find every call site.
- [ ] **Step 2:** For each, verify the `YFloor` field is set to `1.0 / float64(queryCount)` (or equivalent). Fix any that pass 0 or a constant.
- [ ] **Step 3:** Re-render with `PLOT_ONLY=1` (only if any call site changed). Sanity-check one SVG.
- [ ] **Step 4:** Commit only if changes made: `git commit -m "fix(plot): consistent 1/queryCount measurement floor across all callers"`. If everything was already correct: skip step.

### Task 0.3 — Rename stale `bench_results/` and start with clean cache

- [ ] **Step 1:** From repo root: `mv bench_results bench_results_obsolete_pre_1d_opt`. This is a one-line action; takes 5 seconds.
- [ ] **Step 2:** Verify rename succeeded: `ls -d bench_results bench_results_obsolete_pre_1d_opt 2>&1` (first should error, second should exist).
- [ ] **Step 3:** Add the obsolete dir to `.gitignore` so it doesn't pollute git status: `echo "bench_results_obsolete_pre_1d_opt/" >> .gitignore`.
- [ ] **Step 4:** Commit: `git add .gitignore && git commit -m "chore(bench): retire pre-1D-opt results to obsolete dir"`.

(Old data is preserved on disk for reference; the dir can be deleted after defence if desired.)

### Task 0.4 — Extend `bpkSweep` density up to 28

**Files:**
- Modify: `bench/sosd_test.go:33` and any other declarations of `bpkSweep`. Replace `[4, 6, 8, 10, 12, 14, 16, 18, 20]` with `[4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 22, 24, 26, 28]`.

- [ ] **Step 1:** `grep -n "bpkSweep" bench/*.go` — find all declarations; pick one canonical location.
- [ ] **Step 2:** Replace the array.
- [ ] **Step 3:** This **automatically invalidates** Grafite/SNARF/SuRF series (their `paramsBPKSweep` hash changes). On next run they rebuild; k-grid series cache stays untouched.
- [ ] **Step 4:** Commit: `git commit -m "bench: densify bpkSweep up to 28 BPK for CGo filter coverage"`

### Task 0.5 — Drop redundant filters from `allSeries` map

**Files:**
- Modify: `bench/comparison_test.go:52–67`. Remove entries for: `SuRF`, `SuRFHash(8)`, `Truncation`, `Adaptive (t=0)`, `Hybrid`, `CDF-ARE`. Remove their downstream conditional blocks.

Final filter set on FPR-vs-BPK headline plots (**8 series**):

| Group | Series |
|-------|--------|
| Reference (lower bound) | Theoretical |
| Industry baselines | Grafite, SNARF, SuRFReal(8) |
| Default trivial baseline | BloomARE |
| Goswami SODA'15 baseline | SODA |
| Hybrid family (this work) | Greedy+Merge, **Scan-ARE** (headline) |

Rationale for drops:
- `SuRF` (base) and `SuRFHash(8)` — `SuRFReal(8)` dominates them on every benchmark; plotting all three is visual noise.
- `Truncation` and `Adaptive (t=0)` — intermediate hash-design steps building up toward Scan-ARE; not end-products; their dedicated tradeoff plots live in §sec:truncation and §sec:adaptive (chapters 4–5).
- `Hybrid` — gap-percentile predecessor of Scan-ARE; superseded; kept in text Chapter 5 as motivation only.
- `CDF-ARE` — explicitly excluded from defence; in text as documented negative result.

Rationale for keeps:
- `BloomARE` — default reference everyone in storage knows; trivial-baseline anchor.
- `SODA` — direct implementation of the Goswami SODA'15 construction; pairs with `Theoretical` (the gap between them shows ERE metadata overhead in practice).
- `Greedy+Merge` — same hybrid family as Scan-ARE with a different inner ERE backend; carries comparable performance and supports the §sec:eval-ere-backend comparison table.

- [ ] **Step 1:** Read lines 52–67 and downstream conditionals (200–510 area).
- [ ] **Step 2:** Remove the 6 dropped series from `allSeries`, `seriesParams`, the `richData` family switch, and the rebuild groups.
- [ ] **Step 3:** Re-render with `PLOT_ONLY=1`. Verify legend has 8 entries.
- [ ] **Step 4:** Commit: `git commit -m "feat(plot): unified 8-series legend; building blocks moved to chapter-local plots"`

### Task 0.6 — Per-filter bit width: lift global 60-bit cap

**Why:** the global `mask60Keys` (`framework_test.go:21–44`) is applied at `comparison_test.go:594, 622, 876` because **SNARF** overflows when keys are close to UINT64_MAX. Cost: SOSD Wiki at $n=2^{27}$ collapses to ~50% unique keys after the mask (already documented in `distributions.tex` "Note on duplicate keys"). Lifting the cap unlocks honest $n=2^{28}$ for SOSD.

**Strategy:** keep `mask60` helper, but apply it **only inside the SNARF wrapper**, not globally. Other CGo wrappers (Grafite, SuRF) and our own filters (SODA, Truncation, Adaptive, Scan-ARE, Greedy+Merge) get full 64-bit keys.

**Files:**
- Modify: `bench/comparison_test.go:594, 622, 876` — remove global `mask60Keys` calls; pass raw 64-bit keys downstream.
- Modify: `snarf/snarf_cgo.go` (or wherever SNARF's `Build` / `Insert` lives) — mask incoming keys inside the wrapper.
- Modify: SNARF's `Query` / `RangeContains` — mask `[lo, hi]` endpoints similarly.
- Verify: Grafite, SuRF wrappers, all six native ARE filters compile and pass existing unit tests at full 64-bit.

- [ ] **Step 1:** `grep -n "mask60Keys\|mask60Queries" bench/` — list all global call sites; remove them.
- [ ] **Step 2:** Locate SNARF wrapper file (`grep -rn "snarf" snarf/ bench/ | head`). Add `mask60` at insertion and query boundary.
- [ ] **Step 3:** Run unit tests: `cd Thesis && go test ./...` and `go test ./bench/ -run "TestComparison" -short -timeout 5m` (short mode if available; otherwise pick smallest distribution).
- [ ] **Step 4:** If Grafite or SuRF overflow at full 64-bit, scope-creep stop: revert this task, document for post-defence work.
- [ ] **Step 5:** If tests pass, document in `bench/framework_test.go` near the mask helpers: comment that mask60 is now SNARF-only.
- [ ] **Step 6:** Commit: `git commit -m "feat(bench): per-filter bit width — SNARF masks internally, others get full 64-bit"`

This task is the only one in Week 0 that can fail and force scope rollback. Schedule it on a day when you have ~3 hours uninterrupted, ideally before Task 0.7.

### Task 0.7 — Smoke test framework changes on one (dist × L) cell

- [ ] **Step 1:** Pick smallest realistic combination: SOSD Books, n=2²⁰, L=128.
- [ ] **Step 2:** Run cleanly (no env vars needed — `bench_results/` is fresh):
  ```
  go test -run "TestComparison/sosd_books" -v -timeout 1h ./bench/
  ```
- [ ] **Step 3:** Inspect resulting `bench_results/plots/N1048576/sosd_books/L128.svg`. Verify: 8 series in legend, X-axis capped at 25, no premature truncation, SOSD at full 64-bit (large universe span).
- [ ] **Step 4:** If anything looks wrong, fix before launching mass rerun.

### Task 0.8 — Kick off mass rerun (background, multi-day)

**Files:**
- Create: `bench_results/rerun_2026_04_28.log`

- [ ] **Step 1:** Decide scope. Recommended for headline + tables:
  - Distributions: SOSD (4) at $n=2^{24}$ — and try $n=2^{28}$ on Books/FB if Task 0.6 cleared the 64-bit refactor (Wiki still capped by intrinsic dedup); synthetic (5) at $n=2^{20}$.
  - L values: full sweep `{1, 16, 128, 1024, 4096, 16384, 65536}`.
  - Filters: 8 final from Task 0.5.
- [ ] **Step 2:** Launch single-threaded in background (per repo CLAUDE.md, never parallelize benchmarks):
  ```
  go test -run TestComparison -v -timeout 96h ./bench/ 2>&1 \
    | tee bench_results/rerun_2026_04_28.log &
  ```
- [ ] **Step 3:** Capture PID: `echo $! > bench_results/rerun.pid`. Verify alive: `ps -p $(cat bench_results/rerun.pid)`.
- [ ] **Step 4:** Don't block. Move to Week 1 tasks. Check progress daily with `tail -50 bench_results/rerun_2026_04_28.log`.
- [ ] **Step 5:** When rerun finishes (estimate: 3–5 days at $n=2^{24}$, longer if $n=2^{28}$ is included), commit: `git add bench_results/plots/ bench_results/data/ && git commit -m "bench: rerun all FPR-vs-BPK with one-vector ERE + 8-series filter set"`

---

## Week 1: Foundation (May 1 – May 3 — compressed; rerun runs in background)

### Task 1.1 — Audit measurement coverage

**Files:**
- Create: `Thesis/text/defence/measurement-coverage.md`

- [ ] **Step 1:** Iterate over `bench_results/data/N1048576/` and `bench_results/data/N16777216/`; record which (distribution, L) cells have CSVs that contain BPK-vs-FPR sweeps suitable for filling Table~\ref{tab:bpk-fb} … \ref{tab:bpk-temporal} in `evaluation.tex`.
- [ ] **Step 2:** Cross-reference with the 9 distribution × 2 L (128, 65536) target matrix from `talk-structure.md` slide 8 / evaluation §sec:eval-bpk-targets.
- [ ] **Step 3:** Write `measurement-coverage.md` listing every cell as one of: `[OK]` (data complete), `[GAP — re-run needed]`, `[OK at smaller n, accept]`.
- [ ] **Step 4:** Verify: `grep -c "OK\|GAP" Thesis/text/defence/measurement-coverage.md` ≥ 18 (= 9 distributions × 2 L). No cells unaccounted for.
- [ ] **Step 5:** Commit (in submodule only, no parent bump): `git -C Thesis add text/defence/measurement-coverage.md && git -C Thesis commit -m "docs(defence): audit n=2^24 / n=2^20 measurement coverage"`

### Task 1.2 — Kick off B6 measurement (background)

**Files:**
- Modify: `bench/throughput_test.go` (if needed, to cover Grafite/SNARF/SuRF at n=2²⁴)
- Create: `bench_results/b6_latency.log`

- [ ] **Step 1:** Confirm existing test in `bench/performance_test.go` covers Grafite/SNARF/SuRF query latency (read the file). If it does — skip to Step 3.
- [ ] **Step 2:** If missing, add a `TestB6IndustryLatency` covering: build time per key (M keys/s) and query time (ns per probe) at n=2²⁴ on SOSD Books, for {Grafite, SNARF, SuRFReal, SODA, Truncation, Scan-ARE}.
- [ ] **Step 3:** Run in background with long timeout: `go test -v -run TestB6IndustryLatency -timeout 4h ./bench/ 2>&1 | tee bench_results/b6_latency.log &`
- [ ] **Step 4:** Verify the process is alive: `ps -ef | grep go test`. Record the PID.
- [ ] **Step 5:** Don't block — proceed to Task 1.3 while it runs.

### Task 1.3 — Write Introduction chapter

**Files:**
- Create: `Thesis/text/src/introduction.tex`
- Modify: `Thesis/text/practical-range-emptiness.tex` (replace `% TODO: expand introduction` at line 47 with `\input{src/introduction}`)

- [ ] **Step 1:** Draft an outline (15 min) directly into `introduction.tex` as `\section*` placeholders:
  - Motivation (LSM range queries, why range filters)
  - Prior art summary (one paragraph linking SuRF → Rosetta → SNARF → Grafite → Memento → Goswami)
  - Thesis contributions (4 numbered: one-vector + WPS→binsearch, truncation+adaptive, Scan-ARE, negative result CDF)
  - Roadmap (chapter-by-chapter, one line each)
- [ ] **Step 2:** Fill Motivation section (~1 page). Anchor: production LSM workloads from Cao FAST'20.
- [ ] **Step 3:** Fill Prior art (~1 page). Cite all 7 filter papers from `refs.bib`.
- [ ] **Step 4:** Fill Contributions list (1/2 page). For each contribution, lead with the receipt number.
- [ ] **Step 5:** Fill Roadmap (1/3 page). One line per chapter.
- [ ] **Step 6:** Compile: `cd Thesis/text && make thesis`. Expected: 0 errors, intro renders 3–4 pages.
- [ ] **Step 7:** Verify: `grep -c TODO Thesis/text/src/introduction.tex` returns 0.
- [ ] **Step 8:** Commit: `git -C Thesis add text/src/introduction.tex text/practical-range-emptiness.tex && git -C Thesis commit -m "feat(text): add introduction chapter"`

### Task 1.4 — Write Related Work chapter

**Files:**
- Create: `Thesis/text/src/related_work.tex`
- Modify: `Thesis/text/practical-range-emptiness.tex` (insert `\chapter{Related Work}\label{chap:related}\input{src/related_work}` after the introduction chapter)

- [ ] **Step 1:** Outline four sections: 1.1 Point filters (Bloom 1970), 1.2 Trie-based range filters (SuRF, Rosetta), 1.3 Learned range filters (SNARF, Memento), 1.4 Information-theoretic range filters (Goswami, Grafite, GRF).
- [ ] **Step 2:** For each cited paper, write 2–3 sentences: what they do, what they cost, where they fail. ≤ 1/2 page per paper.
- [ ] **Step 3:** Close with a positioning paragraph: "Our work sits at the practical end of the Goswami line — preserving the lower-bound match while paying for non-asymptotic constants in the metadata, hash, and segmentation layers."
- [ ] **Step 4:** Compile: `cd Thesis/text && make thesis`.
- [ ] **Step 5:** Verify: chapter is 2–3 pages; `grep -c TODO Thesis/text/src/related_work.tex` returns 0; every cited paper appears in `refs.bib`.
- [ ] **Step 6:** Commit: `git -C Thesis add text/src/related_work.tex text/practical-range-emptiness.tex && git -C Thesis commit -m "feat(text): add related work chapter"`

### Task 1.5 — Beamer skeleton + slides 1–4

**Files:**
- Create: `Thesis/text/defence/slides/defence.tex`
- Create: `Thesis/text/defence/slides/core/01-title.tex`, `02-lsm.tex`, `03-filters.tex`, `04-problem.tex`
- Modify: `Thesis/text/defence/slides/Makefile` (mirror parent Makefile pattern)

- [ ] **Step 1:** Create `defence.tex` with `\documentclass{beamer}`, theme (Madrid or similar minimal), `\input` directives for `core/0*.tex` files. Title: "Applicability of Non-Asymptotic Optimizations in Range Filters". Author: Andrei Ogurtsov. Programme + supervisor placeholder.
- [ ] **Step 2:** `01-title.tex` — title frame only.
- [ ] **Step 3:** `02-lsm.tex` — LSM-tree slide. Reuse the diagram from `proposal/`. Bullets per `talk-structure.md` row #2.
- [ ] **Step 4:** `03-filters.tex` — combined Filters + Range Filters slide. Two columns: bullets left, LSM-with-filter-blobs picture right. Bottom line: "Bloom = point queries. Range queries → range filter."
- [ ] **Step 5:** `04-problem.tex` — Problem + Goswami baseline (variant β, merged). Lower-bound formula prominent. Pipeline diagram (hash → ERE).
- [ ] **Step 6:** Compile: `cd Thesis/text/defence/slides && latexmk -pdf defence.tex`. Verify: PDF has exactly 4 frames matching the rows.
- [ ] **Step 7:** Commit: `git -C Thesis add text/defence/slides/ && git -C Thesis commit -m "feat(defence): beamer skeleton + slides 1-4"`

---

## Week 2: Tables and core layers (May 4 – May 10)

### Task 2.1 — Fill 9 BPK target tables in evaluation chapter

**Files:**
- Modify: `Thesis/text/src/evaluation.tex` lines ~411–602 (9 tables: tab:bpk-fb, tab:bpk-books, tab:bpk-wiki, tab:bpk-osm, tab:bpk-clustered, tab:bpk-uniform, tab:bpk-spread, tab:bpk-zipfian, tab:bpk-temporal)

- [ ] **Step 1:** From `bench_results/data/N16777216/sosd_*/` extract the 4 (filter × L × FPR-target) tuples per SOSD distribution. Use the existing CSVs.
- [ ] **Step 2:** From `bench_results/data/N1048576/{clustered,uniform,spread,zipfian,temporal}/` extract the same for synthetic at $n=2^{20}$.
- [ ] **Step 3:** Replace each `---` cell with the actual BPK number. Use 2 significant digits.
- [ ] **Step 4:** Update each caption to state "$n=2^{24}$" (SOSD) or "$n=2^{20}$" (synthetic). Add a footnote on the first synthetic table explaining the scale difference.
- [ ] **Step 5:** Compile: `cd Thesis/text && make thesis`. Expected: 9 tables fully populated, no `---` remnants.
- [ ] **Step 6:** Verify: `grep -c '^---' Thesis/text/src/evaluation.tex` returns 0.
- [ ] **Step 7:** Commit: `git -C Thesis add text/src/evaluation.tex && git -C Thesis commit -m "bench(eval): fill BPK target tables (SOSD n=2^24, synthetic n=2^20)"`

### Task 2.2 — Remove all `\colorbox{orange}{TODO}` markers

**Files:**
- Modify: `Thesis/text/src/evaluation.tex`, `Thesis/text/src/ere.tex` (and any others with markers)

- [ ] **Step 1:** Find every instance: `grep -n 'colorbox.*TODO' Thesis/text/src/*.tex`.
- [ ] **Step 2:** For each marker: either resolve the TODO (do the work it describes — usually one paragraph or one missing reference) or, if cosmetic, delete the colorbox while preserving the surrounding sentence.
- [ ] **Step 3:** Verify: `grep -rn 'colorbox.*TODO' Thesis/text/src/` returns nothing.
- [ ] **Step 4:** Compile, sanity-check the affected pages.
- [ ] **Step 5:** Commit: `git -C Thesis add text/src/ && git -C Thesis commit -m "chore(text): resolve all colorbox TODO markers"`

### Task 2.3 — Build core slides 5–7 (Layers 1, 2, 3)

**Files:**
- Create: `defence/slides/core/05-layer1.tex`, `06-layer2.tex`, `07-layer3.tex`

- [ ] **Step 1:** `05-layer1.tex` — table with two rows (Goswami baseline / Ours), two columns (Navigation / Bucket search). Receipts: −24% metadata; 6–110× faster than O(1) WPS. **This is the centrepiece — design it carefully.**
- [ ] **Step 2:** `06-layer2.tex` — phantom comparison figure (reuse `figures/phantom_comparison.pdf`) + density-threshold bullet. Receipts: −log₂L BPK; FPR=0 when ρ > ε/L.
- [ ] **Step 3:** `07-layer3.tex` — segmentation diagram (reuse `figures/segmentation.pdf`). Three-bullet structure: "1D-DBSCAN over sorted keys, O(n)" / "Dense → exact, FPR=0" / "Sparse → truncation fallback". Boxed formula `δ = c · L/ε`.
- [ ] **Step 4:** Compile, time the speech for slides 5–7 (target 1:00 + 0:45 + 1:45 = 3:30).
- [ ] **Step 5:** Commit: `git -C Thesis add text/defence/slides/core/ && git -C Thesis commit -m "feat(defence): core slides 5-7 (three layers)"`

### Task 2.4 — Build DLC modules M1, M2

**Files:**
- Create: `defence/slides/dlc/M1-onevector.tex`, `M2-adaptive.tex`

- [ ] **Step 1:** `M1-onevector.tex` — formulas `|D₁|+|D₂| = B + n + M + 1` vs `|D| = B + n + 1`. One-line derivation Poisson 1−e⁻¹. One-line empirical match.
- [ ] **Step 2:** `M2-adaptive.tex` — derivation of ρ > ε/L. Reuse Table~\ref{tab:exact-threshold} from `hash_truncation.tex`.
- [ ] **Step 3:** Each compiles standalone (`latexmk -pdf M1-onevector.tex`).
- [ ] **Step 4:** Commit: `git -C Thesis add text/defence/slides/dlc/M1-onevector.tex text/defence/slides/dlc/M2-adaptive.tex && git -C Thesis commit -m "feat(defence): DLC modules M1, M2"`

### Task 2.5 — Integrate B6 results into evaluation chapter (if measurements done)

**Files:**
- Modify: `Thesis/text/src/evaluation.tex` (add new section `\section{Build Throughput and Query Latency vs Industry}`)

- [ ] **Step 1:** Check `bench_results/b6_latency.log` for completion. If not done, defer to week 3 and skip this task.
- [ ] **Step 2:** Generate two tables from the log: build throughput (M keys/s), query latency (ns/probe) for {Grafite, SNARF, SuRFReal, SODA, Truncation, Scan-ARE} on SOSD Books $n=2^{24}$.
- [ ] **Step 3:** Write a 1-paragraph reading of each table.
- [ ] **Step 4:** Compile, verify both tables render.
- [ ] **Step 5:** Commit: `git -C Thesis add text/src/evaluation.tex bench_results/b6_latency.log && git -C Thesis commit -m "bench(eval): add build/query latency comparison vs SOTA"`

---

## Week 3: Headline + conclusion + backups (May 11 – May 17)

### Task 3.1 — Choose headline plot, render at $n=2^{24}$

**Files:**
- Modify: `Thesis/text/figures/` — new `headline.pdf`
- Modify: `Thesis/text/Makefile` SVG_MAP if needed

- [ ] **Step 1:** Visually compare `bench_results/plots/N16777216/sosd_books/L65536.svg` and `.../sosd_fb/L65536.svg`. Decide: Books (cleaner separation, more dramatic) or FB (larger universe, more "industrial" feel).
- [ ] **Step 2:** Add the chosen SVG to the Makefile `SVG_MAP` as `headline`.
- [ ] **Step 3:** Run `cd Thesis/text && make figures` to regenerate `figures/headline.pdf`.
- [ ] **Step 4:** Verify: `figures/headline.pdf` exists, opens, looks like the SVG.
- [ ] **Step 5:** Commit: `git -C Thesis add text/Makefile && git -C Thesis commit -m "build(text): wire headline plot to figures pipeline"`

### Task 3.2 — Build core slides 8, 9, 10

**Files:**
- Create: `defence/slides/core/08-headline.tex`, `09-limitations.tex`, `10-conclusion.tex`

- [ ] **Step 1:** `08-headline.tex` — full-width `figures/headline.pdf`. One-line caption with the receipt: "Scan-ARE: FPR<10⁻⁸ at ~5 BPK; SOTA: FPR≈10⁻² or 10–30 BPK". Speaker note: 1:15 spoken.
- [ ] **Step 2:** `09-limitations.tex` — three bullets: uniform data (exact mode never triggers); δ / minClusterSize sensitivity; build cost grows with cluster count.
- [ ] **Step 3:** `10-conclusion.tex` — three numbers in a row: **6–110×** (WPS→binsearch), **FPR=0** (adaptive exact mode), **<10⁻⁸ @ 5 BPK** (Scan-ARE on cluster). One line of future work.
- [ ] **Step 4:** Compile full deck. Verify: 10 frames, total speaker time ≤ 11:00 walked through.
- [ ] **Step 5:** Commit: `git -C Thesis add text/defence/slides/core/ && git -C Thesis commit -m "feat(defence): core slides 8-10 (headline, limitations, conclusion)"`

### Task 3.3 — Build DLC modules M3, M4

**Files:**
- Create: `defence/slides/dlc/M3-lsm-context.tex`, `M4-industry-comparison.tex`

- [ ] **Step 1:** `M3-lsm-context.tex` — per-SSTable n ∈ [2¹⁹, 2²³] from Cao FAST'20; RocksDB BPK budget = 10. Anchors the parameter choices in the benchmarks.
- [ ] **Step 2:** `M4-industry-comparison.tex` — reuse the locate/memory/build complexity table from proposal slide 5. Now placed before headline.
- [ ] **Step 3:** Standalone compile.
- [ ] **Step 4:** Commit: `git -C Thesis add text/defence/slides/dlc/ && git -C Thesis commit -m "feat(defence): DLC modules M3, M4"`

### Task 3.4 — Build Q&A backup slides B1–B10

**Files:**
- Create: 10 files under `defence/slides/backup/B1-phantom.tex` … `B10-codebase.tex`

- [ ] **Step 1:** B1, B7 — reuse figures from `Thesis/text/figures/`. One frame each.
- [ ] **Step 2:** B2, B3, B4, B5 — reuse tables from `src/ere.tex`, `src/hybrid.tex`, `src/evaluation.tex`. One frame each.
- [ ] **Step 3:** B6 — wait for B6 measurement; populate the same tables created in Task 2.5.
- [ ] **Step 4:** B8, B9, B10 — write fresh single-frame slides. B8: "CDF-ARE — what we tried, why it didn't work" (3 bullets). B9: "Future work — dynamic updates, RocksDB integration, n=2²⁷". B10: "Codebase — 6 ARE variants, CGo wrappers, 200+ tests" with a screenshot of `cloc`.
- [ ] **Step 5:** Compile full deck (core + DLC + backup); verify all 24 slides render.
- [ ] **Step 6:** Commit: `git -C Thesis add text/defence/slides/backup/ && git -C Thesis commit -m "feat(defence): Q&A backup slides B1-B10"`

### Task 3.5 — Write Conclusion chapter

**Files:**
- Create: `Thesis/text/src/conclusion.tex`
- Modify: `Thesis/text/practical-range-emptiness.tex` line 147 (replace `% TODO: conclusion and future work` with `\input{src/conclusion}`)

- [ ] **Step 1:** Outline four sections: Summary of contributions (mirror introduction's contribution list, with measured numbers); What we learned (1–2 paragraph reflective); Limitations (forward-pointer to §evaluation); Future work (3–4 directions).
- [ ] **Step 2:** Fill each section. Total 1–2 pages.
- [ ] **Step 3:** Compile, verify ≥ 1 page, ≤ 2 pages.
- [ ] **Step 4:** Verify: `grep -c TODO Thesis/text/src/conclusion.tex` returns 0.
- [ ] **Step 5:** Commit: `git -C Thesis add text/src/conclusion.tex text/practical-range-emptiness.tex && git -C Thesis commit -m "feat(text): add conclusion chapter"`

### Task 3.6 — Write abstract and limitations section

**Files:**
- Modify: `Thesis/text/practical-range-emptiness.tex` lines 35–40 (abstract)
- Modify: `Thesis/text/src/evaluation.tex` (add `\section{Limitations}` near end of chapter)

- [ ] **Step 1:** Replace abstract `TODO:` with one paragraph (~150 words): problem, contributions in 3 lines, headline result with a number.
- [ ] **Step 2:** Limitations section: explicit list of where Scan-ARE fails (uniform, gap-only-at-tail patterns), δ choice sensitivity, current build-time cost vs Grafite.
- [ ] **Step 3:** Compile, verify abstract and limitations render.
- [ ] **Step 4:** Commit: `git -C Thesis add text/practical-range-emptiness.tex text/src/evaluation.tex && git -C Thesis commit -m "feat(text): finalize abstract and limitations"`

---

## Week 4: Text close-out (May 18 – May 20)

The text submission deadline is **2026-05-20**. Tasks 4.1–4.3 must complete by that date. Slide work is decoupled and continues in **Phase B** below — it has its own pacing tied to the defence date.

If you finish text close-out early on May 18 or 19, jump to Phase B; do not let extra time leak into perfectionist text edits.

### Task 4.1 — Final text proofread

**Files:** all of `Thesis/text/src/*.tex` and main `practical-range-emptiness.tex`.

- [ ] **Step 1:** Mechanical scans:
  - `grep -rn 'TODO\|FIXME\|XXX' Thesis/text/src/ Thesis/text/practical-range-emptiness.tex` — must be empty.
  - `grep -rn 'colorbox.*TODO' Thesis/text/` — must be empty.
  - `grep -rn '\\---' Thesis/text/src/` (placeholder dashes in tables) — must be empty.
- [ ] **Step 2:** Read each chapter end-to-end at reading speed; flag awkward sentences. Fix in batch.
- [ ] **Step 3:** Compile final PDF: `cd Thesis/text && make thesis`. Inspect log for warnings: `grep -i 'warning\|undefined' out/practical-range-emptiness.log`. Fix any unresolved references.
- [ ] **Step 4:** Verify: PDF page count is in the program's range; all figures render; bibliography prints.
- [ ] **Step 5:** Commit: `git -C Thesis add . && git -C Thesis commit -m "chore(text): final proofread pass"`

### Task 4.2 — Bump submodule pointer in parent repo

**Files:**
- Modify: parent repo's submodule pointer (single commit, before submission — per standing rule).

- [ ] **Step 1:** From parent repo: `cd /Users/andrei.ogurtsov/Thesis-Bench-industry && git status` — should show `Thesis` modified.
- [ ] **Step 2:** Commit: `git add Thesis && git commit -m "chore: bump Thesis submodule (final text submission)"`
- [ ] **Step 3:** Don't push automatically. User decides when.

### Task 4.3 — Submit text (2026-05-20, hard deadline)

- [ ] **Step 1:** Follow program submission procedure (portal upload, signed forms, etc.). Out of scope for this plan; user knows the procedure.
- [ ] **Step 2:** Save submission confirmation to `defence/submission-receipt.{pdf,eml}`.

---

## Phase B: Defence prep (post-submission, May 20 → defence date)

Slide work continues without text-deadline pressure. Pacing depends on the defence date.

### Task B.1 — Pre-defence dry run (timed) + fix slow spots

**Files:** none — record a stopwatch sheet inline as a markdown comment in `defence/slides/defence.tex`.

- [ ] **Step 1:** Compile final deck (core only, no DLC). Walk through out loud with stopwatch. Record per-slide actual time.
- [ ] **Step 2:** Identify any slide >150% of budget. Cut or rephrase.
- [ ] **Step 3:** Re-time. Target: ≤ 11:00.
- [ ] **Step 4:** Don't commit timing yet — wait until after Task B.2.

### Task B.2 — Pre-defence with supervisor / lab

- [ ] **Step 1:** Schedule the meeting. Aim for ~5–7 days before defence.
- [ ] **Step 2:** Walk through full core (10 slides). Then offer the 4 DLC modules with the question: "Which would you like in the talk?"
- [ ] **Step 3:** Record the chosen DLC list + any direct feedback in `defence/pre-defence-notes.md`.
- [ ] **Step 4:** Commit notes: `git -C Thesis add text/defence/pre-defence-notes.md && git -C Thesis commit -m "docs(defence): pre-defence feedback and DLC selection"`

### Task B.3 — Insert chosen DLC modules and re-time

**Files:**
- Modify: `defence/slides/defence.tex` to `\input` chosen DLC files at the right points.

- [ ] **Step 1:** Wire selected DLC into `defence.tex` according to the M1–M4 insertion points in `talk-structure.md`.
- [ ] **Step 2:** Compile, verify total slide count increased by chosen-DLC count.
- [ ] **Step 3:** Second timed dry run. Target: ≤ 12:30.
- [ ] **Step 4:** Adjust if over.
- [ ] **Step 5:** Commit: `git -C Thesis add text/defence/slides/defence.tex && git -C Thesis commit -m "feat(defence): integrate chosen DLC modules"`

### Task B.4 — Day-before defence prep

- [ ] **Step 1:** Memorize the 4 critical numbers: **−24%** metadata, **6–110×** WPS, **FPR=0** exact mode, **<10⁻⁸ @ 5 BPK**.
- [ ] **Step 2:** Render 3 copies of the Beamer PDF: local laptop, USB stick, cloud (Google Drive).
- [ ] **Step 3:** Render fallback PNG-per-slide in case projector / Beamer breaks: `pdftoppm -r 150 defence.pdf slide -png`.
- [ ] **Step 4:** Sleep.

---

## Risks and contingencies

| Risk | Probability | Impact | Contingency |
|------|-------------|--------|-------------|
| n=2²⁴ data missing for some SOSD cell | Low | Empty cell in table | Run targeted single-cell benchmark in week 1 (1–2 hours) |
| B6 (build/query latency) takes >3 days | Medium | B6 backup unavailable | Skip Task 2.5 / B6 backup; if asked at defence: "ongoing measurement, in next iteration of text" |
| Pre-defence cannot be scheduled | Low | Self-pick DLC | Default selection: **M1 (one-vector detail) + M3 (LSM context)** — covers technical depth + applied framing |
| LaTeX build breaks at submission | Low | Late submission | Render on Overleaf as fallback; fallback PDF rendered earlier in week 4 |
| Beamer figure rendering fails on projector | Low | Slides incomplete | PNG fallback rendered in Task 4.7 |
| Chapter writing slips by >2 days | Medium | Compresses week 3, threatens May 20 deadline | Cut B10 + B9 (cosmetic backups) immediately; if still slipping, cut Task 3.4 entirely (move all backup slides to Phase B); text close-out (Tasks 4.1–4.3) is non-negotiable |
| May 20 deadline missed | Low | Late submission penalty per program rules | Identify gating chapter on May 17 morning; if conclusion or abstract not started, escalate to supervisor by EOD May 17 — do not silently slip |
| Synthetic-at-n=2²⁴ requested by supervisor | Low | Major scope expansion | Push back: "SOSD at industrial scale is the headline; synthetic at smaller scale isolates effects independently — both are reported." |

---

## Daily timeboxing (suggested)

- **6 days/week**, rest 1 (recommend Sunday).
- **4 h/day** focused: writing or measurements.
- **1 h/day** Beamer.
- **1 h/day** miscellaneous: compile, commit, planning hygiene.
- **Saturdays**: dry runs + review pass over the week's commits.

If you fall a day behind on a writing task, immediately cut a backup slide (B8 / B9 / B10), not a core deliverable. Backup slides are the contingency budget.

---

## Self-review (done)

- [x] Spec coverage: every section in `talk-structure.md` (10 core slides, 4 DLC, 10 backup) is mapped to a task in this plan.
- [x] Placeholder scan: no "TBD"/"TODO"/"figure out later" left in this plan.
- [x] Type consistency: file paths and slide IDs match `talk-structure.md` exactly.
- [x] Critical path is contiguous and identified.
- [x] Risks have explicit contingencies, not "hope it works".
