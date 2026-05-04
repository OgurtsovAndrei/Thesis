# Defence Talk — Structure & Slide Plan

Working artefact for the bachelor's defence talk. Captures the narrative
decisions, the slide-by-slide breakdown, the optional DLC modules selectable
on pre-defence, and the Q&A backup pool.

The LaTeX (Beamer) deck will be built from this plan in `defence/slides/`.

---

## Narrative decisions

These are the framing decisions made during planning. They are load-bearing
for everything below — change them only after re-thinking the consequences.

1. **Headline result is Scan-ARE on long-range queries over clustered data.**
   The defining empirical claim: `FPR < 10⁻⁸ at ~5 BPK on SOSD L=65536`,
   while SOTA filters (Grafite, SNARF, SuRF) either degrade to FPR≈1 or
   need 10–30 BPK to reach FPR≈10⁻²–10⁻³.

2. **Framing arc preserves the proposal commitment.**
   The proposal promised "non-asymptotic optimizations in ARE filters".
   The talk delivers exactly that, organized as **three layers** of
   non-asymptotic work, with Layer 3 producing the headline result:
   - **Layer 1**: succinct backend
   - **Layer 2**: hash function design
   - **Layer 3**: distribution-aware segmentation (Scan-ARE)

3. **WPS → binary search is the canonical non-asymptotic example.**
   Theoretically O(1) WPS index (59 BPK auxiliary, 370–567 ns) replaced by
   O(log k) binary search on packed suffixes (5–40 ns). Speedup: 6–110×.
   This is the cleanest illustration of the talk's thesis and lives on
   the Layer 1 slide as a co-equal point with one-vector ERE.

4. **CDF-ARE is excluded from the talk entirely.**
   It remains in the written thesis as a documented negative result.
   Mentioning it on stage burns 30s, dilutes the message, and invites
   questions with no upside.

5. **Modular talk format: 11-min core + 4 DLC modules + Q&A backups.**
   On pre-defence we ask the committee *what to add* (not what to cut).
   They pick 0–3 DLC modules; final talk is 11:00–13:00 by their choice.
   This avoids surrendering control of which content gets cut.

6. **Headline plot uses n = 2²⁴**, not n = 2²⁰.
   Plots already exist in `bench_results/plots/N16777216/`. This is the
   scale at which Grafite/SNARF/SuRF report their headline numbers, so
   matching it removes a likely reviewer objection.

---

## Core slide deck (target 11:50)

