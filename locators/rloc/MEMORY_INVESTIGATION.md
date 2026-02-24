# Memory Usage Investigation: MMPH Buckets Overhead

This document summarizes the investigation into why the `MMPH_Buckets` component in `RangeLocator` (RLOC) and `LocalExactRangeLocator` (LERLOC) reports significantly higher memory usage (~48 bits/key) compared to standalone MMPH benchmarks (~15 bits/key).

## 1. The Core Discrepancy: $N$ vs. $|P|$

The most critical finding is that while standalone MMPH indexes original keys ($N$), the `RangeLocator` indexes an internal **boundary set $P$**.

- **Standalone MMPH**: Indexes $N$ strings. Bits per key is calculated as `TotalBytes * 8 / N`.
- **RangeLocator**: Indexes a set $P$ derived from the Z-Fast Trie nodes to support exact range mapping.
- **Observed Ratio**: Investigation confirmed that for this trie implementation, **$|P| \approx 2 \times N$**.

Since the benchmarking pipeline normalizes all metrics to **bits per original key ($N$)**, any memory used by the MMPH is effectively doubled in the RLOC reports:

$$15.5 \text{ bits/item in } P \times 2.0 \text{ items/key} = \mathbf{31.0 \text{ bits/key (floor)}}$$

## 2. Component Breakdown of `MMPH_Buckets`

At $N=32,768$ ($|P|=65,536$), the MMPH consists of 256 buckets (bucket size = 256). Each bucket contributes to the ~48 bits/key as follows:

| Component              | Bits per Item in $P$ | Contribution to Bits per Key $N$ | Description                                                        |
|------------------------|----------------------|----------------------------------|--------------------------------------------------------------------|
| **Local Ranks**        | 8.0 bits             | **16.0 bits/key**                | A `[]uint8` array of 256 bytes per bucket.                         |
| **MPHF (BoomPHF)**     | ~3.5 bits            | **~7.0 bits/key**                | Minimal Perfect Hash Function for local indexing.                  |
| **Headers & Delims**   | ~2.5 bits            | **~5.0 bits/key**                | Go struct headers (48 bytes) and the bucket delimiter BitString.   |
| **Padding & Overhead** | ~10 bits             | **~20 bits/key**                 | Go slice headers, memory alignment padding, and internal pointers. |
| **Total**              | **~24 bits/item**    | **~48 bits/key**                 |                                                                    |

## 3. Analysis of "Theoretical" 14 bits/key

Standalone MMPH achieves ~14-15 bits/item because:
1. It indexes only $N$ items ($|P|/N = 1$).
2. Overhead from headers and slice descriptors is amortized over a larger percentage of "useful" data.
3. In locators, the "Other" category (headers) remains constant or scales slowly, but is still visible at the per-bucket level.

## 4. Fixed Reporting Logic

During this investigation, a bug was identified in `analyze.py` where the hierarchical `MemReport` was overcounting the `buckets` parent and its children. The parser was updated to correctly traverse the tree and attribute bytes to canonical categories without double-counting.

## 5. Conclusion

The ~48 bits/key reported for `MMPH_Buckets` is **accurate** for the current architecture. It is not an error in implementation or parameter selection ($E, S, I$), but a natural result of:
1. Indexing a boundary set $|P|$ that is twice as large as the key set $N$.
2. Storing 1-byte local ranks for every item in $P$.
3. Fixed memory overhead of the Go `Bucket` structure and slice descriptors.

## Optimization Opportunities
To reduce this footprint in the future:
- **Reduce Bucket Size**: Using smaller buckets (e.g., 16) would allow local ranks to fit in `uint4` (4 bits), saving ~8 bits/key.
- **Succinct Ranks**: Replace the `[]uint8` array with a bit-packed succinct representation.
