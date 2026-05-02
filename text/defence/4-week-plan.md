# Thesis Lock-In Plan (2026-04-27 → 2026-05-20 text submission; defence shortly after)

> Source spec: `Thesis/text/defence/talk-structure.md`
> Adapted from `superpowers:writing-plans` for non-code deliverables.

**Goal:** submit a finished thesis text on 2026-05-20 and present an 11–12 minute defence with modular DLC support, on the structure agreed in `talk-structure.md`.

**SCHEDULE UPDATE (2026-05-02):** pre-defence locked for **Friday 2026-05-08**; preliminary deck must be in supervisor's hands by **Monday 2026-05-04**. Slide work moves *before* text close-out — see Sprint S below. Original Week-1/2/3 slide tasks (1.5, 2.3, 2.4, 3.2, 3.3, 3.4) are absorbed into Sprint S; their checklists remain authoritative for slide content.

---

## Sprint S: Pre-defence (2026-05-02 Sat → 2026-05-08 Fri)

**Hard milestones:**
- **Mon 2026-05-04 EOD** — preliminary deck delivered to supervisor (10 core slides, rough).
- **Fri 2026-05-08** — pre-defence (full deck: 10 core + DLC selection + backups).

**Strategy:** content over polish. Reuse existing plots from `bench_results/plots/N16777216/` and figures from `text/figures/` aggressively — don't redraw. Speaker notes are optional for Mon, mandatory for Fri.

### Task S.1 — Beamer skeleton + 10 core slides (rough cut for Monday)

Compresses Tasks 1.5 + 2.3 + 3.2 into one weekend pass. Use bullet placeholders if a slide needs more thought; supervisor feedback will surface what's worth polishing.

**Files:**
- Create: `Thesis/text/defence/slides/defence.tex` + `Makefile`
- Create: `Thesis/text/defence/slides/core/{01-title,02-lsm,03-filters,04-problem,05-layer1,06-layer2,07-layer3,08-headline,09-limitations,10-conclusion}.tex`

- [ ] **Step 1 (Sat 05-02):** `defence.tex` skeleton with theme + `\input` directives. Title frame `01-title.tex`. Pick headline plot now (Books vs FB at L=65536 from `bench_results/plots/N16777216/`) — locks Task 3.1 visual.
- [ ] **Step 2 (Sat 05-02):** Slides 02–04 (LSM, filters, problem). Reuse proposal diagrams.
- [ ] **Step 3 (Sun 05-03):** Slides 05–07 (three layers). Layer-1 table is the centrepiece — 30+ min on it alone.
- [ ] **Step 4 (Sun 05-03):** Slides 08–10 (headline, limitations, conclusion).
- [ ] **Step 5 (Sun 05-03 EOD):** Compile (`latexmk -pdf defence.tex`), full walkthrough out loud with stopwatch. Target ≤ 12:00 rough.
- [ ] **Step 6 (Mon 05-04):** Send PDF to supervisor before EOD. Don't commit yet — wait for feedback.

### Task S.2 — DLC modules M1–M4 (Tue–Wed 05-05/06)

Absorbs Tasks 2.4 + 3.3.

- [ ] **Step 1:** `dlc/M1-onevector.tex` — Poisson 1−e⁻¹ + empirical match.
- [ ] **Step 2:** `dlc/M2-adaptive.tex` — ρ > ε/L derivation, exact-threshold table.
- [ ] **Step 3:** `dlc/M3-lsm-context.tex` — Cao FAST'20 anchors, RocksDB BPK=10.
- [ ] **Step 4:** `dlc/M4-industry-comparison.tex` — locate/memory/build complexity table from proposal.
- [ ] **Step 5:** Each compiles standalone. Commit batch.

### Task S.3 — Q&A backup slides B1–B10 (Wed–Thu 05-06/07)

Absorbs Task 3.4. **If sprint slips, cut B8/B9/B10 first** — they're cosmetic.