| # | Slide | Time | Bullets / content | Key number / visual |
|---|-------|------|-------------------|---------------------|
| 1 | **Title + arc** | 0:30 | Title, name, supervisor, programme. One sentence promise: "non-asymptotic optimizations on three layers; Layer 3 produces a result beyond the original scope." | — |
| 2 | **LSM-tree** | 1:30 | Memtable + immutable SSTables on levels; write-optimized; lookup at every level. RocksDB, LevelDB, Cassandra, HBase. | LSM diagram (reuse from proposal slide 2). |
| 3 | **Filters in SSTables** *(merged 2+3 from proposal)* | 1:15 | Filters live in SST metadata; small, trivially serializable; avoid disk I/O when key absent. **Bloom = point queries; range queries need a range filter.** | LSM diagram with filter blobs (reuse). |
| 4 | **Problem + Goswami baseline** *(merged: variant β)* | 1:30 | ARE definition; one-sided ε; FPR / BPK metrics. Lower bound `n · log₂(L/ε)` (Goswami, SODA'15). Their construction reaches it via **locality-preserving hash + ERE**. | Lower-bound formula + hash→ERE pipeline diagram. |
| 5 | **Layer 1 — Succinct backend** | 1:00 | Goswami baseline: two bit-vectors (D₁, D₂) for navigation; weak prefix search for bucket-internal queries (theoretically O(1)). Ours: single bit-vector D; binary search on packed w-bit suffixes. | **−24% metadata** (≈ −0.8 BPK); **8–160× faster** than O(1) WPS. |
| **6** | **Real distributions are clustered** *(bridge — new)* | **0:50** | Goswami's bound is worst-case (every S). Real LSM keys are not — dense clusters separated by sparse gaps. Examples: file paths, URLs, timestamps, S2 cells, ISBNs. **L1 cut nanoseconds; L2 + L3 will cut bits, by exploiting this structure.** | `hist_sosd_books.pdf` — visible block + gap + scattered subclusters. |
| 7 | **Dense clusters → exact** | 0:50 | Subtract `k_min` → small `U' = max−min`. ARE size `n·log₂(L/ε)` is independent of `U`; ERE size `n·log₂(U'/n)+O(n)` shrinks with `U'`. Crossover `U' ≤ nL/ε` ⇒ exact mode (FPR = 0). | Two formulas (ARE Goswami / ERE) + boxed crossover. No figure. |
| 8 | **Sparse tail → truncation** | 0:45 | Truncation hash `h(x)=⌊x/2^t⌋`: collision zone `L·2^t` → `L+2^t` (× → +) ⇒ **−log₂(L) BPK**. Cost: phantoms cluster near keys → fails on dense data — but those went to exact on slide 7. | Phantom comparison figure (`phantom_comparison.pdf`). |
| 9 | **Detection (1D-DBSCAN)** | 0:45 | One-pass scan over sorted keys, O(n). Gap ≤ δ ⇒ same cluster. `δ = c·L/ε` from problem parameters; algebraically the same condition as the exact-mode crossover, rewritten in gap form. | Segmentation diagram (`segmentation.pdf`). |
| 10 | **Headline result** | 1:15 | FPR-vs-BPK plot, SOSD Facebook, n = 2²⁴, L = 128. Spoken framing: "BPK on x-axis, FPR on y-axis; legend includes SuRF (SIGMOD'18), SNARF (VLDB'22), Grafite (SIGMOD'24)." | **Scan-ARE hits 0-FP floor at ~11 BPK; SNARF saturates at FPR ≈ 7·10⁻⁴**. Plot file: `figures/headline_fb_L128.pdf`. |
| 11 | **Limitations** | 0:30 | Uniform data: exact mode never triggers, Scan-ARE collapses to SODA fallback. Sensitivity to `δ` and `minClusterSize`. Build cost grows with cluster count. | — |
| 12 | **Conclusion** | 1:00 | Arc closure: "promised non-asymptotic, delivered on three layers." Three numbers: 8–160× WPS, FPR=0 exact mode, FPR<10⁻⁸ at 5 BPK. **One-line latency hook → backup B12.** Future: dynamic updates, RocksDB integration, larger n. | Slide with three numbers in a row + latency-pointer line + future work. |
| | **Total** | **11:40** | | |

### Speaker notes

Per-slide spoken text, key points, and anticipated Q&A live in
[`speaker-notes.md`](speaker-notes.md). That file is the single source
of truth for what gets said on stage; this document stays focused on
deck structure, time budget, and DLC/Q&A planning.

Slide 5 (Layer 1) remains the canonical non-asymptotic example. If a
single slide must survive scope cuts, it is this one.

---

## DLC modules (selected on pre-defence)

Pre-defence question to the committee: **"Which of these would you
like to see in the talk?"** They pick 0–3.

| ID | Insertion point | Time | Content |
|----|-----------------|------|---------|
| **M1** | After #5 | +0:45 | One-vector layout in detail: formulas `\|D₁\|+\|D₂\| = B + n + M + 1` vs `\|D\| = B + n + 1`. Poisson `1 − e⁻¹ ≈ 0.632` as analytic prediction; empirical measurement matches. |
| **M2** | After #7 | +0:30 | Adaptive exact-mode threshold: derivation of `ρ > ε/L`. Table of typical (L, ε) values and corresponding density thresholds. |
| **M3** | Between #3 and #4 | +0:45 | LSM context: per-SSTable n ∈ [2¹⁹, 2²³] from Cao FAST'20; RocksDB BPK budget = 10. Justifies the parameter choices in the benchmarks. |
| **M4** | Before #10 | +0:45 | Industry comparison table: SuRF / SNARF / Grafite / Memento — locate / memory / build complexity. Reuse the table from the proposal (slide 5), now placed where it actually lands. |

Extended talk = 11:50 core + chosen DLC. Maximum (all 4) = 14:35.
Recommended target after pre-defence: 12:30–13:20 with 1–2 DLC selected.

---

## Q&A backup slides (silent reserve)

Not shown during the talk. Triggered only by a question. Each lives on a
single slide; aim for a 30-second answer using the slide as anchor.

| ID | Topic | Source |
|----|-------|--------|
| **B1** | Phantom geometry: SODA scattered vs Truncation contiguous | Reuse `figures/phantom_comparison.pdf` |
| **B2** | WPS variants: 3 implementations, 59/151/179 BPK auxiliary, latency vs binary search | Reuse Tables in `src/ere.tex` (`tab:wps-cost`, `tab:wps-vs-binsearch`) |
| **B3** | DBSCAN δ derivation; sanity table (cluster / uniform / sequential) | Reuse `tab:eps-sanity` from `src/hybrid.tex` |
| **B4** | Bucket occupancy across SOSD: X₅₀, X₉₀, max; saturation | Reuse `tab:eval-bucket-fill` from `src/evaluation.tex` |
| **B5** | ERE bucket Poisson tail; Mitzenmacher Lemma 5.1 | Reuse `tab:ere-bucket-stats`, `tab:ere-poisson-tail` from `src/ere.tex` |
| **B6** | Build throughput vs SOTA | **TODO**: needs new measurements (see week-1 plan) |
| **B7** | SOSD distribution histograms (FB / Books / Wiki / OSM) | Reuse `figures/hist_sosd_*.pdf` |
| **B8** | CDF-ARE as a documented negative result | New slide — short summary of why it didn't work |
| **B9** | Future work expanded: dynamic updates, n = 2²⁷, RocksDB integration | New slide |
| **B10** | Codebase scale: 6 ARE variants, CGo wrappers, test count | New slide |
| **B11** | Goswami bound decomposition + LPH construction sketch | New slide — `log₂(nL/ε)/n = log₂(L/ε)/n + log₂(n)/n` (LPH cost + ERE cost); block-decomposition idea behind the hash; cite Goswami SODA'15. Triggered if asked "where do the bits come from?" or "how does that hash work?" |
| **B12** | **Query latency vs SOTA** — gap-heavy SOSD-FB, n=2²⁴, ε=10⁻². Scan-ARE ~60 ns flat; SNARF ~700 ns; Rosetta degrades to 25 µs at L=4096; Grafite stops at L>8. **Forced by the latency-pointer line on slide 12.** | `figures/latency_fb_gap_heavy.pdf` (rendered from `bench_results/plots/b6_N16777216_gap_heavy/query_latency/sosd_fb.svg`). |

B6 is the only remaining backup that requires fresh measurements (build
throughput). B12 is wired into the deck via `\input{backup/B12-query-latency}`
and has a deliberate hook on slide 12 to surface it during Q&A.

---

## Open questions (to settle in the 4-week plan)

- ~~Which SOSD dataset for slide 10 — Books or Facebook?~~ **Resolved:**
  Facebook for the headline (slide 10, `headline_fb_L128.pdf`); Books for
  the bridge slide 6 (`hist_sosd_books.pdf`, the cleanest visual of
  "dense block + gap + scattered subclusters").
- Build throughput benchmarks against industry filters — still required
  for B6. Estimate runtime and decide whether to commit a week to it or
  accept reviewer ambiguity. Query latency is now done (B12).
- ~~Defence language~~ **Resolved:** English (speaker notes in
  `speaker-notes.md` are written in English).

---

## Deferred: combined headline (memory + latency)

If pre-defence committee asks about query latency in the first 30
seconds of Q&A, that's a signal the question is so natural the
headline should answer it directly. In that case, replace slide 10
with a stacked two-plot version:

- top panel: FPR vs BPK (current `headline_fb_L128.pdf`)
- bottom panel: query latency vs L (`latency_fb_gap_heavy.pdf`)
- single shared legend across both panels

Both plots already use the same filter set (Scan-ARE, SODA, Grafite,
SNARF, SuRF, Rosetta, BloomARE, Greedy+Merge), so the legend collapses
to one block. Required: re-render both through a shared-legend
pipeline. Keep the current single-plot slide 10 + B12 as the safe
default until the pre-defence signal arrives.

---

## File layout

```
defence/
├── talk-structure.md         (deck structure, time budget, DLC/Q&A planning)
├── speaker-notes.md          (per-slide spoken text, key points, anticipated Q&A)
└── slides/                   (Beamer LaTeX deck)
    ├── defence.tex
    ├── core/                 (12 numbered slides; 06-distributions is the bridge)
    ├── dlc/                  (M1–M4, currently empty)
    ├── backup/               (B12-query-latency wired; others as needed)
    └── figures/              (PDFs: headline, limitations, latency, lsm-tikz includes)
```
