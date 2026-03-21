# RSDic Fork

Forked from [github.com/hillbig/rsdic](https://github.com/hillbig/rsdic) (MIT License, Daisuke Okanohara).

## Changes from upstream

### Dependency cleanup (no perf impact)

- Removed `github.com/ugorji/go/codec` — `MarshalBinary`/`UnmarshalBinary` reimplemented with `encoding/binary`
- Removed `github.com/smartystreets/goconvey` — tests rewritten with standard `testing`

### Optimization 1: `popCount` → hardware POPCNT (`util.go`)

**Before (upstream):**

```go
func popCount(x uint64) uint8 {
x = x - ((x & 0xAAAAAAAAAAAAAAAA) >> 1)
x = (x & 0x3333333333333333) + ((x >> 2) & 0x3333333333333333)
x = (x + (x >> 4)) & 0x0F0F0F0F0F0F0F0F
return uint8(x * 0x0101010101010101 >> 56)
}
```

Software bit-parallel popcount: 7 bitwise operations + 1 multiply + 1 shift = **12 instructions**.
The Go compiler does NOT auto-vectorize this into a POPCNT instruction.

**After:**

```go
func popCount(x uint64) uint8 {
return uint8(bits.OnesCount64(x))
}
```

`math/bits.OnesCount64` compiles to a single hardware `CNT` instruction on ARM64 (Apple M-series)
and `POPCNT` on x86-64. **1 instruction** vs 12.

**Where it's called in hot path:**

- `Rank()` line 137: `popCount(rs.lastBlock >> (pos % kSmallBlockSize))` — afterRank for last block
- `enumRank()` raw path (line 76): `popCount(code & ((1 << pos) - 1))` — rank within decoded block
- Every Rank query on dense data (50%/33%) goes through this path

**Measured impact:** Rank ~5% faster (37.4 → 35.3 ns at 50% density). Modest because popCount
is one step among several (block lookup, pointer accumulation, getSlice).

### Optimization 2: `selectRaw` → clear-lowest-bit + TrailingZeros (`enumCode.go`)

**Before (upstream):**

```go
func selectRaw(code uint64, rank uint8) uint8 {
for i := uint8(0); i < kSmallBlockSize; i++ {
if getBit(code, i) {
rank--
if rank == 0 {
return i
}
}
}
return 0
}
```

Scans **every bit position** from 0 to 63, calling `getBit(code, i)` which does `(code >> i) & 1`.
For rank=32 (average case at 50% density), this executes ~63 iterations with a branch per iteration.
The branch predictor struggles because bit values are essentially random.

**After:**

```go
func selectRaw(code uint64, rank uint8) uint8 {
for i := uint8(1); i < rank; i++ {
code &= code - 1 // clear lowest set bit
}
return uint8(bits.TrailingZeros64(code))
}
```

Two key improvements:

1. **`code &= code - 1`**: Clears the lowest set bit in one cycle. This is a well-known bit trick —
   `code - 1` flips all bits from the lowest set bit downward, and AND-ing removes exactly that bit.
   We execute this `rank - 1` times to skip to the rank-th set bit. Each iteration is
   **1 subtract + 1 AND** = 2 instructions with no branch.

2. **`bits.TrailingZeros64(code)`**: Returns the position of the lowest remaining set bit.
   Compiles to a single `RBIT + CLZ` on ARM64 (or `TZCNT`/`BSF` on x86-64).
   **1-2 instructions** vs the original's branch-heavy scan.

For rank=32: old = ~63 iterations × (shift + AND + branch) = ~190 instructions.
New = 31 iterations × (SUB + AND) + 1 CTZ = ~64 instructions, all branchless except the loop counter.

**Where it's called in hot path:**

- `Select1()` → `enumSelect1()` → `selectRaw()` — every Select query on dense data
- `Select0()` → `enumSelect0()` → `selectRaw(^code, rank)` — same for zero-select
- ERE calls `D2.Select()` **twice** per query (in `getBlockRange` to locate block boundaries)

**Measured impact:** Select **2.3–2.5x faster**:

- Dense 50%: 161 → 64 ns
- D2 33%: 145 → 63 ns
- Sparse 1%: 173 → 164 ns (smaller gain — enum path dominates, selectRaw less relevant)

## Benchmark results

Apple M4 Max, Go 1.25, ARM64, GOMAXPROCS=1, 5 runs each.

### Baseline (before optimization, fork = upstream)

| Op     | 50% dense | 33% (D2) | 1% sparse |
|--------|-----------|----------|-----------|
| Bit    | 23.5 ns   | 24.4 ns  | 55.6 ns   |
| Rank   | 25.6 ns   | 27.6 ns  | 57.5 ns   |
| Select | 127 ns    | 116 ns   | 122 ns    |

### After optimization

| Op         | Density | Upstream (ns) | Optimized (ns) | Speedup  |
|------------|---------|---------------|----------------|----------|
| Bit        | 50%     | 33.5          | 33.2           | 1.0x     |
| Rank       | 50%     | 37.4          | 35.3           | 1.06x    |
| **Select** | **50%** | **161**       | **64**         | **2.5x** |
| Bit        | 33%     | 34.1          | 34.9           | 1.0x     |
| Rank       | 33%     | 37.2          | 36.3           | 1.02x    |
| **Select** | **33%** | **145**       | **63**         | **2.3x** |
| Bit        | 1%      | 76.9          | 78.7           | 1.0x     |
| Rank       | 1%      | 76.3          | 79.4           | 1.0x     |
| Select     | 1%      | 173           | 164            | 1.05x    |

Note: absolute numbers vary between benchmark sessions due to system load;
ratios (Orig/Fork within same session) are stable.

### Why Bit and Rank barely changed

At 50%/33% density, most 64-bit blocks have 15–49 ones, hitting the **raw path**
(`kEnumCodeLength[rankSB] == 64`). In the raw path:

- `Bit()` calls `getBit(code, pos)` = single shift+AND — already optimal
- `Rank()` calls `popCount(code & mask)` — popCount improved but it's one step
  among block lookup (pointer accumulation loop of up to 15 iterations) and getSlice

### Why Select improved dramatically

Select's hot path is `Select1()` → linear scan over large blocks → linear scan over small blocks →
`enumSelect1()` → `selectRaw()`. The `selectRaw` function was the **dominant cost** because
it scanned all 64 bit positions with unpredictable branches. The new implementation is
~3x fewer instructions and fully branchless (except the counted loop).

## ERE impact estimate

ERE `IsEmpty()` calls per query:

- `D1.Bit()` × 1-2 (block occupancy check)
- `D1.Rank()` × 1-3 (intermediate block count)
- `D2.Select()` × 2 (in `getBlockRange` — locating block start/end)

With D2.Select going from ~145 ns to ~63 ns, the two Select calls save ~164 ns per query.
For a typical ERE query at ~200 ns, this is a significant fraction.

## Remaining optimization opportunities (not yet implemented)

1. **Small-block pointer accumulation loop** in Rank/Bit: iterates up to 15 small blocks per
   large block. Could be eliminated by storing cumulative pointers per small block (+8 bytes/64 bits
   = 12.5% space overhead).

2. **Enum decode path** (sparse/dense blocks, rankSB outside 15-49): `enumRank`, `enumBit`,
   `enumSelect1` all do O(pos) sequential decode through combinatorial number system.
   Would require changing data layout to fix (e.g., always store raw 64-bit blocks).

3. **`runZerosRaw`**: same bit-by-bit scan pattern as old `selectRaw`. Could use
   `bits.TrailingZeros64(code >> pos)` for O(1).

## Density in ERE context

- **D1** (block occupancy): ~50% ones — half of 2^k blocks occupied (uniform random keys)
- **D2** (block sizes): ~33% ones — n/2 block delimiters among 3n/2 total bits

Both hit the raw fast path most of the time. Sparse enum path matters for clustered distributions.