- [ ] **Step 1:** B1, B7 — figure reuse. B2–B5 — table reuse.
- [ ] **Step 2:** B6 — populate from `bench_results/b6_latency*.log` (data already exists).
- [ ] **Step 3:** B8/B9/B10 — fresh single-frame slides (CDF-ARE negative, future work, codebase stats).
- [ ] **Step 4:** Compile full deck (24 slides). Commit batch.

### Task S.4 — Apply Monday feedback + dry runs (Tue–Thu 05-05/07)

- [ ] **Step 1 (Tue 05-05):** Translate supervisor's notes into a per-slide diff list. Apply same day.
- [ ] **Step 2 (Wed 05-06):** Solo timed dry run with stopwatch. Mark slides over budget.
- [ ] **Step 3 (Thu 05-07):** Second dry run. Memorize 4 critical numbers (−24%, 6–110×, FPR=0, <10⁻⁸ @ 5 BPK).
- [ ] **Step 4 (Thu 05-07 EOD):** Render fallback PNGs (`pdftoppm`); copy deck to laptop + USB + cloud.

### Task S.5 — Pre-defence (Fri 2026-05-08)

- [ ] **Step 1:** Walk through 10 core slides.
- [ ] **Step 2:** Offer 4 DLC modules: "Which would you like in the final talk?" Record selection.
- [ ] **Step 3:** Capture any questions that hit hard — those become future Q&A backups or talk edits.
- [ ] **Step 4:** Save notes to `defence/pre-defence-notes.md`. Commit.

---

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

### Task 0.1 — X-axis cap at 25 BPK (no rerun, render only) ✅ DONE

**Files:**
- Modify: `Thesis/testutils/plot.go:104–115` — add `XMax` field to `PlotConfig`; in linear scale mode clamp `axMaxX = min(axMaxX, XMax)`.
- Modify: callers in `bench/comparison_test.go`, `bench/sosd_test.go` — pass `XMax: 25`.

- [x] **Step 1:** Read `Thesis/testutils/plot.go` lines 100–130 to confirm the auto-scaling block.
- [x] **Step 2:** Add `XMax float64` to `PlotConfig`. In the linear branch: `if cfg.XMax > 0 && axMaxX > cfg.XMax { axMaxX = cfg.XMax }`.
- [x] **Step 3:** In all `GenerateTradeoffSVG` call sites, set `XMax: 25` (or `30`; pick one and stick).
- [x] **Step 4:** Re-render existing plots in plot-only mode: `PLOT_ONLY=1 go test -run TestComparison -v ./bench/ -timeout 5m`. Verify visual change in one SVG.
- [x] **Step 5:** Commit `5dead55 feat(plot): add XMax field to PlotConfig for hard X-axis cap` (in Thesis submodule).

### Task 0.2 — Audit `YFloor` consistency across plot call sites (no rerun) ✅ DONE

**Mechanism is already in place:** `plot.go:39 PlotConfig.YFloor` is a configurable field; lines 119–121 use it when set, fallback `1e-8` otherwise. The "3e-07" we see in the L65536 SVG is a caller passing `1.0/queryCount ≈ 3.3e-7`. This is correct behaviour. The risk is **inconsistency** across callers — some may pass 0 (silent fallback to `1e-8`) or a magic constant.

- [x] **Step 1:** `grep -rn "GenerateTradeoffSVG\|GeneratePerformanceSVG" bench/ Thesis/` — find every call site.
- [x] **Step 2:** For each, verify the `YFloor` field is set to `1.0 / float64(queryCount)` (or equivalent). Fix any that pass 0 or a constant.
- [x] **Step 3:** Re-render with `PLOT_ONLY=1` (only if any call site changed). Sanity-check one SVG.
- [x] **Step 4:** Commit `03217f4 fix(plot): set YFloor consistently across SODA and Truncation tradeoff plots`.

### Task 0.3 — Rename stale `bench_results/` and start with clean cache ✅ DONE (variant)

