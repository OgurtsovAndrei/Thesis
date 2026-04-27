# 4-Week Thesis Lock-In Plan (2026-04-27 → 2026-05-25)

> Source spec: `Thesis/text/defence/talk-structure.md`
> Adapted from `superpowers:writing-plans` for non-code deliverables.

**Goal:** submit a finished thesis text on 2026-05-25 and present an 11–12 minute defence with modular DLC support, on the structure agreed in `talk-structure.md`.

**Architecture:** three parallel streams that converge in week 4.
- **Text stream** — finish missing chapters (intro, related work, conclusion, abstract, limitations); fill 9 empty BPK tables; remove all `\colorbox{orange}{TODO}` markers.
- **Measurement stream** — confirm n=2²⁴ SOSD coverage suffices; run B6 (build throughput + query latency vs Grafite/SNARF/SuRF).
- **Slide stream** — build Beamer deck in `defence/slides/` matching `talk-structure.md` (10 core + 4 DLC + 10 backup).

**Critical path:** data audit → fill BPK tables → finalize evaluation chapter → conclusion text → slide 10 → pre-defence → DLC selection → final talk.

**Compile commands:**
- Thesis: `cd Thesis/text && make thesis`
- Beamer deck: `cd Thesis/text/defence/slides && latexmk -pdf defence.tex`

**Scale decision (locked):** SOSD tables at $n=2^{24}$; synthetic tables at $n=2^{20}$. Caption explicit about the difference. Avoids ~1 week of synthetic-at-$2^{24}$ runs that would not change the headline.

---

## Week 1: Foundation (Apr 27 – May 3)

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

## Week 4: Polish + dry runs + submission (May 18 – May 24; submit May 25)

### Task 4.1 — Pre-defence dry run (timed) + fix slow spots

**Files:** none — record a stopwatch sheet inline as a markdown comment in `defence/slides/defence.tex`.

- [ ] **Step 1:** Compile final deck (core only, no DLC). Walk through out loud with stopwatch. Record per-slide actual time.
- [ ] **Step 2:** Identify any slide >150% of budget. Cut or rephrase.
- [ ] **Step 3:** Re-time. Target: ≤ 11:00.
- [ ] **Step 4:** Don't commit timing yet — wait until after Task 4.2.

### Task 4.2 — Pre-defence with supervisor / lab

- [ ] **Step 1:** Schedule the meeting (calendar permitting; ideally early week 4).
- [ ] **Step 2:** Walk through full core (10 slides). Then offer the 4 DLC modules with the question: "Which would you like in the talk?"
- [ ] **Step 3:** Record the chosen DLC list + any direct feedback in `defence/pre-defence-notes.md`.
- [ ] **Step 4:** Commit notes: `git -C Thesis add text/defence/pre-defence-notes.md && git -C Thesis commit -m "docs(defence): pre-defence feedback and DLC selection"`

### Task 4.3 — Insert chosen DLC modules and re-time

**Files:**
- Modify: `defence/slides/defence.tex` to `\input` chosen DLC files at the right points.

- [ ] **Step 1:** Wire selected DLC into `defence.tex` according to the M1–M4 insertion points in `talk-structure.md`.
- [ ] **Step 2:** Compile, verify total slide count increased by chosen-DLC count.
- [ ] **Step 3:** Second timed dry run. Target: ≤ 12:30.
- [ ] **Step 4:** Adjust if over.
- [ ] **Step 5:** Commit: `git -C Thesis add text/defence/slides/defence.tex && git -C Thesis commit -m "feat(defence): integrate chosen DLC modules"`

### Task 4.4 — Final text proofread

**Files:** all of `Thesis/text/src/*.tex` and main `practical-range-emptiness.tex`.

- [ ] **Step 1:** Mechanical scans:
  - `grep -rn 'TODO\|FIXME\|XXX' Thesis/text/src/ Thesis/text/practical-range-emptiness.tex` — must be empty.
  - `grep -rn 'colorbox.*TODO' Thesis/text/` — must be empty.
  - `grep -rn '\\---' Thesis/text/src/` (placeholder dashes in tables) — must be empty.
- [ ] **Step 2:** Read each chapter end-to-end at reading speed; flag awkward sentences. Fix in batch.
- [ ] **Step 3:** Compile final PDF: `cd Thesis/text && make thesis`. Inspect log for warnings: `grep -i 'warning\|undefined' out/practical-range-emptiness.log`. Fix any unresolved references.
- [ ] **Step 4:** Verify: PDF page count is in the program's range; all figures render; bibliography prints.
- [ ] **Step 5:** Commit: `git -C Thesis add . && git -C Thesis commit -m "chore(text): final proofread pass"`

### Task 4.5 — Bump submodule pointer in parent repo

**Files:**
- Modify: parent repo's submodule pointer (single commit, end of week 4 only — per standing rule).

- [ ] **Step 1:** From parent repo: `cd /Users/andrei.ogurtsov/Thesis-Bench-industry && git status` — should show `Thesis` modified.
- [ ] **Step 2:** Commit: `git add Thesis && git commit -m "chore: bump Thesis submodule (defence prep, final text)"`
- [ ] **Step 3:** Don't push automatically. User decides when.

### Task 4.6 — Submit text (2026-05-25)

- [ ] **Step 1:** Follow program submission procedure (portal upload, signed forms, etc.). Out of scope for this plan; user knows the procedure.
- [ ] **Step 2:** Save submission confirmation to `defence/submission-receipt.{pdf,eml}`.

### Task 4.7 — Day-before defence prep

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
| Chapter writing slips by >2 days | Medium | Compresses week 3 | Cut B10 (codebase backup) and B9 (future work backup) — they are cosmetic |
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
