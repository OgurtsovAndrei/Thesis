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

## Core slide deck (target 11:00)

| # | Slide | Time | Bullets / content | Key number / visual |
|---|-------|------|-------------------|---------------------|
| 1 | **Title + arc** | 0:30 | Title, name, supervisor, programme. One sentence promise: "non-asymptotic optimizations on three layers; Layer 3 produces a result beyond the original scope." | — |
| 2 | **LSM-tree** | 1:30 | Memtable + immutable SSTables on levels; write-optimized; lookup at every level. RocksDB, LevelDB, Cassandra, HBase. | LSM diagram (reuse from proposal slide 2). |
| 3 | **Filters in SSTables** *(merged 2+3 from proposal)* | 1:15 | Filters live in SST metadata; small, trivially serializable; avoid disk I/O when key absent. **Bloom = point queries; range queries need a range filter.** | LSM diagram with filter blobs (reuse). |
| 4 | **Problem + Goswami baseline** *(merged: variant β)* | 1:30 | ARE definition; one-sided ε; FPR / BPK metrics. Lower bound `n · log₂(L/ε)` (Goswami, SODA'15). Their construction reaches it via **locality-preserving hash + ERE**. | Lower-bound formula + hash→ERE pipeline diagram. |
| 5 | **Layer 1 — Succinct backend** | 1:00 | Goswami baseline: two bit-vectors (D₁, D₂) for navigation; weak prefix search for bucket-internal queries (theoretically O(1)). Ours: single bit-vector D; binary search on packed w-bit suffixes. | **−24% metadata** (≈ −0.8 BPK); **6–110× faster** than O(1) WPS. |
| 6 | **Layer 2 — Hash design** | 0:45 | Truncation hash: phantoms contiguous → **−log₂(L) BPK** vs SODA on uniform-like data. Adaptive: when local density `ρ > ε/L`, hash unnecessary → **FPR = 0**. | Phantom comparison figure + density-threshold table. |
| 7 | **Layer 3 — Scan-ARE** | 1:45 | 1D-DBSCAN over sorted keys, O(n). Dense clusters → exact sub-filters (FPR=0). Sparse remainder → truncation fallback. `δ = c · L/ε` derived from problem parameters — no data statistics required. | Segmentation diagram (`segmentation.pdf` from `figures/`). |
| 8 | **Headline result** | 1:15 | FPR-vs-BPK plot, SOSD Books or FB, n = 2²⁴, L = 65536. Spoken framing: "BPK on x-axis, FPR on y-axis; legend includes SuRF (SIGMOD'18), SNARF (VLDB'22), Grafite (SIGMOD'24)." | **Scan-ARE: FPR < 10⁻⁸ at ~5 BPK; SOTA: FPR ≈ 10⁻² or 10–30 BPK**. Plot file: `bench_results/plots/N16777216/sosd_books/L65536.svg` (or `sosd_fb`). |
| 9 | **Limitations** | 0:30 | Uniform data: exact mode never triggers, Scan-ARE collapses to SODA fallback. Sensitivity to `δ` and `minClusterSize`. Build cost grows with cluster count. | — |
| 10 | **Conclusion** | 1:00 | Arc closure: "promised non-asymptotic, delivered on three layers." Three numbers: 6–110× WPS, FPR=0 exact mode, FPR<10⁻⁸ at 5 BPK. Future: dynamic updates, RocksDB integration, larger n. | Slide with three numbers in a row + one line of future work. |
| | **Total** | **11:00** | | |

### Speaker note for slide 5 (Layer 1 — the centrepiece)

Approximate spoken text, ~55 seconds. Not memorized verbatim — used to
calibrate timing and check the argument flows.

> "Goswami's structure has two parts that both come with asymptotic
> guarantees. First, navigation through two bit-vectors. We replaced
> them with a single bit-vector — saves 24% of metadata, predicted
> analytically by the Poisson distribution and confirmed empirically.
>
> The second part is more interesting. Goswami uses a weak prefix
> search structure for bucket-internal queries. It is theoretically
> O(1), but that constant costs 59 bits per key of auxiliary index and
> 370–567 nanoseconds per query — because each "constant-time" step is
> a hash probe plus pointer chasing. We replaced it with a plain
> binary search on packed suffixes. Yes, logarithmic; but on small
> buckets — which is the common case — the entire search fits in L1
> cache and runs in 5 to 40 nanoseconds. Net result: 6 to 110 times
> faster, with no auxiliary index."

This slide is the canonical non-asymptotic example. If a single slide
must survive scope cuts, it is this one.

---

## DLC modules (selected on pre-defence)

Pre-defence question to the committee: **"Which of these would you
like to see in the talk?"** They pick 0–3.

| ID | Insertion point | Time | Content |
|----|-----------------|------|---------|
| **M1** | After #5 | +0:45 | One-vector layout in detail: formulas `\|D₁\|+\|D₂\| = B + n + M + 1` vs `\|D\| = B + n + 1`. Poisson `1 − e⁻¹ ≈ 0.632` as analytic prediction; empirical measurement matches. |
| **M2** | After #6 | +0:30 | Adaptive exact-mode threshold: derivation of `ρ > ε/L`. Table of typical (L, ε) values and corresponding density thresholds. |
| **M3** | Between #3 and #4 | +0:45 | LSM context: per-SSTable n ∈ [2¹⁹, 2²³] from Cao FAST'20; RocksDB BPK budget = 10. Justifies the parameter choices in the benchmarks. |
| **M4** | Before #8 | +0:45 | Industry comparison table: SuRF / SNARF / Grafite / Memento — locate / memory / build complexity. Reuse the table from the proposal (slide 5), now placed where it actually lands. |

Extended talk = 11:00 core + chosen DLC. Maximum (all 4) = 13:45.
Recommended target after pre-defence: 11:30–12:30 with 2 DLC selected.

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
| **B6** | Build throughput + query latency vs SOTA | **TODO**: needs new measurements (see week-1 plan) |
| **B7** | SOSD distribution histograms (FB / Books / Wiki / OSM) | Reuse `figures/hist_sosd_*.pdf` |
| **B8** | CDF-ARE as a documented negative result | New slide — short summary of why it didn't work |
| **B9** | Future work expanded: dynamic updates, n = 2²⁷, RocksDB integration | New slide |
| **B10** | Codebase scale: 6 ARE variants, CGo wrappers, test count | New slide |
| **B11** | Goswami bound decomposition + LPH construction sketch | New slide — `log₂(nL/ε)/n = log₂(L/ε)/n + log₂(n)/n` (LPH cost + ERE cost); block-decomposition idea behind the hash; cite Goswami SODA'15. Triggered if asked "where do the bits come from?" or "how does that hash work?" |

B6 is the only backup that requires fresh measurements. Everything else
reuses material that already exists in the thesis text or in
`bench_results/`.

---

## Open questions (to settle in the 4-week plan)

- Which SOSD dataset for slide 8 — Books or Facebook? Books is the
  cleaner win (`FPR < 10⁻⁸`), FB is the larger universe. Pick after
  re-rendering both at n = 2²⁴ and inspecting.
- Build / query benchmarks against industry filters — required for B6.
  Estimate runtime and decide whether to commit a week to it or accept
  reviewer ambiguity.
- Defence language (English vs Russian) — content-neutral; affects
  speaker notes and slide labels only.

---

## File layout (planned)

```
defence/
├── talk-structure.md         (this file — single source of truth)
├── slides/                   (Beamer LaTeX deck, built from this plan)
│   ├── defence.tex
│   ├── core/
│   ├── dlc/
│   └── backup/
└── notes/                    (speaker notes per slide, optional)
```
