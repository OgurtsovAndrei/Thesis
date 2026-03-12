# ARE Robust (SODA Hash): FPR vs. Space Trade-off

This report analyzes the memory-to-accuracy tradeoff for the SODA 2015 locality-preserving hash implementation. Unlike the basic truncation ARE, this version provides guarantees for ranges up to length $L$.

## 1. Methodology
- **Range Length ($L$):** Fixed at **100**.
- **Dataset Size ($N$):** 100,000 to 200,000 keys.
- **Prefix Length ($K$):** 20 to 32 bits.
- **Query Load:** **1,000,000** random interval queries (length $\le L$) per data point.

## 2. Results Summary (L=100)

| Empirical FPR | Required Space (Robust) | Comparison to Fast ARE |
| :--- | :--- | :--- |
| **1%** (0.01) | **~15.3 bits/key** | +12.8 bits |
| **0.1%** (0.001) | **~18.3 bits/key** | +11.8 bits |

### Critical Observation: The Cost of Interval Guarantees
The significant increase in space (from ~6.5 to ~18.3 bits for 0.1% FPR) is the "theoretical tax" for interval safety. 
- **6.6 bits** come directly from the $\log_2(L)$ factor in the universe reduction formula $r = nL/\epsilon$.
- **~5 bits** are required to compensate for the Union Bound effect across 100 points in the query range and the dual-ERE query overhead.

## 3. Visual Representation (Calculated Points)

| Bits/Key | Observed FPR (L=100) |
| :--- | :--- |
| 6.07 | 90.0% |
| 10.28 | 24.9% |
| 14.31 | 1.85% |
| 16.31 | 0.48% |
| 18.31 | 0.11% |

## 4. Conclusion
ARE Robust is designed for scenarios where range boundaries are critical (sequential keys, LSM-tree iterators). While it requires **~3x more space** than the point-optimized Fast ARE to maintain the same FPR over $L=100$ intervals, it is the only variant that remains reliable under data skew.
