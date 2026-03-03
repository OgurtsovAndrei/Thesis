# Final Refactoring Performance Report: Data Structures (Trie/MPHF/Locators)

**Date:** March 3, 2026  
**Refactoring Hash:** `f0008e80`  
**Hardware:** Intel i9-13900 (P-cores 0-7), 64 GB RAM, Ubuntu Linux.  
**Methodology:** Isolated runs on P-cores via `taskset`, minimum benchmark time: 3 seconds (`-benchtime=3s`).

---

## 1. Executive Summary

The final iteration (`f0008e80`) solidifies the performance gains of the architectural rework. While raw latency for
some locators increased slightly compared to the intermediate `a64c429e` version, it remains drastically faster than the
original baseline while achieving even better performance in core Trie operations.

* **Z-Fast Trie:** Search operations (Hit) are now **17x faster** than the original.
* **Memory Stability:** Allocs/op have been cut by **50-60%** and remain consistent across scaling.
* **Scalability:** Performance remains stable even as key sizes increase to 4096 bits.

---

## 2. Comprehensive lerloc Statistics (Local Exact Range Locator)

The following table shows the full performance profile for `lerloc` in **Compact Mode** across varying key sizes ($L$)
and key counts ($N$).

| Key Size ($L$) | Keys ($N$) | Original (ns/op) | **Newest (ns/op)** | Speedup | Allocs (Old/New) |
|:---------------|:-----------|:-----------------|:-------------------|:--------|:-----------------|
| 64             | 1,024      | 7,120            | **2,662**          | 2.7x    | 42 / 20          |
| 64             | 8,192      | 7,450            | **2,503**          | 3.0x    | 44 / 21          |
| 64             | 262,144    | 8,910            | **2,912**          | 3.1x    | 55 / 24          |
| 128            | 8,192      | 8,100            | **3,006**          | 2.7x    | 48 / 23          |
| 256            | 32,768     | 8,850            | **3,470**          | 2.5x    | 54 / 22          |
| 1024           | 262,144    | 12,400           | **4,374**          | 2.8x    | 72 / 26          |
| 4096           | 1,024      | 8,840            | **4,409**          | 2.0x    | 44 / 21          |
| 4096           | 8,192      | 9,701            | **5,012**          | 1.9x    | 47 / 22          |
| 4096           | 32,768     | 9,480            | **5,159**          | 1.8x    | 53 / 22          |
| 4096           | 262,144    | 10,630           | **5,326**          | 2.0x    | 65 / 26          |

---

## 3. Comparative Results (Other Structures)

### 3.1. Range Locator (rloc)

| Metric (N=8192, L=64) | Original (`summary`) | **Newest (`f0008e80`)** | Diff (%) | Speedup   |
|:----------------------|:---------------------|:------------------------|:---------|:----------|
| **Query Time**        | 5,734.0 ns           | **1,850.0 ns**          | -67.7%   | **~3.1x** |
| **Build Time**        | 37.9M ns             | **8.1M ns**             | -78.4%   | **~4.6x** |
| **Allocs/op (Build)** | 280,056              | **105,852**             | -62.2%   | **~2.6x** |

### 3.2. Z-Fast Trie (zft)

| Operation (100k keys) | Original (`summary`) | **Newest (`f0008e80`)** | Diff (%) | Speedup    |
|:----------------------|:---------------------|:------------------------|:---------|:-----------|
| **Contains (Hit)**    | 1,040.0 ns           | **60.5 ns**             | -94.2%   | **~17.2x** |
| **Contains (Miss)**   | 4,192.0 ns           | **2,150.0 ns**          | -48.7%   | **~1.9x**  |
| **Insert/op**         | 8,893.0 ns           | **2,342.0 ns**          | -73.6%   | **~3.8x**  |

### 3.3. MMPH / Relative Trie

| Test (N=1000)         | Original (`summary`) | **Newest (`f0008e80`)** | Diff (%) | Speedup   |
|:----------------------|:---------------------|:------------------------|:---------|:----------|
| **Lookup Time**       | 2,367.0 ns           | **429.0 ns**            | -81.8%   | **~5.5x** |
| **Allocs/op**         | 14.0                 | **7.0**                 | -50.0%   | **2.0x**  |
| **Trie Rebuild (1k)** | 1,912,042 ns         | **443,172 ns**          | -76.8%   | **~4.3x** |
