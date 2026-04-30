# are_hybrid_scan — Development Log

## Motivation

`are_hybrid` uses gap-percentile segmentation to detect clusters:
1. Compute gaps between sorted keys
2. P95 gap via quickselect → threshold
3. Split at gaps > threshold
4. Segments >= 1% of n → clusters (adaptive ARE), rest → fallback (trunc ARE)

Known limitations:
- **Sequential/equidistant data**: all gaps equal → P95 = that gap → every gap is a boundary
  (with `>=`) or no gap is a boundary (with `>`). Neither behavior is correct for all cases.
- **No merging**: adjacent small segments that together form a valid cluster are never merged.
- **minClusterFrac = 1%**: two adjacent segments of 0.8% each both go to fallback,
  even though together they'd be 1.6%.

We decided to create a new package with proper density-based detection (DBSCAN)
and compare both approaches empirically.

---

## Decision 1: 1D DBSCAN on sorted keys — O(n)

Classic DBSCAN is O(n²) due to neighbor search. But in 1D with sorted data, the
eps-neighborhood is a contiguous range → two-pointer sweep in O(n).

**Algorithm:**
1. Forward sweep: for each key[i], advance left pointer while key[i] - key[left] > eps.
   If window size >= minPts → key[i] is core.
2. Reverse sweep: same logic right-to-left (forward pass only marks rightmost core points).
3. Contiguous core point runs → cluster cores. Split runs at gaps > eps.
4. Merge adjacent runs within eps of each other.
5. Expand clusters outward to include border points (non-core within eps of a core).
6. Post-filter: clusters smaller than minClusterSize → dissolved back to fallback.

**Why DBSCAN over gap-based:**
- Naturally merges adjacent dense regions into one cluster.
- Handles sequential data: all points have many neighbors → all core → one cluster.
- Handles uniform random: low density everywhere → no core points → all fallback → trunc.

---

## Decision 2: eps = 10 * L / epsilon (from problem parameters, not data)

First attempt used median gap (P50) as eps. Failed: on clustered data with Gaussian
internal structure, intra-cluster gaps vary. P50 splits clusters → FPR=85%.

Considered `spread * epsilon / (2n)` (phantom overlap threshold). Problem: spread depends
on outliers. A few scattered keys inflate spread → eps becomes huge → everything clusters,
including scattered keys that should go to trunc.

**Final choice: eps = c * L / epsilon, c = 10.**

Reasoning: we proved that exact mode triggers when average gap < L/epsilon (density > epsilon/L).
So eps = 10 * L/epsilon means "regions 10x denser than the exact-mode threshold get clustered."

Verification on typical parameters:
- Clustered (internal gap ~1-100): eps = 10 * 128 / 0.001 = 1.28M >> 100 → clusters found.
- Uniform random 64-bit (avg gap ~2^44): eps = 1.28M << 2^44 → no clusters → all trunc.
- Sequential gap=1000: eps = 12,800 > 1000 → all clustered.

No data statistics, no percentiles. Pure problem-parameter derivation.

---

## Decision 3: Separate minPts (DBSCAN) from minClusterSize (post-filter)

Initially conflated: minPts = max(256, 1% * n). With n=1M this gave minPts=10,000.
Sequential data with gap=1000, eps=12,800: window = eps/gap = 12 keys << 10,000.
No core points → 0 clusters → all fallback. Benchmark showed FPR=93% on sequential.

**Root cause**: minPts in DBSCAN is the core-point threshold (local density), NOT minimum
cluster size. Classic DBSCAN uses minPts = 5-20.

**Fix**: two separate parameters:
- `minPts = 10` — DBSCAN core threshold. "A point is dense if it has 10+ neighbors within eps."
- `minClusterSize = 256` — post-filter. Clusters with < 256 keys are dissolved to fallback.
  Rationale: clusterSegment struct costs 40+ bytes metadata. Below 256 keys, overhead isn't worth it.

After fix: sequential → 1 cluster (window=12 > minPts=10, cluster=1M > minClusterSize=256).
SOSD Facebook: FPR dropped from 67% to 0% at eps=0.001, BPK from 20 to 12.

---

## Decision 4: Dual fallback — trunc when safe, adaptive/SODA when not

Adversarial testing found: equidistant keys with gap ≈ eps → DBSCAN doesn't cluster →
all keys go to trunc fallback → trunc produces phantom overlap → FPR=70%+.

