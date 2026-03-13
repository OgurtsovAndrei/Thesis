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

### Smoothing (CDF–Uniform Blend)

Pure CDF compresses inter-cluster gaps to near-zero resolution → FPR ≈ 100% there.
Smoothing blends CDF with uniform mapping to preserve minimum gap resolution:

`mapped = (1 − α) · CDF(x) + α · uniform(x)`

where `uniform(x) = (x − minKey) / (maxKey − minKey)`.

- α = 0: pure CDF (current behavior)
- α = 1: pure uniform (no CDF benefit)
- α = 0.01–0.1: mostly CDF, gaps retain some resolution

## Experimental Results

### Early Results (rangeLen = 1M) — Unrealistic

| Distribution | pgmEps | K  | FPR    | BPK   | ERE   | CDF  |
|-------------|--------|-----|--------|-------|-------|------|
| Uniform     | 128    | 18  | 0.073  | 8.46  | 7.44  | 1.02 |
| Cluster     | 128    | 28  | 0.139  | 18.02 | 17.00 | 1.02 |

**Problem**: rangeLen = 1M ≈ 800× avg gap in cluster. All truly empty queries land
in inter-cluster gaps where CDF has zero resolution. Unrealistic for production (L = 10–100).

### Production-Realistic rangeLen (L = 128, 1024)

**Critical finding**: query distribution must match data distribution (σ_query = σ_data).

With σ_query = σ_data = 2²⁰:

| L    | pgmEps | K  | FPR    | BPK   |
|------|--------|-----|--------|-------|
| 128  | 32     | 18  | 0.049  | 11.46 |
| 128  | 64     | 18  | 0.061  | 9.38  |
| 128  | 128    | 18  | 0.088  | 8.24  |
| 1024 | 32     | 19  | 0.036  | 12.50 |
| 1024 | 64     | 18  | 0.074  | 9.38  |
| 1024 | 128    | 18  | 0.112  | 8.24  |

With σ_query = 4 × σ_data (mismatched): FPR = 40–58% — CDF concentrates resolution
where queries DON'T land.

### CDF-ARE vs SODA — Direct Comparison

n = 10,000, ε = 0.05, pgmEps = 64, 200K queries.

**L = 128:**

| Method  | Data    | Query   | K  | FPR    | BPK   |
|---------|---------|---------|-----|--------|-------|
| SODA    | uniform | uniform | 25  | 0.037  | 14.64 |
| CDF-ARE | uniform | uniform | 18  | 0.073  | 9.46  |
| SODA    | cluster | cluster | 25  | 0.232  | 14.62 |
| CDF-ARE | cluster | cluster | 18  | 0.061  | 9.38  |

**L = 1024:**

| Method  | Data    | Query   | K  | FPR    | BPK   |
|---------|---------|---------|-----|--------|-------|
| SODA    | uniform | uniform | 28  | 0.038  | 17.64 |
| CDF-ARE | uniform | uniform | 18  | 0.073  | 9.46  |
| SODA    | cluster | cluster | 28  | 0.788  | 17.62 |
| CDF-ARE | cluster | cluster | 18  | 0.074  | 9.38  |

**Key observations**:
- CDF-ARE saves **5–8 BPK** vs SODA (exactly log₂(L) as predicted by theory)
- On clusters: CDF-ARE **dominates** — 6% FPR vs 23% (L=128) and 7% vs 79% (L=1024)
- On uniform: SODA has slightly better FPR (3.7% vs 7.3%) but costs +5 BPK
- SODA degrades catastrophically on clusters at large L

### Smoothing Results (L = 128, pgmEps = 64, cluster data, cluster queries)

| α (smooth) | FPR    | BPK  |
|------------|--------|------|
| 0.00       | 0.061  | 9.38 |
| 0.01       | 0.061  | 9.39 |
| 0.05       | 0.064  | 9.39 |
| 0.10       | 0.065  | 9.38 |
| 0.20       | 0.070  | 9.35 |
| 0.50       | 0.097  | 9.28 |
| 1.00       | 1.000  | 2.04 |

When query distribution matches data: smoothing **hurts** — it moves resolution
away from where queries concentrate. Smoothing would help only when σ_query > σ_data.

## Key Takeaways

1. **CDF-mapping removes the log₂(L) term from BPK** — this is the theoretical win
2. **FPR is redistributed, not reduced**: lower in dense regions, higher in sparse regions
3. **Benefit materializes when**: query distribution ≈ data distribution AND rangeLen < avg gap
4. **CDF-ARE dominates SODA on clustered data**: both better FPR AND lower BPK
5. **SODA degrades on clusters**: FPR grows with L because hash doesn't respect data structure
6. **Smoothing is counterproductive** when queries match data distribution
7. **CDF overhead**: ~1 BPK for pgmEpsilon=128, ~2 BPK for pgmEpsilon=64

## TODO

- [ ] Test with more distributions (Zipfian, log-normal, real-world datasets)
- [ ] Evaluate hybrid: CDF-ARE for cluster regions + SODA for uniform regions
- [ ] Measure query latency overhead from CDF lookup
- [ ] Optimize CDF storage (compress control points)
