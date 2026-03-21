# Remaining Optimization Opportunities

Found via deep research audit of bit manipulation across the codebase.

## rsdic (build-time)

### `enumEncode` — iterate only set bits

**File:** `enumCode.go:5`

Current: always 64 iterations, checking each bit with `getBit()`.
Fix: iterate only set bits via `TrailingZeros64` + clear-lowest-bit.
Drops from 64 iterations to `popcount(val)` (avg ~32 for random data, much less for sparse).

```go
// Current
for i := uint8(0); i < kSmallBlockSize; i++ {
    if getBit(val, i) {
        code += kCombinationTable64[kSmallBlockSize-i-1][rankSB]
        rankSB--
    }
}

// Proposed
v := val
for v != 0 {
    i := uint8(bits.TrailingZeros64(v))
    code += kCombinationTable64[kSmallBlockSize-i-1][remaining]
    remaining--
    v &= v - 1
}
```

**Impact:** Build-time only, N/64 calls per N-bit bitvector. Medium priority.

### `enumDecode` — early exit when rankSB == 0

**File:** `enumCode.go:19`

Current: always 64 iterations. Once all set bits are found (`rankSB == 0`), remaining iterations do nothing.
Fix: add `if rankSB == 0 { break }`.

**Impact:** Build-time / query-time (block decode). Low priority — only helps for sparse blocks.

## bits package (x86 target)

### `BitString.Sub` / `BitString.Add` — triple `Reverse64` per word

**File:** `Thesis/bits/bit_string.go:39`

Each word does 3× `bits.Reverse64` (reverse both operands + reverse result). On ARM64 this is `RBIT` (1 cycle each), but on x86-64 it's ~6 instructions each = ~18 instructions per word.

For single-word BitStrings (60-bit keys, the common ERE case), could special-case to avoid redundant reverses.

**Impact:** Called in `normalizeToK` (TruncARE query path), Soda hash query path. Medium priority on x86, low on ARM.
