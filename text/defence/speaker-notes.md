# Speaker Notes — Defence Talk

Per-slide notes for the core deck. Single source of truth for what gets
said on stage. Sample-prose blocks are calibration aids for timing —
not memorization scripts.

Conventions per slide:
- **Key points** — what must be said, in order
- **Numbers to nail** — figures that cannot be misquoted
- **Sample prose** — ~30–60s draft of spoken text (calibration only)
- **Anticipated Q&A** — likely committee questions and which backup answers

---

## Slide 1 — Title + arc (0:30)

**Key points:**
- Title, name, supervisor, programme.
- One-sentence promise: Succinct Backend + Scan-ARE, a data-aware filter
  that goes beyond the original scope.

**Sample prose (~25s):**
> "The title of my thesis is Practical Range Emptiness Filters. The work
> splits into two buckets of non-asymptotic optimization: a Succinct
> Backend that improves constants in the known ERE structure, and
> Scan-ARE, a data-aware filter that reaches a regime no industry
> baseline currently reaches on real data."

---

## Slide 2 — Log-Structured Merge-Tree (1:30)

**Key points:**
- Write-optimized; data lives in immutable sorted SSTables on multiple
  levels.
- A point lookup may need to touch every level → I/O amplification.
- Used in production: RocksDB, LevelDB, Cassandra, HBase.

**Sample prose (~75s):**
> "In LSM-tree storage, writes go to a memtable, which periodically
> flushes to immutable sorted SSTables. SSTables accumulate across
> multiple levels. The cost of this design: a single point lookup may
> have to check every level. That's an I/O amplification problem."

**Anticipated Q&A:**
- *Why LSM specifically?* — Filters are most valuable where lookups
  dominate cost. RocksDB's documented BPK budget for filters is 10 BPK;
  that's the operating point we benchmark against.

---

## Slide 3 — Reducing Disk I/O with Filters (1:15)

**Key points:**
- Filters live in SSTable metadata.
- Memory budget: 10–20 bits per key.
- Filters answer "Maybe Contains?" with one-sided error — false positives
  allowed — cutting down disk lookups.
- Bloom filters → point queries; range filters → range queries.

**Sample prose (~60s):**
> "The filter is a small in-memory structure that summarizes which keys
> an SSTable contains. It answers 'Maybe Contains?' queries with
> one-sided error: false positives are allowed, false negatives are not.
> That one-sided guarantee is enough to cut most disk lookups.
> Bloom filters serve point queries: 'is key K in this SSTable?'. For
> range queries — 'does any key in [a,b] live here?' — Bloom doesn't
> suffice; we need a range filter. That's the structure my thesis
> builds."

---

## Slide 4 — Problem and Goswami Baseline (1:30)

**Key points:**
- Define ARE: sorted set, query interval, one-sided ε error, no false
  negatives.
- Goswami SODA'15 lower bound: BPK ≥ log₂(𝓛/ε).
- Their construction reaches it via locality-preserving hash + ERE on
  the shrunk universe.
- Asymptotically optimal — but the constants are large in practice.

**Numbers to nail:**
- Lower bound: log₂(𝓛/ε) bits per key.

**Sample prose (~75s):**
> "The problem: given a sorted set S in a universe of L-bit keys and a
> maximum query range length 𝓛, build a structure that answers 'is
> [a,b] empty in S?' with a one-sided error ε. No false negatives,
> false positive rate at most ε. Goswami in SODA 2015 proved the
> information-theoretic lower bound for any S — log₂(𝓛/ε) bits per
> key. Their construction reaches it: hash the universe down with a
> locality-preserving hash, then run exact range emptiness on the
> shrunk universe. Asymptotically optimal — but the constants behind
> that asymptote are what my thesis attacks."

**Anticipated Q&A:**
- *Where do those bits come from in the bound?* — backup B11.
  BPK ≥ log₂(𝓛/ε) is the per-key form of the total lower bound
  n·log₂(𝓛/ε) bits; O(1) constant factors are absorbed into the bound.

---

## Slide 5 — Succinct Backend (1:00) — centrepiece

**Key points:**
- Goswami's ERE backend uses two bitvectors (D₁, D₂) and a Weak Prefix
  Search inside each bucket.
- Our backend: single bitvector D + binary search on packed suffixes
  inside each bucket.
- Both changes are non-asymptotic. The WPS replacement trades worse
  asymptotics (O(log k) instead of O(1) amortized) for better constants
  on small buckets.
- Bucket index: WPS adds ≥59 BPK of auxiliary index — even the most
  compact variant doesn't fit in the 14–16 BPK industrial budget.

