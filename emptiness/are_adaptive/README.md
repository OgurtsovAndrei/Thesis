# ARE — Adaptive

Adapts to data spread at build time: exact ERE when data is compact, SODA hash when it isn't.

## Core Idea: Skip the Hash When You Can

Both [SODA](../are_soda_hash/) and [Truncation](../are_trunc/) always hash — even when the data
is compact enough to fit in the target universe without any information loss.

The adaptive filter checks first:

1. **Normalize:** subtract $\min(S)$ from all keys, shifting the dataset to start at 0.
2. **Measure spread:** $M = \lceil \log_2(\max(S) - \min(S)) \rceil$ — the number of bits needed
   to represent the normalized data.
3. **Adapt:**
   - $M \leq K$: the entire dataset fits in $K$ bits → **exact mode**.
     Build ERE directly over the normalized keys. No hash, **FPR = 0**.
   - $M > K$: data too spread for $K$ bits → **SODA mode**.
     Apply pairwise-independent hash, same as [`are_soda_hash`](../are_soda_hash/).

The decision is automatic at build time. The caller just sets $\varepsilon$ and $\mathcal{L}$;
the filter computes $K = \lceil \log_2(n\mathcal{L}/\varepsilon) \rceil$ and checks if $M \leq K$.

## When Does Exact Mode Trigger?

Exact mode requires $\max(S) - \min(S) < 2^K = n\mathcal{L}/\varepsilon$.

For example, with $n = 2^{18}$, $\mathcal{L} = 128$, $\varepsilon = 10^{-3}$:
$K = \lceil \log_2(2^{18} \cdot 128 / 10^{-3}) \rceil = \lceil \log_2(2^{35}) \rceil = 35$.
Any dataset with spread $< 2^{35} \approx 34 \cdot 10^9$ gets exact mode for free.

Rewriting the condition in terms of density $\rho = n / (\max - \min)$:

$$\rho > \frac{\varepsilon}{\mathcal{L}}$$

or equivalently, the average gap between consecutive keys must satisfy $\bar{g} < \mathcal{L}/\varepsilon$.
The universe can have up to $\mathcal{L}/\varepsilon$ empty slots per stored key and exact mode still triggers.

| $\mathcal{L}$ | $\varepsilon$ | Max avg gap ($\mathcal{L}/\varepsilon$) | Min density ($\varepsilon/\mathcal{L}$) |
|---|---|---|---|
| 128 | $10^{-3}$ | 128,000 | $7.8 \times 10^{-6}$ |
| 128 | $10^{-6}$ | $1.28 \times 10^{8}$ | $7.8 \times 10^{-9}$ |
| 16  | $10^{-3}$ | 16,000 | $6.25 \times 10^{-5}$ |

The threshold is extremely low. For typical parameters, exact mode fails only when keys are
scattered across a universe much wider than $n \cdot \mathcal{L} / \varepsilon$ — e.g. uniform
random 64-bit keys.

This becomes especially powerful in combination with [`are_hybrid`](../are_hybrid/), which splits
data into dense clusters and builds a separate adaptive filter per cluster. Each cluster has a small
spread → exact mode triggers per-segment, giving 0% FPR on the dense parts of the data.

## Fallback: SODA Mode

When $M > K$, the filter falls back to the [SODA hash](../are_soda_hash/) over the normalized keys.
All SODA guarantees apply: FPR $\leq \varepsilon$ for any distribution,
BPK $= \log_2(\mathcal{L}/\varepsilon) + O(1)$.

Normalization (subtracting $\min$) doesn't affect SODA's guarantees — it's just a shift of the
universe, and the pairwise-independent hash is applied on top.

## Implementation (see [are_adaptive.go](are_adaptive.go))

1. Find $\min(S)$, $\max(S)$. Normalize: $x' = x - \min$.
2. Compute $M = \lceil \log_2(\max') \rceil$.
3. If $M \leq K$: build ERE over $[0, 2^M)$ — exact mode.
4. If $M > K$: apply SODA hash to $x'$, build ERE over $[0, 2^K)$ — approximate mode.
5. Query: normalize endpoints, dispatch to exact ERE or `sodaIsEmpty`.
