# Memory Usage Investigation: Heavy Z-Fast Trie (HZFT) and AZFT

This document summarizes the investigation into the memory consumption of the Trie structures used in the LERLOC and MMPH modules, specifically answering why `HZFastTrie` consumes ~64 bits/key.

## 1. Heavy Z-Fast Trie (HZFT) in LERLOC

The `HZFastTrie` is used in `LERLOC` as the top-level index for fast prefix searching. Benchmarks show it consumes ~64 bits/key for $N=32,768$ and $L=64$.

### Is it Overusage?
**No. The memory consumption is theoretically correct and algorithmically expected.**

Unlike standard trees where 1 node = 1 entry, the Hollow Z-Fast Trie algorithm explicitly generates multiple entries for each internal node. To support "Fat Binary Search", it stores:
1.  **Descriptors**: One for each internal node.
2.  **Pseudo-descriptors**: Up to $\log_2(L)$ additional prefixes per node mapped to $\infty$.

### Experimental Proof
A custom script (`investigate_hzft_lerloc.go`) measured the exact number of entries inserted into the `BoomPHF` for various key lengths ($L$):
- **$L=64$**: $\approx 5.5$ entries per key.
- **$L=256$**: $\approx 7.5$ entries per key.
- **$L=1024$**: $\approx 9.5$ entries per key.

The number of entries perfectly follows the $O(N \log L)$ bound described in `papers/Hollow-Z-Fast-Trie (Fast Prefix Search)/Section-3.md`.

### Component Breakdown ($L=64$)
- **Data Array**: Each entry is an `HNodeData[E]` containing only `extentLen` (e.g., `uint8` for $L=64$).
  - $5.5 	ext{ entries} 	imes 1 	ext{ byte} = 5.5 	ext{ bytes/key} = \mathbf{44 	ext{ bits/key}}$.
- **MPH (BoomPHF)**: ~3.5 bits per entry.
  - $5.5 	ext{ entries} 	imes 3.5 	ext{ bits} = \mathbf{\approx 19.25 	ext{ bits/key}}$.
- **Total**: $44 + 19.25 \approx \mathbf{63.25 	ext{ bits/key}}$.

*(Note: There is zero memory padding or alignment waste in the `HNodeData` struct itself).*

### The Paradox of Long Keys
While HZFT is highly memory efficient for short keys ($L=64, 256$), its $O(N \log L)$ nature means that for very long keys (e.g., $L=1024$), the number of pseudo-descriptors grows, and `E` requires larger types (`uint16`). For $L=1024$, HZFT consumption jumps to **~186 bits/key**. 

### Optimization Path
The original paper ("Section 3.2 Space and time") suggests storing descriptors in a relative dictionary to achieve $O(N \log \log L)$ bits. This would require replacing the simple MPH array with a complex relative data structure.

---

## 2. Approximate Z-Fast Trie (AZFT) in MMPH

The `ApproxZFastTrie` is used internally by `MonotoneHashWithTrie` to bucket keys. It consumes only **~3.0 bits/key** relative to the total dataset, but its per-node efficiency is suboptimal.

### Architectural Anomalies Discovered:
1.  **Memory Padding (Waste)**: The `NodeData[E, S, I]` struct is 20 bytes (for U8, U32, U16), but 5 bytes (25%) are empty padding injected by Go due to field alignment (`extentLen` and `PSig` placement).
2.  **Over-indexing**: The builder (`NewApproxZFastTrieFromIteratorStreaming`) inserts **all** nodes from the compacted trie into the MPH ($2N-1$ nodes). However, for routing, only internal nodes are strictly necessary.

### Why Not Fix It Now?
Because AZFT only indexes **bucket delimiters**, not the full key set $N$. With a bucket size of 256, the AZFT contains only $N/256$ entries. 
Even if we optimize its structure by 50% (fixing padding and removing leaves), we would only save **~1.5 bits/key** globally, making it a low-ROI micro-optimization compared to optimizing the `MMPH_Buckets` local ranks.
