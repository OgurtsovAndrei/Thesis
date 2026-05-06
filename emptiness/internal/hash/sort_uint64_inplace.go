package hash

import (
	mbits "math/bits"
	"slices"
)

// In-place sort+dedup variants for uint64 keys in [0, 2^K).
//
// Both functions here keep the input array as the only large buffer:
//
//   SortAndDedupUint64AmericanFlag — pure MSD radix, ⌈K/8⌉ levels, no
//      heap scratch beyond the 256-entry count/head arrays per recursion.
//
//   SortAndDedupUint64MSDBitmap    — one in-place MSD partition by the
//      top 8 bits (American Flag), then per-bucket bitmap dedup using a
//      *single* reused bitmap of 2^(K-8)/8 bytes.
//
// Compared to LSD radix (radixK), which doubles peak memory, both
// variants stay close to 1× input memory. MSDBitmap additionally folds
// dedup into the per-bucket pass, eliminating a second linear sweep.

// americanFlagThreshold is the cutoff below which a recursive radix call
// switches to slices.Sort. 32 keys is a typical pdqsort sweet spot.
const americanFlagThreshold = 32

// SortAndDedupUint64AmericanFlag sorts and dedups in place using K-aware
// MSD radix. Caller guarantees keys ∈ [0, 2^K).
//
// Memory: only stack + 256-entry count/head arrays per recursion (≈ 8 KiB
// max for K=64). The input slice is permuted in place; no scratch buffer
// of size n is allocated.
func SortAndDedupUint64AmericanFlag(keys []uint64, K uint32) []uint64 {
	if len(keys) <= 1 {
		return keys
	}
	if K == 0 {
		return dedupSortedUint64(keys)
	}
	if K > 64 {
		K = 64
	}
	passes := int((K + 7) / 8)
	topShift := uint((passes - 1) * 8)
	americanFlag(keys, topShift)
	return dedupSortedUint64(keys)
}

// americanFlag does an in-place MSD radix sort by the byte at the given
// shift, then recurses with shift-8 on each non-empty bucket. The input
// is permuted in place using cycle-based bucket walking (American Flag
// Sort, Cormen et al. and McIlroy et al. 1993).
func americanFlag(a []uint64, shift uint) {
	if len(a) < americanFlagThreshold {
		slices.Sort(a)
		return
	}

	var count [256]int
	for _, x := range a {
		count[byte(x>>shift)]++
	}

	// Bucket boundaries: bucketEnd[b] is the exclusive end of bucket b.
	// head[b] is the next free slot in bucket b.
	var bucketEnd [256]int
	var head [256]int
	sum := 0
	for i := 0; i < 256; i++ {
		head[i] = sum
		sum += count[i]
		bucketEnd[i] = sum
	}

	// In-place permutation: for each bucket b, fill it by cycling x
	// through its destination buckets until a member of b is recovered.
	for b := 0; b < 256; b++ {
		for head[b] < bucketEnd[b] {
			x := a[head[b]]
			d := int(byte(x >> shift))
			for d != b {
				tmp := a[head[d]]
				a[head[d]] = x
				head[d]++
				x = tmp
				d = int(byte(x >> shift))
			}
			a[head[b]] = x
			head[b]++
		}
	}

	if shift == 0 {
		return
	}
	nextShift := shift - 8
	start := 0
	for b := 0; b < 256; b++ {
		end := bucketEnd[b]
		if end-start > 1 {
			americanFlag(a[start:end], nextShift)
		}
		start = end
	}
}

