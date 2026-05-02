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

| Op         | Density | Upstream (ns) | Optimized (ns) | Speedup   |
|------------|---------|---------------|----------------|-----------|
| Bit        | 50%     | 23.9          | 24.0           | 1.0x      |
| Rank       | 50%     | 26.7          | 25.2           | **1.06x** |
| **Select** | **50%** | **122**       | **47**         | **2.6x**  |
| Bit        | 33%     | 24.8          | 25.1           | 1.0x      |
| Rank       | 33%     | 26.9          | 26.5           | 1.02x     |
| **Select** | **33%** | **110**       | **47**         | **2.3x**  |
| Bit        | 1%      | 56.3          | 57.2           | 1.0x      |
| Rank       | 1%      | 56.8          | 56.2           | 1.0x      |
| Select     | 1%      | 124           | 122            | 1.02x     |

Measured in same session (Orig vs Fork side-by-side) to eliminate system load variance.

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

With D2.Select going from ~110 ns to ~47 ns, the two Select calls save ~126 ns per query.
For a typical ERE query at ~200 ns, this is a significant fraction.

### Optimization 4: `Select1` inner loop — adaptive linear/binary scan (`rsdic.go`)

**Before (upstream `Select1`, the inner loop):**

```go
selectInd := rank / kSelectBlockSize          // kSelectBlockSize = 4096
lblock := rs.selectOneInds[selectInd]
for ; lblock < uint64(len(rs.rankBlocks)); lblock++ {
    if rank < rs.rankBlocks[lblock] {
        break
    }
}
lblock--
```

`selectOneInds[selectInd]` is a hint pointing to the large block that
contains the $(\textsf{selectInd} \cdot 4096)$-th 1-bit. The loop walks
`rankBlocks` forward from that hint until the running 1-count first
exceeds `rank`. The implicit "O(1) iterations" promise assumes 1-bits
are spread roughly uniformly across the bitvector, so 4096 ones span
~8192 bits ≈ 8 large blocks.

**The pathology.** On clustered bitvectors — e.g. an encoding of the
form $0^{|B_0|} 1\, 0^{|B_1|} 1\, \dots\, 0^{|B_{N-1}|} 1$ where most
$|B_i|=0$ but a small fraction of $|B_i|$ are very large — 1-bits
cluster into long runs separated by long 0-runs. Two consecutive
select-hints (4096 ones apart) can then sit *millions of bits* away
from each other, so the inner scan visits thousands of large blocks
instead of a handful. Empirically: on a workload that produced one
hint-bracket spanning ~$10^6$ large blocks, the original `Select1`
measured **1825 ns/call** versus 67 ns on the same rsdic with
uniform-rank queries — a 27× slowdown driven entirely by the
linear scan.

A second, smaller contribution: `Select1` had a **value receiver**
(`func (rs RSDic) Select1(...)`), which copied a ~104-byte struct
(5 slice headers + 7 scalars) on every call. Under load this showed up
as `runtime.duffcopy` taking ~9% of total time.

**After:**

The inner search is bracketed by `selectOneInds[selectInd]` and
`selectOneInds[selectInd+1]+1`. Below a threshold the bracket is
walked linearly (preserving upstream behaviour on the typical case);
above it, a binary search bounds the worst case at $O(\log\,4096) \le
12$ iterations regardless of clustering. Receiver changed to pointer.

```go
const kSelectLinearThreshold = 128

func (rs *RSDic) Select1(rank uint64) uint64 {
    ...
    selectInd := rank / kSelectBlockSize
    lo := rs.selectOneInds[selectInd]
    hi := /* selectOneInds[selectInd+1]+1, clamped */

    if hi-lo <= kSelectLinearThreshold {
        // small bracket: linear scan, branch predictor friendly
        for lblock = lo; lblock < hi; lblock++ {
            if rank < rs.rankBlocks[lblock] { break }
        }
        lblock--
    } else {
        // large bracket: binary search bounds the worst case
        l, r := lo, hi
        for l < r {
            mid := l + (r-l)/2
            if rs.rankBlocks[mid] <= rank { l = mid + 1 } else { r = mid }
        }
        lblock = l - 1
    }
    ...
}
```

