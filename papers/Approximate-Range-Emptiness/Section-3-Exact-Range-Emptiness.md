# Section 3: Exact Range Emptiness (Succinct)

This section introduces a new succinct data structure for **exact** 1D range emptiness, which serves as a building block for the approximate structure.

## Key Features
- **Mechanism:** It uses a bit-vector representation combined with rank/select operations. It stores the elements of $S$ in a succinct way while still supporting constant-time queries.
- **Space Occupancy:** It occupies $n \log(U/n) + O(n)$ bits.
- **Performance:** It supports $O(1)$ time queries for exact range emptiness.
- **Significance:** This structure is of independent interest as it improves upon previous succinct range query structures by being simpler and more space-efficient while maintaining constant time.
- **Technique:** The structure works by dividing the universe into blocks and using a "summary structure" for large blocks and specific bit-packing for small ones.