- [x] **Step 1–4:** Done as `97cb3fc chore(gitignore): blanket bench_results/` + `e358988 chore(repo): drop legacy Python tooling and stale benchmark outputs` — old results were retired/dropped rather than renamed; current `bench_results/` was rebuilt clean.

### Task 0.4 — Adaptive point density for CGo filter sweeps

**Why dense pre-sweep is wrong:** `[4,5,6,…,28]` would burn compute on uninteresting regions of the curve (high-BPK tail where FPR is already at floor). Better: keep current sparse sweep `[4,6,8,…,20]`, then **after** initial points are measured, refine adaptively only where the descent is steep or the tail hasn't reached the floor.

**Concrete failure mode:** in `bench_results/plots/N16777216/sosd_fb/L1.svg`, Grafite drops from FPR≈10⁻¹ at BPK=10 to floor at BPK≈11.5 in **one sweep step**. The descent shape (BPK=10.5, 11) is invisible because we never measure there.

**Files:**
- Modify: `bench/comparison_test.go` — extend the CGo bpkSweep loop (~lines 414–510) with a single refinement pass per series.

**Algorithm:**
After the initial sweep populates `allSeries[name].Points` for each CGo series:

1. Sort points by BPK ascending.
2. For each consecutive pair `(p_i, p_{i+1})`:
   - If `BPK_{i+1} − BPK_i ≥ 2` **and** `|log₁₀(FPR_i) − log₁₀(FPR_{i+1})| ≥ 1.5` (steep drop), insert midpoint `BPK = (BPK_i + BPK_{i+1}) / 2`.
3. Tail extension: while last point's FPR > floor (≈ `3/queryCount`) **and** last point's BPK < `XMax` (= 25), append `BPK_last + 2`.
4. Run the filter at the resulting `extraBPK` set; append points; re-render.

One refinement pass is enough for thesis quality. Avoid recursion — diminishing returns vs added complexity.

- [x] **Step 1:** Read `bench/comparison_test.go:414–510` to map the current CGo loop structure.
- [x] **Step 2:** Implement the refinement function (separate helper `refineCGoSweep(series *SeriesData, queryCount int, xMax float64) []float64`) — returns extra BPK values.
- [x] **Step 3:** Wire it into the loop after initial sweep completes; re-run filter for `extraBPK` values; append to `Points`.
- [x] **Step 4:** Cache implications: `paramsBPKSweep` hash includes the refinement seed.
- [x] **Step 5:** Commit `f2f18a5 docs(defence): replace dense bpkSweep with adaptive refinement; phased rollout`. ✅ DONE

Effort: ~2–3 hours.

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

- [x] **Step 1:** Read lines 52–67 and downstream conditionals.
- [x] **Step 2:** Remove the 6 dropped series. ✅
- [x] **Step 3:** Re-render with `PLOT_ONLY=1`. Verify legend has 8 entries.
- [x] **Step 4:** Done — chapter-local plots are in place; defence-deck series set is locked. ✅ DONE

### Task 0.6 — Per-filter bit width: lift global 60-bit cap

**Why:** the global `mask60Keys` (`framework_test.go:21–44`) is applied at `comparison_test.go:594, 622, 876` because **SNARF** overflows when keys are close to UINT64_MAX. Cost: SOSD Wiki at $n=2^{27}$ collapses to ~50% unique keys after the mask (already documented in `distributions.tex` "Note on duplicate keys"). Lifting the cap unlocks honest $n=2^{28}$ for SOSD.

**Strategy:** keep `mask60` helper, but apply it **only inside the SNARF wrapper**, not globally. Other CGo wrappers (Grafite, SuRF) and our own filters (SODA, Truncation, Adaptive, Scan-ARE, Greedy+Merge) get full 64-bit keys.

