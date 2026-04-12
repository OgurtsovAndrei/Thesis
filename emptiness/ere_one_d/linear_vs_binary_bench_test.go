package ere_one_d

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"testing"
)

func BenchmarkBucketSearch_LinearVsBinary(b *testing.B) {
	bucketSizes := []int{2, 4, 8, 12, 16, 24, 32, 48, 64, 128, 256}
	wBits := []uint32{8, 11, 16, 20}

	for _, w := range wBits {
		for _, sz := range bucketSizes {
			if sz > (1 << w) {
				continue
			}

			suffixes := genSortedSuffixes(sz, w)
			packed := packUint64Local(suffixes, int(w))

			mask := uint64((1 << w) - 1)
			// query that hits middle of range
			minSuff := suffixes[sz/4]
			maxSuff := suffixes[sz*3/4]

			ere := &ExactRangeEmptiness{
				packedData: packed,
				w:          w,
			}

			b.Run(fmt.Sprintf("w=%d/n=%d/binary", w, sz), func(b *testing.B) {
				_ = mask
				for i := 0; i < b.N; i++ {
					ere.isRangeEmptyInBlock(0, sz, minSuff, maxSuff)
				}
			})

			b.Run(fmt.Sprintf("w=%d/n=%d/linear", w, sz), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					ere.isRangeEmptyInBlockLinear(0, sz, minSuff, maxSuff)
				}
			})
		}
	}
}

func BenchmarkBucketSearch_WorstCase(b *testing.B) {
	// Worst case: very large buckets, as if all keys landed in one block.
	// w=12 (BPK~15), w=15, w=17
	cases := []struct {
		w  uint32
		sz int
	}{
		{12, 1 << 10}, // 1024
		{12, 1 << 12}, // 4096 (full bucket at w=12)
		{15, 1 << 12}, // 4096
		{15, 1 << 15}, // 32768 (full bucket at w=15)
		{17, 1 << 12}, // 4096
		{17, 1 << 15}, // 32768
		{17, 1 << 17}, // 131072 (full bucket at w=17)
	}

	for _, c := range cases {
		suffixes := genSortedSuffixes(c.sz, c.w)
		packed := packUint64Local(suffixes, int(c.w))
		minSuff := suffixes[c.sz/4]
		maxSuff := suffixes[c.sz*3/4]

		ere := &ExactRangeEmptiness{
			packedData: packed,
			w:          c.w,
		}

		b.Run(fmt.Sprintf("w=%d/n=%d/binary", c.w, c.sz), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ere.isRangeEmptyInBlock(0, c.sz, minSuff, maxSuff)
			}
		})
	}
}

func genSortedSuffixes(n int, w uint32) []uint64 {
	mask := uint64((1 << w) - 1)
	seen := make(map[uint64]bool, n)
	suffixes := make([]uint64, 0, n)
	for len(suffixes) < n {
		v := rand.Uint64() & mask
		if !seen[v] {
			seen[v] = true
			suffixes = append(suffixes, v)
		}
	}
	sort.Slice(suffixes, func(i, j int) bool { return suffixes[i] < suffixes[j] })
	return suffixes
}
