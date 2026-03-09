# Section 1: Introduction

This section defines the $\epsilon$-approximate range emptiness (ARE) problem as a generalization of Bloom filters. While Bloom filters handle point queries, ARE handles interval queries.

## Key Points
- **Problem Definition:** Given a set $S$ of $n$ points from a universe $[U]$, determine if an interval $[a, b]$ contains any points from $S$.
- **False Positives:** A false positive probability of at least $1-\epsilon$ is allowed (if the range is empty, the data structure can answer "non-empty" with probability $\epsilon$).
- **Previous Work:** Existing solutions were either space-optimal but slow ($O(\log \log U)$ or $O(\alpha(n))$ time), or fast but space-inefficient.
- **Contribution:** The first data structure to achieve both $O(1)$ query time and $O(n \log(L/\epsilon))$ bits of space, where $L$ is the maximum interval length.