**Numbers to nail:**
- 3.33 BPK (Goswami) → 2.53 BPK (ours) = −0.80 BPK (~24%).
- 1.21–3.00× faster navigation.
- 8–160× faster bucket-internal search.
- ≥59 BPK saved by removing the WPS auxiliary index.

**Sample prose (~55s):**
> "Goswami's structure has two parts that both come with asymptotic
> guarantees. First, navigation through two bit-vectors. We replaced
> them with a single bit-vector — three thirty-three drops to two
> fifty-three bits per key of metadata, that's twenty-four percent
> off, predicted analytically by the Poisson load factor (one minus
> e to the minus one) and confirmed empirically across all n.
>
> The second part is more interesting. Goswami uses a weak prefix
> search structure for bucket-internal queries. It is theoretically
> O(1), but that constant costs 59 bits per key of auxiliary index
> and 370–567 nanoseconds per query — because each 'constant-time'
> step is a hash probe plus pointer chasing. We replaced it with a
> plain binary search on packed suffixes. Yes, logarithmic; but on
> small buckets — which is the common case — the entire search fits
> in L1 cache and runs in 5 to 40 nanoseconds. Net result: 8 to 160
> times faster, with no auxiliary index. The 8–160× I'm quoting is
> the backend microbenchmark; how it translates to end-to-end query
> latency is in the appendix if anyone wants to see."

**Anticipated Q&A:**
- *Why the Poisson factor 0.632 in metadata savings?* — 1 − e⁻¹ at
  λ = 1 (Poisson load); DLC M1 has the formulas, backup B5 has the
  bucket statistics.
- *What about the WPS variants — there are several?* — backup B2.
- *End-to-end latency vs microbenchmark?* — backup B12.

---

## Slide 6 — Real Distributions Are Clustered (0:50) — bridge

**Key points:**
- Goswami's bound makes no assumptions about S — it holds for every
  input.
- Real LSM keys are not arbitrary — they concentrate in dense clusters
  with sparse gaps.
- Examples: file paths, URLs, timestamps, S2 cell IDs, auto-increment
  primary keys.
- The next three slides trade BPK by exploiting this structure:
  clusters → exact ERE, sparse tail → truncation hash, detection via
  1D-DBSCAN.

**Sample prose (~50s):**
> "On the previous slide we saw a backend speedup that's universal —
> works regardless of what the keys look like. The Goswami bound itself
> is the same kind of universal: log₂ of 𝓛 over ε, no assumptions
> about S, holds for every input. That's the bound you get when you
> know nothing about the distribution. Real LSM keys, though, are not
> arbitrary. They concentrate in dense clusters with sparse gaps —
> file paths share directory prefixes, URLs share domains, timestamps
> come consecutive. Here's SOSD Books on the right: a dense block of
> two hundred million keys in the first quarter of the universe, then
> a gap, then regular subclusters above. The next three slides trade
> space-per-key by exploiting this structure. They cut bits, not
> nanoseconds."

**Anticipated Q&A:**
- *What about uniform data?* — Limitations slide 11 addresses it
  directly.
- *How do you measure 'clustered'?* — backup B7 (SOSD distribution
  histograms across all four datasets).

---

## Slide 7 — Dense Clusters: Compress Down to Exact (~0:50)

**Key points:**
- Real keys occupy a narrow window of [0, 2^L) — they cluster, not fill.
  Verbal example: every DB timestamp shares the top bits up to the
  millennium; the rest of the universe is empty.
