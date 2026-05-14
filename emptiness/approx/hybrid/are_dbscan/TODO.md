# TODO — are_hybrid_scan

## truncSafe is too conservative

On uniform/OSM data, `truncSafe` returns false for moderate K values because
`phantomSize = spread >> K` exceeds P5 gap. This forces SODA fallback (+log₂(L) BPK),
even though trunc works fine on these distributions.

Result: Scan-ARE has a plateau near FPR=1 at low BPK, while Hybrid (always-trunc)
drops immediately.

### Options

1. **Relax threshold**: `p5Gap > phantomSize / c` with tunable c (e.g. 10).
   Easy to experiment with different constants.

2. **Remove truncSafe entirely**: always use trunc (like Hybrid).
   Adversarial S2 worst case: FPR=0.006 — acceptable for 0.01 target.

3. **Direct FPR estimate**: instead of gap proxy, compute
   `fpr ≈ n * (L + 2^(M-K)) / 2^M`. Use SODA only when estimated fpr > ε.
   More principled than P5 gap heuristic.

### Target

Beat SNARF on FPR/BPK tradeoff across all distributions.
Industry target FPR: 0.01 (RocksDB standard), 0.001 for premium workloads.

## eps multiplier tuning

Current `c = 10` in `eps = c * L / ε` is arbitrary. Could derive optimal c from
theory or tune empirically across SOSD datasets.
