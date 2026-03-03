# Refactoring Performance Report: Data Structures (Trie/MPHF/Locators)

**Date:** March 3, 2026  
**Hardware:** Intel i9-13900 (P-cores 0-7), 64 GB RAM, Ubuntu Linux.  
**Methodology:** Isolated runs on P-cores via `taskset`, minimum benchmark time: 3 seconds (`-benchtime=3s`).

---

## 1. Executive Summary
The refactoring resulted in a significant performance boost across all metrics. Execution speed increased by **3–5x** on average, with heap allocations reduced by more than **2x**.

*   **Peak Speedup:** Up to **8.4x** in search operations (Z-Fast Trie).
*   **Memory Efficiency:** Allocation count reduced by up to **60%** in structure constructors.

---

## 2. Comparative Results

### 2.1. Local Exact Range Locator (lerloc)
| Metric (N=8192, L=4096) | OLD (`summary`) | NEW (`a64c429e`) | Diff (%) | Speedup |
| :--- | :--- | :--- | :--- | :--- |
| **Query Time (Compact)** | 9,701.0 ns | 3,295.0 ns | **-66.0%** | **~3x** |
| **Query Time (Fast)** | 7,850.0 ns | 2,150.0 ns | **-72.6%** | **~3.6x** |
| **Allocs/op** | 47.0 | 22.0 | **-53.2%** | **~2.1x** |
| **Build Time (Fast)** | 20.2M ns | 5.3M ns | **-73.6%** | **~3.8x** |

### 2.2. Range Locator (rloc)
| Metric (N=8192, L=64) | OLD (`summary`) | NEW (`a64c429e`) | Diff (%) | Speedup |
| :--- | :--- | :--- | :--- | :--- |
| **Query Time** | 5,734.0 ns | 1,273.0 ns | **-77.8%** | **~4.5x** |
| **Allocs/op (Build)** | 280,056 | 105,852 | **-62.2%** | **~2.6x** |
| **Build Time** | 37.9M ns | 8.1M ns | **-78.4%** | **~4.6x** |

### 2.3. Z-Fast Trie (zft)
| Operation (100k keys) | OLD (`summary`) | NEW (`a64c429e`) | Diff (%) | Speedup |
| :--- | :--- | :--- | :--- | :--- |
| **Contains (Hit)** | 1,040.0 ns | 124.1 ns | **-88.1%** | **~8.4x** |
| **Contains (Miss)** | 4,192.0 ns | 1,171.0 ns | **-72.1%** | **~3.6x** |
| **Insert/op** | 8,893.0 ns | 2,342.0 ns | **-73.6%** | **~3.8x** |

### 2.4. MMPH / Relative Trie
| Test (N=1000) | OLD (`summary`) | NEW (`a64c429e`) | Diff (%) | Speedup |
| :--- | :--- | :--- | :--- | :--- |
| **Lookup Time** | 2,367.0 ns | 360.1 ns | **-84.8%** | **~6.5x** |
| **Allocs/op** | 14.0 | 7.0 | **-50.0%** | **~2x** |
| **Trie Rebuild (1k)** | 1.9M ns | 0.4M ns | **-76.8%** | **~4.3x** |
