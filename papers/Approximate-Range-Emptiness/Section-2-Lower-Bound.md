# Section 2: Preliminaries and Lower Bound

This section establishes a theoretical lower bound for the $\epsilon$-approximate range emptiness problem.

## Theoretical Benchmarks
- **Space Complexity:** The authors prove that any data structure capable of answering approximate range emptiness queries for intervals of length up to $L$ with a false positive probability $\epsilon$ must use at least:
  $$\Omega(n \log(L/\epsilon)) - O(n) \text{ bits.}$$
- **Optimal Space:** This lower bound is independent of the universe size $U$ and provides a benchmark for the authors' proposed data structure.
- **Problem Statement:** The lower bound covers both the case where the range length is fixed ($L=U$) and the case where it's restricted ($L < U$).
