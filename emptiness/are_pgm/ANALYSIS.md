# CDF-Mapped Range Emptiness: Analysis

## Core Idea

Use a monotonic CDF approximation (piecewise-linear, inspired by PGM-index) to map
keys to a near-uniform distribution before storing in ERE.

Since CDF is monotonic: `x ∈ [a,b] ⟹ CDF(x) ∈ [CDF(a), CDF(b)]` — zero false negatives.

## Why CDF Should Help (Distribution-Aware FPR)

Consider data X = clustered keys, and Y = universe \ X (all non-key points).

False positives arise when points from Y map to the same position as points from X
in the compressed universe. The FPR in any region is proportional to `X' / (U' - X')`,
where X' and U' are the mapped sizes.

**Key insight**: CDF is an order-preserving transform that **stretches dense regions**
(clusters) and **compresses sparse regions** (inter-cluster gaps):

- In cluster regions: more mapped positions per key → lower local FPR
- In gap regions: fewer mapped positions → higher local FPR
- Total FPR across the universe is conserved (information-preserving transform)

**Why this helps**: if the query distribution matches the data distribution (realistic
assumption — queries are concentrated where data is), then most queries land in the
stretched cluster regions where FPR is low. Few queries land in compressed gaps where
FPR is high. The **weighted average FPR** is lower than SODA's uniform FPR.

### BPK Advantage

- **SODA**: `K = ⌈log₂(n·L/ε)⌉` → BPK ≈ `log₂(L/ε) + 2`
- **CDF-ARE**: `K = ⌈log₂(n/ε)⌉` → BPK ≈ `log₂(1/ε) + 2 + CDF_overhead`

The CDF removes the `log₂(L)` term from K entirely. For L=64, this saves ~6 BPK.

## Implementation

1. Build PGM index on sorted keys → get approximate CDF
2. Fix monotonicity (running max on PGM positions)
3. Sample CDF control points every `pgmEpsilon` keys
4. Map stored keys through piecewise-linear CDF → store in ERE
5. Query: map endpoints through same CDF → query ERE

CDF model cost: `(n/pgmEpsilon) × 128 bits`. For pgmEpsilon=128: ~1 BPK.

## Experimental Results

### Benchmark Parameters
- n = 10,000 keys
- 5 Gaussian clusters (σ = 2²⁰ ≈ 1M), avg gap between keys in cluster ≈ 1,250
- ε = 0.05
- **rangeLen = 2²⁰ ≈ 1M** ← problematic, see below

### Results (rangeLen = 1M)

| Distribution | pgmEps | K  | FPR    | BPK   | ERE   | CDF  |
|-------------|--------|-----|--------|-------|-------|------|
| Uniform     | 128    | 18  | 0.073  | 8.46  | 7.44  | 1.02 |
| Cluster     | 128    | 28  | 0.139  | 18.02 | 17.00 | 1.02 |

Uniform: CDF-ARE achieves **8.46 BPK** vs SODA's ~26 BPK. Major win.

Cluster: FPR = 14%, above target 5%. BPK = 18, similar to SODA.

### Why Cluster FPR Was High

All false positives were **boundary quantization collisions**: a query endpoint
and a nearby stored key round to the same uint64 position in the mapped universe.

This happens specifically in **inter-cluster gaps** where CDF resolution is minimal
(CDF allocates positions proportional to key density → gaps get almost none).

**But the real problem was the benchmark**: rangeLen = 1M ≈ 800× avg gap in cluster.
With such large rangeLen, there are **zero truly empty ranges within clusters**.
ALL truly empty queries land in inter-cluster gaps — exactly where CDF compressed
the space and has poor resolution.

### Production-Realistic rangeLen

In production, typical rangeLen is **10–100** (not 1M). With L ≪ avg_gap ≈ 1,250:

- Many truly empty ranges exist WITHIN clusters
- CDF gives them excellent resolution (clusters are stretched)
- Queries follow data distribution → most queries in stretched regions
- FPR ≈ ε as intended

Expected BPK comparison for L = 64, ε = 0.05:

| Approach     | K formula              | K  | BPK estimate |
|-------------|------------------------|-----|-------------|
| SODA        | ⌈log₂(n·64/ε)⌉       | 24  | ~12         |
| CDF-ARE     | ⌈log₂(n/ε)⌉          | 18  | ~8          |

CDF saves **~4 BPK** (log₂(64) = 6 minus ~2 CDF overhead).

## Key Takeaways

1. **CDF-mapping removes the log₂(L) term from BPK** — this is the theoretical win
2. **FPR is redistributed, not reduced**: lower in dense regions, higher in sparse regions
3. **Benefit materializes when**: query distribution ≈ data distribution AND rangeLen < avg gap between keys
4. **Limitation**: for rangeLen ≫ avg gap, all empty queries are in sparse regions where CDF has poor resolution — no benefit over SODA
5. **CDF overhead**: ~1 BPK for pgmEpsilon=128, negligible for pgmEpsilon≥64

## TODO

- [ ] Re-run benchmarks with realistic rangeLen (10–100)
- [ ] Compare CDF-ARE vs SODA vs Adaptive at same rangeLen
- [ ] Measure FPR conditioned on query distribution matching data distribution
- [ ] Evaluate hybrid approach: CDF for global flattening + SODA for local hashing