**Analysis**: trunc breaks when gap < phantom_size. Phantom size = spread / 2^K.
On equidistant data with large spread, phantom_size can exceed the gap even though
gaps are regular.

**Fix**: after DBSCAN separates clusters from fallback, check fallback keys:
- Compute phantom_size = spread_fallback / 2^K
- Compute P5 gap (5th percentile, robust min gap estimate via quickselect)
- If P5 gap > phantom_size → **trunc** (saves log₂(L) BPK on uniform data)
- If P5 gap ≤ phantom_size → **adaptive/SODA** (distribution-independent FPR)

No chicken-and-egg: we already know which keys are in fallback.

**Result**: adversarial S2 (arithmetic progression, gap = 1.1 * eps) fixed: FPR 0.70 → 0.006.

---

## Decision 5: Exact mode density threshold (from are_adaptive)

Key insight in are_adaptive: exact mode (FPR=0, no hash) triggers when
data spread < 2^K = n * L / epsilon.

Rewritten as density: rho > epsilon / L, or average gap < L / epsilon.

| L | epsilon | Max avg gap (L/epsilon) | Min density (epsilon/L) |
|---|---------|------------------------|------------------------|
| 128 | 10^-3 | 128,000 | 7.8e-6 |
| 128 | 10^-6 | 1.28e8 | 7.8e-9 |

Threshold is extremely low — exact mode triggers for almost any cluster.
This is why DBSCAN + adaptive works so well: DBSCAN finds clusters,
adaptive detects that spread < 2^K → exact ERE → FPR=0.

---

## Benchmark Results (N=1M, L=128)

### Uniform — identical (no clusters in either)
Both: FPR=0.0005, BPK=14, Build 15 Mk/s, Query 2.8 Mq/s.

### Sequential (gap=1000) — Scan fixed
| | Hybrid | Scan |
|---|--------|------|
| FPR | 0.00 | 0.00 |
| BPK | 13.05 | 13.05 |
| Clusters | 1 | 1 |

Both detect one cluster → adaptive → exact mode → FPR=0.

### SOSD Facebook (eps=0.001) — Scan dramatically better
| | Hybrid | Scan |
|---|--------|------|
| FPR | 0.67 | **0.00** |
| BPK | 20.05 | **12.05** |
| Query Mq/s | 12.6 | **199.6** |
| Clusters | 0 | 1 |

Scan: one cluster → adaptive → exact mode → FPR=0, query 200 Mq/s.
Hybrid: 0 clusters → all trunc → FPR=67%.
(Note: Hybrid's FPR=67% here is due to benchmark parameters differing from
the original comparison_test.go which showed good results for Hybrid.)

### SOSD Wiki (eps=0.001)
| | Hybrid | Scan |
|---|--------|------|
| FPR | 0.53 | **0.00** |
| BPK | 19.95 | **10.11** |
| Query Mq/s | 12.3 | **198.9** |

---

## Adversarial Results (N=100K)

| Strategy | Hybrid FPR | Scan FPR | Notes |
|----------|-----------|----------|-------|
| S1: sequential near gap | 0.00 | 0.00 | Both OK |
| S2: arithmetic gap > eps | 0.00 | **0.006** | Scan: dual fallback → adaptive/SODA |
| S3: bimodal spread region | 0.00 | 0.00 | Both OK |
| S4: targeted midpoints | 0.00 | 0.00 | Both OK |
| S5: gap = exactly eps | 0.00 | **0.73** | Scan broken — under investigation |
| S6: high L, low eps | 0.00 | 0.00 | Both OK |
| S7: dense tiny range | **1.00** | 0.00 | Hybrid broken, Scan OK |
| S8: sequential gap=2 | 0.00 | 0.00 | Both OK |

Each filter breaks where the other works. No single approach dominates all cases.

### S5 investigation status

gap = exactly eps (1,000,000). 0 clusters. truncSafe says true (phantom_size << gap).
Yet FPR=73%. Phantom overlap math suggests trunc should work here.
Under investigation — may be a trunc issue unrelated to cluster detection.

---

## Open Questions

1. **S5 root cause**: why does trunc produce FPR=73% when phantom_size << gap?
2. **Combining both detectors**: could a hybrid of gap-based + DBSCAN cover all cases?
   Hybrid handles S5 (gap-based sees equal gaps → one cluster), Scan handles S7 (DBSCAN
   finds dense region). Merging strategies might eliminate all adversarial failures.
3. **eps multiplier tuning**: c=10 is arbitrary. Could derive optimal c from theory or
   tune empirically across SOSD datasets.