- Subtract k_min: the same n keys live in [0, U'), where
  U' = k_max − k_min ≪ 2^L.
- ARE size (Goswami lower bound): n · log₂(𝓛/ε) bits — **independent of U**.
- ERE size: n · log₂(U'/n) + O(n) bits — **shrinks with U'**.
- Crossover at U' ≤ n𝓛/ε: ERE is cheaper than any ARE *and* gives FPR = 0
  (exact mode).

**Numbers to nail:**
- ARE size: n · log₂(𝓛/ε) — does not depend on U.
- ERE size: n · log₂(U'/n) + O(n).
- Crossover threshold: U' ≤ n𝓛/ε.

**Sample prose (~50s):**
> "Real keys occupy a narrow window of the universe. Take any
> database — every timestamp shares the top bits up to the millennium;
> the rest is empty. So before building any filter, subtract the
> minimum key. The same n keys now live in [0, U-prime), where U-prime
> equals k-max minus k-min — far less than 2 to the L.
>
> What did the smaller universe buy us? Two formulas, side by side.
> Goswami's lower bound for any approximate filter is n times log of
> 𝓛 over ε — it does not depend on U at all. The exact filter, ERE,
> costs n times log of U-prime over n — and *that* one shrinks as the
> cluster gets denser.
>
> So at small enough U-prime — concretely, U-prime below n times 𝓛
> over ε — ERE is cheaper than any approximate filter we could build,
> AND it gives false-positive rate exactly zero. That's exact mode."

**Anticipated Q&A:**
- *How small is "small enough U-prime"?* — boxed condition U' ≤ n𝓛/ε.
  For 𝓛 = 128, ε = 10⁻³, n = 2²⁰: any cluster narrower than ~10¹¹.
- *What if a segment is too spread?* — fall through to truncation
  fallback on the next slide.
- *Why subtract min and not, say, divide?* — affine shift is free
  (one stored constant), ERE size only cares about spread, not
  absolute position.

---

## Slide 8 — Sparse Tail: Truncate Low Bits (~0:45)

**Key points:**
- On the sparse tail (between clusters) ERE would be expensive — wide U.
- Truncation hash h(x) = ⌊x / 2ᵗ⌋: drop the bottom t bits before
  fingerprinting.
- Phantom geometry differs from a random hash:
  - SODA random hash: phantoms scattered uniformly across U;
    collision zone per key = 𝓛 × 2ᵗ (multiplicative).
  - Truncation: all phantoms of one key sit in a single block of
    width 2ᵗ next to it; collision zone = 𝓛 + 2ᵗ (additive).
- × → + costs us log₂(𝓛) fewer bits per key.
- Cost: phantoms cluster near keys → on dense data they overlap and
  FPR → 1. So truncation only on sparse regions; dense clusters went
  to exact mode on the previous slide.

**Numbers to nail:**
- BPK saving: log₂(𝓛). At 𝓛 = 128, that's −7 bits per key.
- Collision zone: 𝓛·2ᵗ → 𝓛 + 2ᵗ.

**Sample prose (~45s):**
> "On the sparse tail between clusters, ERE would be expensive — the
> universe is wide. Use a truncation hash instead: drop the bottom t
> bits. The phantom geometry changes completely.
>
> With a random hash like SODA, every key produces phantoms scattered
> uniformly across U — its collision zone is 𝓛 times two-to-the-t,
> multiplicative. With truncation, all phantoms of one key sit in a
> single block of width two-to-the-t right next to it — the collision
> zone is 𝓛 plus two-to-the-t, additive.
>
> Times-to-plus is exactly log of 𝓛 fewer bits per key. At 𝓛 equal
> to one hundred and twenty-eight, that's seven bits per key cheaper
> than Goswami SODA.
>
> Cost: phantoms cluster near keys, so on dense data they overlap and
> FPR blows up. That's why truncation only on the sparse part — the
> dense clusters already went to exact mode on the previous slide."

**Anticipated Q&A:**
- *Why does truncation save log₂(𝓛)?* — multiplicative-to-additive
  collision zone (B1).
- *In `𝓛 + 2ᵗ`, why does 𝓛 disappear from BPK and not 2ᵗ?* —
  asymmetry of choice. 2ᵗ is the parameter we *pick* (minimising
  K = 𝓛 − t means maximising t); 𝓛 is fixed by the problem.
  In the optimal regime ε·2^L/n ≫ 𝓛, the constraint
  𝓛 + 2ᵗ ≤ ε·2^L/n becomes 2ᵗ ≈ ε·2^L/n with 𝓛 a negligible
  additive correction — it falls into the O(1) of the log. In
  the multiplicative form 𝓛·2ᵗ, by contrast, 𝓛 *divides* the
  budget, forcing us to truncate log₂(𝓛) bits less. One-liner:
  "× is symmetric, + is dominated by the larger term — and we
  always pick 2ᵗ to be that larger term."
- *What if I truncate dense data?* — phantoms overlap, FPR → 1.
  Limitations slide directly demonstrates this.

---

## Slide 9 — Detection: 1D-DBSCAN over Sorted Keys (~0:45)

**Key points:**
- Last piece: given a key set, decide which segments go to exact
  (slide 7) and which to truncation (slide 8).
- One pass over sorted keys, O(n).
- Gap ≤ δ ⇒ same cluster; gap > δ ⇒ cluster boundary.
- δ = c · 𝓛/ε derived from problem parameters.
- Fingerprint width K = ⌈log₂(n·𝓛/ε)⌉ — define K on first use.
- Algebraically the same condition as the exact-mode crossover
  ρ > ε/𝓛, just rewritten in gap form.
- No learning, no training data.

**Numbers to nail:**
- Build complexity: O(n).
- δ = c · 𝓛/ε.

**Sample prose (~45s):**
> "Last piece: given a key set, how do we decide which segments go to
> exact mode and which to truncation? One pass over the sorted keys —
> gap below δ stays inside a cluster, gap above δ is a boundary.
> Linear time.
>
> The threshold δ equals c times 𝓛 over ε. We don't learn it — there
> is nothing to learn. It comes from the problem parameters directly,
> and it is algebraically the same condition as the exact-mode
> crossover from the clusters slide, just rewritten in gap form
> instead of density form. The constant c we set empirically.
>
> Each segment that falls to the truncation path stores fingerprints
> of width K = ⌈log₂(n·𝓛/ε)⌉ — K is the fingerprint width, set so
> that the false-positive probability matches ε.
>
> Compared to learned filters like SNARF, this is a different bargain.
> SNARF learns a CDF and uses it for layout; we use the structural
> fact that LSM keys cluster, without committing to any specific
> distribution shape."

**Anticipated Q&A:**
- *Sensitivity to δ?* — backup B3.
- *What's c?* — empirical, between 0.5 and 2 across SOSD; doesn't
  matter much in practice (B3).
- *vs learned filters?* — we don't model the distribution, only
  cluster it. Robust to distribution drift.
- *Why DBSCAN and not k-means?* — k unknown, densities differ.

---

## Slide 10 — Headline: SOSD Facebook, 𝓛=128, n=2²⁴ (1:15)

**Key points:**
- FPR vs BPK plot on real data, n=2²⁴, 𝓛=128.
- Scan-ARE hits the 0-FP floor at ~11 BPK.
- SNARF saturates at FPR ≈ 7·10⁻⁴.

**Numbers to nail:**
- Scan-ARE: FPR < 10⁻⁸ (the 0-FP floor) at ~11 BPK on this dataset.
- SNARF saturation: FPR ≈ 7·10⁻⁴ irrespective of memory.

**Sample prose (~55s):**
> "Headline. SOSD Facebook, n equals two to the twenty-four — same
> scale at which Grafite, SNARF, and SuRF report their headline
> numbers. X axis: bits per key. Y axis: false positive rate, log
> scale. Scan-ARE in magenta hits the zero-FP floor at around eleven
> bits per key. SNARF, the strongest learned filter from VLDB 2022,
> saturates at seven times ten to the minus four — it doesn't matter
> how much memory you give it, that's where it stops."

**Anticipated Q&A:**
- *Why this dataset?* — backup B7 (SOSD distribution histograms).
- *Build cost?* — backup B6 (build throughput).
- *Latency?* — backup B12.

---

## Slide 11 — Limitations: Uniform, 𝓛=1, n=2²⁴ (0:30)

**Key points:**
- On uniform data with point queries, Scan-ARE has nothing to segment.
- Grafite, designed for adversarial point queries, hits 0-FP at
  ~19 BPK and wins.
- Honest reading: our gain lives where real data lives.

**Numbers to nail:**
- Grafite reaches 0-FP at ~19 BPK on uniform.
- Scan-ARE pays segmentation overhead with no segmentation benefit.

**Sample prose (~30s):**
> "Honest limitation: where we don't win. Uniform keys, point queries.
> No clusters to segment, no range to leverage truncation. Grafite,
> designed exactly for this regime, hits zero-FP at nineteen bits per
> key. We pay segmentation overhead with nothing to segment. Our gain
> lives where real data lives."

---

## Slide 12 — Conclusion: Three Receipts (1:00)

**Key points:**
- Three numbers matching the slide columns: −24% metadata (Backend —
  one-vector ERE), 8–160× query latency improvement (Backend — query
  latency), FPR < 10⁻⁸ at ~11 BPK on Facebook user IDs (Scan-ARE).
- Arc closure: "Succinct backend, plus a data-aware Scan-ARE filter."
- One-line hook to backup: end-to-end latency in appendix.

**Sample prose (~50s):**
> "Three receipts. Twenty-four percent fewer metadata bits — Backend,
> one-vector ERE. Eight to one-hundred-sixty times faster query
> latency — Backend again, replacing weak prefix search with binary
> search. FPR below ten to the minus eight at around eleven bits per
> key on Facebook user IDs — Scan-ARE. End-to-end query latency
> results are in the appendix. Future work: dynamic updates, RocksDB
> integration, larger n. Thank you."

**Anticipated Q&A:**
- *Latency?* — appendix B12, ready.
- *RocksDB integration plan?* — discuss verbally; no slide.
- *Dynamic updates?* — discuss verbally; future work.

---

## Deferred — Combined headline (memory + latency)

If pre-defence committee asks about query latency in the first 30
seconds of Q&A, that's a signal the question is so natural the
headline should answer it directly. In that case, replace slide 10
with a stacked two-plot version: top panel FPR-vs-BPK, bottom panel
latency-vs-𝓛, single shared legend (both plots use the same filter
set, so the legend is one block). Requires re-rendering both plots
through a shared-legend pipeline. Keep the current single-plot
slide 10 as the safe default until the pre-defence signal arrives.