**Picking the threshold.** A `BenchmarkSelect1ThresholdSweep` (Apple
M4 Max, 2 s/cell) compared Linear, Binary, and Adaptive across
brackets $\{1, 4, 16, 64, 128, 256, 1024, 4096\}$ on a deterministic
clustered-unary pattern:

| bracket | Linear (ns) | Binary (ns) | Adaptive (ns) | Winner |
|--------:|------------:|------------:|--------------:|--------|
|       1 |        51.9 |        56.6 |          52.9 | Linear |
|      16 |        32.0 |        40.6 |          31.7 | Linear |
|     128 |        74.7 |        80.1 |          78.1 | Linear |
|     256 |       100.2 |        89.7 |          89.7 | Binary |
|    1024 |       218.7 |       101.4 |          97.6 | Binary |
|    4096 |       633.5 |       112.1 |         113.9 | Binary |

Crossover sits between 128 and 256. We pick `kSelectLinearThreshold =
128` — keeps the upstream regime intact and engages Binary slightly
before it strictly wins, since the Linear pathology grows
superlinearly past the crossover.

A separate `BenchmarkSelect1MixedBrackets` / `…AlternatingPattern`
checks that the `if hi-lo <= threshold` branch is well predicted when
queries interleave small- and large-bracket rsdics; on M4 Max the
branch is free.

**Correctness.** `TestSelect1ThresholdEquivalence` cross-checks the
Linear, Binary, and production Adaptive variants for full agreement
across all ranks, on both uniform-density and clustered-unary
patterns, for thresholds $\{1, 4, 8, 16, 32, 64, 256, 1024\}$.

**Measured impact.**

- Microbench, single Select on a clustered bitvector, replaying a
  real small-rank stream that exercised the worst case:
  **1825 ns → ~80 ns** (~23×).
- A consumer of this rsdic that issues two `Select1` calls per query
  on such a bitvector dropped from 4391 ns/op to 374 ns/op end-to-end
  (~11.7×), with no change in any other behaviour.
- No regression on the uniform-density baseline used in §"Benchmark
  results" above: bracket sizes there sit in the small-bracket regime
  and take the linear path.

### Optimization 3: `runZerosRaw` → `bits.TrailingZeros64` (`enumCode.go`)

**Before:**

```go
func runZerosRaw(code uint64, pos uint8) uint8 {
i := uint8(pos)
for; i < kSmallBlockSize && !getBit(code, i); i++ {
}
return i - pos
}
```

Bit-by-bit scan from `pos` looking for the first set bit — up to 64 iterations with branch per bit.

**After:**

```go
func runZerosRaw(code uint64, pos uint8) uint8 {
shifted := code >> pos
if shifted == 0 {
return kSmallBlockSize - pos
}
return uint8(bits.TrailingZeros64(shifted))
}
```

Single shift + `TrailingZeros64` (`TZCNT` on x86 / `RBIT+CLZ` on ARM). O(1) vs O(64).

## Remaining optimization opportunities (not yet implemented)

1. **Small-block pointer accumulation loop** in Rank/Bit: iterates up to 15 small blocks per
   large block. Could be eliminated by storing cumulative pointers per small block (+8 bytes/64 bits
   = 12.5% space overhead). Decided against: +2.5 bpk for ~5% of ERE memory.

2. **Enum decode path** (sparse/dense blocks, rankSB outside 15-49): `enumRank`, `enumBit`,
   `enumSelect1` all do O(pos) sequential decode through combinatorial number system.
   Would require changing data layout to fix (e.g., always store raw 64-bit blocks).

## Density in ERE context

- **D1** (block occupancy): ~50% ones — half of 2^k blocks occupied (uniform random keys)
- **D2** (block sizes): ~33% ones — n/2 block delimiters among 3n/2 total bits

Both hit the raw fast path most of the time. Sparse enum path matters for clustered distributions.
