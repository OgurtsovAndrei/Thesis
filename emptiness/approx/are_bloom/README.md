# ARE — Bloom Filter Baseline

A trivial range filter built on top of a standard
[Bloom filter](https://en.wikipedia.org/wiki/Bloom_filter).

This package exists as a **baseline** for benchmarking — it is the simplest possible
approach to approximate range emptiness and is not space-competitive with the other
ARE implementations.

## Complexity

- **Query time:** $O(\mathcal{L})$ — one Bloom lookup per point in the range.
- **Space:** standard Bloom filter sizing: $\approx -n \ln\varepsilon / \ln^2 2$ bits
  for per-point FPR $\varepsilon$.
- **FPR:** per-query FPR grows with range length since each point is an independent
  trial. For per-point FPR $p$, the probability of a false positive on a truly empty
  range of length $\mathcal{L}$ is $1 - (1-p)^\mathcal{L} \approx p\mathcal{L}$
  for small $p$.

## Limitations

- Query cost is linear in $\mathcal{L}$ — all other ARE implementations are $O(1)$ or $O(\log n)$.
- Not applicable when keys are not integers or when $\mathcal{L}$ is large.
- Exists purely as a performance and space baseline for comparison.
