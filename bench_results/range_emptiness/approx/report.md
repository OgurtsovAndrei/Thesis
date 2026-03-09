# Approximate Range Emptiness: Large Grid Report

## 1. Summary Statistics (epsilon = 0.001)

### Dataset: N = 262,144

| Key Size (L) | Query Time | Build Time | Throughput (Keys/sec) | Bits/Key |
| :--- | :--- | :--- | :--- | :--- |
| 64 bits | 142.6 ns | 40.6 ms | 6.46 M | 14.21 |
| 128 bits | 139.5 ns | 41.7 ms | 6.29 M | 14.21 |
| 256 bits | 144.1 ns | 42.1 ms | 6.23 M | 14.21 |
| 512 bits | 139.5 ns | 45.8 ms | 5.72 M | 14.21 |
| 1024 bits | 144.5 ns | 47.9 ms | 5.47 M | 14.21 |


### Dataset: N = 1,048,576

| Key Size (L) | Query Time | Build Time | Throughput (Keys/sec) | Bits/Key |
| :--- | :--- | :--- | :--- | :--- |
| 64 bits | 148.3 ns | 176.1 ms | 5.95 M | 14.21 |
| 128 bits | 147.7 ns | 182.9 ms | 5.73 M | 14.21 |
| 256 bits | 142.9 ns | 189.6 ms | 5.53 M | 14.21 |
| 512 bits | 148.2 ns | 184.0 ms | 5.70 M | 14.21 |
| 1024 bits | 147.4 ns | 181.7 ms | 5.77 M | 14.21 |


### Dataset: N = 4,194,304

| Key Size (L) | Query Time | Build Time | Throughput (Keys/sec) | Bits/Key |
| :--- | :--- | :--- | :--- | :--- |
| 64 bits | 155.8 ns | 750.4 ms | 5.59 M | 14.21 |
| 128 bits | 165.0 ns | 743.2 ms | 5.64 M | 14.21 |
| 256 bits | 158.1 ns | 754.1 ms | 5.56 M | 14.21 |
| 512 bits | 156.7 ns | 827.9 ms | 5.07 M | 14.21 |
| 1024 bits | 156.8 ns | 1116.0 ms | 3.76 M | 14.21 |


### Dataset: N = 16,777,216

| Key Size (L) | Query Time | Build Time | Throughput (Keys/sec) | Bits/Key |
| :--- | :--- | :--- | :--- | :--- |
| 64 bits | 164.4 ns | 3002.8 ms | 5.59 M | 14.21 |
| 128 bits | 170.6 ns | 3124.3 ms | 5.37 M | 14.21 |
| 256 bits | 164.6 ns | 4302.0 ms | 3.90 M | 14.21 |
| 512 bits | 165.3 ns | 3274.8 ms | 5.12 M | 14.21 |
| 1024 bits | 169.7 ns | 3688.9 ms | 4.55 M | 14.21 |


## 2. Visualizations

- [Query Latency Plot](are_large_query_latency.svg)
- [Space Efficiency Plot](are_large_bits_per_key.svg)
- [Build Throughput Plot](are_large_build_throughput.svg)

## 3. Observations

- **Constant Space**: The `Bits/Key` is remarkably stable at ~14.2 regardless of both $N$ and $L$.
- **Constant Time Query**: Query latency scales slightly with $N$ due to CPU cache effects but remains within 140-170ns range.
- **Throughput**: Build throughput is consistently in the range of 4-7 Million keys/sec.
