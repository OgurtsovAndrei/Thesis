# Future Research: Distribution-Aware Range Emptiness (DARE)

## 1. The Core Idea: CDF-Mapping
Current Range Emptiness filters (SODA, Truncation) assume a relatively uniform distribution of keys or use randomized hashing to achieve it. However, real-world data is often skewed (Normal, Power-law, or Clustered distributions).

The proposed optimization is to transform the input space $X$ into a uniform space $Y \in [0, 1]$ using a Cumulative Distribution Function (CDF):
$$y = F(x) = P(X \le x)$$

Since $F(x)$ is a monotonically increasing function, it **preserves the order** of keys, which is critical for range queries:
$$x_1 < x_2 \implies F(x_1) < F(x_2)$$

## 2. Advantages
- **Optimal Entropy**: Mapping to a uniform distribution ensures that every bit of the hashed universe ($2^K$) carries the same amount of information.
- **Consistent FPR**: Eliminates "hotspots" where high-density clusters in the original space cause localized FPR spikes.
- **Space Efficiency**: By flattening the distribution, we can potentially use a smaller $K$ to achieve the same target $\epsilon$.

## 3. Implementation Paths
- **Parametric (Log/Power)**: For known skews (e.g., timestamps often follow predictable patterns), a simple logarithmic or power-law transform is computationally cheap.
- **Piecewise-Linear (PL)**: Approximating the CDF with a small set of linear segments (splines). This is a "mid-way" between SODA and full Learned Indexes.
- **Learned Models**: Using a small neural network or a compact decision tree to approximate $F(x)$.

## 4. Challenges
- **Model Storage**: The parameters of $F(x)$ must be stored alongside the filter. The bits saved in the filter must outweigh the bits spent on the model.
- **Query Latency**: Computing $F(x)$ for every `IsEmpty(a, b)` call adds overhead compared to simple bitwise operations.
- **Data Drift**: If the distribution of incoming keys changes, the model becomes suboptimal, potentially increasing the False Positive Rate.

## 5. Relationship to SODA Hash
SODA Hash can be viewed as a **stochastic piecewise-constant** approximation of a distribution. Instead of learning the "true" shape of data, it breaks the universe into blocks and applies random shifts to destroy any structured regularities. A true Distribution-Aware approach would be the deterministic, "learned" evolution of this idea.
