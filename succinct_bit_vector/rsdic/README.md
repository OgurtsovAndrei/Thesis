# RSDic Fork

Forked from [github.com/hillbig/rsdic](https://github.com/hillbig/rsdic) (MIT License, Daisuke Okanohara).

## Changes from upstream

- Removed `github.com/ugorji/go/codec` dependency — MarshalBinary/UnmarshalBinary reimplemented with `encoding/binary`
- Removed `github.com/smartystreets/goconvey` dependency — tests rewritten with standard `testing`
- No algorithmic changes yet — this is a baseline fork for optimization

## Baseline benchmarks

Apple M4 Max, Go 1.25, GOMAXPROCS=1, 5 runs. Compared against upstream `github.com/hillbig/rsdic`:

| Op | Density | Upstream (ns/op) | Fork (ns/op) |
|---|---|---|---|
| Bit | 50% | 23.5 | 23.7 |
| Rank | 50% | 25.6 | 25.3 |
| Select | 50% | 127 | 125 |
| Bit | 33% | 24.4 | 25.0 |
| Rank | 33% | 27.6 | 26.6 |
| Select | 33% | 116 | 111 |
| Bit | 1% | 55.6 | 56.5 |
| Rank | 1% | 57.5 | 56.8 |
| Select | 1% | 122 | 123 |

Fork is identical to upstream within noise — confirms clean copy.

## Optimization targets

CPU profiling of ERE queries shows ~80% time in rsdic Rank/Select/Bit. Key bottlenecks:

1. **`popCount`**: software bit-parallel implementation (12 ops). Replace with `math/bits.OnesCount64` (1 hardware POPCNT instruction).
2. **`selectRaw`**: bit-by-bit scan through uint64 (up to 64 iterations). Replace with `bits.TrailingZeros64` + clear-lowest-bit.
3. **`enumRank`/`enumBit`/`enumSelect1`**: O(pos) sequential decode through combinatorial number system for sparse/dense blocks (rankSB outside 15-49).
4. **Small-block accumulation loop**: up to 15 iterations in Rank/Bit/Select to compute pointer offset within large block.

## Density in ERE context

- **D1** (block occupancy): ~50% ones (half of 2^k blocks are occupied with uniform random keys)
- **D2** (block sizes): ~33% ones (n/2 block delimiters among 3n/2 total bits)

Both hit the "raw" fast path (rankSB 15-49) most of the time, bypassing enum coding. Sparse path (1%) matters for clustered distributions.