// SortAndDedupUint64MSDBitmap performs one in-place 8-bit MSD partition
// (American Flag) by the top 8 bits of K, then deduplicates each bucket
// with a single reused bitmap of size 2^(K-8) bits = 2^(K-11) bytes.
//
// Caller guarantees keys ∈ [0, 2^K). For K ≤ 16 this falls back to
// SortAndDedupUint64Bitmap (the partition overhead is not amortized).
//
// Memory: 2^(K-11) bytes for the inner bitmap (reused across all 256
// buckets) plus 256-entry count array. The keys slice is permuted in
// place. For K=40 the inner bitmap is 512 MiB — still 4× less than
// radixK's 2 GiB scratch at n=2²⁸.
func SortAndDedupUint64MSDBitmap(keys []uint64, K uint32) []uint64 {
	n := len(keys)
	if n == 0 {
		return keys
	}
	if K == 0 {
		return dedupSortedUint64(keys)
	}
	if K > 64 {
		K = 64
	}
	if K <= 16 {
		return SortAndDedupUint64Bitmap(keys, K)
	}
	if K > 40 {
		// 2^(K-8) bits would exceed 256 GiB — fall back to fully in-place
		// MSD radix instead.
		return SortAndDedupUint64AmericanFlag(keys, K)
	}

	const bucketBits = uint32(8)
	innerBits := K - bucketBits
	shift := uint(innerBits)

	// Phase 1: count by top 8 bits.
	var count [256]int
	for _, x := range keys {
		count[byte(x>>shift)]++
	}

	// Phase 2: in-place American Flag partition by top 8 bits.
	var bucketEnd [256]int
	var head [256]int
	sum := 0
	for i := 0; i < 256; i++ {
		head[i] = sum
		sum += count[i]
		bucketEnd[i] = sum
	}
	for b := 0; b < 256; b++ {
		for head[b] < bucketEnd[b] {
			x := keys[head[b]]
			d := int(byte(x >> shift))
			for d != b {
				tmp := keys[head[d]]
				keys[head[d]] = x
				head[d]++
				x = tmp
				d = int(byte(x >> shift))
			}
			keys[head[b]] = x
			head[b]++
		}
	}

	// Phase 3: per-bucket bitmap dedup, compact unique into keys[:out].
	innerSize := uint64(1) << innerBits
	innerMask := innerSize - 1
	bmWords := (innerSize + 63) >> 6
	bm := make([]uint64, bmWords)

	// Extraction below clears each bm word as it walks (bm[i]=0 before
	// pulling its set bits out), so the bitmap is implicitly reset for
	// the next bucket. This avoids a 2^(K-8)/64-word wipe per bucket
	// (256 × that cost would dominate at K ≥ 36).

	out := 0
	bucketStart := 0
	for b := 0; b < 256; b++ {
		bucketSize := count[b]
		if bucketSize == 0 {
			continue
		}

		// Dispatch by per-bucket density. If the bucket is too sparse for
		// a bitmap pass to amortize (bucketSize × 64 < bmWords ⇒ scanning
		// the bitmap is more work than sort+dedup), fall back to slices.Sort
		// inside this bucket. This handles small-n / large-K cases where
		// each bucket has only a handful of keys against a multi-MiB
		// bitmap.
		if uint64(bucketSize)*64 < bmWords {
			sub := keys[bucketStart : bucketStart+bucketSize]
			slices.Sort(sub)
			prev := uint64(0)
			hasFirst := false
			for _, x := range sub {
				if !hasFirst || x != prev {
					keys[out] = x
					out++
					prev = x
					hasFirst = true
				}
			}
			bucketStart += bucketSize
			continue
		}

		for i := 0; i < bucketSize; i++ {
			v := keys[bucketStart+i] & innerMask
			bm[v>>6] |= uint64(1) << (v & 63)
		}

		prefix := uint64(b) << innerBits
		for i := range bm {
			w := bm[i]
			if w == 0 {
				continue
			}
			bm[i] = 0
			base := uint64(i) << 6
			for w != 0 {
				bit := mbits.TrailingZeros64(w)
				keys[out] = prefix | (base + uint64(bit))
				out++
				w &= w - 1
			}
		}

		bucketStart += bucketSize
	}
	return keys[:out]
}

// SortAndDedupUint64Adaptive picks the best implementation by K, chosen
// from the benchmark Pareto front:
//
//	K ≤ 28          → SortAndDedupUint64Bitmap        (≤ 32 MiB scratch, fastest)
//	29 ≤ K ≤ 36     → SortAndDedupUint64MSDBitmap     (≤ 32 MiB scratch, in-place keys)
//	K ≥ 37          → SortAndDedupUint64AmericanFlag  (zero scratch, in-place keys)
//
// This is the recommended entry point for the SODA build path: it
// avoids the 8n scratch of LSD radix while staying within ~2× of the
// fastest variant at every K.
func SortAndDedupUint64Adaptive(keys []uint64, K uint32) []uint64 {
	if len(keys) == 0 {
		return keys
	}
	switch {
	case K == 0:
		return dedupSortedUint64(keys)
	case K <= 28:
		return SortAndDedupUint64Bitmap(keys, K)
	case K <= 36:
		return SortAndDedupUint64MSDBitmap(keys, K)
	default:
		return SortAndDedupUint64AmericanFlag(keys, K)
	}
}