**Files:**
- Modify: `bench/comparison_test.go:594, 622, 876` — remove global `mask60Keys` calls; pass raw 64-bit keys downstream.
- Modify: `snarf/snarf_cgo.go` (or wherever SNARF's `Build` / `Insert` lives) — mask incoming keys inside the wrapper.
- Modify: SNARF's `Query` / `RangeContains` — mask `[lo, hi]` endpoints similarly.
- Verify: Grafite, SuRF wrappers, all six native ARE filters compile and pass existing unit tests at full 64-bit.

- [x] **Steps 1–6:** Done — no `mask60` calls remain in `bench/`; SNARF wrapper masks internally. Confirmed by `20a2aff fix(bench): handle span > 2^63 in smart-query generator (sosd_fb at n=2^28)` running successfully at full 64-bit. ✅ DONE

This task is the only one in Week 0 that can fail and force scope rollback. Schedule it on a day when you have ~3 hours uninterrupted, ideally before Task 0.7.

### Task 0.7 — Phase 1: single distribution end-to-end validation

Don't launch a multi-day mass rerun until everything works on one ground-truth cell. This is the framework smoke test.

**Target:** SOSD FB at $n=2^{20}$, all L values `{1, 16, 128, 1024, 4096, 16384, 65536}`, 8 final filters.

Why FB: it's the headline distribution; visible Grafite descent and Scan-ARE dominance both happen here. If anything is broken (filter rendering, X-cap, adaptive refinement, 64-bit refactor), it shows up.

- [x] **Steps 1–4:** Phase 1 validated; data exists at `bench_results/data/N1048576/sosd_fb/`. ✅ DONE

### Task 0.8 — Phase 2: all distributions at $n=2^{20}$

After Phase 1 passes. End-to-end at small scale across the full distribution matrix; catches distribution-specific issues before scaling N.

**Target:** all 9 distributions (4 SOSD + 5 synthetic) at $n=2^{20}$, all L values, 8 filters.

- [x] **Steps 1–4:** Phase 2 complete; all distributions at n=2²⁰ in `bench_results/data/N1048576/`. ✅ DONE

### Task 0.9 — Phase 3: SOSD scale-up to $n=2^{24}$ (final headline data)

After Phase 2 passes. This produces the headline numbers for the thesis tables.

**Target:** 4 SOSD distributions at $n=2^{24}$, all L values, 8 filters. Synthetic stays at $n=2^{20}$ per scale decision in plan header.

If Task 0.6 (per-filter bit width) succeeded, optionally also run SOSD Books and FB at $n=2^{28}$ — Wiki and OSM keep $n=2^{24}$ due to intrinsic dedup.

- [x] **Steps 1–4:** Phase 3 complete — SOSD scaled all the way to n=2²⁸ (`bench_results/data/N16777216/`, `N268435456/`, plus `b6_N268435456_gap_heavy/`). ✅ DONE — exceeded plan target.

---

## Week 1: Foundation (May 1 – May 3 — compressed; rerun runs in background)

### Task 1.1 — Audit measurement coverage  *(deferred — fold into Task 2.1)*

**Files:**
- Create: `Thesis/text/defence/measurement-coverage.md`

The standalone audit doc is no longer worth its overhead — coverage will be exercised directly when filling Task 2.1's BPK tables. Skip and absorb.

- [ ] **Step 1:** ~~Iterate over `bench_results/data/...`~~ → folded into Task 2.1.
- [ ] **Step 2:** ~~Cross-reference target matrix~~ → folded into Task 2.1.
- [ ] **Step 3:** ~~Write `measurement-coverage.md`~~ — skipped.
- [ ] **Step 4:** N/A
- [ ] **Step 5:** N/A

### Task 1.2 — Kick off B6 measurement (background) ✅ DONE

**Files:**
- Modify: `bench/throughput_test.go` (if needed, to cover Grafite/SNARF/SuRF at n=2²⁴)
- Create: `bench_results/b6_latency.log`

- [x] **Step 1:** Confirmed via `bench/throughput_test.go` and follow-on B6 framework.
- [x] **Step 2:** `TestB6IndustryLatency` added (`a9024ce bench(b6): add TestB6IndustryLatency at n=2^24 SOSD Books`).
- [x] **Step 3:** Multi-distribution + multi-N runs done; logs in `bench_results/b6_latency.log`, `b6_latency_multidist.log`, `b6_latency_fb.log`, `b6_latency_rest.log`.
- [x] **Step 4:** Process completed; results processed.
- [x] **Step 5:** B6 final report at `Thesis/text/defence/b6-final-report.md`. ✅ DONE — N coverage extended to 2²⁸.

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

### Task 1.5 — Beamer skeleton + slides 1–4  *(→ absorbed into Sprint S.1)*

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

### Task 2.3 — Build core slides 5–7 (Layers 1, 2, 3)  *(→ absorbed into Sprint S.1)*

**Files:**
- Create: `defence/slides/core/05-layer1.tex`, `06-layer2.tex`, `07-layer3.tex`

- [ ] **Step 1:** `05-layer1.tex` — table with two rows (Goswami baseline / Ours), two columns (Navigation / Bucket search). Receipts: −24% metadata; 6–110× faster than O(1) WPS. **This is the centrepiece — design it carefully.**
- [ ] **Step 2:** `06-layer2.tex` — phantom comparison figure (reuse `figures/phantom_comparison.pdf`) + density-threshold bullet. Receipts: −log₂L BPK; FPR=0 when ρ > ε/L.
- [ ] **Step 3:** `07-layer3.tex` — segmentation diagram (reuse `figures/segmentation.pdf`). Three-bullet structure: "1D-DBSCAN over sorted keys, O(n)" / "Dense → exact, FPR=0" / "Sparse → truncation fallback". Boxed formula `δ = c · L/ε`.
- [ ] **Step 4:** Compile, time the speech for slides 5–7 (target 1:00 + 0:45 + 1:45 = 3:30).
- [ ] **Step 5:** Commit: `git -C Thesis add text/defence/slides/core/ && git -C Thesis commit -m "feat(defence): core slides 5-7 (three layers)"`

### Task 2.4 — Build DLC modules M1, M2  *(→ absorbed into Sprint S.2)*

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

### Task 3.2 — Build core slides 8, 9, 10  *(→ absorbed into Sprint S.1)*

**Files:**
- Create: `defence/slides/core/08-headline.tex`, `09-limitations.tex`, `10-conclusion.tex`

- [ ] **Step 1:** `08-headline.tex` — full-width `figures/headline.pdf`. One-line caption with the receipt: "Scan-ARE: FPR<10⁻⁸ at ~5 BPK; SOTA: FPR≈10⁻² or 10–30 BPK". Speaker note: 1:15 spoken.
- [ ] **Step 2:** `09-limitations.tex` — three bullets: uniform data (exact mode never triggers); δ / minClusterSize sensitivity; build cost grows with cluster count.
- [ ] **Step 3:** `10-conclusion.tex` — three numbers in a row: **6–110×** (WPS→binsearch), **FPR=0** (adaptive exact mode), **<10⁻⁸ @ 5 BPK** (Scan-ARE on cluster). One line of future work.
- [ ] **Step 4:** Compile full deck. Verify: 10 frames, total speaker time ≤ 11:00 walked through.
- [ ] **Step 5:** Commit: `git -C Thesis add text/defence/slides/core/ && git -C Thesis commit -m "feat(defence): core slides 8-10 (headline, limitations, conclusion)"`

### Task 3.3 — Build DLC modules M3, M4  *(→ absorbed into Sprint S.2)*

**Files:**
- Create: `defence/slides/dlc/M3-lsm-context.tex`, `M4-industry-comparison.tex`

- [ ] **Step 1:** `M3-lsm-context.tex` — per-SSTable n ∈ [2¹⁹, 2²³] from Cao FAST'20; RocksDB BPK budget = 10. Anchors the parameter choices in the benchmarks.
- [ ] **Step 2:** `M4-industry-comparison.tex` — reuse the locate/memory/build complexity table from proposal slide 5. Now placed before headline.
- [ ] **Step 3:** Standalone compile.
- [ ] **Step 4:** Commit: `git -C Thesis add text/defence/slides/dlc/ && git -C Thesis commit -m "feat(defence): DLC modules M3, M4"`

### Task 3.4 — Build Q&A backup slides B1–B10  *(→ absorbed into Sprint S.3)*

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

> **Note (2026-05-02 update):** the original Phase B *pre-defence* (Task B.2) is now Sprint S.5 and runs **before** text submission. Phase B is now strictly post-submission polish + final dry run.

Slide work continues without text-deadline pressure. Pacing depends on the defence date.

### Task B.1 — Pre-defence dry run (timed) + fix slow spots

**Files:** none — record a stopwatch sheet inline as a markdown comment in `defence/slides/defence.tex`.

- [ ] **Step 1:** Compile final deck (core only, no DLC). Walk through out loud with stopwatch. Record per-slide actual time.
- [ ] **Step 2:** Identify any slide >150% of budget. Cut or rephrase.
- [ ] **Step 3:** Re-time. Target: ≤ 11:00.
- [ ] **Step 4:** Don't commit timing yet — wait until after Task B.2.

### Task B.2 — Pre-defence with supervisor / lab  *(→ now Sprint S.5, scheduled 2026-05-08)*

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

## Phase C: Post-defence release cleanup (deferred)

Issues spotted during defence prep that don't block submission or the
talk, but are real performance / API debt to clear before any public
release of the codebase. Recorded here so we don't lose them.

### Task C.1 — Lift ARE/ERE public API to `uint64` keys  ✅ DONE (ahead of schedule)

Completed during Week 0 work:
- `b7c19c9 feat(ere): drop BitString API, uint64-only`
- `b22ae24 feat(ere_one_d): drop BitString API, uint64-only`
- `21cf015 feat(are_trunc): drop BitString API, uint64-only with Config{Eps}`
- `9a2a575 feat(are_adaptive): drop BitString API`
- `cdb3550 feat(are_greedy_scan): drop BitString API`
- `dbb270b feat(are_hybrid_scan): drop BitString API`
- `e1bd71e refactor(bench): adopt uint64 ARE/ERE API; drop TrieBS trampolines`

**Problem.** Every public `IsEmpty(a, b bits.BitString) bool` allocates
two `bits.BitString` values per query (and the build path allocates one
per key). On the FPR-vs-BPK benchmark this is ~6e6 allocations per
configuration just from the trampoline `func(a, b uint64) bool { return
f.IsEmpty(testutils.TrieBS(a), testutils.TrieBS(b)) }`. Heap pressure
fragments the GC pacer and competes with the actual filter work.

**Reality check.** Every benchmark we run — synthetic and SOSD —
produces keys that fit in `uint64`. The `BitString`-typed surface is
only useful inside the trie internals (variable-width hashing,
prefix/suffix splits). It does not need to be the API contract.

**Plan.**
- Add `IsEmptyU64(lo, hi uint64) bool` and `BuildU64([]uint64, ...)`
  fast paths to every ARE/ERE package whose underlying logic accepts
  uint64-shaped keys (trunc, adaptive, soda_hash, hybrid_scan,
  greedy_scan, bloom — almost all of them).
- Convert internally where width-conversion is unavoidable; do it once
  per call instead of inside the user-facing closure.
- Update `bench/comparison_test.go` and the Thesis test suites to call
  the uint64 path first; keep `BitString` versions as a thin wrapper.
- Re-run the throughput / latency benchmarks (B6 backup) and quantify
  the saved time — this is presentation-grade evidence for the
  conclusion slide's "future work" line.

**Why not now.** Touches every ARE filter package and a lot of test
fixtures. Risk of breaking the cached JSON used by the headline plot
(any signature change forces a re-measurement). Schedule **after** the
text submission and after the headline data lock so we can refactor
freely without invalidating the defence run.

### Task C.2 — Audit other allocation hot spots flagged during the
performance review (TBD list — fill in as we spot them).

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
